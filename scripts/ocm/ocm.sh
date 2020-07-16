#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly OCM_DIR="${REPO_DIR}/ocm"

readonly TEMPLATES_DIR="${REPO_DIR}/templates/ocm"
readonly CLUSTER_TEMPLATE_FILE="${TEMPLATES_DIR}/cluster-template.json"
readonly CR_AWS_STRATEGIES_CONFIGMAP_FILE="${TEMPLATES_DIR}/cr-aws-strategies.yml"
readonly LB_CLUSTER_QUOTA_FILE="${TEMPLATES_DIR}/load-balancer-cluster-quota.json"
readonly CLUSTER_STORAGE_QUOTA_FILE="${TEMPLATES_DIR}/cluster-storage-quota.json"
readonly CLUSTER_KUBECONFIG_FILE="${OCM_DIR}/cluster.kubeconfig"
readonly CLUSTER_CONFIGURATION_FILE="${OCM_DIR}/cluster.json"
readonly CLUSTER_DETAILS_FILE="${OCM_DIR}/cluster-details.json"
readonly CLUSTER_CREDENTIALS_FILE="${OCM_DIR}/cluster-credentials.json"
readonly CLUSTER_LOGS_FILE="${OCM_DIR}/cluster.log"

readonly RHMI_OPERATOR_NAMESPACE="redhat-rhmi-operator"

readonly ERROR_MISSING_AWS_ENV_VARS="ERROR: Not all required AWS environment are set. Please make sure you've exported all following env vars:"
readonly ERROR_MISSING_CLUSTER_JSON="ERROR: ${CLUSTER_CONFIGURATION_FILE} file does not exist. Please run 'make ocm/cluster.json' first"

check_aws_credentials_exported() {
    if [[ -z "${AWS_ACCOUNT_ID:-}" || -z "${AWS_SECRET_ACCESS_KEY:-}" || -z "${AWS_ACCESS_KEY_ID:-}" ]]; then
        printf "%s\n" "${ERROR_MISSING_AWS_ENV_VARS}"
        printf "AWS_ACCOUNT_ID='%s'\n" "${AWS_ACCOUNT_ID:-}"
        printf "AWS_ACCESS_KEY_ID='%s'\n" "${AWS_ACCESS_KEY_ID:-}"
        printf "AWS_SECRET_ACCESS_KEY='%s'\n" "${AWS_SECRET_ACCESS_KEY:-}"
        exit 1
    fi
}

get_latest_generated_access_key_id() {
    local access_keys="${1}"
    firstAccessKeyDateAndId="$(jq -r .[0].CreateDate <<< "${access_keys}")|$(jq -r .[0].AccessKeyId <<< "${access_keys}")"
    secondAccessKeyDateAndId="$(jq -r .[1].CreateDate <<< "${access_keys}")|$(jq -r .[1].AccessKeyId <<< "${access_keys}")"
    # Compare "CreateDate" and return newer "AccessKeyId" (the latest generated one)
    if [[ "${firstAccessKeyDateAndId%|*}" > "${secondAccessKeyDateAndId%|*}" ]]; then
        printf "%s" "${firstAccessKeyDateAndId#*|}"
    else
        printf "%s" "${secondAccessKeyDateAndId#*|}"
    fi
}

create_cluster_configuration_file() {
    local timestamp
    local listening="external"

    : "${OCM_CLUSTER_LIFESPAN:=4}"
    : "${OCM_CLUSTER_NAME:=rhmi-$(date +"%y%m%d-%H%M")}"
    : "${OCM_CLUSTER_REGION:=eu-west-1}"
    : "${BYOC:=false}"
    : "${OPENSHIFT_VERSION:=}"
    : "${PRIVATE:=false}"

    timestamp=$(get_expiration_timestamp "${OCM_CLUSTER_LIFESPAN}")

    if [ "${PRIVATE}" = true ]; then
        listening="internal"
    fi

    jq ".expiration_timestamp = \"${timestamp}\" | .name = \"${OCM_CLUSTER_NAME}\" | .region.id = \"${OCM_CLUSTER_REGION}\" | .api.listening = \"${listening}\"" \
        < "${CLUSTER_TEMPLATE_FILE}" \
        > "${CLUSTER_CONFIGURATION_FILE}"
	
    if [ "${BYOC}" = true ]; then
        check_aws_credentials_exported
        update_configuration_with_aws_credentials
    fi

    if [[ -n "${OPENSHIFT_VERSION}" ]]; then
        update_configuration_with_openshift_version
    fi
    cat "${CLUSTER_CONFIGURATION_FILE}"
}

