#!/bin/sh

REPO_ROOT=$(git rev-parse --show-toplevel)

CFG=${REPO_ROOT}/min-config.yaml
CMD="go run ./cmd/samctr --config=${CFG} --to-stdout shell"

exec ${REPO_ROOT}/scripts/interactive/runner.sh ${CMD}
