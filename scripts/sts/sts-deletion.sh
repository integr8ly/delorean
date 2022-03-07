#!/bin/bash
# Usage :
# ./sts-deletion.sh <Cluster name>
set -eux


CLUSTER_NAME=$1
if [[ -z $CLUSTER_NAME ]]; then
  echo "usage: $0 <cluster name>"
  exit 1
fi

CLUSTER_ID=$(ocm get clusters --parameter search="name like '%$CLUSTER_NAME%'" | jq '.items[].id' -r)

rosa delete cluster --cluster=$CLUSTER_NAME --watch  -y

rosa delete oidc-provider -c $CLUSTER_ID --mode auto -y

rosa delete operator-roles -c $CLUSTER_ID --mode auto -y

#Do not run if other ROSA sts clusters are still in use
rosa delete account-roles --prefix ManagedOpenShift --mode auto -y