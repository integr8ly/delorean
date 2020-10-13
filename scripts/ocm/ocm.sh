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
readonly ERROR_CREATING_SECRET=" secret was not created. This could be caused by unstable connection between the client and OpenShift cluster"

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
    local cluster_display_name
    local cluster_name_length

    : "${OCM_CLUSTER_LIFESPAN:=4}"
    : "${OCM_CLUSTER_NAME:=rhmi-$(date +"%y%m%d-%H%M")}"
    : "${OCM_CLUSTER_REGION:=eu-west-1}"
    : "${BYOC:=false}"
    : "${OPENSHIFT_VERSION:=}"
    : "${PRIVATE:=false}"
    : "${MULTI_AZ:=false}"

    timestamp=$(get_expiration_timestamp "${OCM_CLUSTER_LIFESPAN}")

    if [ "${PRIVATE}" = true ]; then
        listening="internal"
    fi

    # Set cluster display name (a name that's visible in OCM UI)
    cluster_display_name="${OCM_CLUSTER_NAME}"
    cluster_name_length=$(echo -n "${OCM_CLUSTER_NAME}" | wc -c | xargs)

    # Limit for a cluster name is 15 characters - shorten it if it's longer
    if [ "${cluster_name_length}" -gt 15 ]; then
        OCM_CLUSTER_NAME="${OCM_CLUSTER_NAME:0:15}"
        # Remove the last character from a cluster name if it's "-"
        OCM_CLUSTER_NAME="${OCM_CLUSTER_NAME%-}"
    fi

    jq ".expiration_timestamp = \"${timestamp}\" | .name = \"${OCM_CLUSTER_NAME}\" | .display_name = \"${cluster_display_name}\" | .region.id = \"${OCM_CLUSTER_REGION}\" | .api.listening = \"${listening}\"" \
        < "${CLUSTER_TEMPLATE_FILE}" \
        > "${CLUSTER_CONFIGURATION_FILE}"
	
    if [ "${BYOC}" = true ]; then
        check_aws_credentials_exported
        update_configuration "aws"
    fi

    if [[ -n "${OPENSHIFT_VERSION}" ]]; then
        update_configuration "openshift_version"
    fi

    if [[ "${MULTI_AZ}" = true ]]; then
        update_configuration "multi_az"
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

    wait_for "ocm get /api/clusters_mgmt/v1/clusters/${cluster_id}/status | jq -r .state | grep -q ready" "cluster creation" "180m" "300"
    wait_for "ocm get /api/clusters_mgmt/v1/clusters/${cluster_id}/credentials | jq -r .admin | grep -q admin" "fetching cluster credentials" "10m" "30"

    save_cluster_credentials "${cluster_id}"


    printf "Login credentials: \n%s\n" "$(jq -r < "${CLUSTER_CREDENTIALS_FILE}")"
    printf "Log in to the OSD cluster using oc:\noc login --server=%s --username=kubeadmin --password=%s\n" "$(jq -r .api.url < "${CLUSTER_DETAILS_FILE}")" "$(jq -r .password < "${CLUSTER_CREDENTIALS_FILE}")"
}

install_addon() {
    local cluster_id
    local rhmi_name
    local infra_id
    local addon_id
    local completion_phase

    addon_id="${1}"
    completion_phase="${2}"

    : "${USE_CLUSTER_STORAGE:=true}"
    : "${PATCH_CR_AWS_CM:=true}"
    cluster_id=$(get_cluster_id)

    echo "Applying RHMI Addon on a cluster with ID: ${cluster_id}"
    echo "{\"addon\":{\"id\":\"${addon_id}\"}}" | ocm post "/api/clusters_mgmt/v1/clusters/${cluster_id}/addons"

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
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch rhmi "${rhmi_name}" -n ${RHMI_OPERATOR_NAMESPACE} \
            --type=merge -p "{\"spec\":{ \"alertingEmailAddress\": \"${ALERTING_EMAIL_ADDRESS}\"}}"
    fi

    create_secrets

    if [[ "${PATCH_CR_AWS_CM}" == true ]]; then
        echo "Patching Cloud Resources AWS Strategies Config Map"
        wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get configMap cloud-resources-aws-strategies -n ${RHMI_OPERATOR_NAMESPACE} | grep -q cloud-resources-aws-strategies" "cloud-resources-aws-strategies ready" "5m" "20"
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch configMap cloud-resources-aws-strategies -n "${RHMI_OPERATOR_NAMESPACE}" --type='json' -p '[{"op": "add", "path": "/data/_network", "value":"{ \"production\": { \"createStrategy\": { \"CidrBlock\": \"'10.1.0.0/23'\" } } }"}]'
    fi

    wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get rhmi ${rhmi_name} -n ${RHMI_OPERATOR_NAMESPACE} -o json | jq -r ${completion_phase} | grep -q completed" "rhmi installation" "90m" "300"
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get rhmi "${rhmi_name}" -n ${RHMI_OPERATOR_NAMESPACE} -o json | jq -r '.status.stages'
}

install_rhmi() {
    install_addon "rhmi" ".status.stages.\\\"solution-explorer\\\".phase"
}

