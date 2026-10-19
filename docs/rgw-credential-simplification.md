# rgw credential setup: simplification options via rivet-tools

Exploration for TODO 3f7a2c91. Written 2026-10-19. No sam/rivet code
was changed; this is the comparison + recommendation. The token
retrieval leg is demonstrated (offline) at the bottom.

## The current workflow (what we want to simplify)

Manual, per user, from the internal docs:

1. Look up access key + secret in the password store, write
   `~/.config/rgw/sam.env` with `AWS_ACCESS_KEY_ID`,
   `AWS_SECRET_ACCESS_KEY`, `AWS_ENDPOINT_URL`, `AWS_DEFAULT_REGION`.
2. Copy the CA bundle out of `/etc/pki/ca-trust` into
   `~/.config/rgw/`, plus a shell fork covering the container-view path
   vs host-view path.
3. Edit `~/.config/samctr/config.yaml` bind mounts so the container
   sees `rgw` and the config dir.
4. Remember to source `sam.env` in every shell; the build's sbatch
   wrapper forwards the values into apptainer via `APPTAINERENV_`.

Pain: the secret sits at rest in an env file on every machine; rotation
means editing every copy; the CA-bundle fork is two shell conditionals
hopeing `/opt/rgw` exists; nothing verifies the setup until the first
failed `rgw object put`.

## What rivet-tools already provides

Verified against the source at
`~/src/gitlab.tuwien.ac.at/vsc/virtualisation/rivet-tools` (c22d303):

- **`papiere bitte`** — one-word login: OIDC device flow (RFC 8628)
  against authentik → root session on the ladder; then RFC 8693
  exchange + JWT login → vault token at `~/.vault-token` (0600) with a
  scope-keyed sidecar. Kept/renewed/exchanged state machine, renewal is
  headless-friendly; full headless mint needs `RIVET_OIDC_CLIENT_SECRET`
  or a fresh stored session (fails fast, before any network leg).
- **`keks get ref+vault://<instance>/<mount>/<path>[#<key>]`** — resolve
  one KV row through the dosn cache: version-pinned probe; fetch only
  on version change or `-R`; serve stale on vault failure with a
  warning. No masking law: a keyed bare get prints the bare line —
  suited for var=$(...) composition.
- **`pkg/vault`, `pkg/refs`, `pkg/ladder`** — importable per the import
  laws (pkg sees only pkg, rivet-blind). `internal/` is never
  importable. Module path is `gitlab.tuwien.ac.at/...`; see constraint
  C2 below.
- Refusal texts carry the login hint (`run "papiere bitte"`), so
  failure modes stay uniform if we surface them deliberately.

## Constraints

- **C1 — headless at runtime.** Token acquisition happens on login
  nodes (device flow needs a browser leg once per session); builds and
  `sami publish` run inside apptainer on compute nodes. Anything
  requiring a tty inside the job is a non-starter.
- **C2 — module provenance.** sam lives at `github.com/asc-ac-at/sam`
  (public, `go install @latest`). Importing a `gitlab.tuwien.ac.at`
  module makes public installs resolve an internal host — the go proxy
  will not serve it. Any strategy that compiles rivet code *into* sam
  pays this provenance cost.
- **C3 — import laws.** Only `pkg/` may be imported; `oidclogin`
  (the session machinery) is `internal/` and off-limits. So strategy B
  cannot reuse the login state machine at all — only its aftermath
  (`~/.vault-token` + sidecar).
- **C4 — container contract.** `buildcmd.tmpl` already says "creds
  provisioned by the sbatch wrapper via APPTAINERENV_ forwarding (no
  creds file bind-mounted into the container)". The container surface
  is plain env. We should not regress to mounting secret files.
- **C5 — settled 2026-10-19.** The row is authored at
  `deployment/rgw/sam` with keys `access_key` / `secret_key`; reads
  work from a papiere-minted JWT-session token. One open item: the
  sam-admin token's policy answers 403 on *write* to
  `deployment/data/rgw/sam`, so key rotation needs either a policy
  grant on that path or a vault-operator step — decide before the
  first rotation.

## Strategies

### A. Generate `sam.env` from vault refs (a renderer)

