#!/bin/sh

if [ "$#" != 2 ]; then
    echo "usage "${BASH_SOURCE[0]}" <resume_dir> <apptainer_cmd>"
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)

RUNNER=${REPO_ROOT}/scripts/samctr/interactive/runner.sh

RESUME="$1"
APPTAINER_CMD="$2"
CFG=${REPO_ROOT}/examples/samctr/config/min-config.yaml
CMD="go run ${REPO_ROOT}/cmd/samctr --config=${CFG} --resume=${RESUME} --to-stdout exec -- ${APPTAINER_CMD}"

exec ${RUNNER} ${CMD}
