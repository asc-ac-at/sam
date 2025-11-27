// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"github.com/spf13/cobra"
)

var (
	cfgFile               string
	FuseCmdRW             string
	HostInjections        string
	Image                 string
	ExtraBindPaths        string
	Nvidia                string
	ResumePath            string
	RootTmpDirPrefix      string
	ToStdout              bool
	WriteableRepositories []string
)

const DefaultHostInjections = "/opt/eessi"

func registerFlags(root *cobra.Command) {
	root.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "Config file")
	root.PersistentFlags().StringVarP(&Image, "image", "c", "", "Container image to use in the buildenv")
	root.PersistentFlags().StringVarP(&ExtraBindPaths, "extra-bind-paths", "b", "", "Extra bind mounts to pass to apptainer")
	root.PersistentFlags().StringVarP(&HostInjections, "host-injections", "i", DefaultHostInjections, "Valid path to EESSI's host injections")
	root.PersistentFlags().StringVarP(&ResumePath, "resume", "r", "", "Resume path for the container")
	root.PersistentFlags().StringVarP(&RootTmpDirPrefix, "root-tmp-dir-prefix", "p", "sam.", "Prefix to use for the root tmp directory on host")
	root.PersistentFlags().StringVarP(&Nvidia, "nvidia", "n", "all", "Enable container for use with Nvidia gpu")
	root.PersistentFlags().BoolVar(&ToStdout, "to-stdout", false, "Do not run final command. Print to stdout instead")
	root.PersistentFlags().StringSliceVarP(&WriteableRepositories, "writeable-repositories", "w", []string{}, "CVMFS repository mouned with writeable overlay filesystem")
	root.PersistentFlags().StringVar(&FuseCmdRW, "fuse", "fuse-overlayfs", "Fuse implementation to use for overlay (writeable) filesystem")
}
