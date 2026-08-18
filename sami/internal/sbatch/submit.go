package sbatch

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command"
)

const defaultSubmitTimeout = 30 * time.Second

// Submitter submits one fully rendered submit script to slurm. It is an
// interface so the cvmfs control flow can be tested without a live slurm.
type Submitter interface {
	Submit(script []byte) (jobID string, err error)
}

// SbatchSubmitter pipes the rendered script to `sbatch --parsable` on stdin
// (the heredoc pattern from the POC) and returns the job ID from stdout.
type SbatchSubmitter struct {
	Runner command.CmdRunner
}

// NewSbatchSubmitter creates a Submitter backed by the real sbatch command.
// runner may be nil, in which case a default one with a 30s timeout is used.
func NewSbatchSubmitter(runner command.CmdRunner) *SbatchSubmitter {
	if runner == nil {
		runner = command.NewRunner(command.WithTimeout(defaultSubmitTimeout))
	}
	return &SbatchSubmitter{Runner: runner}
}

// Submit returns the job ID reported by sbatch --parsable.
func (s *SbatchSubmitter) Submit(script []byte) (string, error) {
	out, err := s.Runner.New("sbatch", "--parsable").
		WithStdin(bytes.NewReader(script)).
		Output()
	if err != nil {
		return "", fmt.Errorf("sbatch submission failed: %w", err)
	}
	jobID := strings.TrimSpace(string(out))
	if jobID == "" {
		return "", fmt.Errorf("sbatch --parsable returned no job id")
	}
	return jobID, nil
}
