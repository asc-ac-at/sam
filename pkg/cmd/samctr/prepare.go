// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"bytes"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	isamctr "github.com/asc-ac-at/sam/internal/samctr"
	"github.com/asc-ac-at/sam/pkg/subproc"
)

// basically, we need to create a list of fusemounts that appeared in the
// config file (these are the ro fusemounts)
//
//	  then we process the writeable-repository flag
//	  on finding writeable repository, we should:
//
//	a) copy the "ro" fusemount (it must exist, else err)
//	b) create a new FuseMount "fmNew" from the existing "fm":
//	        fmNew.Type = fm.Type
//	        fmNew.Cmd  = fuse-overlayfs | unionfs   (which ?)
//	        fmNew.Args = args formatted from the helper
//	        fmNew.CtrMountpoint = fm.CtrMountpoint
//	c) update the CtrMountpoint of fm
//	        fm.CtrMountpoint = "/cvmfs_ro"
func PrepareFuseMounts(config *Config) ([]isamctr.FuseMount, error) {
	result := []isamctr.FuseMount{}

	// initially set up the read only fusemounts
	for i := range config.FuseMounts {
		fm := &isamctr.FuseMount{
			Type:          config.FuseMounts[i].Type,
			FuseCmd:       config.FuseMounts[i].FuseCmd,
			FuseArg:       config.FuseMounts[i].FuseArg,
			CtrMountpoint: config.FuseMounts[i].CtrMountpoint,
		}
		result = append(result, *fm)
	}
	// then process the rw mounts
	ro_default := "/cvmfs_ro"
	for _, r := range config.WriteableRepos {
		for i := range result {
			rofm := result[i]
			if r == rofm.FuseArg {
				// make it writeable, use the implementation type from config
				newMountpoint := filepath.Clean(fmt.Sprintf("/cvmfs/%s", strings.TrimPrefix(rofm.CtrMountpoint, ro_default)))
				rwfm := &isamctr.FuseMount{
					Type:          rofm.Type,
					FuseCmd:       config.FuseCmdRW,
					FuseArg:       rofm.FuseArg,
					CtrMountpoint: newMountpoint,
				}
				result = append(result, *rwfm)
			}

			// remember to update the CtrMountpoint of the rofm
			rofm.CtrMountpoint = ro_default
		}
	}
	return result, nil
}

// parse config for host injections,
func parseHostInjections(c *Config) string {
	result := ""
	if HostInjections != DefaultHostInjections { // user specified a flag
		result = HostInjections
	} else if c.HostInjections != "" { // check config
		result = c.HostInjections
	} else { // use default
		result = DefaultHostInjections
	}
	slog.Debug("parsed host injections", "path", result)
	return result
}

var apptainerCmdOpts []string

