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
readonly CLUSTER_INSTALLATION_LOGS_FILE="${OCM_DIR}/cluster-installation.log"
readonly CLUSTER_SUBSCRIPTION_DETAILS_FILE="${OCM_DIR}/cluster-subscription.json"

readonly ERROR_MISSING_AWS_ENV_VARS="ERROR: Not all required AWS environment are set. Please make sure you've exported all following env vars:"
readonly ERROR_MISSING_CLUSTER_JSON="ERROR: ${CLUSTER_CONFIGURATION_FILE} file does not exist. Please run 'make ocm/cluster.json' first"
readonly ERROR_CREATING_SECRET=" secret was not created. This could be caused by unstable connection between the client and OpenShift cluster"
readonly ERROR_MISSING_CLUSTER_ID="ERROR: OCM_CLUSTER_ID was not specified"

readonly WARNING_CLUSTER_HEALTH_CHECK_FAILED="WARNING: Cluster was not reported as healthy. You might expect some issues while working with the cluster."

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
    : "${COMPUTE_NODES_COUNT:=}"

    timestamp=$(get_expiration_timestamp "${OCM_CLUSTER_LIFESPAN}")

    if [ "${PRIVATE}" = true ]; then
        listening="internal"
    fi

    # Set cluster display name (a name that's visible in OCM UI)
    cluster_display_name="${OCM_CLUSTER_NAME}"
    cluster_name_length=$(echo -n "${OCM_CLUSTER_NAME}" | wc -c | xargs)

    # Limit for a cluster name is 15 characters - shorten it if it's longer
    if [ "${cluster_name_length}" -gt 15 ]; then
        OCM_CLUSTER_NAME="${OCM_CLUSTER_NAME:0:11}${RANDOM:0:4}"
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

    if [[ -n "${COMPUTE_NODES_COUNT}" ]]; then
        update_configuration "compute_nodes_count"
    fi


    cat "${CLUSTER_CONFIGURATION_FILE}"
}

create_cluster() {
    local cluster_id

    if ! [[ -e "${CLUSTER_CONFIGURATION_FILE}" ]]; then
        printf "%s\n" "${ERROR_MISSING_CLUSTER_JSON}"
        exit 1
    fi

    : "${TIMEOUT_CLUSTER_CREATION:=180}"
    : "${TIMEOUT_CLUSTER_HEALTH_CHECK:=30}"

    echo "Sending a request to OCM to create an OSD cluster"
    send_cluster_create_request
    cluster_id=$(get_cluster_id)

    echo "Cluster ID: ${cluster_id}"

    wait_for "ocm get /api/clusters_mgmt/v1/clusters/${cluster_id}/status | jq -r .state | grep -q ready" "cluster creation" "${TIMEOUT_CLUSTER_CREATION}m" "300"
    wait_for "ocm get subs --parameter search=\"cluster_id = '${cluster_id}'\" | jq -r .items[0].metrics[0].health_state | grep -q healthy" "cluster to be healthy" "${TIMEOUT_CLUSTER_HEALTH_CHECK}m" "30" \
        || echo "${WARNING_CLUSTER_HEALTH_CHECK_FAILED}"
    wait_for "ocm get /api/clusters_mgmt/v1/clusters/${cluster_id}/credentials | jq -r .admin | grep -q admin" "fetching cluster credentials" "10m" "30"

    save_cluster_credentials "${cluster_id}"


    printf "Login credentials: \n%s\n" "$(jq -r < "${CLUSTER_CREDENTIALS_FILE}")"
    printf "Log in to the OSD cluster using oc:\noc login --server=%s --username=kubeadmin --password=%s\n" "$(jq -r .api.url < "${CLUSTER_DETAILS_FILE}")" "$(jq -r .password < "${CLUSTER_CREDENTIALS_FILE}")"

    echo "To log into the web console as kubeadmin use the link below:"
    base_domain=$(jq -r .console_url < "${CLUSTER_CREDENTIALS_FILE}" | cut -d '.' -f 3,4,5,6,7 | cut -d '/' -f 1)
    echo "https://oauth-openshift.apps.${base_domain}/login/kube:admin?then=%2Foauth%2Fauthorize%3Fclient_id%3Dconsole%26idp%3Dkubeadmin%26redirect_uri%3Dhttps%253A%252F%252Fconsole-openshift-console.apps.${base_domain}%252Fauth%252Fcallback%26response_type%3Dcode%26scope%3Duser%253Afull"
}

