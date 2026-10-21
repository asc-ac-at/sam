/*
// SPDX-License-Identifier: MIT
*/
package creds

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stubKeks writes an executable PATH stub named "keks" whose behavior is
// scripted by body (a /bin/sh script receiving "$@" in the usual way).
func stubKeks(t *testing.T, body string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("PATH stub requires /bin/sh")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "keks")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("writing keks stub: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestKeksResolver_HappyPath(t *testing.T) {
	stubKeks(t, `echo "SECRET-VALUE-$2"`)

	got, err := KeksResolver{}.Resolve("ref+vault://vault/deployment/x#k")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := "SECRET-VALUE-ref+vault://vault/deployment/x#k"
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}
}

func TestKeksResolver_StderrPassesThroughVerbatim(t *testing.T) {
	stubKeks(t, `echo "no vault token (VAULT_TOKEN / ~/.vault-token) - run \"papiere bitte\"" >&2
exit 1`)

	_, err := KeksResolver{}.Resolve("ref+vault://vault/deployment/x#k")
	if err == nil {
		t.Fatal("expected refusal, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, `run "papiere bitte"`) {
		t.Errorf("refusal lost the byte-pinned login hint; got: %s", msg)
	}
	if !strings.Contains(msg, "keks get ref+vault://vault/deployment/x#k") {
		t.Errorf("refusal does not name the ref; got: %s", msg)
	}
}

func TestKeksResolver_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir) // empty PATH legs: keks cannot exist here

	_, err := KeksResolver{}.Resolve("ref+vault://vault/deployment/x#k")
	if err == nil {
		t.Fatal("expected refusal, got nil")
	}
	if !strings.Contains(err.Error(), "keks binary not found") {
		t.Errorf("missing-binary refusal got mangled: %v", err)
	}
	if !strings.Contains(err.Error(), "env backend") {
		t.Errorf("refusal lacks the downgrade hint: %v", err)
	}
}

func TestKeksResolver_EmptyValueIsLoud(t *testing.T) {
	stubKeks(t, `echo ""`)
	if _, err := (KeksResolver{}).Resolve("ref+vault://vault/deployment/x#k"); err == nil {
		t.Fatal("expected refusal for empty value, got nil")
	}
}

type fakeResolver struct {
	vals map[string]string
	err  error
}

func (f fakeResolver) Resolve(ref string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.vals[ref], nil
}

func TestResolvePair_DefaultRefs(t *testing.T) {
	fr := fakeResolver{vals: map[string]string{
		DefaultAccessKeyRef: "AKIAEXAMPLE",
		DefaultSecretKeyRef: "sekret",
	}}
	pair, err := ResolvePair(fr, "", "")
	if err != nil {
		t.Fatalf("ResolvePair: %v", err)
	}
	if pair.AccessKeyID != "AKIAEXAMPLE" || pair.SecretAccessKey != "sekret" {
		t.Errorf("pair = %+v", pair)
	}
}

func TestResolvePair_ExplicitRefs(t *testing.T) {
	fr := fakeResolver{vals: map[string]string{
		"ref+vault://vault/deployment/custom#a": "a-val",
		"ref+vault://vault/deployment/custom#s": "s-val",
	}}
	pair, err := ResolvePair(fr, "ref+vault://vault/deployment/custom#a", "ref+vault://vault/deployment/custom#s")
	if err != nil {
		t.Fatalf("ResolvePair: %v", err)
	}
	if pair.AccessKeyID != "a-val" || pair.SecretAccessKey != "s-val" {
		t.Errorf("pair = %+v", pair)
	}
}

func TestFillMissingEnv_EnvWins(t *testing.T) {
	t.Setenv(EnvAccessKeyID, "prepared")
	t.Setenv(EnvSecretAccessKey, "prepared-too")
	// resolver that would fail if consulted (env must win without any call)
	fr := fakeResolver{err: errTestResolver}
	if err := FillMissingEnv(fr, "", ""); err != nil {
		t.Fatalf("FillMissingEnv with prepared env: %v", err)
	}
	if os.Getenv(EnvAccessKeyID) != "prepared" {
		t.Error("prepared env was overwritten")
	}
}

var errTestResolver = &testResolverError{"resolver must not be consulted"}

type testResolverError struct{ s string }

func (e *testResolverError) Error() string { return e.s }

func TestFillMissingEnv_FillsOnlyGaps(t *testing.T) {
	t.Setenv(EnvAccessKeyID, "prepared")
	t.Setenv(EnvSecretAccessKey, "")
	fr := fakeResolver{vals: map[string]string{
		DefaultAccessKeyRef: "from-vault",
		DefaultSecretKeyRef: "from-vault-secret",
	}}
	if err := FillMissingEnv(fr, "", ""); err != nil {
		t.Fatalf("FillMissingEnv: %v", err)
	}
	if got := os.Getenv(EnvAccessKeyID); got != "prepared" {
		t.Errorf("%s = %q, want prepared (env wins)", EnvAccessKeyID, got)
	}
	if got := os.Getenv(EnvSecretAccessKey); got != "from-vault-secret" {
		t.Errorf("%s = %q, want gap filled from vault", EnvSecretAccessKey, got)
	}
}

func TestFillMissingEnv_ResolverFailurePropagates(t *testing.T) {
	t.Setenv(EnvAccessKeyID, "")
	t.Setenv(EnvSecretAccessKey, "")
	fr := fakeResolver{err: errTestResolver}
	if err := FillMissingEnv(fr, "", ""); err == nil {
		t.Fatal("expected resolver error to propagate, got nil")
	}
}
