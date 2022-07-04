# Delorean

Delorean CLI

### Docs

- [Early Warning System (EWS)](./docs/ews/README.md)
- [Integreatly CI](./docs/integreatly-ci/README.md)
- [Provision RHMI Cluster using OCM](./docs/ocm/README.md)

### Building

To build the CLI, run from the root of this repo:

```
make build/cli
```

A binary will be created in the root directory of the repo, which can be run:

```
./delorean
```

To build a delorean container image:

```
make image/build
```

To run some basic tests against the image:

```
make image/test
```

To build and test using a container engine other than docker (podman):

```
make image/test CONTAINER_ENGINE=podman
```

## Testing

To run unit tests, run:

```
make test/unit
```

`test`