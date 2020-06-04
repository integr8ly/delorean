## Debugging Prow PR Failures
Prow is used to run various checks on every Pull Request in a number of Integreatly repos. These checks can fail for various reasons. Below is a list of steps to help debug failures in Prow.
### Prerequisites
* Ensure you are a member of Openshift Origin, follow [this guide](https://mojo.redhat.com/docs/DOC-1081313#jive_content_id_Github_Access) and when you get as far as `sending the email request`, use the `the openshift org so I can generate an api.ci key` option. Once a member of this organisation you will gain access to the resources used to perform Prow checks.
### e2e check

First of all, you need to check if the failure is caused by problems with Prow itself, or our own tests. In most cases, you can figure it out by looking at the output messages. If the e2e check failed before the tests are started, it normally means the problem is caused by Prow. In this case, you should:

* Check the [#announce-testplatform](https://app.slack.com/client/T027F3GAJ/CFUGK0K9R/thread/CBN38N3MW-1590397619.005400) Slack channel to see if there's any annoucement about Prow having issues.
* If there isn't any issues, try paste the error in the [#forum-testplatform](https://app.slack.com/client/T027F3GAJ/CBN38N3MW/thread/CBN38N3MW-1590397619.005400) and ask for help. They are quite responsive to questions/issues and you should get an answer from them soon.

If the problem is caused by the tests, the following steps will help debug failing e2e tests.
* *NOTE* This cluster will be deprovisioned once the `e2e timeout` has exceeded.

* Request [OpenShift GitHub Access](https://mojo.redhat.com/docs/DOC-1217273) if you have not done so already.
* Login to the CI openshift cluster using GitHub. Depending on how the CI job is configured it could be running on one of [api.ci](https://api.ci.openshift.org) or [build01](https://console-openshift-console.apps.build01.ci.devcluster.openshift.com).
* Here you will have access to a namespace used to for running the checks on your Pull Request (e.g. ci-op-pqnjzfbw).
* In this namespace you will see an `e2e` pod
* There are numerous containers running in this pod. The first one of interest is the `setup` container. Here we can see in the log the cluster being provisioned to execute the e2e tests against. Once this cluster has provisioned, a `url` to the cluster will be output.
* You will also see output in the logs the location to the cluster credentials.

To make accessing test cluster credentials and logs easier you can use this [script](../../scripts/prow/e2e-test-extract-creds.sh). This can be executed as soon as you know what the namespace name is and it will output the credentials once the `setup` container is complete.

*NOTE* Make sure you are first oc logged into the correct cluster.

Extract cluster credentials:
```
$ ./scripts/prow/e2e-test-extract-creds.sh ci-op-pqnjzfbw
Retrieving test cluster kubeconfig and console details from container 'test' in pod 'e2e' in namespace 'ci-op-pqnjzfbw' on server 'https://api.ci.openshift.org:443'

....

KUBECONFIG: /tmp/kubeconfig.ci-op-pqnjzfbw/kubeconfig (Example: oc --kubeconfig=/tmp/kubeconfig.ci-op-pqnjzfbw/kubeconfig whoami)
URL: https://console-openshift-console.apps.ci-op-pqnjzfbw-4750b.origin-ci-int-aws.dev.rhcloud.com
Password: XXXXX-XXXXX-XXXXX-XXXXX
```

If you are a developer of the [Integreatly Operator](https://github.com/integr8ly/integreatly-operator), you can alternatively use the prow make targets to run this script. This will allow it to be run without the need of downloading this repo/script and installing any dependencies it might require.

Extract cluster credentials:
```
$ make prow/e2e/credentials CI_NAMESPACE=ci-op-pqnjzfbw
docker pull quay.io/integreatly/delorean-cli:master
Trying to pull repository quay.io/integreatly/delorean-cli ... 
sha256:553e75c44231211e8285fc9ca7bc47205bd53c72838804528b274a0d8d4b3081: Pulling from quay.io/integreatly/delorean-cli

...


KUBECONFIG: /tmp/kubeconfig.ci-op-pqnjzfbw/kubeconfig (Example: oc --kubeconfig=/tmp/kubeconfig.ci-op-pqnjzfbw/kubeconfig whoami)
URL: https://console-openshift-console.apps.ci-op-pqnjzfbw-4750b.origin-ci-int-aws.dev.rhcloud.com
Password: XXXXX-XXXXX-XXXXX-XXXXX
```

Tail e2e test logs:

```
make prow/e2e/tail CI_NAMESPACE=ci-op-pqnjzfbw
```

Here are some links that could be useful to find out tests failures:

* [Prow Status page](https://deck-ci.apps.ci.l2s4.p1.openshiftapps.com/)
* [OpenShift CI Search page](https://search.svc.ci.openshift.org/)

