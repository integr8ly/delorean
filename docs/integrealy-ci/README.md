# Integreatly CI 

OpenShift 4 repositories are using [prow](https://github.com/kubernetes/test-infra/tree/master/prow), a kubernetes based CI/CD system, used by both kubernetes itself and OpenShift. The prow instance we are using is maintained by the OpenShift [DPTP](https://mojo.redhat.com/docs/DOC-1177573) team.

## CI Workflow/Usage

* [Prow Pull Request Workflow](./prow-pr-workflow.md)
* [Prow Commands](./prow-commands.md)
* [Prow Debugging](./prow-pr-debugging.md)

## CI Development

* [Adding new repos to prow](./prow-adding-repos.md)
* [Adding new jobs to prow](./prow-adding-jobs.md)
* [Adding adding secrets](./prow-adding-secrets.md)
* [Adding image mirroring](./prow-adding-image-mirroring.md)

## Useful Links

* [Developer Productivity Test Platform (DPTP) Mojo Page](https://mojo.redhat.com/docs/DOC-1177573)
* [Prow Overview](https://github.com/kubernetes/test-infra/tree/master/prow)
* [OpenShift CI Prow Configuration](https://github.com/openshift/release)
* [Code Review Process](https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md#the-code-review-process)

