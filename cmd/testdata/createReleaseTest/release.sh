#!/usr/bin/env bash
echo "$SEMVER" > VERSION.txt
if [[ "$SERVICE_AFFECTING" == true ]]; then
 echo "ServiceAffecting=true" >> VERSION.txt
else
 echo "ServiceAffecting=false" >> VERSION.txt
fi