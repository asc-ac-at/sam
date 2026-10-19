Development process for sam tools
=================================

Sam tools is a small project that has been primarily developed by a
single developer. The workflow described below reflects that. This
process can be reviewed as needed in the future. Development process
refers to the specific practice employed when using git. It also tries
to place this within the context of the actual workflow used to develop
the project up to this point, so that future contributors might better
understand how we got here.

## Branches

- `main` is the trunk. This should remain releasable; CI must be green.
- One short-lived branch pepr change, named after the Conventional Commit type:
  `feat/...`, `fix/...`, `test/...`, `refactor/...`, `chore/...`.
- Merge to `main` when CI passes; delete the branch. No release branches
  only the latest version is maintained.

## Commits 
- Conventional Commits, tool name as scope:
  `feat(sami): ...`, `fix(crtar): ...`, `test(rgw): ...`, `chore(ci): ...`.
- AI-assisted commits carry one of the following markers:
  + `Co-Authored-by: <AGENT-NAME> (<provider>/<model>) <agent-sig>

## Working with AI agents

1. Human writes the brief: intent, design answers, constraints.
2. Agent implements on a topic branch within the agent-sandbox
   (typically a VM where it cannot push directly to remote).
3. Agent runs `go vet ./...`, `go test -race ./...`, `gofmt -l .`
   and reports the results before handing off.
4. The agent prepares a commit (and possibly emits a `git format-patch`).
5. Human reviews the diff. Small, single-purpose commits only.
6. Merge through the CI gate. Agents never comit to `main` directly.

Context for agents lives in `AGENTS.md` (repo rules) and an (internal)
agent-context repo, where design docs are collected.

## Releases

- Features collect on `main` through seperate merges; one merge target,
  one tag per release.
- When the collection is done:
```
git switch main && git pull
git tag -a v0.3.0 -m "sam build tools 0.3.0"
git push origin v0.3.0
```
- Version in binaries comes from the `git describe --tags --always --dirty` 
  in the Makefile - no manual version commits.
- `vX.Y.Z-rcN` tags are allowed only to produce installable test builds
  for the cluster; they are not a record of merged features.


## Patches

`git format-patch` output is used as a transport mechanism for surfacing
changes from the sandbox to the developer machine that will ultimately
review and push changes.
