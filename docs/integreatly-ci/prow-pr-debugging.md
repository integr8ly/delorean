## Debugging Prow PR Failures
Prow is used to run various checks on every Pull Request in a number of Integreatly repos. These checks can fail for various reasons. Below is a list of steps to help debug failures in Prow.
### Prerequisites
* Ensure you are a member of Openshift Origin, follow [this guide](https://source.redhat.com/groups/public/atomicopenshift/atomicopenshift_wiki/openshift_onboarding_checklist_for_github).
### e2e check

First of all, you need to check if the failure is caused by problems with Prow itself, or our own tests. In most cases, you can figure it out by looking at the output messages. If the e2e check failed before the tests are started, it normally means the problem is caused by Prow. In this case, you should:

* Check the [#announce-testplatform](https://app.slack.com/client/T027F3GAJ/CFUGK0K9R/thread/CBN38N3MW-1590397619.005400) Slack channel to see if there's any annoucement about Prow having issues.
* If there isn't any issues, try paste the error in the [#forum-testplatform](https://app.slack.com/client/T027F3GAJ/CBN38N3MW/thread/CBN38N3MW-1590397619.005400) and ask for help. They are quite responsive to questions/issues and you should get an answer from them soon.

If the problem is caused by the tests, the following steps will help debug failing e2e tests.
* *NOTE* This cluster will be deprovisioned once the `e2e timeout` has exceeded and won't be usable.

* Request OpenShift GitHub Access from the guide above if you have not done so already.
* In your pull request, click the details on your failed e2e test and find `using namespace` in your logs.
* Log in using your SSO.
* Inside the UI navigate towards `Workloads -> Secrets`.
* After a while a secret will be created called `rhoam-e2e-hive-admin-kubeconfig`.
* In this secret at the bottom you will find the link to the secondary cluster. The link will look something like this : `https://api.ci-ocp-4-10-amd64-aws-us-west-1-7msbg.hive.aws.ci.openshift.org:6443`
* Paste this URL into anew tab and replace the `api` part with `console-openshift-console.apps`. You will also need to remove the port number.
* This will require you to log in. The user will be `kubeadmin`. To find the log in password navigate back to `Secrets` and find `rhoam-e2e-hive-admin-password`.
* From here you will have access to the cluster to debug whatever is needed.

### Getting the server API token

* In your terminal write `oc login <SERVER-LINK>` eg: https://api.ci-ocp-4-10-amd64-aws-us-west-1-7msbg.hive.aws.ci.openshift.org:6443.
* Accept the insecure connection.
* Click the link it generates.
* This will redirect you to a page displaying a generated API token.

Here are some links that could be useful to find out tests failures:

* [Prow Status page](https://deck-ci.apps.ci.l2s4.p1.openshiftapps.com/)
* [OpenShift CI Search page](https://search.ci.openshift.org/)

