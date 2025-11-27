// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"
)

// e.g.
// Type          container
// FuseCmd       cvmfs2
// FuseArg       software.eessi.io
// CtrMountpoint /cvmfs/software.eessi.io
type FuseMount struct {
	Type          string `mapstructure:"type"`
	FuseCmd       string `mapstructure:"fuse_cmd"`
	FuseArg       string `mapstructure:"fuse_arg"`
	CtrMountpoint string `mapstructure:"ctr_mountpoint"`
}

func (fm *FuseMount) FuseCmdForExec() string {
	return fmt.Sprintf(`"%s:%s %s %s" `,
		fm.Type,
		fm.FuseCmd,
		fm.FuseArgFmt(),
		fm.CtrMountpoint,
	)
}

// Format the fuse arg of a --fusemount command
// basically we are interested in creating a writeable overlay
func (fm *FuseMount) FuseArgFmt() string {
	switch fm.FuseCmd {
	case "fuse-overlayfs":
		arg := fmt.Sprintf("-o lowerdir=/cvmfs_ro/%s -o upperdir=/tmp/%s/overlay-upper -o workdir=/tmp/%s/overlay-work", fm.FuseArg, fm.FuseArg, fm.FuseArg)
		return arg
	case "unionfs":
		arg := fmt.Sprintf("-o cow /tmp/%s/overlay-upper=RW:/cvmfs_ro/%s=RO", fm.FuseArg, fm.FuseArg)
		return arg
	default:
		// cvmfs2
		return fm.FuseArg
	}
}
