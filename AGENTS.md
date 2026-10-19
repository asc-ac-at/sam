AGENTS.md for sam
=================

Build tools for the Software and Modules group. Go monorepo, one module,
four tools: sami (builds), samctr (containers), crtar (tarballs), rgw
(radosgw client). Distributed via `go install @<tag>`


## Hard rules

- Ask before making direct code changes. Propose a design first;
  wait for answers to open questions before implementing.
- Never commit to `main`. Work on a topic branch, or emit a
  `git format-patch` into the agent-context patch archive.
- Before handing off, run and report:
  - `go build ./...`
  - `go vet ./...`
  - `go test -race ./...`
  - `gofmt -l .`
- Keep changes small and single-purpose. 400 lines of diff is
  reviewable; 4000 is not.

## Conventions

- Commits: Conventional Commits, tool as scope
  (`feat(sami): ...`, `fix(crtar): ...`, `test(rgw): ...`).
  AI-assisted commits carry:
  `Co-Authored-by: Pi (<provider>/<model>) <pi-coding-agent>`
- Branches named after the commit type: `feat/...`, `fix/...`.
- New sami subcommands live in `internal/sami/cli/<name>/`
  (logic and cobra wiring together) and are registered in
  `pkg/cmd/sami/root.go`.
- Core rgw operations go through `internal/rgw/client.Client`.
- License headers: match the neighboring files (MIT for sami,
  GPL-2.0 for rgw and crtar).
- Tests: stdlib `testing` only; no external assertion libraries.

## Environment

- rgw credentials come from a prepared environment
  (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_ENDPOINT_URL).
  Tools must not read credential files themselves.
- CI gate: `.gitlab-ci.yml` runs `go vet` and `go test -race`;
  merge only when green.

## Releases

- Features collect on `main`; one tag per release, tagged from
  `main`. `vX.Y.Z-rcN` tags only for installable cluster test builds.
- Version comes from `git describe` (see Makefile); never bump a
  hardcoded version string.

## Context

- Design docs and patch archive: ~/agent-context/asc-software-stack/
  (sami/design.md, sam/*.patch).
- Related repos: asc-software-layer, asc-software-layer-scripts
  (gitlab.tuwien.ac.at/vsc/software-stacks).
- More broadly related: <github.com/EESSI> software-layer, software-layer-scripts
