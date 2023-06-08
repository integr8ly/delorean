#!/usr/bin/env bash

# Script gather the following information from Cluster
# API calls being sent through RHOAM (?)
# user count
# tenant
# activedoc count
# backend count
# product count
# application count
# application plan count
# if they are using the developer portal
# policy created count
# Run script Example:  ./scripts/delorean/get-cluster-data.sh
# Need be logged into the Openshift cluster as kubeadmin before running this script

# Please note that following counts provided only for admin tenant:
# application_plans, services / products, backend_apis, active_docs, policies, authentication_providers/dev portal.
# Tenants access token can't be retrieved by script;
# provider_verification_key, that probably should be used instead of Tenant account token -  is not working, maybe 3scale bug.


oc_whoami=$(oc whoami)
if [[ -z "${oc_whoami}" ]]; then
  echo "Need be logged into the Openshift cluster before running this script. Please check: oc whoami. Exiting";
  exit
fi

logfile="./cluster_data_log.txt"
touch $logfile

prometheus_route=$(oc get route -n redhat-rhoam-observability |grep prometheus |awk '{print $2}')
threescale_admin_url=$(oc get route -n redhat-rhoam-3scale |grep 3scale-admin |awk '{print $2}')
threescale_admin_access_token=$(oc get secret system-seed -oyaml -n redhat-rhoam-3scale |grep ADMIN_ACCESS_TOKEN |awk '{print $2}' |base64 -d)
master_url=$(oc get route -n redhat-rhoam-3scale |grep master |awk '{print $2}')
master_access_token=$(oc get secret system-seed -oyaml -n redhat-rhoam-3scale |grep MASTER_ACCESS_TOKEN |awk '{print $2}' |base64 -d)
oc_server=$(oc whoami --show-server)

echo "**** Get cluster data - $(date) ****" |tee -a $logfile
echo "**** oc user: ${oc_whoami} | server: ${oc_server}" |tee -a $logfile

# ?? API calls being sent through RHOAM  /WIP
count=$(curl -s  "https://${prometheus_route}/metrics"  |grep redhat-rhoam-marin3r-ratelimit |grep net_conntrack_dialer_conn_attempted_total |awk '{print $2}')
echo "* API calls being sent through RHOAM: ${count}" |tee -a $logfile

# user count
# 3scale API name: Account List
count=$(curl -sb  GET "https://${master_url}/admin/api/accounts.xml?access_token=${master_access_token}" |grep -o "<username>" |wc -l)
echo "* Users count (for all tenants): ${count}" |tee -a $logfile


# backend count
# 3scale API name: Backend List
count=$(curl -sb  GET "https://${threescale_admin_url}/admin/api/backend_apis.json?access_token=${threescale_admin_access_token}" |grep -o system_name |wc -l)
echo "* Backend count (for admin): ${count}" |tee -a  $logfile

# product count
# 3scale API name: Service List
count=$(curl -sb  GET "https://${threescale_admin_url}/admin/api/services.json?access_token=${threescale_admin_access_token}" |grep -o system_name |wc -l)
echo "* Product count (for admin): ${count}" |tee -a $logfile

# application plan count - for all services
# 3scale API name: Application Plan List (all services)
count=$(curl -sb  GET "https://${threescale_admin_url}/admin/api/application_plans.json?access_token=${threescale_admin_access_token}" |grep -o system_name |wc -l)
echo "* Application plan count (for admin): ${count}" |tee -a $logfile

# applications count - for all services
# 3scale API name: Application List (all services)
count=$(curl -sb  GET "https://${master_url}/admin/api/applications.json?access_token=${master_access_token}" |grep -o user_key |wc -l)
echo "* Application count (for all tenants): ${count}" |tee -a $logfile

# activedoc count
# 3scale API name: ActiveDocs Spec List
count=$(curl -sb  GET "https://${threescale_admin_url}/admin/api/active_docs.json?access_token=${threescale_admin_access_token}" |grep -o system_name |wc -l)
echo "* Activedoc count (for admin): ${count}" |tee -a $logfile

# policy created count
# 3scale API name: APIcast Policy Registry List
count=$(curl -sb  GET "https://${threescale_admin_url}/admin/api/registry/policies.json?access_token=${threescale_admin_access_token}" |grep -o policy |wc -l)
echo "* Policy created count (for admin): ${count}" |tee -a $logfile

# Developer Portal
# 3scale API name: Authentication Providers Developer Portal List
count=$(curl -sb  GET "https://${threescale_admin_url}/admin/api/authentication_providers.json?access_token=${threescale_admin_access_token}" |grep -o system_name |wc -l)
echo "* Authentication Providers Developer Portal count: ${count}" |tee -a $logfile

# tenant
# 3scale API name: Account List
count=$(curl -sb  GET "https://${master_url}/admin/api/accounts.json?access_token=${master_access_token}" |grep -o admin_base_url |wc -l)
echo "* Tenant count: ${count}" |tee -a $logfile

echo "****  Please see logs in  ${logfile} ****" |tee -a $logfile
echo "" >> $logfile






