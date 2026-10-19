/*
Package rgwenv implements `sami rgw-env`: the sam.env renderer (L2 of
the credential ladder - a uniform way to produce the L1 fallback file).

Laws (rgw-credential-simplification design):

  - keys resolve via the external keks binary; this tool never imports
    rivet libraries and never touches vault itself;
  - the file is derived, not authored: overwrite without prompting;
  - write atomically (same-dir tempfile + rename), mode 0600 explicitly;
  - nothing half-written: resolve all legs before composing, compose
    before renaming;
  - only non-secret lines are printed; key material goes to the file.
*/
package rgwenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/asc-ac-at/sam/internal/sami/creds"
)

// DefaultBundlePath is where the renderer and the site's submit wrapper
// both look for the TLS bundle (the translation law: containers see
// /opt/rgw/... regardless of the host path).
const bundleRelPath = ".config/rgw/tls-ca-bundle.pem"

// SystemBundleSource is the one-time copy source (--setup).
const SystemBundleSource = "/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem"

// Data is everything the render needs; callers fold config and flags.
type Data struct {
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // empty: no endpoint line (refusal law stays upstream)
	Region          string // empty: no region line
	Home            string // user home; bundle checks resolve under it
}

// Render composes the sam.env content. Secret values are written as
// exports, never quoted into comments; AWS_CA_BUNDLE only appears when
// the bundle exists under Home at generation time.
func (d Data) Render() string {
	var b strings.Builder
	b.WriteString("# Rendered by `sami rgw-env`; derived, not authored. Re-run to rotate.\n")
	b.WriteString("export AWS_ACCESS_KEY_ID=" + d.AccessKeyID + "\n")
	b.WriteString("export AWS_SECRET_ACCESS_KEY=" + d.SecretAccessKey + "\n")
	if d.Endpoint != "" {
		b.WriteString("export AWS_ENDPOINT_URL=" + d.Endpoint + "\n")
	}
	if d.Region != "" {
		b.WriteString("export AWS_DEFAULT_REGION=" + d.Region + "\n")
	}
	if _, err := os.Stat(filepath.Join(d.Home, bundleRelPath)); err == nil {
		b.WriteString("export AWS_CA_BUNDLE=${HOME}/" + bundleRelPath + "\n")
	}
	return b.String()
}

// WriteAtomic writes content to path with mode 0600, atomically:
// same-dir tempfile + explicit chmod + rename; the tempfile is
// removed on any failure.
func WriteAtomic(path, content string) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".sam.env-*")
	if err != nil {
		return fmt.Errorf("creating tempfile for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if _, werr := tmp.WriteString(content); werr != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", tmpName, werr)
	}
	if cerr := tmp.Close(); cerr != nil {
		return fmt.Errorf("closing %s: %w", tmpName, cerr)
	}
	if cerr := os.Chmod(tmpName, 0o600); cerr != nil {
		return fmt.Errorf("chmod %s: %w", tmpName, cerr)
	}
	if rerr := os.Rename(tmpName, path); rerr != nil {
		return fmt.Errorf("renaming %s -> %s: %w", tmpName, path, rerr)
	}
	return nil
}

// Run executes the rgw-env mechanics: resolve all legs via r, compose,
// write. Nothing is written when any leg fails (the nothing-half-written
// law). Returns the composed (secret-bearing) content count for logging.
func Run(r creds.Resolver, accessRef, secretRef, out string, d *Data) error {
	pair, err := creds.ResolvePair(r, accessRef, secretRef)
	if err != nil {
		return err
	}
	d.AccessKeyID = pair.AccessKeyID
	d.SecretAccessKey = pair.SecretAccessKey
	content := d.Render()
	if err := WriteAtomic(out, content); err != nil {
		return err
	}
	return nil
}

// Setup performs the one-time bundle copy: SystemBundleSource -> the
// bundle slot under home. An existing bundle is kept (not refreshed),
// matching its role as a manually cached artifact. Idempotent.
func Setup(home string) (dest string, err error) {
	dest = filepath.Join(home, bundleRelPath)
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat %s: %w", dest, err)
	}
	src, err := os.ReadFile(SystemBundleSource)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", SystemBundleSource, err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", filepath.Dir(dest), err)
	}
	if err := os.WriteFile(dest, src, 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", dest, err)
	}
	return dest, nil
}