create_cluster() {
    local cluster_id

    if ! [[ -e "${CLUSTER_CONFIGURATION_FILE}" ]]; then
        printf "%s\n" "${ERROR_MISSING_CLUSTER_JSON}"
        exit 1
    fi

    echo "Sending a request to OCM to create an OSD cluster"
    send_cluster_create_request
    cluster_id=$(get_cluster_id)

    echo "Cluster ID: ${cluster_id}"

    wait_for "ocm get /api/clusters_mgmt/v1/clusters/${cluster_id} | jq -r .api.url | grep -q https://.*:6443" "cluster creation" "180m" "300"
    wait_for "ocm get /api/clusters_mgmt/v1/clusters/${cluster_id}/credentials | jq -r .admin | grep -q admin" "fetching cluster credentials" "10m" "30"

    save_cluster_credentials "${cluster_id}"


    printf "Login credentials: \n%s\n" "$(jq -r < "${CLUSTER_CREDENTIALS_FILE}")"
    printf "Log in to the OSD cluster using oc:\noc login --server=%s --username=kubeadmin --password=%s\n" "$(jq -r .api.url < "${CLUSTER_DETAILS_FILE}")" "$(jq -r .password < "${CLUSTER_CREDENTIALS_FILE}")"
}

install_rhmi() {
    local cluster_id
    local rhmi_name
    local infra_id
    local csv_name

    : "${USE_CLUSTER_STORAGE:=true}"
    : "${PATCH_CR_AWS_CM:=true}"

    cluster_id=$(get_cluster_id)

    wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get rhmi -n ${RHMI_OPERATOR_NAMESPACE} | grep -q NAME" "rhmi installation CR to be created" "15m" "30"

    rhmi_name=$(get_rhmi_name)

    if [[ "${USE_CLUSTER_STORAGE}" == false ]]; then
        echo "Creating cluster resource quotas and AWS backup strategies"
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" create -f \
        "${CR_AWS_STRATEGIES_CONFIGMAP_FILE},${LB_CLUSTER_QUOTA_FILE},${CLUSTER_STORAGE_QUOTA_FILE}"
    fi

    echo "Patching RHMI CR"
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch rhmi "${rhmi_name}" -n ${RHMI_OPERATOR_NAMESPACE} \
        --type=merge -p "{\"spec\":{\"useClusterStorage\": \"${USE_CLUSTER_STORAGE}\", \"selfSignedCerts\": ${SELF_SIGNED_CERTS:-true} }}"

    # Change alerting email address is ALERTING_EMAIL_ADDRESS variable is set
    if [[ -n "${ALERTING_EMAIL_ADDRESS:-}" ]]; then
        echo "Changing alerting email address to: ${ALERTING_EMAIL_ADDRESS}"
        csv_name=$(oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get csv -n ${RHMI_OPERATOR_NAMESPACE} | grep integreatly-operator | awk '{print $1}')
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch csv "${csv_name}" -n ${RHMI_OPERATOR_NAMESPACE} \
            --type=json -p "[{\"op\": \"replace\", \"path\": \"/spec/install/spec/deployments/0/spec/template/spec/containers/0/env/4/value\", \"value\": \"${ALERTING_EMAIL_ADDRESS}\" }]"
    fi
    # Create a valid SMTP secret if SENDGRID_API_KEY variable is exported
    if [[ -n "${SENDGRID_API_KEY:-}" ]]; then
        infra_id=$(get_infra_id)
        echo "Creating SMTP secret with Sendgrid API key with ID: ${infra_id}"
        smtp-service create "${infra_id}" | oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" create -n "${RHMI_OPERATOR_NAMESPACE}" -f -
    else
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" create secret generic redhat-rhmi-smtp -n "${RHMI_OPERATOR_NAMESPACE}" \
            --from-literal=host=smtp.example.com \
            --from-literal=username=dummy \
            --from-literal=password=dummy \
            --from-literal=port=587 \
            --from-literal=tls=true
    fi
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" create secret generic redhat-rhmi-pagerduty -n ${RHMI_OPERATOR_NAMESPACE} \
        --from-literal=serviceKey=dummykey
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" create secret generic redhat-rhmi-deadmanssnitch -n ${RHMI_OPERATOR_NAMESPACE} \
        --from-literal=url=https://dms.example.com

    if [[ "${PATCH_CR_AWS_CM}" == true ]]; then
        echo "Patching Cloud Resources AWS Strategies Config Map"
        wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get configMap cloud-resources-aws-strategies -n ${RHMI_OPERATOR_NAMESPACE} | grep -q cloud-resources-aws-strategies" "cloud-resources-aws-strategies ready" "5m" "20"
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch configMap cloud-resources-aws-strategies -n "${RHMI_OPERATOR_NAMESPACE}" --type='json' -p '[{"op": "add", "path": "/data/_network", "value":"{ \"production\": { \"createStrategy\": { \"CidrBlock\": \"'10.1.0.0/23'\" } } }"}]'
    fi

    wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get rhmi ${rhmi_name} -n ${RHMI_OPERATOR_NAMESPACE} -o json | jq -r .status.stages.\\\"solution-explorer\\\".phase | grep -q completed" "rhmi installation" "90m" "300"
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get rhmi "${rhmi_name}" -n ${RHMI_OPERATOR_NAMESPACE} -o json | jq -r '.status.stages'
}

