# Testing

See [Install guide](INSTALL.md) for development install.

## Prerequisites

- Go 1.25+ (see [Install guide](INSTALL.md))

## Unit tests

Run unit tests (default, no envtest assets needed):

```bash
go test ./...
```

Coverage includes template rendering for k0s and k3s and bootstrap controller logic.

## Envtest

Envtest is optional and downloads assets automatically:

```bash
make test-envtest
```

`make test-envtest` installs setup-envtest if needed, downloads assets, and runs envtest-tagged tests.

CI currently tests against both Go 1.24 and 1.25 for one-cycle compatibility. After confirming stability, the matrix can be reduced to Go 1.25 only.
