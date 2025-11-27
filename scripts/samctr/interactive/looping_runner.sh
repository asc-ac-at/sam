#!/bin/sh

#
# This script uses inotifywait tools to set up a loop where the program is run
# on any change found in the go files.

REPO_ROOT=$(git rev-parse --show-toplevel)

CMD="$@"

while inotifywait -q $(find ${REPO_ROOT}/pkg/cmd/samctr/ ${REPO_ROOT}/internal/samctr/ -name "*.go");
do
    echo -e "\n\n"
    sleep 0.01
    ${CMD}
done
