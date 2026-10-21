/*
Copyright © 2026 Adam McCartney
// SPDX-License-Identifier: MIT
*/
package publish

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
	"github.com/asc-ac-at/sam/internal/sami/cli/shared"
	"github.com/asc-ac-at/sam/internal/sami/config"
	"github.com/asc-ac-at/sam/internal/sami/creds"
)

var publishExample = `
	# publish the tarball discovered from a build's slurm output
	sami publish --logdir /opt/adm/asc-software-stack/logs/me/20260114/sami.xyz.some-sw-default

	# publish an explicit tarball path
	sami publish --tarball /opt/adm/asc-software-stack/logs/me/20260114/sami.xyz.some-sw-default/sw.tar.gz

	# override the destination bucket and endpoint
	sami publish --tarball /path/to/sw.tar.gz --bucket sam-archives --endpoint https://rgw.example.org
`

var (
	logdir   string
	tarball  string
	bucket   string
	endpoint string
)

func NewCommand(opts *shared.Options, logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish an existing build tarball to the rados gateway",
		Long: `Publish a build artifact directly, post-build.

A successful build will produce a tarball before exiting. Sometimes
a user will prefer not to pass the "--publish" flag. There may be a
specific reason for doing this, such as manually testing the software
before publishing it to the rados gateway, for ingestion by the stratum0
server. If the manual checks are successful, the user may want to manually
publish the tarball.

The tarball path can be given directly using the "--tarball" flag, or else
recovered from the "TARBALL=" line printed by crtar in the build log.
The procedure for publishing a tarball mirrors that in the standard run:
a sha256 checksum of the tarball is written next to it and both files
are uploaded keyed by basename; the checksum goes last so consumers only
see complete pairs.

Configuration for the rados gateway come from the prepared environment.`,
		Example: publishExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if logdir == "" && tarball == "" {
				return errors.New("one of --logdir or --tarball is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// resolve the tarball: explicit flag wins, else recover from logs
			tb := tarball
			if tb == "" {
				logger.Debug("no --tarball given, scanning slurm output", "logdir", logdir)
				var err error
				tb, err = TarballFromLogs(logdir)
				if err != nil {
					return err
				}
			}
			logger.Debug("resolved tarball", "tarball", tb)

			// resolve bucket and endpoint: flags win, else fall back to config
			cfg, err := config.Load()
			if err != nil && !errors.Is(err, config.ErrNotFound) {
				return fmt.Errorf("loading sami config: %w", err)
			}
			if bucket == "" {
				if cfg != nil {
					bucket = cfg.RGW.Bucket
				}
				if bucket == "" {
					return errors.New("no bucket given: pass --bucket or set rgw.bucket in the sami config")
				}
			}
			if endpoint == "" && cfg != nil {
				endpoint = cfg.RGW.Endpoint
			}
			// the rgw client resolves the endpoint from the environment
			if endpoint != "" {
				if err := os.Setenv("AWS_ENDPOINT_URL", endpoint); err != nil {
					return fmt.Errorf("setting AWS_ENDPOINT_URL: %w", err)
				}
			}

			// C-host gap-fill: on the keks lane, resolve the credential
			// pair now; a prepared environment always wins. A dead papiere
			// session refuses before any upload attempt, carrying keks'
			// login hint verbatim.
			if cfg != nil {
				keks, err := cfg.RGW.Credentials.KeksBackend()
				if err != nil {
					return err
				}
				if keks {
					c := cfg.RGW.Credentials
					if err := creds.FillMissingEnv(creds.KeksResolver{}, c.AccessKeyRef, c.SecretKeyRef); err != nil {
						return fmt.Errorf("keks backend: %w", err)
					}
					logger.Debug("rgw credentials resolved via keks")
				}
			}

			c, err := client.New(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating rgw client: %w", err)
			}

			sidecar, err := Publish(cmd.Context(), c, tb, bucket)
			if err != nil {
				return err
			}
			logger.Info("published tarball", "bucket", bucket, "tarball", tb, "checksum", sidecar)
			return nil
		},
	}

	cmd.Flags().StringVarP(&logdir, "logdir", "l", "", "Build log directory; the tarball path is recovered from its slurm*.out files")
	cmd.Flags().StringVarP(&tarball, "tarball", "t", "", "Path to the tarball to publish (overrides --logdir discovery)")
	cmd.Flags().StringVar(&bucket, "bucket", "", "Destination bucket (default: rgw.bucket from the sami config)")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "rgw endpoint (default: rgw.endpoint from the sami config, else AWS_ENDPOINT_URL)")

	return cmd
}