// Intended to be used as a cobra PreRunE for subcommands that require the
// container SIF to be available
func PrepareContainerPreRun(cmd *cobra.Command, args []string) error {

	// setup fusemounts (try config, then optional --writeable-repository flag)
	fm, fm_err := PrepareFuseMounts(AppConfig)
	if fm_err != nil {
		return fmt.Errorf("prepare fusemount %w", fm_err)
	}

	hostInjections := parseHostInjections(AppConfig)

	// try to determine rootTmpDir name (resume flag has precedence)
	rootTmpDir := ResumePath
	// if rootTmpDir is "", then use the following prefix to create a random
	// dir during the flow of SetupStorage
	rootTmpDirPrefix := RootTmpDirPrefix

	// Prepare options for SetupStorage (note: we intentionally do not write back to AppConfig)
	opts := isamctr.StorageOptions{
		RootTmpDir:           rootTmpDir,
		RootTmpDirPrefix:     rootTmpDirPrefix,
		HostInjections:       hostInjections,
		ApptainerVarHome:     AppConfig.ApptainerVarHome,
		ApptainerVarCacheDir: AppConfig.ApptainerVarCachedir,
		FuseMounts:           fm,
	}

	// call setup storage, here we create rootTmpDir if needed
	state, err := isamctr.SetupStorage(opts)
	if err != nil {
		return fmt.Errorf("storage setup failed: %w", err)
	}

	// parse bind mounts coming from config
	var configBinds []isamctr.BindMount
	for _, spec := range AppConfig.BindPaths {
		if strings.TrimSpace(spec) == "" { // nothing to do
			continue
		}
		b, perr := isamctr.ParseBindSpec(spec)
		if perr != nil {
			return fmt.Errorf("invalid bind path in config %q: %w", spec, perr)
		}
		configBinds = append(configBinds, b)
	}

	// parse bind mounts coming from CLI provided flags
	// "-b" or "--extra-bind-paths" flags
	var cliBinds []isamctr.BindMount
	if strings.TrimSpace(ExtraBindPaths) != "" {
		parts := splitCommaList(ExtraBindPaths)
		for _, p := range parts {
			b, perr := isamctr.ParseBindSpec(p)
			if perr != nil {
				return fmt.Errorf("invalid bind path from CLI %q: %w", p, perr)
			}
			cliBinds = append(cliBinds, b)
		}
	}

	var apptainerCmdOpts []string

	// hermetic container env: pass --cleanenv when requested via CLI or config
	// (build jobs go through sami → exec; interactive shells stay permissive
	// unless explicitly requested)
	if CleanEnv || AppConfig.CleanEnv {
		apptainerCmdOpts = append(apptainerCmdOpts, "--cleanenv")
	}

	// nvidia setup (optional)
	// check for nvidia-smi, if present:
	//  + setup nvidia flag for apptainer
	//  + setup bind mount
	var nvidiaBinds []isamctr.BindMount

	cfg := subproc.New([]string{"which", "nvidia-smi"})
	cfg.Timeout = 3 * time.Second
	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr
	if err := cfg.Run(); err != nil {
		return fmt.Errorf(`which nvidia-smi failed: %q`, err)
	}
	nvidiaSmiPath := strings.TrimSpace(stdout.String())

	nvidiaFlag := "--nv"
	apptainerCmdOpts = append(apptainerCmdOpts, nvidiaFlag)
	// which returns a linebreak
	nvSafePath := strings.TrimSuffix(nvidiaSmiPath, "\n")
	nvBm := isamctr.NewBindMount(nvSafePath, nvSafePath, "ro")
	nvidiaBinds = append(nvidiaBinds, *nvBm)

	// Merge bind paths
	allBinds := make([]isamctr.BindMount, 0, len(configBinds)+len(cliBinds)+len(state.BindMounts)+len(nvidiaBinds))
	allBinds = append(allBinds, configBinds...)
	allBinds = append(allBinds, cliBinds...)
	allBinds = append(allBinds, state.BindMounts...)
	allBinds = append(allBinds, nvidiaBinds...)

	// at this point rootTmpDir has been created, storageState generated
	runtime := &RuntimeState{
		Storage:          state,
		ContainerSif:     "",
		AllBindMounts:    allBinds,
		Environ:          append([]string{}, state.RuntimeEnv...),
		ApptainerCmdOpts: append([]string{}, apptainerCmdOpts...),
	}

	// Legacy --nv path (use nvidia-container-cli = no): apptainer's
	// action-script helper set_default_ld_library_path() prepends the
	// container's own ldconfig dirs (/lib:/lib64) whenever LD_LIBRARY_PATH
	// equals the default "/.singularity.d/libs". Those resolve to the
	// image's native glibc, which may be older than what EESSI compat-layer
	// binaries require (LD_LIBRARY_PATH is searched before RUNPATH) --
	// symptom: lua5.1: /lib64/libm.so.6: version `GLIBC_2.38' not found.
	//
	// Setting the env-provided value to anything NOT exactly the default
	// suppresses that prepend; apptainer then appends :/.singularity.d/libs
	// itself (process_linux.go injectEnvHandler), so the *doubled*
	// "/.singularity.d/libs:/.singularity.d/libs" is expected and harmless.
	// Do NOT "fix" the doubling: an empty value instead suppresses the
	// libs-dir append entirely, and the exact default brings back the
	// /lib:/lib64 prepend. Only the injected GPU libs dir may precede
	// RUNPATH resolution.
	if nvidiaFlag != "" {
		runtime.Environ = append(runtime.Environ,
			"APPTAINERENV_LD_LIBRARY_PATH=/.singularity.d/libs")
	}

	// 8) setup container
	image := Image
	if image == "" {
		image = AppConfig.Image
	}
	if image == "" {
		return fmt.Errorf("no container image specified (flag or config)")
	}
	runtime.Image = image
	runtime, err = CtrSetup(runtime, PullRunner)
	if err != nil {
		return fmt.Errorf("ctr setup failed %w", err)
	}

	// 9) save runtime state for use by Run/RunE
	Runtime = runtime
	return nil
}

// splitCommaList splits a comma-separated list while trimming whitespace and ignoring empty parts.
func splitCommaList(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
