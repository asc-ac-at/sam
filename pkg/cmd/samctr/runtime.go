// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"
	"log"

	"github.com/asc-ac-at/sam/internal/samctr"
)

// holds ephemeral state computed during Prepare and consumed by Run
type RuntimeState struct {
	// host-side storage details created by setup storage
	Storage *samctr.StorageState

	// container image
	Image string

	// path to converted sif file
	ContainerSif string

	// merged list of bind mounts (config + generated)
	AllBindMounts      []samctr.BindMount
	ApptainerBindPaths []string

	// Any additional options to pass to apptainer
	ApptainerCmdOpts []string

	Environ []string

	ArgsAfterDash []string
}

var Runtime *RuntimeState

// This should be called _once_ after all the setup is complete, before
// invoking the final "runner" call
func (rs *RuntimeState) SetApptainerBindPaths() {
	// bind mounts
	bindCmds := []string{}
	for _, bm := range rs.AllBindMounts {
		log.Printf("SetApptainerBindPaths %s", bm.Fmt())
		bindCmds = append(bindCmds, fmt.Sprintf("-B %s", bm.Fmt()))
	}
	rs.ApptainerBindPaths = append(rs.ApptainerBindPaths, bindCmds...)
}
