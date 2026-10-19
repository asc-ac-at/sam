/*
// SPDX-License-Identifier: MIT
*/
package rgwenv

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/sami/cli/shared"
	"github.com/asc-ac-at/sam/internal/sami/config"
	"github.com/asc-ac-at/sam/internal/sami/creds"
)

var rgwenvExample = `
	# render ~/.config/rgw/sam.env from the estate's vault row
	sami rgw-env

	# one-time setup: copy the TLS CA bundle next to sam.env, then render
	sami rgw-env --setup

	# render somewhere else (tests, staging)
	sami rgw-env --out /tmp/sam.env
`

var (
	outPath  string
	endpoint string
	region   string
	doSetup  bool
)

// DefaultOutPath is where the render lands without --out.
func defaultOutPath(home string) string {
	return filepath.Join(home, ".config", "rgw", "sam.env")
}

func NewCommand(opts *shared.Options, logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rgw-env",
		Short: "Render the rgw credential env file (sam.env) from vault",
		Long: `Render ~/.config/rgw/sam.env from the vault row via keks.

This is the uniform way to produce the rgw credential fallback file:
keys come from the vault row (default refs ref+vault://vault/deployment/
rgw/sam#...), endpoint and region from the sami config (overridable by
flags). The file is overwritten atomically, mode 0600, on every run -
re-run it to rotate keys.

Requires a live papiere session and VAULT_ADDR exported (keks' lanes);
keks' own refusal texts surface verbatim, including the login hint.`,
		Example: rgwenvExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolving home dir: %w", err)
			}

			if doSetup {
				dest, err := Setup(home)
				if err != nil {
					return err
				}
				logger.Info("CA bundle ready", "path", dest)
			}

			cfg, err := config.Load()
			if err != nil && !errors.Is(err, config.ErrNotFound) {
				return fmt.Errorf("loading sami config: %w", err)
			}
			var accessRef, secretRef string
			if cfg != nil {
				c := cfg.RGW.Credentials
				accessRef, secretRef = c.AccessKeyRef, c.SecretKeyRef
				if endpoint == "" {
					endpoint = cfg.RGW.Endpoint
				}
				if region == "" {
					region = cfg.RGW.Region
				}
			}

			out := outPath
			if out == "" {
				out = defaultOutPath(home)
			}
			if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
				return fmt.Errorf("preparing %s: %w", filepath.Dir(out), err)
			}

			d := &Data{Endpoint: endpoint, Region: region, Home: home}
			if err := Run(creds.KeksResolver{}, accessRef, secretRef, out, d); err != nil {
				return err
			}
			logger.Info("rendered rgw env file", "path", out)
			fmt.Printf("rendered %s (mode 0600)\n", out)
			return nil
		},
	}

	cmd.Flags().StringVar(&outPath, "out", "", "Output path (default ~/.config/rgw/sam.env)")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "rgw endpoint (default: rgw.endpoint from the sami config)")
	cmd.Flags().StringVar(&region, "region", "", "AWS region to render (default: rgw.region from the sami config)")
	cmd.Flags().BoolVar(&doSetup, "setup", false, "Also perform one-time setup: copy the TLS CA bundle next to sam.env")

	return cmd
}
