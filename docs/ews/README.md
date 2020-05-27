# Early Warning System (EWS)

Multiple products are installed as part of the Integreatly(RHMI) environment. 
It is important to move quickly to new releases of these products to enable proof of concepts and ensure any new managed environment is always using the latest versions. 
The Delorean Early Warning System(EWS) consists of tooing and pipelines which together provide an automated way of discovering, updating and testing new versions of these products.

### Repos and Pipelines

The current set of relevant repos for the EWS are:

* [Delorean CLI](https://github.com/integr8ly/delorean)  - Tooling and scripts used by the EWS
* [Jenkins Config](https://gitlab.cee.redhat.com/integreatly-qe/ci-cd/tree/master/jobs/Delorean/ews) - (Internal Only) Jenkins job configuration
* [Jenkins](https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/Delorean/job/ews/) - (Internal Only) Jenkins instance running the EWS jobs


### Release Discovery Pipeline

The main component of the EWS is the release discovery pipeline, the currently configured pipelines can all be found [here](https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/Delorean/job/ews/).
Every component/product that is enabled on the EWS has it's own discovery job, and it's role is to search for new versions of a particular product, and update the installer([integreatly operator](https://github.com/integr8ly/integreatly-operator)).

It has the following responsibilities:
* Discovery of new versions.
* Create a new branch against the installer if one does not already exist (\<product-name\>-next-\<version\> i.e. 3scale-next-0.5.2)
* Mirror any pre release images (operator and operand) to a private quay registry
* Process the discovered products OLM manifests cluster service version (CSV)
* Update the OLM manifest in the installer with the newly discovered version
* Push branch  
* Create pull request(PR) if one does not already exist
* Manage labels on the PR indicating GA status
* Create JIRA if one does not already exist


### Delorean Branches (next branches)

The main output from the EWS is a branch on the installer that includes all the updates required to install the newly discovered version in a cluster.
All the current next branches can be seen [here](https://github.com/integr8ly/integreatly-operator/branches/all?query=next)

The next branch is intended to track and receive updates for a single version of a products OLM manifest from beginning to final GA status. 
As updates for a particular version are discovered, the appropriate branch will be updated with what is discovered (updated OLM bundle with new operand images versions etc..)
The version of the OLM bundle is therefore embedded in the branch name (\<product-name\>-next-\<version\> i.e. 3scale-next-0.5.2), and should not be confused with the product version which is a completely separate piece of metadata not contained in the operator itself. 

The EWS automation does a best effort job at updating everything required to install and ultimately consider the change for merge into the main line branch (master).
It doesn't however do everything, and it's the responsibility of the reviewers/approvers to ensure everything is as expected. 

The automation will not:

* Update GO Module dependencies
* Update Operator/Product versions hard coded in the installer

As well as automated updates, these branches would expect to be updated by members of the development team to resolve any installation issues arising form the new version of the product and operator.
These updates should be applied to the next branch in the same way you would update master by creating a PR against the branch and following the usual PR workflow for getting the changes approved and merged.


#### Installation

The EWS generated branches start getting created as soon as we discover any new version of a product, including versions that have not yet released.
Since these versions are not released and their operators and operand images will not yet be available from the normal RH Registries (registry.redhat.io), 
we need to prepare a cluster to pull from a different registry that the EWS ensures all images are available in.

The script [here](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/setup-delorean-pullsecret.sh) can be used to do this.

You must be logged into the OCP4 cluster you intend to install on, and the script will expect access to a local pull secret that allows read access to the private quay registry containing the images. 
If you do not have access to this registry, and require a pull secret, you should contact a member of the [Delorean team](https://chat.google.com/room/AAAAEPzaAc0) 

The installation from any EWS generated branch should work in the same way as any other installation once this script has been executed.

Note: This is only required for pre release versions of products, once a product GA's there should be no need to execute this script.

### GitHub Pull Requests

A GitHub pull request is created as one of the final steps of the release discovery process. These PRs are created as soon as any new version, pre release or GA, is discovered and a valid branch created against the installer.
The PR executes the same tests that get executed on any other PR that is opened against this repo, including the e2e test, that ensures the entire installation completes and the functional test suite passes. 

The triggered CI is already setup to allow the testing of pre release versions of products by applying the delorean pull secret as described [here](#installation) 

#### PR labels

As with all prow enabled repos, labels are used extensively to control what happens, and give the status of the state of these PRs.

Along with the usual prow workflow labels there also these labels managed by delorean:

* do-not-merge/product-pre-release - Used as a tide merge blocker, applied to all EWS PRs that include products that are not yet GA, removed automatically when a product is GA.
* is-ga - Information only, applied to all EWS PRs that include products that are GA, added automatically when a product is GA.

#### PR Jobs

All the usual CI (prow) jobs will execute against the EWS PRs, but some jobs have been updated to prevent accidental inclusion of files/updates solely for the testing of product pre release versions:

* ci/prow/manifest - fails if image mirror mapping files exists or any references to internal registries exist in any CSVs. It would be expected that this will fail for all pre release PRs.


### Adding new products

To add a new component/product to the EWS pipelines, it must:

* Productized (Go through the RH internal build system)
* Produce an OLM installation manifest bundle image
* Included as part of an installation in the installer (integreatly operator)

Any new product that meets these requirements can be added to the EWS by updating a single configuration [here](https://gitlab.cee.redhat.com/integreatly-qe/ci-cd/blob/master/jobs/Delorean/ews/delorean-ews-umb-discovery.groovy)

Options:

* productName - The product name
* createJIRA - Whether or not to automatically create a JIRA for the version update
* createPR - Whether or not to automatically create a PR for the version update
* brewPkg - The internal brew package ID for the OLM manifest container image build