A small tool or `sami` subcommand resolves the refs and rewrites
`sam.env` (0600, atomic write): keys from vault, endpoint/region from
the sami config, CA bundle path resolved once at generation time.

- **UX:** one command, e.g. `sami rgw-env`, replaces the password-store
  copy-paste; existing shells and wrappers keep working unchanged.
- **Moving parts:** the renderer (can shell out to keks — no imports);
  the vault row; keks present on the login node.
- **Headless:** generation runs on the login node where the papiere
  session lives; jobs see no difference.
- **Failure modes:** stale file after radosgw key rotation (re-run the
  renderer; add an mtime check if that bites); vault unreachable and
  cache cold → keks refuses with the login hint; vault unreachable and
  cache warm → stale serve still yields a file.
- **Not fixed:** secrets still at rest in a sourced file; still per-
  shell sourcing discipline.

### B. rgw-native vault resolution (imports)

rgw imports `pkg/vault` + `pkg/refs`, resolves refs at startup from
`~/.vault-token`; `sam.env` shrinks to non-secret config
(endpoint/region live in the sami config's `rgw:` section already).

- **UX:** nothing to source; `rgw object put ...` just works on any
  login node with a live papiere session.
- **Moving parts:** +1 module dependency crossing the provenance
  boundary (C2 — public github repo gains an internal-gitlab import;
  `go install github.com/asc-ac-at/sam/cmd/rgw@latest` fails off the
  internal network unless GOPRIVATE/Athens is arranged); vault
  client + token chain inside rgw.
- **Headless:** fails on compute nodes: the job needs the vault token
  forwarded (`APPTAINERENV_VAULT_TOKEN` — exporting a reusable vault
  token into every job container is a *bigger* exposure than today's
  narrowly-scoped S3 keys), plus vault reachable from the container's
  network namespace, plus CA handling again. Doable; heavy.
- **Failure modes:** expired/unrenewable vault token inside a 10-hour
  build job; refusal surfaces inside the build log, after eb already
  spent hours. The C3 limit also means rgw can never *fix* the session
  itself, only refuse.

### C. keks as an external broker at the submission boundary

Resolve on the login node, forward as plain env — the shape
`buildcmd.tmpl` already assumes. Concretely: `sami build` (and
`sami publish`) runs `keks get
ref+vault://vault/deployment/rgw/sam#access_key` etc. at submission
time and exports the values into the sbatch wrapper's forward list
(exactly where `APPTAINERENV_AWS_ENDPOINT_URL` is handled today).

- **UX:** zero files, zero sourcing; the login node's papiere session
  is the only credential the admin manages; `sami build` refuses early
  (before eb consumes hours) with the byte-pinned login hint when the
  session is dead.
- **Moving parts:** keks binary on login nodes (one more `go install`
  or rpm); ~20 lines of resolve-and-export in sami's publish path; the
  vault row. No imports — sam stays public-github-pure.
- **Headless:** login node = interactive-capable; the job container
  receives plain env vars (today's C4 exactly). Long jobs are safe:
  resolution finished before `sbatch` returns.
- **Failure modes:** keks binary absent → loud refusal at submit; vault
  down + cache cold → same; vault down + cache warm → stale serve (a
  rotated key would slip through — bounded by KV version pinning, and
  `-R` exists); keys land in the job env as today (no new exposure over
  `sam.env`-sourced shells).
- **CA bundle:** rgw (host-side, at submit time) uses the system trust
  store; the container path fork moves into... nothing, because no
  in-container fetch happens. The container's rgw call still needs
  `AWS_CA_BUNDLE` — that single-resolved path rides the same forward
  list (strategy D, below).

### D. CA-bundle consolidation (orthogonal, do regardless)

One canonical bundle path delivered by the same resolution step
(strategy-dependent: written by the renderer in A, resolved by sami and
forwarded in C); the two shell conditionals in `sam.env` die. Cost:
near zero; benefit: half the current setup ceremony.

## Comparison

|                        | A: render sam.env | B: rgw-native      | C: keks broker      |
|------------------------|-------------------|--------------------|---------------------|
| secrets at rest        | yes (0600 file)   | vault token only   | no                  |
| new sam imports        | none (exec keks)  | pkg/vault+pkg/refs | none                |
| provenance cost (C2)   | none              | **blocks go install** off-net | none   |
| compute-node exposure  | none (env only)   | vault token + reachability | none (env only) |
| early failure          | at generate       | **deep in the job** | at submit          |
| IRC: rotation handling | manual re-run     | fresh each run     | cache-verity        |
| work to land           | small             | large + fork risk  | small-medium        |

## Recommendation

**C, with D folded in.** It matches the contract `buildcmd.tmpl`
already documents (creds forwarded by the wrapper), fixes the
configuration ceremony at the layer where the ceremony lives (the
login node), keeps sam dependency-clean, and keeps the blast radius of
a leaked job environment identical to today. A is the honest fallback
if keks-on-login-nodes proves socially hard to provision (it is one
binary; I doubt it). B fails on C2, C3, and the compute-node exposure —
recorded and rejected.

**Review outcome 2026-10-19: the operator chose A.** Rationale (their
words): it fits the bill without the env-propagation and
public/private-export tangles of C. The C design below is kept on
record as the documented fallback; the A design follows it.

Landing order:

1. ~~Author the vault row and confirm the role binding can read it.~~
   Done 2026-10-19: row at `deployment/rgw/sam`; `keks get
   ref+vault://vault/deployment/rgw/sam#{access_key,secret_key}`
   manually verified on the musica login nodes (the papiere login used
   `--audience openbao-internal-vault-adm-int-k8s-vie`).
2. Teach sami's publish path to resolve-and-forward via keks (submit
   boundary; refuse early with the login hint).
3. Fold D: one CA-bundle path in the forward list.
4. Update the internal docs; retire the sam.env recipe (keep it as
   documented fallback for hosts without keks).

## Demo: the token retrieval leg (offline, verified)

The retrieval machinery a strategy C submit boundary relies on,
exercised against the runbook's fake vault
(`docs/runbooks/fakevault.py`, stdlib python; serves one KV v2 row):

```console
$ VAULT_ADDR=http://127.0.0.1:8421 VAULT_TOKEN=dev-token-123 \
  HOME=$(mktemp -d) ./build/keks get "ref+vault://vault/deployment/dns/tsig#sync"
AAAAsekret==                                  # cold: probe + fetch (2 legs)
$ ... same command, warm
AAAAsekret==                                  # warm: probe only (1 leg)
$ ... get --json "ref+vault://vault/deployment/dns/tsig"
{
  "externaldns": "BBBBsekret==",
  "sync": "AAAAsekret=="
}
```

Vault-side log confirms the cache law: 2 legs cold, then 1 leg per warm
call (version probe, no refetch). The keyed bare get prints one bare
line — directly composable as
`export AWS_ACCESS_KEY_ID=$(keks get ref+vault://...#access_key)`.

**Live chain, verified 2026-10-19 on musica login nodes** (manually,
by the operator): `papiere vault login --audience
openbao-internal-vault-adm-int-k8s-vie` minted the token;
`keks get ref+vault://vault/deployment/rgw/sam#<key>` resolves
both keys. Write to the path 403s under the admin token's policy —
functionally irrelevant for the read-at-submit design, but rotation
will need a grant or an operator (see C5).

---

# Design: resolve-and-forward at the submit boundary (strategy C)

Status: **revisited 2026-10-19** — split into two lanes below; the job
lane is now recorded as declined (exposure law), the host lane
survives as `sami publish`'s gap-fill. Original text kept for the
reasoning trail.

## Current mechanism (what we replace)

`sbatch-script.tmpl`, the `Publish` block: the job script sources
`~/.config/rgw/sam.env` **on the compute node** (NFS home is visible
there), re-exports the AWS chain as `APPTAINERENV_AWS_*`, and
translates the CA bundle to the container-view `/opt/rgw/` path. The
template law is explicit: secrets are never written into the job
script itself. Failure mode today: a missing `sam.env` aborts the
*job* on the compute node — after queue time, not at submit.

## Proposed mechanics

**Where:** on the login node, inside sami, before `sbatch` is invoked.
sami execs `keks get` (a PATH binary; no imports) once per key. On
success it sets `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` in the
environment of the `sbatch` subprocess. sbatch propagates the
submission environment to the job (its default `--export=ALL`
behavior), so the values arrive on the compute node without ever
being written down.

**Template change:** the sam.env sourcing block becomes a
presence-check block: if `AWS_ACCESS_KEY_ID` is already in the job
environment (the propagated path), re-export `APPTAINERENV_AWS_*`
from it; **else** fall back to sourcing `sam.env` when it exists
(transition compat); else abort with a refusal naming both missing
lanes. The CA-bundle translation to `/opt/rgw/` is untouched — it is
already the canonical law, and with strategy A env file gone the host
-side fork dies with it.

**Early refusal:** resolution failure — keks absent from PATH — refuses
the submit with an install hint; keks non-zero exit passes its stderr
through verbatim, which by byte-pin carries the login hint
`run "papiere bitte"`. The operator learns the session is dead in a
second, on the login node, before eb burns hours.

**`sami publish` (standalone):** same resolver, in-process env
(`os.Setenv` before `client.New`). Precedence law: an explicitly
prepared environment wins (an operator who already exported `AWS_*`
gets exactly what they exported); the keks lane fills only what is
absent.

**Cache interplay:** repeat submits pay keks' version probe only (one
leg; the never-fetch-when-warm law measured in the demo). Rotation
converges by itself: the next probe after a KV version bump yields
the new pair; nothing is edited on any machine.

