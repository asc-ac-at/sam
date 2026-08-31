// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
)

func TestArgCounts(t *testing.T) {
	opts := NewOptions()
	c := &client.Client{}

	cases := []struct {
		name    string
		command *cobra.Command
		valid   []string
		invalid []string
	}{
		{"object put", newPutCmd(opts, c), []string{"b", "k", "f"}, []string{"b", "k"}},
		{"object get", newGetCmd(opts, c), []string{"b", "k", "f"}, []string{"b"}},
		{"object delete", newDeleteObjectCmd(opts, c), []string{"b", "k"}, []string{"b"}},
		{"object list", newListObjectsCmd(opts, c), []string{"b"}, []string{}},
		{"bucket create", newCreateCmd(opts, c), []string{"b"}, []string{}},
		{"bucket delete", newDeleteBucketCmd(opts, c), []string{"b"}, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/valid", func(t *testing.T) {
			if err := tc.command.Args(tc.command, tc.valid); err != nil {
				t.Errorf("Args(%v) = %v, want nil", tc.valid, err)
			}
		})
		t.Run(tc.name+"/invalid", func(t *testing.T) {
			if err := tc.command.Args(tc.command, tc.invalid); err == nil {
				t.Errorf("Args(%v) = nil, want error", tc.invalid)
			}
		})
	}
}

func TestRegisterFlagsBindsRegion(t *testing.T) {
	opts := NewOptions()
	command := newCreateCmd(opts, &client.Client{})
	if err := command.ParseFlags([]string{"--region", "eu-west-1"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if opts.Region != "eu-west-1" {
		t.Errorf("Region = %q, want eu-west-1", opts.Region)
	}
}
