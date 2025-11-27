#!/bin/sh

CFG=${REPO_ROOT}/examples/samctr/config/min-config.yaml

# exec command
samctr --config=${CFG} --to-stdout exec -- "$@"

