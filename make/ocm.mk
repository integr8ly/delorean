#OCM_IMAGE=registry.svc.ci.openshift.org/openshift/release:intly-golang-1.12
#OCM=docker run --rm -it -u 1000 -v "/home/mnairn/go/src/github.com/integr8ly/integreatly-operator:/integreatly-operator/" -w "/integreatly-operator" -v "${HOME}/tmp-home:/myhome:z" -e "HOME=/myhome" --entrypoint=/usr/local/bin/ocm ${OCM_IMAGE}
OCM=ocm
OCM_SH=scripts/ocm/ocm.sh

.PHONY: ocm/version
ocm/version:
	@${OCM} version

ocm/login: export OCM_URL := https://api.stage.openshift.com/
.PHONY: ocm/login
ocm/login:
	@${OCM} login --url=$(OCM_URL) --token=$(OCM_TOKEN)

.PHONY: ocm/whoami
ocm/whoami:
	@${OCM} whoami

.PHONY: ocm/execute
ocm/execute:
	${OCM} ${CMD}

.PHONY: ocm/get/current_account
ocm/get/current_account:
	@${OCM} get /api/accounts_mgmt/v1/current_account

.PHONY: ocm/cluster/list
ocm/cluster/list:
	@${OCM} cluster list

.PHONY: ocm/cluster/logs
ocm/cluster/logs:
	@${OCM_SH} get_cluster_logs

.PHONY: ocm/cluster/create
ocm/cluster/create:
	@${OCM_SH} create_cluster

.PHONY: ocm/install/rhmi-addon
ocm/install/rhmi-addon:
	@${OCM_SH} install_rhmi

.PHONY: ocm/cluster/delete
ocm/cluster/delete:
	@${OCM_SH} delete_cluster

.PHONY: ocm/cluster.json
ocm/cluster.json:
	@${OCM_SH} create_cluster_configuration_file

.PHONY: ocm/aws/create_access_key
ocm/aws/create_access_key:
	@${OCM_SH} create_access_key

.PHONY: ocm/cluster/upgrade
ocm/cluster/upgrade:
	@${OCM_SH} upgrade_cluster

.PHONY: ocm/help
ocm/help:
	@${OCM_SH} -h