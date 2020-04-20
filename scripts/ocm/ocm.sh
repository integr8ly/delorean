#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly OCM_DIR="${REPO_DIR}/ocm"

readonly AWS_CREDENTIALS_FILE="${OCM_DIR}/aws.json"
readonly CLUSTER_KUBECONFIG_FILE="${OCM_DIR}/cluster.kubeconfig"
readonly CLUSTER_CONFIGURATION_FILE="${OCM_DIR}/cluster.json"
readonly CLUSTER_DETAILS_FILE="${OCM_DIR}/cluster-details.json"
readonly CLUSTER_CREDENTIALS_FILE="${OCM_DIR}/cluster-credentials.json"
readonly CLUSTER_LOGS_FILE="${OCM_DIR}/cluster.log"

readonly RHMI_OPERATOR_NAMESPACE="redhat-rhmi-operator"

readonly ERROR_MISSING_AWS_ENV_VARS="ERROR: Not all required AWS environment are set. Please make sure you've exported all following env vars:"
readonly ERROR_MISSING_AWS_JSON="ERROR: ${AWS_CREDENTIALS_FILE} file does not exist. Please run 'make ocm/aws/create_access_key' first"
readonly ERROR_MISSING_CLUSTER_JSON="ERROR: ${CLUSTER_CONFIGURATION_FILE} file does not exist. Please run 'make ocm/cluster.json' first"
readonly PROMPT_ACCESS_KEY_LIMIT="WARNING
Number of access keys for user osdCcsAdmin has reached its limit (2).
Do you want to delete latest generated key and create a new one? (y/n): "

create_access_key() {
    if [[ -z "${AWS_ACCOUNT_ID:-}" || -z "${AWS_SECRET_ACCESS_KEY:-}" || -z "${AWS_ACCESS_KEY_ID:-}" ]]; then
        printf "%s\n" "${ERROR_MISSING_AWS_ENV_VARS}"
        printf "AWS_ACCOUNT_ID='%s'\n" "${AWS_ACCOUNT_ID:-}"
        printf "AWS_ACCESS_KEY_ID='%s'\n" "${AWS_ACCESS_KEY_ID:-}"
        printf "AWS_SECRET_ACCESS_KEY='%s'\n" "${AWS_SECRET_ACCESS_KEY:-}"
        exit 1
    fi

    local available_access_keys
    local number_of_access_keys_present
    local latest_generated_access_key_id

    available_access_keys=$(aws iam list-access-keys --user-name osdCcsAdmin | jq -r '.AccessKeyMetadata')
    number_of_access_keys_present=$(jq length <<< "${available_access_keys}")

    if [[ "${number_of_access_keys_present}" = 2 ]]; then
        read -rp "${PROMPT_ACCESS_KEY_LIMIT}" user_input
        if [[ "${user_input}" = "y" ]]; then
            latest_generated_access_key_id=$(get_latest_generated_access_key_id "${available_access_keys}")
            printf "Deleting following access key id: %s\n" "${latest_generated_access_key_id}"
            aws iam delete-access-key --user-name osdCcsAdmin --access-key-id "${latest_generated_access_key_id}"
        else
            printf 'Access key was not generated.\n'
            exit 1
        fi
    fi
    printf "Generating new access key:\n"
    aws iam create-access-key --user-name osdCcsAdmin | jq -r .AccessKey | tee "${AWS_CREDENTIALS_FILE}"
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

    : "${OCM_CLUSTER_LIFESPAN:=4}"
    : "${OCM_CLUSTER_NAME:=rhmi-$(date +"%y%m%d-%H%M")}"
    : "${OCM_CLUSTER_REGION:=eu-west-1}"
    : "${BYOC:=false}"
    : "${OPENSHIFT_VERSION:=}"

    timestamp=$(get_expiration_timestamp "${OCM_CLUSTER_LIFESPAN}")

    jq ".expiration_timestamp = \"${timestamp}\" | .name = \"${OCM_CLUSTER_NAME}\" | .region.id = \"${OCM_CLUSTER_REGION}\"" \
        < "${REPO_DIR}/templates/ocm/cluster-template.json" \
        > "${CLUSTER_CONFIGURATION_FILE}"
	
    if [ "${BYOC}" = true ]; then
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

    send_cluster_create_request
    cluster_id=$(get_cluster_id)

    wait_for "ocm get /api/clusters_mgmt/v1/clusters/${cluster_id}/status | jq -r .state | grep -q ready" "cluster creation" "120m" "300"
    wait_for "ocm get /api/clusters_mgmt/v1/clusters/${cluster_id}/credentials | jq -r .admin | grep -q admin" "fetching cluster credentials" "10m" "30"

    save_cluster_credentials "${cluster_id}"
    printf "Console URL: %s\nLogin credentials: \n%s\n" "$(jq -r .console.url < "${CLUSTER_DETAILS_FILE}")" "$(jq -r < "${CLUSTER_CREDENTIALS_FILE}")"
}

