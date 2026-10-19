/*
Copyright © 2026 Adam McCartney
// SPDX-License-Identifier: MIT
*/
package publish

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestTarballFromLogs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "slurm-123.out"), "some build output\nTARBALL=/path/to/sw.tar.gz\nmore output\n")

	got, err := TarballFromLogs(dir)
	if err != nil {
		t.Fatalf("TarballFromLogs returned err %v", err)
	}
	if got != "/path/to/sw.tar.gz" {
		t.Errorf("TarballFromLogs = %q, want %q", got, "/path/to/sw.tar.gz")
	}
}

func TestTarballFromLogs_MultipleFilesSameTarball(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "slurm-1.out"), "TARBALL=/path/to/sw.tar.gz\n")
	writeFile(t, filepath.Join(dir, "slurm-2.out"), "TARBALL=/path/to/sw.tar.gz\n")

	got, err := TarballFromLogs(dir)
	if err != nil {
		t.Fatalf("TarballFromLogs returned err %v", err)
	}
	if got != "/path/to/sw.tar.gz" {
		t.Errorf("TarballFromLogs = %q, want %q", got, "/path/to/sw.tar.gz")
	}
}

func TestTarballFromLogs_MultipleTarballsFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "slurm-1.out"), "TARBALL=/path/to/a.tar.gz\nTARBALL=/path/to/b.tar.gz\n")

	_, err := TarballFromLogs(dir)
	if err == nil {
		t.Fatal("expected error for multiple distinct tarballs, got nil")
	}
	if !strings.Contains(err.Error(), "multiple tarballs") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTarballFromLogs_NoSlurmOutFails(t *testing.T) {
	dir := t.TempDir()
	if _, err := TarballFromLogs(dir); err == nil {
		t.Fatal("expected error for missing slurm output, got nil")
	}
}

func TestTarballFromLogs_NoTarballLineFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "slurm-1.out"), "build log without tarball\nTARBALL=\n")

	if _, err := TarballFromLogs(dir); err == nil {
		t.Fatal("expected error for missing TARBALL= line, got nil")
	}
}

func TestTarballFromLogs_NotADirFails(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "not-a-dir")
	writeFile(t, f, "x")

	if _, err := TarballFromLogs(f); err == nil {
		t.Fatal("expected error for non-directory logdir, got nil")
	}
}

func TestTarballFromLogs_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "slurm-1.out"), "  TARBALL=/path/to/sw.tar.gz  \n")

	got, err := TarballFromLogs(dir)
	if err != nil {
		t.Fatalf("TarballFromLogs returned err %v", err)
	}
	if got != "/path/to/sw.tar.gz" {
		t.Errorf("TarballFromLogs = %q, want %q", got, "/path/to/sw.tar.gz")
	}
}

func TestWriteChecksum(t *testing.T) {
	dir := t.TempDir()
	tb := filepath.Join(dir, "sw.tar.gz")
	writeFile(t, tb, "hello\n")

	sidecar, err := WriteChecksum(tb)
	if err != nil {
		t.Fatalf("WriteChecksum returned err %v", err)
	}
	wantPath := filepath.Join(dir, "sw.tar.gz.sha256")
	if sidecar != wantPath {
		t.Errorf("sidecar path = %q, want %q", sidecar, wantPath)
	}

	content, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatalf("reading sidecar: %v", err)
	}
	// sha256 of "hello\n", matching `sha256sum` output format
	want := "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03  sw.tar.gz\n"
	if string(content) != want {
		t.Errorf("sidecar content = %q, want %q", content, want)
	}
}

// fakeUploader records UploadFile calls so tests can assert on order and
// arguments without touching an object store.
type fakeUploader struct {
	calls [][3]string
	err   error
}

func (f *fakeUploader) UploadFile(ctx context.Context, bucket, key, file string) error {
	f.calls = append(f.calls, [3]string{bucket, key, file})
	return f.err
}

func TestPublish_UploadOrder(t *testing.T) {
	dir := t.TempDir()
	tb := filepath.Join(dir, "sw.tar.gz")
	writeFile(t, tb, "hello\n")

	fu := &fakeUploader{}
	sidecar, err := Publish(context.Background(), fu, tb, "sam-archives")
	if err != nil {
		t.Fatalf("Publish returned err %v", err)
	}

	want := [][3]string{
		{"sam-archives", "sw.tar.gz", tb},
		{"sam-archives", "sw.tar.gz.sha256", sidecar},
	}
	if !reflect.DeepEqual(fu.calls, want) {
		t.Errorf("upload calls = %v, want %v", fu.calls, want)
	}
	if _, err := os.Stat(sidecar); err != nil {
		t.Errorf("sidecar not written: %v", err)
	}
}

func TestPublish_MissingTarballFails(t *testing.T) {
	fu := &fakeUploader{}
	_, err := Publish(context.Background(), fu, filepath.Join(t.TempDir(), "nope.tar.gz"), "sam-archives")
	if err == nil {
		t.Fatal("expected error for missing tarball, got nil")
	}
	if len(fu.calls) != 0 {
		t.Errorf("expected no uploads on failure, got %v", fu.calls)
	}
}

func TestPublish_UploadErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	tb := filepath.Join(dir, "sw.tar.gz")
	writeFile(t, tb, "hello\n")

	fu := &fakeUploader{err: errors.New("boom")}
	if _, err := Publish(context.Background(), fu, tb, "sam-archives"); err == nil {
		t.Fatal("expected upload error to propagate, got nil")
	}
}
