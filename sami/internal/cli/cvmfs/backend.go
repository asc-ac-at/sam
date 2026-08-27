package cvmfs

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/config"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/logging/buildlog"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/sbatch"
)

// runBackend dispatches on the build backend after build_cmd.sh has been
// rendered. This is step 4+5 of the cvmfs control flow.
//
//   - slurm: compose the submit script (headers + samctr exec wrapper) and
//     submit it via sbatch, logging the job ID. If no config exists in any
//     standard location, warn and return render-only.
//   - local: the rendered build_cmd.sh already stands alone; execution is
//     left to a future BuildRunner.
func runBackend(opts *shared.Options, blPath *buildlog.BuildLogPaths, logger *slog.Logger, sub sbatch.Submitter) error {
	backend, err := shared.ParseBackend(opts.BuildBackend)
	if err != nil {
		return err
	}
	switch backend {
	case shared.BackendSlurm:
		return runSlurmBackend(opts, blPath, logger, sub)
	case shared.BackendLocal:
		logger.Info("local build backend: rendered build_cmd.sh, execution not yet implemented",
			"path", blPath.BuildCmd)
		return nil
	}
	return nil // unreachable: ParseBackend rejects unknown values
}

// runSlurmBackend loads the sbatch config, composes the submit script and
// submits it. When no config is found anywhere it logs a warning and leaves
// the run at render-only — submission is skipped, not failed.
func runSlurmBackend(opts *shared.Options, blPath *buildlog.BuildLogPaths, logger *slog.Logger, sub sbatch.Submitter) error {
	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			logger.Warn("no sami config found; skipping sbatch submission (render-only)",
				"partition", opts.Partition)
			return nil
		}
		return fmt.Errorf("loading sami config: %w", err)
	}

	var headers bytes.Buffer
	if err := sbatch.RenderHeaders(&cfg.Sbatch, opts.Partition, blPath, &headers); err != nil {
		return err
	}

	var script bytes.Buffer
	if err := sbatch.RenderScript(sbatch.ScriptData{
		Headers:      headers.String(),
		BuildCmdPath: blPath.BuildCmd,
	}, &script); err != nil {
		return err
	}

	jobID, err := sub.Submit(script.Bytes())
	if err != nil {
		return err
	}
	outfile := fmt.Sprintf("%s/slurm-%s.out", blPath.BuildLog, jobID)
	logger.Info("submitted build job", "partition", opts.Partition, "jobID", jobID, "output", outfile)
	return nil
}
