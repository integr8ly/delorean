#!/bin/bash
# USAGE
#
# ~ Cluster Creation Prerequisites ~
#
#  make ocm/sts/rhoam-prerequisites (optional <ROLE_NAME= > <FUNCTIONAL_TEST_ROLE_NAME= >)
#
# <Default values>
#  ROLE_NAME=rhoam_role
#  FUNCTIONAL_TEST_ROLE_NAME=functional_test_role
#
#
# ~ Creation ~
#
# make ocm/rosa/cluster/create (optional <CLUSTER_NAME= > <AWS_REGION= > <COMPUTE_NODES= > <MACHINE_TYPE= > <ENABLE_AUTOSCALING=true/false > <STS_ENABLED=true/false > <ROLE_NAME= > <FUNCTIONAL_TEST_ROLE_NAME= >)
#
# <Default values>
#  CLUSTER_NAME=default-rosa
#  AWS_REGION=eu-west-1
#  COMPUTE_NODES=4
#  MACHINE_TYPE=m5.xlarge
#  ENABLE_AUTOSCALING=false
#  STS_ENABLED=true
#
#
#
# ^C to break
# Create a STS cluster
#
# ~ Deletion ~
#
# make ocm/rosa/cluster/delete (optional <CLUSTER_NAME= > <AWS_REGION= > <PREFIX= >)
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
ROLE_NAME="${ROLE_NAME:-rhoam_role}"
FUNCTIONAL_TEST_ROLE_NAME="${FUNCTIONAL_TEST_ROLE_NAME:-functional_test_role}"
FUNCTIONAL_TEST_MINIMAL_POLICY_NAME="${FUNCTIONAL_TEST_ROLE_NAME}_minimal_policy"
OCM_ENV="${OCM_ENV:-staging}"
CLUSTER_NAME="${CLUSTER_NAME:-default-rosa}"
AWS_REGION="${AWS_REGION:-eu-west-1}"
PREFIX="${PREFIX:-ManagedOpenShift}"
COMPUTE_NODES="${COMPUTE_NODES:-4}"
MACHINE_TYPE="${MACHINE_TYPE:-m5.xlarge}"
ENABLE_AUTOSCALING="${ENABLE_AUTOSCALING:-false}"
MIN_REPLICAS="${MIN_REPLICAS:-4}"
MAX_REPLICAS="${MAX_REPLICAS:-6}"
STS_ENABLED="${STS_ENABLED:-true}"


provision_rosa_cluster() {
    rosa login --env=$OCM_ENV
    args=(--cluster-name $CLUSTER_NAME --region $AWS_REGION --compute-machine-type $MACHINE_TYPE)
    if [[ $ENABLE_AUTOSCALING == 'true' ]]; then
        args+=(--enable-autoscaling --min-replicas $MIN_REPLICAS --max-replicas $MAX_REPLICAS)
    else
        args+=(--replicas $COMPUTE_NODES)
    fi
    if [[ $STS_ENABLED == 'true' ]]; then
        args+=(--sts --mode auto)
        rosa create account-roles --mode auto -y
        sleep 30
    else
        args+=(--non-sts)
    fi
    args+=( -y)
    rosa create cluster "${args[@]}"
    rosa describe cluster --cluster $CLUSTER_NAME
    rosa logs install --cluster $CLUSTER_NAME --watch
}

delete_rosa_cluster() {
    CLUSTER_ID=$(get_cluster_id)
    rosa delete cluster --cluster=$CLUSTER_NAME --watch -y
    rosa delete oidc-provider -c $CLUSTER_ID --mode auto -y
    rosa delete operator-roles -c $CLUSTER_ID --mode auto -y
    if rosa list clusters | grep -q 'No clusters available'; then
        rosa delete account-roles --prefix $PREFIX --mode auto -y
    fi
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

get_oidc_provider() {
    local OIDC_PROVIDER=$(aws iam list-open-id-connect-providers | jq -r --arg CLUSTER_ID "$(get_cluster_id)" '.OpenIDConnectProviderList[] | select(.Arn | endswith($CLUSTER_ID)).Arn')
    echo "$OIDC_PROVIDER"
}

sts_cluster_prerequisites() {
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
              "$(get_oidc_provider)"
          ]
      },
      "Action": ["sts:AssumeRole", "sts:AssumeRoleWithWebIdentity"],
      "Condition": {}
    }
  ]
}
EOM

    # Role and policy for functional tests
    aws iam delete-role-policy --role-name $FUNCTIONAL_TEST_ROLE_NAME --policy-name $FUNCTIONAL_TEST_MINIMAL_POLICY_NAME || true
    aws iam delete-role --role-name $FUNCTIONAL_TEST_ROLE_NAME || true

    aws iam create-role --role-name $FUNCTIONAL_TEST_ROLE_NAME --assume-role-policy-document "file://$ROLE_NAME.json" || true
    cat <<EOM >"$FUNCTIONAL_TEST_MINIMAL_POLICY_NAME.json"
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeRouteTables",
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeSubnets",
                "ec2:DescribeVpcPeeringConnections",
                "ec2:DescribeVpcs",
                "elasticache:DescribeCacheClusters",
                "elasticache:DescribeReplicationGroups",
                "elasticache:DescribeCacheSubnetGroups",
                "elasticache:ListTagsForResource",
                "rds:DescribeDBInstances",
                "rds:DescribeDBSubnetGroups",
                "s3:GetBucketTagging",
                "s3:GetBucketPublicAccessBlock",
                "s3:GetEncryptionConfiguration"
            ],
            "Resource": "*"
        }
    ]
}
EOM
    aws iam put-role-policy --role-name $FUNCTIONAL_TEST_ROLE_NAME --policy-name $FUNCTIONAL_TEST_MINIMAL_POLICY_NAME --policy-document "file://$FUNCTIONAL_TEST_MINIMAL_POLICY_NAME.json" || true
}

main() {
    while :; do
        case "${1:-}" in
        provision_rosa_cluster)
            provision_rosa_cluster
            exit 0
            ;;
        delete_rosa_cluster)
            delete_rosa_cluster
            exit 0
            ;;
        sts_cluster_prerequisites)
            sts_cluster_prerequisites
            exit 0
            ;;
        esac
    done
}

main "${@}"
