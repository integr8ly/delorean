#!/usr/bin/env bash
echo $SEMVER > VERSION.txt
if [[ ! -z "$NON_SERVICE_AFFECTING" ]]; then
 echo "ServiceAffecting=false" >> VERSION.txt
fi