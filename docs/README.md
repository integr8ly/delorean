# Maintenance

## Updating the go version
To update the go version, the go version in the system path should be the required version.
Release notes for the go version being updated to should be reviewed before starting the update.

Updating the `go.mod` file.
```shell
go mod tidy -go=<go version major.minor>
```

Re-vendor the project.
```shell
go mod vendor
```

Test that update were successfully.
```shell
make build/cli
make test/unit
```

There are two Dockerfiles that also require updating.
The go version in these Dockerfiles should be updated to the current version of go used in development.


[openshift-ci/Dockerfile.tools](../openshift-ci/Dockerfile.tools) which is used by [prow](https://github.com/openshift/release/blob/master/ci-operator/config/integr8ly/delorean/integr8ly-delorean-master.yaml) for running tests against any PR's. 
To test this image was updated correctly please follow the instructions in [openshift-ci/README](../openshift-ci/README.md).

_NOTE: As the `Dockerfile.tools` is used to run the checks with in prow, the checks on the PR updating this file will use the older version.
Normally causing the vendor check to fail._


[build/Dockerfile](../build/Dockerfile) is used to build the images that are pushed [quay.io](https://quay.io/repository/integreatly/delorean-cli).
These images are built and [mirrored](https://github.com/openshift/release/blob/master/core-services/image-mirroring/integr8ly/mapping_integr8ly_delorean) by prow.
Test the image builds correctly.
```shell
make image/build
make image/test
```

### Other Places
The Delorean CLI tool is used by the RHOAM pipelines.
There are more container images there that maybe affected by an update to the go version here.
Confirm changes are addressed for the pipelines while doing maintenance work on this project. 