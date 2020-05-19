## Using `ocm` for installation of RHMI

If you want to test your changes on a cluster, the easiest solution would be to spin up OSD 4 cluster using `ocm`. If you want to spin up a cluster using BYOC (your own AWS credentials), follow the additional steps marked as **BYOC**.

#### Prerequisites
* [OCM CLI](https://github.com/openshift-online/ocm-cli/releases)
* [AWS CLI v1.18](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html)
* [jq](https://stedolan.github.io/jq/)
* [cluster-service](https://github.com/integr8ly/cluster-service/releases)
* [smtp-service](https://github.com/integr8ly/smtp-service/releases)
* :apple: (Mac users) - install `gtimeout` util and create a symbolic link (so it can be referenced as `timeout`): 
  * `brew install coreutils && sudo ln -s /usr/local/bin/gtimeout /usr/local/bin/timeout`

#### Steps

1. Download the OCM CLI tool and add it to your PATH
2. Export [OCM_TOKEN](https://github.com/openshift-online/ocm-cli#log-in): `export OCM_TOKEN="<TOKEN_VALUE>"`
3. Login via OCM: 
```
make ocm/login
```

**BYOC**
Credentials for **your own IAM user** with admin access to AWS are required. These are used to create a new access key for the "osdCcsAdmin" user that provisions the cluster.  
Export the credentials for **your own IAM user**, set BYOC variable to `true` and create a new access key for "osdCcsAdmin" user:
```
export AWS_ACCOUNT_ID=<REPLACE_ME>
export AWS_ACCESS_KEY_ID=<REPLACE_ME>
export AWS_SECRET_ACCESS_KEY=<REPLACE_ME>
export BYOC=true
make ocm/aws/create_access_key
```

4. Create cluster template: `make ocm/cluster.json`

This command will generate `ocm/cluster.json` file with generated cluster name. This file will be used as a template to create your cluster via OCM CLI.
By default, it will set the expiration timestamp for a cluster for 4 hours, meaning your cluster will be automatically deleted after 4 hours after you generated this template. If you want to change the default timestamp, you can update it in `ocm/cluster.json` or delete the whole line from the file if you don't want your cluster to be deleted automatically at all. 

5. Create the cluster: `make ocm/cluster/create`

This command will send a request to [Red Hat OpenShift Cluster Manager](https://cloud.redhat.com/) to spin up your cluster and waits until it's ready. You can see the details of your cluster in `ocm/cluster-details.json` file

6. Once your cluster is ready, OpenShift Console URL will be printed out together with the `kubeadmin` user & password. These are also saved to `ocm/cluster-credentials.json` file. Also there will be `ocm/cluster.kubeconfig` file created that you can use for running `oc` commands right away, for example, for listing all projects on your OpenShift cluster:

```
oc --config ocm/cluster.kubeconfig projects
```

7. If you want to install the latest released RHMI, you can trigger it by applying an RHMI addon.
Run `make ocm/install/rhmi-addon` to trigger the installation. Once the installation is completed, the installation CR with RHMI components info will be printed to the console.

8. If you want to delete your cluster, run `make ocm/cluster/delete`