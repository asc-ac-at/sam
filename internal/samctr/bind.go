// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

type BindMount struct {
	Host  string // host path (absolute)
	Ctr   string // container path (absolute)
	Perms string // permissions ro, rw
}

func NewBindMount(host, ctr, perms string) *BindMount {
	bm := BindMount{
		Host: host,
		Ctr:  ctr,
	}
	// set up sensible default for perms
	perms_default := "rw"
	switch perms {
	case "ro":
		perms_default = "ro"
	case "rw":
		perms_default = "rw"
	default:
		log.Printf("Could not create bindmount with %s perms, using default %s", perms, perms_default)
	}
	bm.Perms = perms_default
	return &bm
}

func (bm *BindMount) Fmt() string {
	result := fmt.Sprintf("%s:%s:%s ", bm.Host, bm.Ctr, bm.Perms)
	return result
}

func BindMountsApptainerFmt(bindmounts []BindMount) string {
	result := ""
	for _, bm := range bindmounts {
		result = result + "-B " + bm.Fmt()
	}
	return result
}

var InvalidBindSpecError = errors.New("invalid bind spec")

// ParseBindSpec parses a bind specification string (from config or CLI) into a BindMount.
func ParseBindSpec(spec string) (BindMount, error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return BindMount{}, errors.New("empty bind spec")
	}
	// Split into at most 3 parts: host, ctr, perms
	parts := strings.SplitN(s, ":", 3)
	switch len(parts) {
	case 1:
		// single path -> same host and ctr
		p := strings.TrimSpace(parts[0])
		if p == "" {
			return BindMount{}, fmt.Errorf("%w: %q", InvalidBindSpecError, spec)
		}
		return BindMount{Host: p, Ctr: p, Perms: "rw"}, nil
	case 2:
		host := strings.TrimSpace(parts[0])
		ctr := strings.TrimSpace(parts[1])
		if host == "" || ctr == "" {
			return BindMount{}, fmt.Errorf("%w: %q", InvalidBindSpecError, spec)
		}
		return BindMount{Host: host, Ctr: ctr, Perms: "rw"}, nil
	case 3:
		host := strings.TrimSpace(parts[0])
		ctr := strings.TrimSpace(parts[1])
		perms := strings.TrimSpace(parts[2])
		if host == "" || ctr == "" || perms == "" {
			return BindMount{}, fmt.Errorf("%w: %q", InvalidBindSpecError, spec)
		}
		// permisssions normalization: accept "ro" or "rw" only
		if perms != "ro" && perms != "rw" {
			return BindMount{}, fmt.Errorf("invalid bind perms %q in %q (expected ro or rw)", perms, spec)
		}
		return BindMount{Host: host, Ctr: ctr, Perms: perms}, nil
	default:
		return BindMount{}, fmt.Errorf("%w: %q", InvalidBindSpecError, spec)
	}
}
