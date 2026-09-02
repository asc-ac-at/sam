package sbatch

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/asc-ac-at/sam/internal/sami/command"
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
	Cfg *command.CmdConfig
}

// NewSbatchSubmitter creates a Submitter backed by the real sbatch command.
// runner may be nil, in which case a default one with a 30s timeout is used.
func NewSbatchSubmitter() *SbatchSubmitter {
	cfg := command.NewCmdConfig([]string{"sbatch", "--parsable"})
	cfg.Timeout = defaultSubmitTimeout

	var stderr bytes.Buffer

	cfg.Stdout = io.Discard
	cfg.Stderr = &stderr

	return &SbatchSubmitter{Cfg: cfg}
}

// Submit returns the job ID reported by sbatch --parsable.
func (s *SbatchSubmitter) Submit(script []byte) (string, error) {

	var stdout, stderr bytes.Buffer
	s.Cfg.Stdin = bytes.NewReader(script)
	s.Cfg.Stdout = &stdout
	s.Cfg.Stderr = &stderr

	if err := s.Cfg.Run(); err != nil {
		return "", fmt.Errorf("sbatch --parsable: %w: %s",
			err, strings.TrimSpace(stderr.String()))
	}

	jobID := strings.TrimSpace(stdout.String())
	if jobID == "" {
		return "", fmt.Errorf("sbatch --parsable returned no job id; stderr: %s",
			strings.TrimSpace(stderr.String()))
	}
	return jobID, nil
}