## Configuration

Extend the existing `rgw:` section of the sami config:

```yaml
rgw:
  bucket: sam-archives
  endpoint: https://s3.adm.asc
  credentials:
    backend: keks            # keks | env (default: env = today's behavior)
    access-key-ref: ref+vault://vault/deployment/rgw/sam#access_key
    secret-key-ref: ref+vault://vault/deployment/rgw/sam#secret_key
```

- `backend: env` keeps everything as-is (opt-in law: nobody is
  surprised by a missing keks on a fresh login node).
- The refs are config data, not code: a future per-admin account
  model spells `ref+vault://vault/userkv/{user}/rgw/sam#access_key`
  and the `{user}` placeholder law expands it per caller at no extra
  code cost.
- `VAULT_ADDR` is required in the submitting shell for keks'
  no-workspace lane; document it next to the config.

## Failure matrix

| situation | behavior | surface |
|---|---|---|
| keks binary missing | refuse submit, install hint | login node, immediate |
| dead papiere session | refuse submit, `run "papiere bitte"` (keks stderr verbatim) | login node, immediate |
| vault unreachable, cold cache | refuse submit | login node, immediate |
| vault unreachable, warm cache | stale serve, warning, submit proceeds | login node, immediate |
| stale cache after rotation | old keys one submit cycle; probe refreshes | job log |
| backend=env, sam.env missing | unchanged from today (job aborts) | compute node |
| env override set by operator | wins everywhere (precedence law) | both |