delete_cluster() {
    local cluster_id
    local infra_id
    local cluster_region

    cluster_id=$(get_cluster_id)
    infra_id=$(get_infra_id)

    # Delete SMTP API key if SENDGRID_API_KEY is defined and infra_id is not empty
    # infra_id would be empty if the cluster was not fully provisioned
    if [[ -n "${SENDGRID_API_KEY:-}" ]] && [[ -n "${infra_id:-}" ]]; then
        echo "Deleting Sendgrid sub user and API key with ID: ${infra_id}"
        smtp-service delete "${infra_id}"
    fi

    echo "Deleting the cluster with ID: ${cluster_id}"
    ocm delete "/api/clusters_mgmt/v1/clusters/${cluster_id}"

    # Use cluster-service to cleanup AWS resources
    if [[ $(is_byoc_cluster) == true ]] && [[ -n "${infra_id:-}" ]]; then
        check_aws_credentials_exported

        cluster_region=$(get_cluster_region)
        echo "Cleaning up RHMI AWS resources for the cluster with infra ID: ${infra_id}, region: ${cluster_region}, AWS Account ID: ${AWS_ACCOUNT_ID}"
        cluster-service cleanup "${infra_id}" --region="${cluster_region}" --dry-run=false
    fi
}

upgrade_cluster() {
    local cluster_id
    cluster_id=$(get_cluster_id)

    upgradeAvailable=$(ocm get cluster "${cluster_id}" | jq -r .metrics.upgrade.available)
    
    if [[ $upgradeAvailable == true ]]; then
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" adm upgrade --to-latest=true
        sleep 600 # waiting 10 minutes to allow for '.metrics.upgrade.state' to appear
        wait_for "ocm get cluster ${cluster_id} | jq -r .metrics.upgrade.state | grep -q completed" "OpenShift upgrade" "90m" "300"
    else
        echo "No upgrade available for cluster with id: ${cluster_id}"
    fi
}

get_cluster_logs() {
    ocm get "/api/clusters_mgmt/v1/clusters/$(get_cluster_id)/logs/hive" | jq -r .content | tee "${CLUSTER_LOGS_FILE}"
}

get_cluster_id() {
    jq -r .id < "${CLUSTER_DETAILS_FILE}"
}

get_cluster_region() {
    jq -r .region.id < "${CLUSTER_DETAILS_FILE}"
}

is_byoc_cluster() {
    jq -r .byoc < "${CLUSTER_DETAILS_FILE}"
}

get_rhmi_name() {
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get rhmi -n "${RHMI_OPERATOR_NAMESPACE}" -o jsonpath='{.items[*].metadata.name}'
}

get_infra_id() {
    # 'values' function evaluates null as empty string
    jq -r '.infra_id | values' < "${CLUSTER_DETAILS_FILE}"
}

send_cluster_create_request() {
    local cluster_details
    cluster_details=$(ocm post /api/clusters_mgmt/v1/clusters --body="${CLUSTER_CONFIGURATION_FILE}" | jq -r | tee "${CLUSTER_DETAILS_FILE}")
    if [[ -z "${cluster_details:-}" ]]; then
        printf "Something went wrong with cluster create request\n"
        exit 1
    fi
}

wait_for() {
    local command="${1}"
    local description="${2}"
    local timeout="${3}"
    local interval="${4}"

    printf "Waiting for %s for %s...\n" "${description}" "${timeout}"
    timeout --foreground "${timeout}" bash -c "
    until ${command}
    do
        printf \"Waiting for %s... Trying again in ${interval}s\n\" \"${description}\"
        sleep ${interval}
    done
    "
    printf "%s finished!\n" "${description}"
}

