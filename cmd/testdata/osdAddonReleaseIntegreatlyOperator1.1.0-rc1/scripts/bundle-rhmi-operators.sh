#!/usr/bin/env bash

if [ BUNDLE_ONLY ]; then
    mkdir -p ./packagemanifests/$OLM_TYPE/1.1.0/1.1.0/manifests
    mkdir -p ./packagemanifests/$OLM_TYPE/1.1.0/1.1.0/metadata
    cp packagemanifests/$OLM_TYPE/1.1.0/managed-api-service.clusterserviceversion.yaml packagemanifests/$OLM_TYPE/1.1.0/1.1.0/manifests
    cp packagemanifests/$OLM_TYPE/1.1.0/integreatly.org_rhmis_crd.yaml packagemanifests/$OLM_TYPE/1.1.0/1.1.0/manifests
    cp scripts/annotations.yaml packagemanifests/$OLM_TYPE/1.1.0/1.1.0/metadata
fi