#!/usr/bin/env bash

# Dependencies used


case $OLM_TYPE in
  "integreatly-operator")
    echo "dummy manifest dependency" > "prodsec-manifests/rhmi-production-release-manifest.txt"
    ;;
  "managed-api-service")
    echo "dummy manifest dependency" > "prodsec-manifests/rhoam-production-release-manifest.txt"
    ;;
   *)
    echo "Invalid OLM_TYPE set"
    echo "Use \"integreatly-operator\" or \"managed-api-service\""
    exit 1
    ;;
esac
