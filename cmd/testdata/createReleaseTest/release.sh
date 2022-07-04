#!/usr/bin/env bash
echo "$SEMVER" > VERSION.txt
if [[ "$SERVICE_AFFECTING" == true ]]; then
 echo "ServiceAffecting=true" >> VERSION.txt
else
 echo "ServiceAffecting=false" >> VERSION.txt
fi
if [[ "$PREPARE_FOR_NEXT_RELEASE" == true ]]; then
 echo "prepareForNextRelease=true" >> VERSION.txt
else
 echo "prepareForNextRelease=false" >> VERSION.txt
fi