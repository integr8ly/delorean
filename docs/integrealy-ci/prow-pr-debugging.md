## Debugging Prow PR Failures
Prow is used to run various checks on every Pull Request in a number of Integreatly repos. These checks can fail for various reasons. Below is a list of steps to help debug failures in Prow.
### Prerequisites
* Ensure you are a member of Openshift Origin, follow [this guide](https://mojo.redhat.com/docs/DOC-1081313#jive_content_id_Github_Access) and when you get as far as `sending the email request`, use the `the openshift org so I can generate an api.ci key` option. Once a member of this organisation you will gain access to the resources used to perform Prow checks.
### e2e check
The following steps will help debug failing e2e tests.
* Visit [this cluster](https://api.ci.openshift.org/console/catalog) and log in via your GitHub. 
* Here you will have access to Namespace used to for running the checks on your Pull Request.
* In this Namespace you will see an `e2e` pod
* There are numerous containers running in this pod. The first one of interest is the `setup` container. Here we can see in the log the cluster being provisioned to execute the e2e tests against. Once this cluster has provisioned, a `url` to the cluster will be output.
* You will also see output in the logs the location to the cluster credentials.
* To retrieve the cluster credentials, you can simply `cat` the location in the `test` container.
* Logging into the cluster with these credentials you will have `admin` access allowing you to debug the issue while the tests are being executed.
* *NOTE* This cluster will be deprovisioned once the `e2e timeout` has exceeded. 