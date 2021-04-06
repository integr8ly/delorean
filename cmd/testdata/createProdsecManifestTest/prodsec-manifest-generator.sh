#!/usr/bin/env bash

# Dependencies used
case $TYPE_OF_MANIFEST in 
  "production")
  RHMIfileName="rhmi-production-release-manifest.txt"
  RHOAMfileName="rhoam-production-release-manifest.txt"
  ;;
  "master")
  RHMIfileName="rhmi-master-manifest.txt"
  RHOAMfileName="rhoam-master-manifest.txt"
  ;;
  "compare")
    case $OLM_TYPE in
      "integreatly-operator")
        exit 0
        ;;
      "managed-api-service")
        exit 1
        ;;
       *)
        ;;
      esac
  ;;
  *)
esac

case $OLM_TYPE in
  "integreatly-operator")
    echo "dummy manifest dependency" > "prodsec-manifests/$RHMIfileName"
    ;;
  "managed-api-service")
    echo "dummy manifest dependency" > "prodsec-manifests/$RHOAMfileName"
    ;;
   *)
    echo "Invalid OLM_TYPE set"
    echo "Use \"integreatly-operator\" or \"managed-api-service\""
    exit 1
    ;;
esac
