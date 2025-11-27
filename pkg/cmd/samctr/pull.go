// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

func ApptainerPullArgs(rs *RuntimeState) []string {
	args := []string{"pull"}

	// containerSif
	args = append(args, rs.ContainerSif)

	// image
	args = append(args, rs.Image)

	return args

}
