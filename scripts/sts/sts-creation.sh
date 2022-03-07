#!/bin/bash
# Usage :
# ./sts-creation.sh <Cluster name>
set -eux


CLUSTER_NAME=$1
if [[ -z $CLUSTER_NAME ]]; then
  echo "usage: $0 <cluster name>"
  exit 1
fi

name=$CLUSTER_NAME
aws_account_id=$(aws sts get-caller-identity | jq -r .Account)
export AWS_REGION=eu-west-1

rosa login

rosa create account-roles --mode auto -y

rosa create cluster --cluster-name $CLUSTER_NAME --sts --mode auto  -y 

rosa describe cluster --cluster $CLUSTER_NAME

rosa logs install --cluster $CLUSTER_NAME --watch 

