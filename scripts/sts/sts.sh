#!/bin/bash
# USAGE
#
# ~ Creation ~
#
# make ocm/provision_sts (optional <CLUSTER_NAME= > <AWS_REGION= > <NODE_COUNT= > <MACHINE_TYPE= >)
#
# <Default values>
#  CLUSTER_NAME=defaultsts
#  AWS_REGION=eu-west-1
#  NODE_COUN=4
#  MACHINE_TYPE=m5.xlarge
#
# ^C to break
# Create a STS cluster
#
# ~ Deletion ~
#
# make ocm/delete_sts (optional <CLUSTER_NAME= > <AWS_REGION= > <PREFIX= >)
#
# <Default values>
#  PREFIX=ManagedOpenShift
#
# ^C to break
# Delete a STS cluster
#
# PREREQUISITES
# - ROSA CLI
# - AWS CLI
# - jq
# - aws configuration with valid permissions. Run `aws configure`
#
# MORE INFO
# https://docs.openshift.com/rosa/rosa_getting_started_sts/rosa-sts-setting-up-environment.html

set -eux

# Prevents aws cli from opening editor on responses - https://github.com/aws/aws-cli/issues/4992
export AWS_PAGER=""
ROLE_NAME="rhoam_role"
OCM_ENV="${OCM_ENV:-production}"
CLUSTER_NAME="${CLUSTER_NAME:-defaultsts}"
AWS_REGION="${AWS_REGION:-eu-west-1}"
PREFIX="${PREFIX:-ManagedOpenShift}"
NODES_COUNT="${NODES_COUNT:-4}"
MACHINE_TYPE="${MACHINE_TYPE:-m5.xlarge}"

provision_sts_cluster() {
    rosa login --env=$OCM_ENV
    rosa create account-roles --mode auto -y
    rosa create cluster --cluster-name $CLUSTER_NAME --compute-nodes=$NODES_COUNT --compute-machine-type=$MACHINE_TYPE --sts --mode auto -y
    rosa describe cluster --cluster $CLUSTER_NAME
    rosa logs install --cluster $CLUSTER_NAME --watch
}

delete_sts_cluster() {
    CLUSTER_ID=$(get_cluster_id)

    rosa delete cluster --cluster=$CLUSTER_NAME --watch -y
    rosa delete oidc-provider -c $CLUSTER_ID --mode auto -y
    rosa delete operator-roles -c $CLUSTER_ID --mode auto -y
    #Do not run if other ROSA sts clusters are still in use
    rosa delete account-roles --prefix $PREFIX --mode auto -y
}

get_cluster_id() {
  local CLUSTER_ID=$(ocm get clusters --parameter search="name like '%$CLUSTER_NAME%'" | jq '.items[].id' -r)
  echo "$CLUSTER_ID"
}

# Gets the local aws account id
get_account_id() {
    local ACCOUNT_ID=$(aws sts get-caller-identity | jq -r .Account)
    echo "$ACCOUNT_ID"
}

get_role_arn() {
    echo "arn:aws:iam::$(get_account_id):role/$ROLE_NAME"
}

rhoam-prerequisites() {
    # Delete policy and role
    # TODO - detach policy with only the required permissions by CRO
    aws iam detach-role-policy --role-name $ROLE_NAME --policy-arn arn:aws:iam::aws:policy/AdministratorAccess || true
    aws iam delete-role --role-name $ROLE_NAME || true

    # Create policy and role
    # sts:AssumeRole with iam to allow for running CRO locally with this specific iam user
    # sts:AssumeRoleWithWebIdentity with federated oidc provider to allow assuming role when running on cluster in pod
    # Allows osdCcsAdmin IAM user to assume this role
    cat <<EOM >"$ROLE_NAME.json"
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
          "AWS": [
              "arn:aws:iam::$(get_account_id):user/osdCcsAdmin"
          ],
          "Federated": [
              "arn:aws:iam::$(get_account_id):oidc-provider/rh-oidc.s3.us-east-1.amazonaws.com/$(get_cluster_id)"
          ]
      },
      "Action": ["sts:AssumeRole", "sts:AssumeRoleWithWebIdentity"],
      "Condition": {}
    }
  ]
}
EOM
    aws iam create-role --role-name $ROLE_NAME --assume-role-policy-document "file://$ROLE_NAME.json" || true
    # TODO - attach policy with only the required permissions by CRO
    aws iam attach-role-policy --role-name $ROLE_NAME --policy-arn arn:aws:iam::aws:policy/AdministratorAccess || true
}

main() {
    while :; do
        case "${1:-}" in
        provision_sts_cluster)
            provision_sts_cluster
            exit 0
            ;;
        delete_sts_cluster)
            delete_sts_cluster
            exit 0
            ;;
        rhoam-prerequisites)
            rhoam-prerequisites
            exit 0
            ;;
        esac
    done
}

main "${@}"
