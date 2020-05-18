#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

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

namespace="${1:-}"
if [[ -z "${namespace}" ]]; then
	echo "USAGE: $0 <namespace>"
	exit 1
fi

server=$(oc whoami --show-server)

echo "Retrieving test cluster kubeconfig and console details from container 'test' in pod 'e2e' in namespace '${namespace}' on server '${server}'"

wait_for "oc get project/${namespace}" "find namespace" "10m" "100"
wait_for "oc get pod/e2e -n ${namespace}" "find e2e pod" "20m" "100"

#ToDo Wait for setup container to complete here

output="/tmp/kubeconfig.${namespace}"
mkdir -p ${output}

oc -n ${namespace} rsync e2e:/tmp/artifacts/installer/auth/kubeconfig ${output} -c test
oc -n ${namespace} rsync e2e:/tmp/artifacts/installer/auth/kubeadmin-password ${output} -c test

wait_for "oc --kubeconfig=${output}/kubeconfig get route console -n openshift-console" "find console route" "40m" "100"
echo "https://$(oc --kubeconfig=${output}/kubeconfig get route console -n openshift-console -o jsonpath="{.status.ingress[].host}")" >> ${output}/console-url

if [[ -s "${output}/kubeconfig" ]]; then
    echo "KUBECONFIG: ${output}/kubeconfig (Example: oc --kubeconfig=${output}/kubeconfig whoami)"
fi

if [[ -s "${output}/console-url" ]]; then
    echo "URL: $(cat ${output}/console-url)"
fi

if [[ -s "${output}/kubeadmin-password" ]]; then
    echo "Password: $(cat ${output}/kubeadmin-password)"
fi

#echo "Tail test logs: oc -n ${namespace} logs -f e2e -c test --tail=15"
