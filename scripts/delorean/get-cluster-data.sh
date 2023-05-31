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
# Need be logged into the Openshift cluster before running this script


oc_whoami=$(oc whoami)
if [[ -z "${oc_whoami}" ]]; then
  echo "Need be logged into the Openshift cluster before running this script. Please check: oc whoami. Exiting";
  exit
fi

logfile="./cluster_data_log.txt"
touch $logfile

threescale_url=$(oc get route -n redhat-rhoam-3scale |grep admin |awk '{print $2}')
#echo "threescale_url: ${threescale_url}"
threescale_access_token=$(oc get secret system-seed -oyaml -n redhat-rhoam-3scale |grep ADMIN_ACCESS_TOKEN |awk '{print $2}' |base64 -d)
prometheus_route=$(oc get route -n redhat-rhoam-observability |grep prometheus |awk '{print $2}')
#echo "prometheus_route: ${prometheus_route}"

echo "**** Get cluster data - $(date) ****" |tee -a $logfile

# ?? API calls being sent through RHOAM  /WIP
count=$(curl -s  "https://${prometheus_route}/metrics"  |grep redhat-rhoam-marin3r-ratelimit |grep net_conntrack_dialer_conn_attempted_total |awk '{print $2}')
echo "* API calls being sent through RHOAM: ${count}" |tee -a $logfile

# user count
# 3scale API name: User List
#account_id="3"
#count=$(curl -sb  GET "https://${threescale_url}/admin/api/accounts/${account_id}/users.json?access_token=${threescale_access_token}" |grep -o username |wc -l)
#echo "* User count: ${count}"

#Count of users CRs
count=$(oc get User --all-namespaces |wc -l)
if [[ $count -gt 1 ]]; then
  count=$(($count-2))
fi
echo "* User CRs count: ${count}" |tee -a $logfile

# backend count
# 3scale API name: Backend List
count=$(curl -sb  GET "https://${threescale_url}/admin/api/backend_apis.json?access_token=${threescale_access_token}" |grep -o system_name |wc -l)
echo "* Backend count: ${count}" |tee -a  $logfile

# product count
# 3scale API name: Service List
count=$(curl -sb  GET "https://${threescale_url}/admin/api/services.json?access_token=${threescale_access_token}" |grep -o system_name |wc -l)
echo "* Product count: ${count}" |tee -a $logfile

# application plan count - for all services
# 3scale API name: Application Plan List (all services)
count=$(curl -sb  GET "https://${threescale_url}/admin/api/application_plans.json?access_token=${threescale_access_token}" |grep -o system_name |wc -l)
echo "* Application plan count: ${count}" |tee -a $logfile

# applications count - for all services
# 3scale API name: Application List (all services)
count=$(curl -sb  GET "https://${threescale_url}/admin/api/application_plans.json?access_token=${threescale_access_token}" |grep -o system_name |wc -l)
echo "* Application count: ${count}" |tee -a $logfile

# activedoc count
# 3scale API name: ActiveDocs Spec List
count=$(curl -sb  GET "https://${threescale_url}/admin/api/active_docs.json?access_token=${threescale_access_token}" |grep -o system_name |wc -l)
echo "* Activedoc count: ${count}" |tee -a $logfile

# policy created count
# 3scale API name: APIcast Policy Registry List
count=$(curl -sb  GET "https://${threescale_url}/admin/api/registry/policies.json?access_token=${threescale_access_token}" |grep -o policy |wc -l)
echo "* Policy created count: ${count}" |tee -a $logfile

# Developer Portal
# 3scale API name: Authentication Providers Developer Portal List
count=$(curl -sb  GET "https://${threescale_url}/admin/api/authentication_providers.json?access_token=${threescale_access_token}" |grep -o system_name |wc -l)
echo "* Authentication Providers Developer Portal count: ${count}" |tee -a $logfile

# tenant
# Check Tenant CRs count in all namespaces
count=$(oc get Tenants --all-namespaces |wc  -l)
count=$(($count-1))
echo "* Tenant count: ${count}" |tee -a $logfile

echo "****  Please see logs in  ${logfile} ****" |tee -a $logfile
echo "" >> $logfile