## Sub-system sketch (no code yet)

- New package `internal/sami/creds`: `Resolve(cfg, ref) (string, error)`
  wrapping an exec call; an interface seam for tests. Zero new module
  imports. Unit tests use a PATH-stub fake keks.
- `internal/sami/sbatch`: submitter needs env propagation to the
  `sbatch` child process (one field on the submit call).
- `internal/sami/cli/build/backend.go`: call Resolve when
  backend=keks and publish=true; refuse early per matrix.
- `internal/sami/cli/publish`: same resolver, same precedence law.
- Templates: the presence-check/fallback block replaces the
  source-or-die block; existing render tests grow compat cases.

## Open questions for review

1. Fallback retention: keep the `sam.env` sourcing branch in the
   template during transition (proposed yes), or rip it out now?
2. Backend default `env` vs `keks` — default stays `env`
   (opt-in) until the musica login nodes all have keks?
3. Rotation grant: do we pursue the vault policy write grant for
   `deployment/data/rgw/sam` for sam admins, or accept the
   operator-authored rotation procedure?
4. Ref spelling: `deployment/rgw/sam` (shared pair, current row) is
   the default; note the userkv per-admin spelling is a config edit
   away when per-person rgw accounts land.

---

# Design: `sami rgw-env` (strategy A)

