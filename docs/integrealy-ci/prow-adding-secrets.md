# Adding secrets to Prow

## Updating an existing secrets values

### integr8tly-ci-secrets values

Create a new env file (integr8ly-ci-secrets.env) with all env vars listed below:

```bash
HEIMDALL_REGISTRY_PASSWORD=
HEIMDALL_REGISTRY_USERNAME=
INTLY_OPERATOR_COVERALLS_TOKEN=
ocm-refresh-token=
```

### integr8ly-tower-secrets values

Create a new env file (integr8ly-tower-secrets.env) with all env vars listed below:

```bash
OPENSHIFT_MASTER=
TOWER_OPENSHIFT_USERNAME=
TOWER_OPENSHIFT_PASSWORD=
TOWER_LICENSE=
TOWER_USERNAME=
TOWER_PASSWORD=
```

### Create the secret
```bash
oc create secret generic <secret-name> --from-env-file=./<file-name> --dry-run -o yaml | oc replace -f -
```

An example of the changes required to make this secret available in the test container can be found [here](https://github.com/openshift/release/pull/5083/files).

An example of how to access the contents of the secret in your test scripts can be found [here](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/ci/unit_test.sh#L8).

Note:

Injecting secrets into the test container this way only works when using [container tests](https://github.com/openshift/release/blob/master/ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-master.yaml#L47-L51)
This approach of injecting extra secrets into the test container is not supported by [template tests](https://github.com/openshift/release/blob/master/ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-master.yaml#L60-L63) used for e2e testing.