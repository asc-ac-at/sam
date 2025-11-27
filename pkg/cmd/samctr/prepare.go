// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	isamctr "github.com/asc-ac-at/sam/internal/samctr"
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
	log.Printf("parse host injections: %s\n", result)
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

	// nvidia setup (optional)
	// check for nvidia-smi, if present:
	//  + setup nvidia flag for apptainer
	//  + setup bind mount
	var nvidiaBinds []isamctr.BindMount
	nvidiaSmiPath, n_err := IoRunner("which", "nvidia-smi")
	nvidiaFlag := ""
	if n_err != nil { // setup nvidia
		return fmt.Errorf("failed to find host nvidia: %w", n_err)
	} else {
		nvidiaFlag = "--nv"
		apptainerCmdOpts = append(apptainerCmdOpts, nvidiaFlag)
		// which returns a linebreak
		nvSafePath := strings.TrimSuffix(nvidiaSmiPath, "\n")
		nvBm := isamctr.NewBindMount(nvSafePath, nvSafePath, "ro")
		nvidiaBinds = append(nvidiaBinds, *nvBm)
	}

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