install_addon() {
    local cluster_id
    local rhmi_name
    local infra_id
    local addon_id
    local completion_phase
    local addon_payload

    addon_id="${1}"
    completion_phase="${2}"

    : "${USE_CLUSTER_STORAGE:=true}"
    : "${PATCH_CR_AWS_CM:=true}"
    : "${WAIT:=true}"
    : "${QUOTA:=20}"
    cluster_id=$(get_cluster_id)
    addon_payload="{\"addon\":{\"id\":\"${addon_id}\"}}"

    # Add mandatory "cidr-range" and "addon-managed-api-service" (quota) params with default value in case of rhoam (managed-api-service) addon 
    if [[ "${addon_id}" == "managed-api-service" ]]; then
    	addon_payload="{\"addon\":{\"id\":\"${addon_id}\"}, \"parameters\": { \"items\": [{\"id\": \"cidr-range\", \"value\": \"10.1.0.0/26\"}, {\"id\": \"addon-managed-api-service\", \"value\": \"${QUOTA}\"}] }}"
    fi

    echo "Applying ${addon_id} Add-on on a cluster with ID: ${cluster_id}"
    echo "${addon_payload}" | ocm post "/api/clusters_mgmt/v1/clusters/${cluster_id}/addons"

    wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get rhmi -n ${OPERATOR_NAMESPACE} | grep -q NAME" "rhmi installation CR to be created" "15m" "30"

    rhmi_name=$(get_rhmi_name)

    # Apply cluster resource quotas and AWS backup strategies only in case of RHMI installation (with AWS cloud resources)
    if [[ "${USE_CLUSTER_STORAGE}" == false && "${NS_PREFIX}" == "redhat-rhmi" ]]; then
        echo "Creating cluster resource quotas and AWS backup strategies"
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" -n "${OPERATOR_NAMESPACE}" create -f \
        "${CR_AWS_STRATEGIES_CONFIGMAP_FILE},${LB_CLUSTER_QUOTA_FILE},${CLUSTER_STORAGE_QUOTA_FILE}"
    fi

    echo "Patching RHMI CR"
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch rhmi "${rhmi_name}" -n "${OPERATOR_NAMESPACE}" \
        --type=merge -p "{\"spec\":{\"useClusterStorage\": \"${USE_CLUSTER_STORAGE}\", \"selfSignedCerts\": ${SELF_SIGNED_CERTS:-false} }}"

    # Change alerting email address is ALERTING_EMAIL_ADDRESS variable is set
    if [[ -n "${ALERTING_EMAIL_ADDRESS:-}" ]]; then
        echo "Changing alerting email address to: ${ALERTING_EMAIL_ADDRESS}"
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch rhmi "${rhmi_name}" -n "${OPERATOR_NAMESPACE}" \
            --type=merge -p "{\"spec\":{ \"alertingEmailAddress\": \"${ALERTING_EMAIL_ADDRESS}\"}}"
    fi

    create_secrets

    if [[ "${PATCH_CR_AWS_CM}" == true ]]; then
        echo "Patching Cloud Resources AWS Strategies Config Map"
        wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get configMap cloud-resources-aws-strategies -n ${OPERATOR_NAMESPACE} | grep -q cloud-resources-aws-strategies" "cloud-resources-aws-strategies ready" "5m" "20"
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" patch configMap cloud-resources-aws-strategies -n "${OPERATOR_NAMESPACE}" --type='json' -p '[{"op": "add", "path": "/data/_network", "value":"{ \"production\": { \"createStrategy\": { \"CidrBlock\": \"'10.1.0.0/23'\" } } }"}]'
    fi

    if [[ "${WAIT}" == true ]]; then
        wait_for "oc --kubeconfig ${CLUSTER_KUBECONFIG_FILE} get rhmi ${rhmi_name} -n ${OPERATOR_NAMESPACE} -o json | jq -r ${completion_phase} | grep -q completed" "rhmi installation" "90m" "300"
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get rhmi "${rhmi_name}" -n "${OPERATOR_NAMESPACE}" -o json | jq -r '.status.stages'
    fi
}

install_rhmi() {
    NS_PREFIX="redhat-rhmi"
    OPERATOR_NAMESPACE="${NS_PREFIX}-operator"
    install_addon "rhmi" ".status.stages.\\\"solution-explorer\\\".phase"
}

install_rhoam() {
    NS_PREFIX="redhat-rhoam"
    OPERATOR_NAMESPACE="${NS_PREFIX}-operator"
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
    if [[ $(is_ccs_cluster) == true ]] && [[ -n "${infra_id:-}" ]]; then
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
    ocm get cluster "$(get_cluster_id)/logs/install" | jq -r .content > "${CLUSTER_INSTALLATION_LOGS_FILE}"
    printf "Cluster installation logs saved to %s\n" "${CLUSTER_INSTALLATION_LOGS_FILE}"
    ocm get subscription "$(get_cluster_subscription_id)" | jq -r > "${CLUSTER_SUBSCRIPTION_DETAILS_FILE}"
    printf "Cluster subscription details saved to %s\n" "${CLUSTER_SUBSCRIPTION_DETAILS_FILE}"
}

get_cluster_id() {
    jq -r .id < "${CLUSTER_DETAILS_FILE}"
}

get_cluster_subscription_id() {
    jq -r .subscription.id < "${CLUSTER_DETAILS_FILE}"
}

