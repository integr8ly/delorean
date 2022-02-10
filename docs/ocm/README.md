## Using `ocm` for installation of RHMI or Managed API Service (RHOAM)

If you want to test your changes on a cluster, the easiest solution would be to spin up OSD 4 cluster using `ocm`. If you want to spin up a cluster using CCS (Cloud Customer Subscription, previously called "BYOC"), follow the additional steps marked as **BYOC**.

### Prerequisites
* [OCM CLI](https://github.com/openshift-online/ocm-cli/releases)
* [jq](https://stedolan.github.io/jq/)
* [cluster-service](https://github.com/integr8ly/cluster-service/releases)
* [smtp-service](https://github.com/integr8ly/smtp-service/releases)
* :apple: (Mac users) - install `gtimeout` util and create a symbolic link (so it can be referenced as `timeout`): 
  * `brew install coreutils && sudo ln -s /usr/local/bin/gtimeout /usr/local/bin/timeout`

NOTE: Due to a change in how networking is configured for openshift in v4.4.6 (mentioned in the [cloud resource operator](https://github.com/integr8ly/cloud-resource-operator#supported-openshift-versions)) there is a limitation on the version of Openshift that RHMI can be installed on for BYOC (CCS) clusters.
Due to this change the use of integreatly-operator <= v2.4.0 on Openshift >= v4.4.6 is unsupported. Please use >= v2.5.0 of integreatly-operator for Openshift >= v4.4.6.

### Steps

1. Download the OCM CLI tool and add it to your PATH
2. Export [OCM_TOKEN](https://github.com/openshift-online/ocm-cli#log-in): `export OCM_TOKEN="<TOKEN_VALUE>"`
3. Login via OCM: 
```
make ocm/login
```

**BYOC/CCS**

If you want to set up a BYOC (CCS) cluster, you will need an AWS root account with no OSD cluster running in it. The AWS account needs an IAM user named `osdCcsAdmin` and this user needs the AdministratorAccess permission.

Once this user is in place and no other OSD cluster is running in the account, you will need the AWS credentials (`AWS_ACCOUNT_ID`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`) for the `osdCcsAdmin` user to use in your cluster request.

Export the AWS credentials for `osdCcsAdmin` user and set BYOC variable to `true`:
```
export AWS_ACCOUNT_ID=<REPLACE_ME>
export AWS_ACCESS_KEY_ID=<REPLACE_ME>
export AWS_SECRET_ACCESS_KEY=<REPLACE_ME>
export BYOC=true
```

**Multiple AZ**

If you want your BYOC cluster to span multiple availability zones then set the `MULTI_AZ` environment variable:

```
export MULTI_AZ=true
```

**BYOVPC/CCS**

If you want to set up a BYOVCP (Bring Your Own VPC) cluster, you will need an AWS account where a VPC can be pre-created. The AWS account needs an IAM user named `osdCcsAdmin` and this user needs the AdministratorAccess permission.

If you do not already have a VPC created go to aws and create a VPC using the VPC Wizard available from the VPC Dashboard.

Create a VPC with Public and Private Subnets. Note the CIDR Range ip as you will need this for your machine CIDR, or alternatively use 10.11.0.0/16 which will satisfy the default machine_cidr

Provide a name, chose an availability zone and update the subnets to suitable values. Chose an elastic ip allocation and chose CreateVPC

Export the following values
```bash
export BYOVPC=true
export CREATE_CUSTOM_VPC=<true/false>   # optional - let the script create a custom VPC for you
export PRIVATE_SUBNET_ID=<REPLACE_ME>   # not required if CREATE_CUSTOM_VPC=true
export PUBLIC_SUBNET_ID=<REPLACE_ME>    # not required if CREATE_CUSTOM_VPC=true
export AVAILABILITY_ZONES=<REPLACE_ME>  # not required if CREATE_CUSTOM_VPC=true
export OCM_CLUSTER_REGION=<REPLACE_ME>  # optional - default eu-west-1
```

To create the cluster.json correctly, please make sure you have exported the variables mentioned in BYOVPC/CCS section above also.

4. Create cluster template: `make ocm/cluster.json`

This command will generate `ocm/cluster.json` file with generated cluster name. This file will be used as a template to create your cluster via OCM CLI.
By default, it will set the expiration timestamp for a cluster for 4 hours, meaning your cluster will be automatically deleted after 4 hours after you generated this template. If you want to change the default timestamp, you can update it in `ocm/cluster.json` or delete the whole line from the file if you don't want your cluster to be deleted automatically at all.
If you are creating BYOVPC cluster and exported `CREATE_CUSTOM_VPC=true`, this command will also create a custom VPC from cloudforms template in specified AWS region

**/BYOC (CCS)**

If you exported AWS credentials (like described in the previous step), your cluster configuration will also include the AWS credentials and `ccs.enabled` field set to `true`.

**/Multiple AZ**

If you set `MULTI_AZ` to `true`, your cluster configuration will include the `multi_az` field set to true and the `nodes.compute_machine_type.id` field will be set to `r5.xlarge`. 

5. Create the cluster: `make ocm/cluster/create`

This command will send a request to [Red Hat OpenShift Cluster Manager](https://cloud.redhat.com/) to spin up your cluster and waits until it's ready. You can see the details of your cluster in `ocm/cluster-details.json` file

6. Once your cluster is ready, OpenShift Console URL will be printed out together with the `kubeadmin` user & password. These are also saved to `ocm/cluster-credentials.json` file. Also there will be `ocm/cluster.kubeconfig` file created that you can use for running `oc` commands right away, for example, for listing all projects on your OpenShift cluster:

```
oc --config ocm/cluster.kubeconfig projects
```

7. If you want to install the latest release, you can trigger it by applying an addon
    1. For **RHMI**: Run `make ocm/install/rhmi-addon` to trigger the installation
    2. For **Managed API Service (RHOAM)**: Run `make ocm/install/rhoam-addon` to trigger the installaion
  
    Once the installation is completed, the installation CR with RHMI components info will be printed to the console

8. If you want to delete your cluster, run `make ocm/cluster/delete`

### Help

To get more information about the OCM tooling, run `make ocm/help`