#!/usr/bin/env bash

# This script mirrors all image mappings found in any image_mirror_mapping files
# inside the given folder.
# Authentication is required for any repository referenced in a mapping from the running machine.
#
# Example:
# $ scripts/mirror-images.sh
#

set -e

WORK_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ARGS=
# ARGS is a fix for running it locally on mac. It is defaulted to empty for other OSs.
if [[ "$OSTYPE" == "darwin"* ]]; then
    ARGS=--filter-by-os=.*
fi

mirror_images() {
    set -o errexit
    failures=0
    files=$(find ${MIRROR_MAPPING_DIR} -name "image_mirror_mapping")
    for mapping in $files; do
        echo "Running: oc image mirror -f=$mapping --skip-multiple-scopes $ARGS"
        if ! oc image mirror -f="$mapping" --skip-multiple-scopes --insecure $ARGS; then
            echo "ERROR: Failed to mirror images from $mapping"
            failures=$((failures+1))
        fi
    done
    exit $failures
}

if [ -z "${MIRROR_MAPPING_DIR}" ]; then
    echo "MIRROR_MAPPING_DIR is not set!!"
    exit 1
else
    mirror_images
fi