save_cluster_credentials() {
    local cluster_id="${1}"
    local api_url
    local console_url

    api_url=$(jq .api.url < "${CLUSTER_DETAILS_FILE}")
    # Update cluster details (with master & console URL)
    ocm get "/api/clusters_mgmt/v1/clusters/${cluster_id}" | jq -r > "${CLUSTER_DETAILS_FILE}"
    # Create kubeconfig file & save admin credentials
    ocm get "/api/clusters_mgmt/v1/clusters/${cluster_id}/credentials" | jq -r .kubeconfig > "${CLUSTER_KUBECONFIG_FILE}"

    wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get route console -n openshift-console -o jsonpath='{.spec.host}' | grep -q console" "OpenShift console to be ready" "10m" "20"
    console_url=\"https://$(oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get route console -n openshift-console -o jsonpath='{.spec.host}')\"

    ocm get "/api/clusters_mgmt/v1/clusters/${cluster_id}/credentials" | jq -r ".admin | .api_url = ${api_url} | .console_url = ${console_url}" > "${CLUSTER_CREDENTIALS_FILE}"
}

get_expiration_timestamp() {
    local os
    os=$(uname)

    if [[ $os = Linux ]]; then
        date --date="${1:-4} hour" "+%FT%TZ"
    elif [[ $os = Darwin ]]; then
        date -v+"${1:-4}"H "+%FT%TZ"
    fi
}

update_configuration_with_aws_credentials() {
    local updated_configuration

    updated_configuration=$(jq ".byoc = true | .aws.access_key_id = \"${AWS_ACCESS_KEY_ID}\" | .aws.secret_access_key = \"${AWS_SECRET_ACCESS_KEY}\" | .aws.account_id = \"${AWS_ACCOUNT_ID}\"" < "${CLUSTER_CONFIGURATION_FILE}")
    printf "%s" "${updated_configuration}" > "${CLUSTER_CONFIGURATION_FILE}"
}

update_configuration_with_openshift_version() {
    local updated_configuration

    updated_configuration=$(jq ".version = {\"kind\": \"VersionLink\",\"id\": \"openshift-v${OPENSHIFT_VERSION}\", \"href\": \"/api/clusters_mgmt/v1/versions/openshift-v${OPENSHIFT_VERSION}\"}" < "${CLUSTER_CONFIGURATION_FILE}")
    printf "%s" "${updated_configuration}" > "${CLUSTER_CONFIGURATION_FILE}"
}

display_help() {
    printf \
"Usage: %s <command>

Commands:
==========================================================================================
create_cluster_configuration_file - create cluster.json
------------------------------------------------------------------------------------------
Optional exported variables:
- OCM_CLUSTER_LIFESPAN              How many hours should cluster stay until it's deleted?
- OCM_CLUSTER_NAME                  e.g. my-cluster (lowercase, numbers, hyphens)
- OCM_CLUSTER_REGION                e.g. eu-west-1
- BYOC                              true/false (default: false)
- OPENSHIFT_VERSION                 to get OpenShift versions, run: ocm cluster versions
- PRIVATE                           Cluster's API and router will be private
==========================================================================================
create_cluster                    - spin up OSD cluster
==========================================================================================
install_rhmi                      - install RHMI using addon-type installation
------------------------------------------------------------------------------------------
Optional exported variables:
- USE_CLUSTER_STORAGE               true/false - use OpenShift/AWS storage (default: true)
- SENDGRID_API_KEY                  a token for creating SMTP secret
- ALERTING_EMAIL_ADDRESS            email address for receiving alert notifications
- SELF_SIGNED_CERTS                 true/false - cluster certificate can be invalid
- PATCH_CR_AWS_CM                   true/false - set to true from 2.5.0 (standalone VPC)
==========================================================================================
upgrade_cluster                   - upgrade OSD cluster to latest version (if available)
==========================================================================================
delete_cluster                    - delete RHMI product & OSD cluster
Optional exported variables:
- SENDGRID_API_KEY                  a token for creating SMTP secret
- AWS_ACCOUNT_ID
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
==========================================================================================
get_cluster_logs                  - get logs from hive and save them to ${CLUSTER_LOGS_FILE}
==========================================================================================
" "${0}"
}

main() {
    # Create a folder for storing cluster details
    mkdir -p "${OCM_DIR}"

    while :
    do
        case "${1:-}" in
        create_cluster_configuration_file)
            create_cluster_configuration_file
            exit 0
            ;;
        create_cluster)
            create_cluster
            exit 0
            ;;
        install_rhmi)
            install_rhmi
            exit 0
            ;;
        delete_cluster)
            delete_cluster
            exit 0
            ;;
        upgrade_cluster)
            upgrade_cluster
            exit 0
            ;;
        get_cluster_logs)
            get_cluster_logs
            exit 0
            ;;
        -h | --help)
            display_help
            exit 0
            ;;
        -*)
            echo "Error: Unknown option: ${1}" >&2
            exit 1
            ;;
        *)
            echo "Error: Unknown command: ${1}" >&2
            exit 1
            ;;
        esac
    done
}

main "${@}"
