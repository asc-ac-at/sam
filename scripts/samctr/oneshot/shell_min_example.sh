#!/bin/sh

REPO_ROOT=$(git rev-parse --show-toplevel)

CFG=${REPO_ROOT}/examples/samctr/config/min-config.yaml

./samctr --config=${CFG} shell

