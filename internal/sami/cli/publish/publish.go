/*
Copyright © 2026 Adam McCartney
// SPDX-License-Identifier: MIT
*/
package publish

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Uploader abstracts the subset of internal/rgw/client.Client used here; a
// fake is substituted in tests.
type Uploader interface {
	UploadFile(ctx context.Context, bucket, key, file string) error
}

// TarballFromLogs scans the slurm*.out files under logdir for the
// "TARBALL=<path>" line crtar prints at the end of a build and returns the
// tarball path. It fails when no tarball is found, or when several distinct
// ones appear (e.g. a logdir shared by more than one build).
func TarballFromLogs(logdir string) (string, error) {
	info, err := os.Stat(logdir)
	if err != nil {
		return "", fmt.Errorf("logdir %q: %w", logdir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("logdir %q is not a directory", logdir)
	}
	matches, err := filepath.Glob(filepath.Join(logdir, "slurm*.out"))
	if err != nil {
		return "", fmt.Errorf("globbing slurm output in %q: %w", logdir, err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no slurm*.out files found in %q", logdir)
	}

	seen := make(map[string]struct{})
	for _, m := range matches {
		tarballs, err := tarballsFromSlurmOut(m)
		if err != nil {
			return "", err
		}
		for _, tb := range tarballs {
			seen[tb] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return "", fmt.Errorf("no TARBALL= line found in slurm output under %q", logdir)
	}
	tarballs := make([]string, 0, len(seen))
	for tb := range seen {
		tarballs = append(tarballs, tb)
	}
	sort.Strings(tarballs)
	if len(tarballs) > 1 {
		return "", fmt.Errorf("found multiple tarballs in slurm output under %q: %s", logdir, strings.Join(tarballs, ", "))
	}
	return tarballs[0], nil
}

// tarballsFromSlurmOut extracts the values of all TARBALL=<path> lines.
func tarballsFromSlurmOut(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading slurm output %q: %w", path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	// increase buffer size to handle scanning long lines in build log
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	var out []string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "TARBALL=") {
			continue
		}
		if tb := strings.TrimSpace(strings.TrimPrefix(line, "TARBALL=")); tb != "" {
			out = append(out, tb)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scanning slurm output %q: %w", path, err)
	}
	return out, nil
}

// WriteChecksum computes the sha256 of tarball and writes it in
// `sha256sum --check` compatible format to "<basename>.sha256" next to the
// tarball, mirroring `(cd $(dirname $tb) && sha256sum $bn > $bn.sha256)`.
// Returns the sidecar path.
func WriteChecksum(tarball string) (string, error) {
	f, err := os.Open(tarball)
	if err != nil {
		return "", fmt.Errorf("opening tarball %q: %w", tarball, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing tarball %q: %w", tarball, err)
	}

	base := filepath.Base(tarball)
	sidecar := filepath.Join(filepath.Dir(tarball), base+".sha256")
	content := fmt.Sprintf("%x  %s\n", h.Sum(nil), base)
	if err := os.WriteFile(sidecar, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing checksum sidecar %q: %w", sidecar, err)
	}
	return sidecar, nil
}

// Publish uploads tarball and its sha256 sidecar to bucket, keyed by
// basename. The sidecar is uploaded last: pullers ingest pairs only, so the
// checksum is the sentinel marking a complete upload.
func Publish(ctx context.Context, u Uploader, tarball, bucket string) (string, error) {
	if _, err := os.Stat(tarball); err != nil {
		return "", fmt.Errorf("tarball %q: %w", tarball, err)
	}
	sidecar, err := WriteChecksum(tarball)
	if err != nil {
		return "", err
	}
	base := filepath.Base(tarball)
	if err := u.UploadFile(ctx, bucket, base, tarball); err != nil {
		return "", fmt.Errorf("uploading tarball to bucket %q: %w", bucket, err)
	}
	if err := u.UploadFile(ctx, bucket, base+".sha256", sidecar); err != nil {
		return "", fmt.Errorf("uploading checksum to bucket %q: %w", bucket, err)
	}
	return sidecar, nil
}
