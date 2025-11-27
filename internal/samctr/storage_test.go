// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"errors"
	"testing"
)

var setupStorageTests = []struct {
	rtd   string
	pfx   string
	hi    string
	vhome string
	vcd   string
	fm    []FuseMount
	oe    error
	ok    bool
}{
	{"", "", "", "", "", []FuseMount{}, nil, true},
}

// TODO -> improve these tests!
func TestSetupStorage(t *testing.T) {
	for _, e := range setupStorageTests {
		opts := &StorageOptions{e.rtd, e.pfx, e.hi, e.vhome, e.vcd, e.fm}
		_, err := SetupStorage(*opts)
		if err != e.oe {
			t.Errorf("SetupStorage(%s)", opts)
		}
	}
}

// Based on BindMount type
var parseBindSpecTests = []struct {
	in  string
	err error
	w   BindMount
}{
	{"/opt/somewhere", nil, BindMount{"/opt/somewhere", "/opt/somewhere", "rw"}},
	{"/opt/somewhere:/opt/anywhere", nil, BindMount{"/opt/somewhere", "/opt/anywhere", "rw"}},
	{"/opt/somewhere:/opt/anywhere:ro", nil, BindMount{"/opt/somewhere", "/opt/anywhere", "ro"}},
	{":/uho:ro", InvalidBindSpecError, BindMount{}},
}

func TestParseBindSpec(t *testing.T) {
	for _, p := range parseBindSpecTests {
		out, err := ParseBindSpec(p.in)
		if !errors.Is(err, p.err) {
			t.Errorf("ParseBindSpec(%s) got %s, want %s", p.in, out, p.w)
		}

		if out != p.w {
			t.Errorf("ParseBindSpec(%s) got %s, want %s", p.in, out, p.w)
		}
	}
}
