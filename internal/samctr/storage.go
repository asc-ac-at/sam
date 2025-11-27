// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Helper to make sure path components become relative to rootTmpDir
func safeRel(p string) string {
	if p == "" {
		return ""
	}
	// normalize slashes
	p = filepath.Clean(p)
	// if p is absolute path, strip leading "/"
	p = strings.TrimLeft(p, string(os.PathSeparator))
	return p
}

type StorageOptions struct {
	// if RootTmpDir is empty, SetupStorage will create one
	RootTmpDir string

	RootTmpDirPrefix string

	// eessi's host injetions path on host
	HostInjections string

	// Apptainer specific variables
	ApptainerVarHome     string
	ApptainerVarCacheDir string

	// Carries info about the mounted (cvmfs) repositories
	// Note that this is the format passed by the user config
	FuseMounts []FuseMount
}

// Returned by SetupStorage and includes computed/runtime values
type StorageState struct {
	RootTmpDir           string
	ApptainerVarHome     string
	ApptainerVarCacheDir string
	BindMounts           []BindMount
	WriteableRepos       []string
	FuseMounts           []FuseMount
	RuntimeEnv           []string
}

// create CVMFS repo dirs from the Fusemounts and overlay dirs from
// perform a copy-on-write before creating any dirs (to avoid changing config)
func (opts *StorageOptions) setupStorageFuseMounts() error {
	if opts.FuseMounts != nil {
		for i := range opts.FuseMounts {
			log.Printf("setupStorageFuseMounts: %s %s %s %s\n",
				opts.FuseMounts[i].Type,
				opts.FuseMounts[i].FuseCmd,
				opts.FuseMounts[i].FuseArg,
				opts.FuseMounts[i].CtrMountpoint,
			)

			switch opts.FuseMounts[i].FuseCmd {
			case "unionfs":
				fallthrough
			case "fuse-overlayfs":
				// use a relative path component derived from FuseArg
				repoCandidate := safeRel(opts.FuseMounts[i].FuseArg)
				if repoCandidate == "" { // fall back to safe base name
					repoCandidate = safeRel(filepath.Base(opts.FuseMounts[i].CtrMountpoint))
				}
				if repoCandidate == "" {
					return fmt.Errorf("cannot determine repository name for fuse-overlayfs entry")
				}
				upper := filepath.Join(opts.RootTmpDir, repoCandidate, "overlay-upper")
				work := filepath.Join(opts.RootTmpDir, repoCandidate, "overlay-work")
				dirs := []string{upper, work}
				for i := range dirs {
					log.Printf("setupStorageFuseMounts mkdir: %s", dirs[i])
					d_err := os.MkdirAll(dirs[i], 0o755)
					if d_err != nil {
						log.Printf("failed to created fuse overlay %s", dirs[i])
						log.Println(d_err)
					}
				}
			case "cvmfs2":
				fallthrough
			default:
				rel := safeRel(opts.FuseMounts[i].FuseArg)
				if rel == "" {
					rel = safeRel(filepath.Base(opts.FuseMounts[i].CtrMountpoint))
				}
				d := filepath.Join(opts.RootTmpDir, rel)
				d_err := os.MkdirAll(d, 0o755)
				if d_err != nil {
					// failing to set up the basis is enough reason to bail out
					// at this point
					return fmt.Errorf("mkdir %s: %w", opts.FuseMounts[i].FuseArg, d_err)
				}
				log.Printf("storage fusemount cvmfs2 (default) created: %s", d)
			}
		}
	}
	return nil
}

