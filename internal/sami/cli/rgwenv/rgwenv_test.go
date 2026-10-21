/*
// SPDX-License-Identifier: MIT
*/
package rgwenv

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asc-ac-at/sam/internal/sami/creds"
)

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

func vaultFake() fakeResolver {
	return fakeResolver{vals: map[string]string{
		creds.DefaultAccessKeyRef: "AKIAEXAMPLE",
		creds.DefaultSecretKeyRef: "sekret",
	}}
}

func TestRender_ComposesEnvExports(t *testing.T) {
	home := t.TempDir()
	d := &Data{
		AccessKeyID:     "AKIAEXAMPLE",
		SecretAccessKey: "sekret",
		Endpoint:        "https://rgw.example.org",
		Region:          "default",
		Home:            home,
	}
	content := d.Render()
	for _, want := range []string{
		"export AWS_ACCESS_KEY_ID=AKIAEXAMPLE\n",
		"export AWS_SECRET_ACCESS_KEY=sekret\n",
		"export AWS_ENDPOINT_URL=https://rgw.example.org\n",
		"export AWS_DEFAULT_REGION=default\n",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("render missing %q\ngot:\n%s", want, content)
		}
	}
	// no bundle under home: no CA line
	if strings.Contains(content, "AWS_CA_BUNDLE") {
		t.Errorf("CA line rendered without a bundle on disk:\n%s", content)
	}
}

func TestRender_CABundleLineWhenBundleExists(t *testing.T) {
	home := t.TempDir()
	bundleDir := filepath.Join(home, filepath.Dir(bundleRelPath))
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "tls-ca-bundle.pem"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := &Data{Home: home}
	content := d.Render()
	want := "export AWS_CA_BUNDLE=" + filepath.Join(home, bundleRelPath) + "\n"
	if !strings.Contains(content, want) {
		t.Errorf("CA line not the absolute host path %q:\n%s", want, content)
	}
	if strings.Contains(content, "${HOME}") || strings.Contains(content, "$HOME") {
		t.Errorf("render defers expansion to source time (must be plain KEY=VALUE):\n%s", content)
	}
}

func TestRun_ResolvesWritesMode0600(t *testing.T) {
	home := t.TempDir()
	out := filepath.Join(t.TempDir(), "sam.env")
	d := &Data{Endpoint: "https://rgw.example.org", Home: home}

	if err := Run(vaultFake(), "", "", out, d); err != nil {
		t.Fatalf("Run: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat rendered file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 0600", info.Mode().Perm())
	}
	content, _ := os.ReadFile(out)
	if !strings.Contains(string(content), "export AWS_ACCESS_KEY_ID=AKIAEXAMPLE\n") {
		t.Errorf("missing resolved access key:\n%s", content)
	}
}

func TestRun_NothingHalfWrittenOnResolveFailure(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sam.env")
	d := &Data{Home: t.TempDir()}
	err := Run(fakeResolver{err: errors.New("boom")}, "", "", out, d)
	if err == nil {
		t.Fatal("expected resolve error, got nil")
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Errorf("file must not exist after a failed resolve: %v", statErr)
	}
	// no stray tempfile in the dir
	entries, _ := os.ReadDir(filepath.Dir(out))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".sam.env-") {
			t.Errorf("stray tempfile leaked: %s", e.Name())
		}
	}
}

func TestWriteAtomic_OverwriteIdempotent(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sam.env")
	if err := WriteAtomic(out, "first"); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(out, "second"); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(out)
	if string(content) != "second" {
		t.Errorf("content = %q, want second", content)
	}
}

func TestSetup_CopiesBundleOnce(t *testing.T) {
	// SystemBundleSource is the machine's real pki path; skip when absent
	if _, err := os.Stat(SystemBundleSource); err != nil {
		t.Skip("no pki bundle on this host")
	}
	home := t.TempDir()
	dest, err := Setup(home)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("bundle not at %s: %v", dest, err)
	}
	// idempotent: second call keeps the existing file
	if err := os.WriteFile(dest, []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Setup(home); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(dest)
	if string(content) != "user" {
		t.Error("Setup overwrote an existing bundle (must keep)")
	}
}
