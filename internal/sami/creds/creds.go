/*
Package creds resolves secret references by execing the external keks
binary (keks get). No rivet-tools imports: the binary is the contract.

The lookup laws (from the rgw-credential-simplification design):

  - a reference resolves to exactly one trimmed stdout line;
  - keks' stderr is refusal text and passes through verbatim (it carries
    the byte-pinned login hint `run "papiere bitte"`);
  - keks absent from PATH refuses with an install hint, loudly — a
    requested backend must never downgrade silently to a weaker lane.
*/
package creds

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DefaultRefs is the estate's shared rgw credential row (settled
// 2026-10-19); a config may override each ref.
const (
	DefaultAccessKeyRef = "ref+vault://vault/deployment/rgw/sam#access_key"
	DefaultSecretKeyRef = "ref+vault://vault/deployment/rgw/sam#secret_key"
)

// Resolver resolves one full secret reference to its value.
type Resolver interface {
	Resolve(ref string) (string, error)
}

// KeksResolver execs `keks get` from PATH (or Bin) per reference.
type KeksResolver struct {
	Bin string // keks binary name or path; "" means "keks" on PATH
}

var errKeksNotFound = errors.New("keks binary not found")

// Resolve answers the value of ref via a bare `keks get`. The keyed bare
// get prints exactly one line (the machine lane law); anything else is
// a refusal. Exit failures surface keks' stderr verbatim.
func (k KeksResolver) Resolve(ref string) (string, error) {
	bin := k.Bin
	if bin == "" {
		bin = "keks"
	}
	if _, err := exec.LookPath(bin); err != nil {
		return "", fmt.Errorf("%w on PATH - install rivet-tools keks (go install .../rivet-tools/cmd/keks), or use the env backend", errKeksNotFound)
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, "get", ref)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		text := strings.TrimSpace(stderr.String())
		if text == "" {
			text = err.Error()
		}
		return "", fmt.Errorf("keks get %s: %s", ref, text)
	}
	value := strings.TrimSpace(stdout.String())
	if value == "" {
		return "", fmt.Errorf("keks get %s returned no value", ref)
	}
	return value, nil
}

// Pair is one resolved rgw credential pair.
type Pair struct {
	AccessKeyID     string
	SecretAccessKey string
}

// ResolvePair fetches both legs of the pair; the access leg first, so a
// dead session answers on the cheapest leg (probe laws make the second
// leg a cache serve).
func ResolvePair(r Resolver, accessRef, secretRef string) (Pair, error) {
	if accessRef == "" {
		accessRef = DefaultAccessKeyRef
	}
	if secretRef == "" {
		secretRef = DefaultSecretKeyRef
	}
	access, err := r.Resolve(accessRef)
	if err != nil {
		return Pair{}, err
	}
	secret, err := r.Resolve(secretRef)
	if err != nil {
		return Pair{}, err
	}
	return Pair{AccessKeyID: access, SecretAccessKey: secret}, nil
}

// FillEnvLaw returns the four (env var, ref) bindings of the AWS chain
// that PrepareEnv and FillMissingEnv manage.
const (
	EnvAccessKeyID     = "AWS_ACCESS_KEY_ID"
	EnvSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

// FillMissingEnv resolves the pair via r and sets only the parts of the
// AWS chain that are currently unset in the process environment; a
// prepared environment wins (the env precedence law).
func FillMissingEnv(r Resolver, accessRef, secretRef string) error {
	needAccess := os.Getenv(EnvAccessKeyID) == ""
	needSecret := os.Getenv(EnvSecretAccessKey) == ""
	if !needAccess && !needSecret {
		return nil
	}
	pair, err := ResolvePair(r, accessRef, secretRef)
	if err != nil {
		return err
	}
	if needAccess {
		if err := os.Setenv(EnvAccessKeyID, pair.AccessKeyID); err != nil {
			return fmt.Errorf("setting %s: %w", EnvAccessKeyID, err)
		}
	}
	if needSecret {
		if err := os.Setenv(EnvSecretAccessKey, pair.SecretAccessKey); err != nil {
			return fmt.Errorf("setting %s: %w", EnvSecretAccessKey, err)
		}
	}
	return nil
}