get_cluster_region() {
    jq -r .region.id < "${CLUSTER_DETAILS_FILE}"
}

is_ccs_cluster() {
    jq -r .ccs.enabled < "${CLUSTER_DETAILS_FILE}"
}

get_rhmi_name() {
    oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get rhmi -n "${OPERATOR_NAMESPACE}" -o jsonpath='{.items[*].metadata.name}'
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
    " || return 1
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
        updated_configuration=$(jq ".multi_az = true | .nodes.compute = 9 | .nodes.compute_machine_type.id = \"m5.xlarge\"" < "${CLUSTER_CONFIGURATION_FILE}")
        ;;

    compute_nodes_count)
        updated_configuration=$(jq ".nodes.compute = ${COMPUTE_NODES_COUNT}" < "${CLUSTER_CONFIGURATION_FILE}")
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
    secrets=$(oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get secrets -n "${OPERATOR_NAMESPACE}" || true)

    # Create DMS secret if it's not present in the "redhat-rhmi-operator" namespace
    # This secret should be created only for RHMI (the creation of this secret is automated for RHOAM)
    # The rest of the secrets (SMTP, Pagerduty) are also auto-created
    if [[ $NS_PREFIX = "redhat-rhmi" ]] && ! grep -q deadmanssnitch <<< "${secrets}"; then
        oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" create secret generic ${NS_PREFIX}-deadmanssnitch -n ${OPERATOR_NAMESPACE} \
            --from-literal=url=https://dms.example.com \
            || echo "DMS ${ERROR_CREATING_SECRET}"
    fi

    if [[ $(oc --kubeconfig "${CLUSTER_KUBECONFIG_FILE}" get secrets -n ${OPERATOR_NAMESPACE} | grep -cE "${NS_PREFIX}-((.*smtp|.*pagerduty|.*deadmanssnitch))" || true) != 3 ]]; then
        printf "Waiting for DMS, Pagerduty and SMTP secrets to be created. Found the following secrets in %s namespace:\n%s\n" "${OPERATOR_NAMESPACE}" "${secrets}"
        create_secrets
    fi
}

display_help() {
    printf \
"Usage: %s <command>

Commands:
==========================================================================================================
create_cluster_configuration_file - create cluster.json
----------------------------------------------------------------------------------------------------------
Optional exported variables:
- OCM_CLUSTER_LIFESPAN              How many hours should cluster stay until it's deleted?
- OCM_CLUSTER_NAME                  e.g. my-cluster (lowercase, numbers, hyphens)
- OCM_CLUSTER_REGION                e.g. eu-west-1
- BYOC                              Cloud Customer Subscription: true/false (default: false)
- OPENSHIFT_VERSION                 to get OpenShift versions, run: ocm cluster versions
- PRIVATE                           Cluster's API and router will be private
- MULTI_AZ                          true/false (default: false)
- COMPUTE_NODES_COUNT               number of cluster's compute nodes (default: single-az: 4, multi-az: 9)
==========================================================================================================
create_cluster                    - spin up OSD cluster
----------------------------------------------------------------------------------------------------------
Optional exported variables:
- TIMEOUT_CLUSTER_CREATION          Timeout for cluster creation (in minutes, default: 180)
- TIMEOUT_CLUSTER_HEALTH_CHECK      Timeout for cluster health check (in minutes, default: 30)
==========================================================================================================
install_rhmi                      - install RHMI using addon-type installation
install_rhoam                     - install RHOAM using addon-type installation
------------------------------------------------------------------------------------------
Optional exported variables:
- USE_CLUSTER_STORAGE               true/false - use OpenShift/AWS storage (default: true)
- ALERTING_EMAIL_ADDRESS            email address for receiving alert notifications
- SELF_SIGNED_CERTS                 true/false - cluster certificate can be invalid
- WAIT                              true/false - wait for install to complete (default: true)
- QUOTA                             Ratelimit quota. Allowed values: 1,5,10,20,50 (default: 20)
==========================================================================================================
upgrade_cluster                   - upgrade OSD cluster to latest version (if available)
==========================================================================================================
delete_cluster                    - delete RHMI product & OSD cluster
Optional exported variables:
- AWS_ACCOUNT_ID
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
==========================================================================================================
get_cluster_logs                  - save cluster install logs and subscription details to ${OCM_DIR}
==========================================================================================================
save_cluster_credentials          - save cluster credentials to ./ocm folder
----------------------------------------------------------------------------------------------------------
Required variables:
- OCM_CLUSTER_ID                  - your cluster's ID
==========================================================================================================
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
        install_rhoam)
            install_rhoam
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
        save_cluster_credentials)
            if [[ -z "${OCM_CLUSTER_ID:-}" ]]; then
                printf "%s\n" "${ERROR_MISSING_CLUSTER_ID}"
                exit 1
            fi
            save_cluster_credentials "${OCM_CLUSTER_ID}"
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