install_managed_api() {
    install_addon "managed-api-service" ".status.stages.products.phase"
}


delete_cluster() {
    local cluster_id
    local infra_id
    local cluster_region

    cluster_id=$(get_cluster_id)
    infra_id=$(get_infra_id)

    echo "Deleting the cluster with ID: ${cluster_id}"
    ocm delete "/api/clusters_mgmt/v1/clusters/${cluster_id}"

    # Use cluster-service to cleanup AWS resources
    if [[ $(is_byoc_cluster) == true ]] && [[ -n "${infra_id:-}" ]]; then
        check_aws_credentials_exported

        cluster_region=$(get_cluster_region)
        echo "Cleaning up RHMI AWS resources for the cluster with infra ID: ${infra_id}, region: ${cluster_region}, AWS Account ID: ${AWS_ACCOUNT_ID}"
        cluster-service cleanup "${infra_id}" --region="${cluster_region}" --dry-run=false --watch
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
    jq -r .ccs.enabled < "${CLUSTER_DETAILS_FILE}"
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
    ocm get "/api/clusters_mgmt/v1/clusters/${cluster_id}/credentials" | jq -r ".admin | .api_url = $(jq .api.url < "${CLUSTER_DETAILS_FILE}") | .console_url = $(jq .console.url < "${CLUSTER_DETAILS_FILE}")" > "${CLUSTER_CREDENTIALS_FILE}"
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

update_configuration() {
    local param="${1}"
    local updated_configuration

    case $param in

    aws)
        updated_configuration=$(jq ".ccs.enabled = true | .aws.access_key_id = \"${AWS_ACCESS_KEY_ID}\" | .aws.secret_access_key = \"${AWS_SECRET_ACCESS_KEY}\" | .aws.account_id = \"${AWS_ACCOUNT_ID}\"" < "${CLUSTER_CONFIGURATION_FILE}")
        ;;

    openshift_version)
        updated_configuration=$(jq ".version = {\"kind\": \"VersionLink\",\"id\": \"openshift-v${OPENSHIFT_VERSION}\", \"href\": \"/api/clusters_mgmt/v1/versions/openshift-v${OPENSHIFT_VERSION}\"}" < "${CLUSTER_CONFIGURATION_FILE}")
        ;;

    multi_az)
        updated_configuration=$(jq ".multi_az = true | .nodes.compute = 9 | .nodes.compute_machine_type.id = \"r5.xlarge\"" < "${CLUSTER_CONFIGURATION_FILE}")
        ;;

    *)
        echo "Error: Invalid parameter: '${param}' passed to a function '${FUNCNAME[0]}'" >&2
        exit 1
        ;;
    esac

    printf "%s" "${updated_configuration}" > "${CLUSTER_CONFIGURATION_FILE}"
}

create_secrets() {
    local secrets
    secrets=$(oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get secrets -n "${RHMI_OPERATOR_NAMESPACE}" || true)

    # Pagerduty secret
    if ! grep -q pagerduty <<< "${secrets}"; then
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" create secret generic redhat-rhmi-pagerduty -n ${RHMI_OPERATOR_NAMESPACE} \
            --from-literal=serviceKey=dummykey \
            || echo "Pagerduty ${ERROR_CREATING_SECRET}"
    fi

    # DMS secret
    if ! grep -q deadmanssnitch <<< "${secrets}"; then
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" create secret generic redhat-rhmi-deadmanssnitch -n ${RHMI_OPERATOR_NAMESPACE} \
            --from-literal=url=https://dms.example.com \
            || echo "DMS ${ERROR_CREATING_SECRET}"
    fi

    # Keep trying creating secrets until all of them are present in RHMI operator namespace
    # SMTP Secret should be automatically created (and deleted) by a Sendgrid Service
    # https://gitlab.cee.redhat.com/service/ocm-sendgrid-service
    if [[ $(oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get secrets -n ${RHMI_OPERATOR_NAMESPACE} | grep -cE "redhat-rhmi-((.*pagerduty|.*deadmanssnitch))" || true) != 2 ]]; then
        create_secrets
    fi
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
- BYOC                              Cloud Customer Subscription: true/false (default: false)
- OPENSHIFT_VERSION                 to get OpenShift versions, run: ocm cluster versions
- PRIVATE                           Cluster's API and router will be private
- MULTI_AZ                          true/false (default: false)
==========================================================================================
create_cluster                    - spin up OSD cluster
==========================================================================================
install_rhmi                      - install RHMI using addon-type installation
==========================================================================================
install_managed_api               - install Managed API Service using addon-type installation
------------------------------------------------------------------------------------------
Optional exported variables:
- USE_CLUSTER_STORAGE               true/false - use OpenShift/AWS storage (default: true)
- ALERTING_EMAIL_ADDRESS            email address for receiving alert notifications
- SELF_SIGNED_CERTS                 true/false - cluster certificate can be invalid
==========================================================================================
upgrade_cluster                   - upgrade OSD cluster to latest version (if available)
==========================================================================================
delete_cluster                    - delete RHMI product & OSD cluster
Optional exported variables:
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
        install_managed_api)
            install_managed_api
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
