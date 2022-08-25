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
ROLE_NAME="${ROLE_NAME:-rhoam_role}"
MINIMAL_POLICY_NAME="${ROLE_NAME}_minimal_policy"
FUNCTIONAL_TEST_ROLE_NAME="${FUNCTIONAL_TEST_ROLE_NAME:-functional_test_role}"
FUNCTIONAL_TEST_MINIMAL_POLICY_NAME="${FUNCTIONAL_TEST_ROLE_NAME}_minimal_policy"
OCM_ENV="${OCM_ENV:-staging}"
CLUSTER_NAME="${CLUSTER_NAME:-defaultsts}"
AWS_REGION="${AWS_REGION:-eu-west-1}"
PREFIX="${PREFIX:-ManagedOpenShift}"
NODES_COUNT="${NODES_COUNT:-4}"
MACHINE_TYPE="${MACHINE_TYPE:-m5.xlarge}"

provision_sts_cluster() {
    rosa login --env=$OCM_ENV
    rosa create account-roles --mode auto -y
    sleep 30s
    rosa create cluster --cluster-name $CLUSTER_NAME --compute-nodes=$NODES_COUNT --compute-machine-type=$MACHINE_TYPE --sts --mode auto -y
    rosa describe cluster --cluster $CLUSTER_NAME
    rosa logs install --cluster $CLUSTER_NAME --watch
}

delete_sts_cluster() {
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

get_oidc_provider_env() {
  if [[ "$OCM_ENV" == "staging" ]]; then
      echo "rh-oidc-staging"
  else
    echo "rh-oidc"
  fi
}

rhoam-prerequisites() {
    # Delete policy and role
    aws iam delete-role-policy --role-name $ROLE_NAME --policy-name $MINIMAL_POLICY_NAME || true
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
              "arn:aws:iam::$(get_account_id):oidc-provider/$(get_oidc_provider_env).s3.us-east-1.amazonaws.com/$(get_cluster_id)"
          ]
      },
      "Action": ["sts:AssumeRole", "sts:AssumeRoleWithWebIdentity"],
      "Condition": {}
    }
  ]
}
EOM
    aws iam create-role --role-name $ROLE_NAME --assume-role-policy-document "file://$ROLE_NAME.json" || true

    cat <<EOM >"$MINIMAL_POLICY_NAME.json"
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "cloudwatch:GetMetricData",
                "ec2:CreateRoute",
                "ec2:DeleteRoute",
                "ec2:DescribeAvailabilityZones",
                "ec2:DescribeInstanceTypeOfferings",
                "ec2:DescribeInstanceTypes",
                "ec2:DescribeRouteTables",
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeSubnets",
                "ec2:DescribeVpcPeeringConnections",
                "ec2:DescribeVpcs",
                "elasticache:CreateReplicationGroup",
                "elasticache:DeleteReplicationGroup",
                "elasticache:DescribeCacheClusters",
                "elasticache:DescribeCacheSubnetGroups",
                "elasticache:DescribeReplicationGroups",
                "elasticache:DescribeSnapshots",
                "elasticache:DescribeUpdateActions",
                "rds:DescribeDBInstances",
                "rds:DescribeDBSnapshots",
                "rds:DescribeDBSubnetGroups",
                "rds:DescribePendingMaintenanceActions",
                "rds:ListTagsForResource",
                "s3:CreateBucket",
                "s3:DeleteBucket",
                "s3:ListAllMyBuckets",
                "s3:ListBucket",
                "s3:PutBucketPublicAccessBlock",
                "s3:PutBucketTagging",
                "s3:PutEncryptionConfiguration"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:CreateSecurityGroup",
                "ec2:CreateSubnet",
                "ec2:CreateTags",
                "ec2:CreateVpc",
                "ec2:CreateVpcPeeringConnection",
                "elasticache:AddTagsToResource",
                "elasticache:CreateCacheSubnetGroup",
                "elasticache:CreateSnapshot",
                "rds:AddTagsToResource",
                "rds:CreateDBInstance",
                "rds:CreateDBSnapshot",
                "rds:CreateDBSubnetGroup"
            ],
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "aws:RequestTag/red-hat-managed": "true"
                }
            }
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:AcceptVpcPeeringConnection",
                "ec2:AuthorizeSecurityGroupIngress",
                "ec2:CreateSecurityGroup",
                "ec2:CreateSubnet",
                "ec2:CreateVpcPeeringConnection",
                "ec2:DeleteSecurityGroup",
                "ec2:DeleteSubnet",
                "ec2:DeleteVpc",
                "ec2:DeleteVpcPeeringConnection",
                "elasticache:BatchApplyUpdateAction",
                "elasticache:CreateSnapshot",
                "elasticache:DeleteCacheSubnetGroup",
                "elasticache:DeleteSnapshot",
                "elasticache:ModifyCacheSubnetGroup",
                "elasticache:ModifyReplicationGroup",
                "rds:DeleteDBInstance",
                "rds:DeleteDBSnapshot",
                "rds:DeleteDBSubnetGroup",
                "rds:ModifyDBInstance",
                "rds:RemoveTagsFromResource"
            ],
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "aws:ResourceTag/red-hat-managed": "true"
                }
            }
        }
    ]
}
EOM
    # attach policy with only the required permissions by CRO
    aws iam put-role-policy --role-name $ROLE_NAME --policy-name $MINIMAL_POLICY_NAME --policy-document "file://$MINIMAL_POLICY_NAME.json" || true

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