// SetupStorage configures all necessary host-side preparations for container
// storage. It them prepares a state object that can be used to derive flags to
// pass to apptainer.
// It will:
// + pick or create a root temp directory (contains the rest of the hierarchy)
// + ensure that vars APPTAINER_HOME, APPTAINER_CACHE_DIR exist
// set up common vars and directories
//
//	directory structure should be:
//	  ${ROOT_TMP_DIR}
//	  |-singularity_cache
//	  |-home
//	  |-repos_cfg
//	  |-${CVMFS_VAR_LIB}
//	  |-${CVMFS_VAR_RUN}
//	  |-CVMFS_REPO_1
//	  |   |-repo_settings.sh (name, id, access, host_injections)
//	  |   |-overlay-upper
//	  |   |-overlay-work
//	  |   |-opt-eessi (unless otherwise specificed for host_injections)
//	  |-CVMFS_REPO_n
//	      |-repo_settings.sh (name, id, access, host_injections)
//	      |-overlay-upper
//	      |-overlay-work
//	      |-opt-eessi (unless otherwise specificed for host_injections)
//
// TODO: repos_cfg
func SetupStorage(opts StorageOptions) (*StorageState, error) {
	state := &StorageState{
		BindMounts: []BindMount{},
	}

	// 1) root tmpdir
	rootTmpDir := opts.RootTmpDir
	var err error
	if rootTmpDir == "" {
		prefix := opts.RootTmpDirPrefix
		if prefix == "" {
			// sensible default
			prefix = "sam."
		}
		rootTmpDir, err = os.MkdirTemp("", prefix)
		if err != nil {
			return nil, fmt.Errorf("create root tmpdir: %w", err)
		}
	}
	// make sure it exists (in case caller provided a path)
	if err = os.MkdirAll(rootTmpDir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure tmpdir exists: %w", err)
	}
	state.RootTmpDir = rootTmpDir
	opts.RootTmpDir = state.RootTmpDir
	// then make sure to mount this globally as "/tmp" in the container
	state.BindMounts = append(state.BindMounts, *NewBindMount(rootTmpDir, "/tmp", "rw"))

	// Update the fusemounts and create any required files
	f_err := opts.setupStorageFuseMounts()
	if f_err != nil {
		return nil, f_err
	}

	// export the updated fusemount state
	state.FuseMounts = opts.FuseMounts

	// 3) APPTAINER_HOME env var
	if opts.ApptainerVarHome != "" {
		state.ApptainerVarHome = opts.ApptainerVarHome
	} else {
		hostHome := filepath.Join(rootTmpDir, "home")
		if err := os.MkdirAll(hostHome, 0o755); err != nil {
			return nil, fmt.Errorf("create apptainer host home failed: %w", err)
		}
		// Prefer USER env, fall back to current user lookup
		userName := os.Getenv("USER")
		if userName == "" {
			if u, uerr := user.Current(); uerr == nil {
				userName = u.Username
			} else {
				return nil, fmt.Errorf("failed to determine user home")
			}
		}
		ctr := fmt.Sprintf("/home/%s", userName)
		state.ApptainerVarHome = fmt.Sprintf("%s:%s", hostHome, ctr)
		// also register this as a rw bind mount
		state.BindMounts = append(state.BindMounts, *NewBindMount(hostHome, ctr, "rw"))
	}
	state.RuntimeEnv = append(state.RuntimeEnv, fmt.Sprintf("APPTAINER_HOME=%s", state.ApptainerVarHome))

	// 4) APPTAINER_CACHE_DIR
	if opts.ApptainerVarCacheDir != "" {
		state.ApptainerVarCacheDir = opts.ApptainerVarCacheDir
	} else {
		cacheDir := filepath.Join(rootTmpDir, "apptainer_cache")
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return nil, fmt.Errorf("create apptainer cache dir failed: %w", err)
		}
		state.ApptainerVarCacheDir = cacheDir
	}
	state.RuntimeEnv = append(state.RuntimeEnv, fmt.Sprintf("APPTAINER_CACHEDIR=%s", state.ApptainerVarCacheDir))

	// 5) host_injections ->
	// if provided by user config, take that (and register bind),
	// otherwise create /opt/eessi under tmpRoot and use that
	ctrPath := "/opt/eessi"
	if opts.HostInjections != "" {
		hostPath := opts.HostInjections
		if _, err := os.Stat(hostPath); err != nil {
			return nil, fmt.Errorf("host injections path %s does not exist: %w", hostPath, err)
		}
		state.BindMounts = append(state.BindMounts, *NewBindMount(hostPath, ctrPath, "rw"))
	} else {
		hi := filepath.Join(rootTmpDir, "opt-eessi")
		if err := os.MkdirAll(hi, 0o755); err != nil {
			return nil, fmt.Errorf("create host injections dir %s: %w", hi, err)
		}
		// derive container path as relative portion under rootTmpDir
		rel := strings.TrimPrefix(hi, rootTmpDir)
		rel = strings.TrimPrefix(rel, string(os.PathSeparator))
		ctrPath := "/" + rel
		state.BindMounts = append(state.BindMounts, *NewBindMount(hi, ctrPath, "rw"))
	}

	// 6) Create var-lib/run cvmfs dirs and register binds
	for _, part := range []string{"lib", "run"} {
		hostPath := filepath.Join(rootTmpDir, fmt.Sprintf("var-%s-cvmfs", part))
		if err := os.MkdirAll(hostPath, 0o755); err != nil {
			return nil, fmt.Errorf("create cvmfs dir %s: %w", hostPath, err)
		}
		// normalize container path "var/lib/cvmfs/" "var/run/cvmfs"
		normalized := strings.ReplaceAll(filepath.Base(hostPath), "-", "/")
		// TODO: these should be added to singularity bind
		ctrPath := "/" + normalized
		state.BindMounts = append(state.BindMounts, *NewBindMount(hostPath, ctrPath, "rw"))
	}
	// TODO: create comma seperatd list of BindMounts and use in APPTAINER_BINDPATH
	return state, nil
}