install_rhmi() {
    local cluster_id
    local rhmi_name
    local infra_id
    local csv_name

    cluster_id=$(get_cluster_id)

    echo '{"addon":{"id":"rhmi"}}' | ocm post "/api/clusters_mgmt/v1/clusters/${cluster_id}/addons"

    wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get rhmi -n ${RHMI_OPERATOR_NAMESPACE} | grep -q NAME" "installation CR created" "10m" "30"

    rhmi_name=$(get_rhmi_name)

    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch rhmi "${rhmi_name}" -n ${RHMI_OPERATOR_NAMESPACE} \
        --type=merge -p "{\"spec\":{\"useClusterStorage\": \"${USE_CLUSTER_STORAGE:-true}\", \"selfSignedCerts\": ${SELF_SIGNED_CERTS:-true} }}"

    # Change alerting email address is ALERTING_EMAIL_ADDRESS variable is set
    if [[ -n "${ALERTING_EMAIL_ADDRESS:-}" ]]; then
        csv_name=$(oc get csv -n ${RHMI_OPERATOR_NAMESPACE} | grep integreatly-operator | awk '{print $1}')
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch csv "${csv_name}" -n ${RHMI_OPERATOR_NAMESPACE} \
            --type=json -p "[{\"op\": \"replace\", \"path\": \"/spec/install/spec/deployments/0/spec/template/spec/containers/0/env/4/value\", \"value\": \"${ALERTING_EMAIL_ADDRESS}\" }]"
    fi
    # Create a valid SMTP secret if SENDGRID_API_KEY variable is exported
    if [[ -n "${SENDGRID_API_KEY:-}" ]]; then
        infra_id=$(get_infra_id)
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

    wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get rhmi ${rhmi_name} -n ${RHMI_OPERATOR_NAMESPACE} -o json | jq -r .status.stages.\\\"solution-explorer\\\".phase | grep -q completed" "rhmi installation" "90m" "300"
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get rhmi "${rhmi_name}" -n ${RHMI_OPERATOR_NAMESPACE} -o json | jq -r '.status.stages'
}

delete_cluster() {
    local rhmi_name
    local cluster_id
    local infra_id

    rhmi_name=$(get_rhmi_name || true)

    # Delete SMTP API key if SENDGRID_API_KEY variable is exported
    if [[ -n "${SENDGRID_API_KEY:-}" ]]; then
        infra_id=$(get_infra_id)
        # Check if infra_id is not empty (would happen when the cluster-details.json file is not updated after creating a cluster)
        if [[ $infra_id ]]; then
            smtp-service delete "${infra_id}"
        fi
    fi
    
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" delete rhmi "${rhmi_name}" -n "${RHMI_OPERATOR_NAMESPACE}" || true

    cluster_id=$(get_cluster_id)
    ocm delete "/api/clusters_mgmt/v1/clusters/${cluster_id}"
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
    # Update cluster details (with master & console URL)
    ocm get "/api/clusters_mgmt/v1/clusters/${cluster_id}" | jq -r > "${CLUSTER_DETAILS_FILE}"
    # Create kubeconfig file & save admin credentials
    ocm get "/api/clusters_mgmt/v1/clusters/${cluster_id}/credentials" | jq -r .kubeconfig > "${CLUSTER_KUBECONFIG_FILE}"
    ocm get "/api/clusters_mgmt/v1/clusters/${cluster_id}/credentials" | jq -r .admin > "${CLUSTER_CREDENTIALS_FILE}"
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
    local access_key
    local secret_key
    local updated_configuration

    if [[ -z "${AWS_ACCOUNT_ID:-}" ]]; then
            printf "%s" "${ERROR_MISSING_AWS_ENV_VARS}"
            printf "AWS_ACCOUNT_ID='%s'\n" "${AWS_ACCOUNT_ID:-}"
            exit 1
    fi
    if ! [[ -e "${AWS_CREDENTIALS_FILE}" ]]; then
        printf "%s\n" "${ERROR_MISSING_AWS_JSON}"
        exit 1
    fi

    access_key=$(jq -r .AccessKeyId < "${AWS_CREDENTIALS_FILE}")
    secret_key=$(jq -r .SecretAccessKey < "${AWS_CREDENTIALS_FILE}")
    updated_configuration=$(jq ".byoc = true | .aws.access_key_id = \"${access_key}\" | .aws.secret_access_key = \"${secret_key}\" | .aws.account_id = \"${AWS_ACCOUNT_ID}\"" < "${CLUSTER_CONFIGURATION_FILE}")
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
create_access_key                 - create aws access key (required for BYOC-type cluster)
------------------------------------------------------------------------------------------
Required exported variables:
- AWS_ACCOUNT_ID
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
==========================================================================================
create_cluster_configuration_file - create cluster.json
------------------------------------------------------------------------------------------
Optional exported variables:
- OCM_CLUSTER_LIFESPAN              How many hours should cluster stay until it's deleted?
- OCM_CLUSTER_NAME                  e.g. my-cluster (lowercase, numbers, hyphens)
- OCM_CLUSTER_REGION                e.g. eu-west-1
- BYOC                              true/false (default: false)
- OPENSHIFT_VERSION                 to get OpenShift versions, run: ocm cluster versions
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
==========================================================================================
upgrade_cluster                   - upgrade OSD cluster to latest version (if available)
==========================================================================================
delete_cluster                    - delete RHMI product & OSD cluster
Optional exported variables:
- SENDGRID_API_KEY                  a token for creating SMTP secret
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
        create_access_key)
            create_access_key
            exit 0
            ;;
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