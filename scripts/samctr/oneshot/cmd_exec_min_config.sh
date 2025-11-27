#!/bin/sh

CFG=${REPO_ROOT}/examples/samctr/config/min-config.yaml

samctr --config=${CFG} exec -- "$@"

