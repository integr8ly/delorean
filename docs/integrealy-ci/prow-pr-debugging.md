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
* Visit [this cluster](https://api.ci.openshift.org/console/catalog) and log in via your GitHub. 
* Here you will have access to Namespace used to for running the checks on your Pull Request.
* In this Namespace you will see an `e2e` pod
* There are numerous containers running in this pod. The first one of interest is the `setup` container. Here we can see in the log the cluster being provisioned to execute the e2e tests against. Once this cluster has provisioned, a `url` to the cluster will be output.
* You will also see output in the logs the location to the cluster credentials.
* To retrieve the cluster credentials, you can simply `cat` the location in the `test` container.
* Logging into the cluster with these credentials you will have `admin` access allowing you to debug the issue while the tests are being executed.

Here are some links that could be useful to find out tests failures:

* [Prow Status page](https://deck-ci.apps.ci.l2s4.p1.openshiftapps.com/)
* [OpenShift CI Search page](https://search.svc.ci.openshift.org/)