Status: proposed, chosen by the operator 2026-10-19, awaiting answers
on the open questions. No code yet.

## The one-line version

A renderer: `sami rgw-env` writes `~/.config/rgw/sam.env` (atomic,
0600) with keys resolved from vault; nothing downstream changes — the
sbatch template keeps sourcing the file, the wrappers keep the
APPTAINERENV_ re-export, `sami publish` keeps its prepared-env
contract. Rotation becomes "run it again."

## Mechanics

1. Exec `keks get` twice (access-key ref, secret-key ref). No imports.
2. Compose the file: keys from keks; endpoint from the sami config's
   `rgw.endpoint`; `AWS_DEFAULT_REGION` for CLI parity with the old
   recipe; CA bundle line resolved once at generation time (see below).
3. Write: same-dir tempfile + rename; mode 0600 via explicit chmod,
   never at umask's mercy. Overwrite without prompting — the file is
   derived, not authored; a `--force` gate would only guard against
   regenerating what we regenerate anyway.
4. Print non-secret results only: target path, mode, which legs ran.
   Key material goes to the file, never to the terminal (the one
   deliberate soften of keks' no-masking law).

**CA bundle, resolved once:** if `$HOME/.config/rgw/tls-ca-bundle.pem`
exists at generation time, the file gets
`export AWS_CA_BUNDLE="${HOME}/.config/rgw/tls-ca-bundle.pem"`; else no
line. The `/opt/rgw` leg of the old recipe dies: it existed only for
in-container sourcing, which never happens — the sbatch template sets
the container path itself. The one-time `cp` of the bundle into
~/.config/rgw stays a documented manual step (open question 4).

## Configuration

Under the existing `rgw:` section; flags override:

```yaml
rgw:
  bucket: sam-archives
  endpoint: https://s3.adm.asc
  region: default                                # optional
  credentials:
    access-key-ref: ref+vault://vault/deployment/rgw/sam#access_key
    secret-key-ref: ref+vault://vault/deployment/rgw/sam#secret_key
```

No `backend` knob — running `rgw-env` IS the opt-in. Per-admin refs
(`userkv/{user}/rgw/sam`) remain a config edit away.

Operator requirement at generate time: `VAULT_ADDR` exported (keks'
no-workspace lane) and a live papiere session.

## Failure matrix

| situation | behavior |
|---|---|
| keks missing from PATH | refuse, install hint, nothing written |
| dead papiere session | keks stderr verbatim (carries `run "papiere bitte"`), nothing written |
| vault unreachable, cold cache | refuse, nothing written |
| vault unreachable, warm cache | stale serve + warning, file written from cache |
| existing sam.env present | overwritten atomically (renderer law) |
| target unwritable | write/rename refusal, nothing half-written |
| rotation happened upstream | next run fetches new value (KV version probe law) |

Nothing written ever means nothing half-written: keks resolves first,
then the file is composed, then the rename.

## Sub-system sketch

- `internal/sami/creds`: thin exec seam over keks (interface for
  tests; PATH-stub fakes). This package is spelled so C could reuse it
  later — A does not foreclose C.
- `internal/sami/cli/rgwenv/`: command + renderer + atomic-write
  helper, mirroring the cli/publish layout.
- `pkg/cmd/sami/root.go`: one registration line.
- Tests: golden file content with stub keys, explicit 0600 check,
  tempfile cleanup on failure, keks-stderr pass-through, overwrite
  idempotence.

Not in scope: `sami publish` learning to source sam.env (prepared-env
contract stays), CA-bundle auto-refresh from the vault row, any change
to sbatch templates or wrappers.

## Open questions

These belong to L2 (the renderer); L3's questions are settled by the
final decision above.

1. ~~Name/shape~~ → **`sami rgw-env`** (flat).
2. ~~CA bundle one-time copy~~ → **behind `--setup`**.
3. ~~`--out`~~ → **yes, `--out` for tests/debug**; default
   `~/.config/rgw/sam.env`.
4. ~~Region~~ → **flag to configure** (`--region`), static default;
   also configurable via `rgw.region`.
5. ~~Backend default~~ → **`env` for now** (static file is default
   until keks is on all login nodes).

---

# Strategy C revisited (2026-10-19)

The operator asked for another look. With A chosen, C decomposes into
two lanes that deserve separate verdicts — the original design treated
them as one.

## C-job: submit-boundary resolution for the slurm build path — ACCEPTED (2026-10-19)

The re-look surfaced an exposure law the first pass glossed over:
resolving at submit time and forwarding via the `sbatch` child's
environment means the keys travel **inside the slurm job record**
(submission environment propagation is how they reach the compute
node). Job environments are durable scheduler state — visibility varies
by site slurm version and config, and accounting retention makes the
copy long-lived. Today's mechanism reads a 0600 file on NFS *on the
compute node*: exactly one controlled resting place. C-job adds a
second resting place, in a store we do not administer.

**The operator weighed this and accepted it** (2026-10-19), choosing
C+D with A as fallback. The exposure law stays on record here as the
known cost of the ladder's top rung; the scoped-to-a-bucket S3 pair is
the least-blast-radius credential this system moves, and slurm
visibility posture is a site property the operator owns. If a site
audit ever finds the posture unacceptable, the per-run-file variant
(resolve at submit into a 0600 file in the run's logdir, cleaned on
job end) is the recorded escape hatch.

## C-host: resolve-and-fill for standalone `sami publish` — SURVIVES

`publish` runs on the login node; there is no propagation channel, no
scheduler record, no container. Resolving in-process to **fill gaps**
in the operator's environment is strictly better than refusing when
`AWS_*` is absent:

- precedence law unchanged: explicitly prepared env wins, keks fills
  only what is missing;
- no secrets at rest beyond today;
- early refusal on submit disappears for publish entirely — a dead
  papiere session means keks' stderr (`run "papiere bitte"`) verbatim,
  on the login node, before any upload attempt.

This is the same resolver seam A needs (`internal/sami/creds`), so
C-host lands as a by-product of A rather than a separate project.

# Final decision: C+D with A as fallback (the ladder), 2026-10-19

The operator's framing, a three-rung maturity ladder — each rung stays
available; higher rungs strictly *add* capability:

1. **L1 — file fallback (permanent).** The 0600 `sam.env` on NFS,
   sourced on the compute node by the template, remains a usable lane
   forever. Unsafe-but-practical; never removed.
2. **L2 — uniform generation.** `sami rgw-env` renders L1's file from
   vault (strategy A). This is what makes L1 safe*ish* to keep: no more
   password-store copy-paste, rotation is a re-run.
3. **L3 — runtime resolution (C+D).** With `credentials.backend: keks`,
   sami resolves the keys at submit time on the login node and
   forwards them via the job environment; `sami publish` gap-fills the
   operator's shell. Nothing secret at rest beyond the vault row.

## Landing order (final)

1. ~~Author the vault row~~ — done (2026-10-19, `deployment/rgw/sam`).
2. `internal/sami/creds` — exec seam over keks; shared by L2 and L3.
3. **L3 (C+D):** config `credentials.backend: keks|env` under `rgw:`,
   template presence-check block with sam.env fallback (the L1 lane),
   submitter env forwarding, `sami publish` gap-fill with env-wins
   precedence, single CA-bundle path in the forward list. Refusals per
   the C-design failure matrix.
4. **L2 (A):** `sami rgw-env` renderer per its design section
   (open questions 1-4 there). Maintains the L1 file for hosts
   without keks and for the fallback lane.
5. Docs: internal recipe becomes papiere login → (build/publish just
   works); `rgw-env` for the file lane; sam.env recipe demoted to
   "what rgw-env writes."

## Backend law

- `backend: env` (default until keks is on all login nodes): unchanged
  behavior, L1 only.
- `backend: keks`: resolve at submit; keks missing or a dead session
  refuses the submit on the login node, stderr verbatim (`run "papiere
  bitte"`). No silent downgrade to the file lane — a requested
  backend failing must answer loudly, not quietly change safety
  class. Operators who want the file lane say `env`.
- `sami publish`: env-wins precedence over keks gap-fill, as designed
  under C-host.
