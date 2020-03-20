# Adding Prow Jobs

Once you have access to https://api.ci.openshift.org by joining the OpenShift GitHub organisation, you can test your release repo configuration locally before submitting a PR like:

```bash
docker run -it -e KUBECONFIG=/kube.config -v "${HOME}/.kube/config":/kube.config:z -v "$(pwd)/ci-operator/config/integr8ly/integreatly-operator":/config:z registry.svc.ci.openshift.org/ci/ci-operator:latest --config "/config/integr8ly-integreatly-operator-master.yaml" --target "unit" --git-ref "integr8ly/integreatly-operator@openshift-ci"
```

You can also use prowgen to generate appropriate job files based on the config with:

```bash
docker run -it -v "$(pwd)/ci-operator:/ci-operator:z" registry.svc.ci.openshift.org/ci/ci-operator-prowgen:latest --from-dir /ci-operator/config --to-dir /ci-operator/jobs integr8ly
```

To test a job running in the actual CI OpenShift, you can run the following command to test a PR:

```bash
hack/mkpjpod.sh pull-ci-integr8ly-integreatly-operator-master-unit 115 | oc -n ci-stg create -f -
```

## Useful Links

* [Onboarding A New Component for Testing and Merge Automation](https://docs.google.com/document/d/1SQ_qlkcplqhe8h6ONXdgBr7YUVbs4oRSj4ISl3gpLW4)
* [CI Operator Prowgen](https://github.com/openshift/ci-tools/blob/master/CI_OPERATOR_PROWGEN.md)
