# golang-osc

`golang-osc` is a Go implementation of the OpenStack `openstack` command. It is built on [Gophercloud](https://github.com/gophercloud/gophercloud) and targets command compatibility with [python-openstackclient](https://docs.openstack.org/python-openstackclient/latest/) where the implemented OpenStack APIs and documented plugin paths provide coverage.

The local binary is built as `bin/openstack`.

## Goals

The default output is intended to match Python OpenStackClient behavior for compatible commands. A separate human-readable mode is available through `--pretty`, `--format=pretty`, or `OS_PRETTY=1`; Pretty output adds color, richer tables, and progress displays without changing the default compatibility surface.

The current command catalog has no generated "not implemented" command stubs. Remaining work is focused on compatibility validation, edge cases, output parity, and cloud-specific fixture coverage.

## Build

Use the top-level `Makefile` for the common workflow:

```sh
make help
make build
make test
make smoke
```

Build directly with Go:

```sh
go build -o bin/openstack ./cmd/openstack
```

Run basic checks:

```sh
./bin/openstack --version
./bin/openstack command list
./bin/openstack server list --help
```

## Configuration

Use normal OpenStack configuration inputs such as `OS_CLOUD`, `clouds.yaml`, and standard `OS_*` environment variables. The project tracks Python OpenStackClient compatibility for config precedence and auth behavior in the active plan.

Example:

```sh
OS_CLOUD=mycloud ./bin/openstack server list
```

## Output

Default output is the compatibility mode:

```sh
./bin/openstack server list
./bin/openstack server show my-server -f json
```

Pretty output is opt-in:

```sh
./bin/openstack --pretty server list
OS_PRETTY=1 ./bin/openstack volume list
```

Pretty output supports compact table rendering:

```sh
./bin/openstack --pretty --compact server list
OS_COMPACT=1 ./bin/openstack --pretty server list
./bin/openstack --pretty --no-compact server list
```

## Compatibility Reports

Generate the compatibility matrix and cloud test metadata:

```sh
make matrix
```

Generate a Markdown report comparing the pinned Python OpenStackClient command catalog with current Go CLI status:

```sh
make report
```

The generated matrix is written to `compat/matrix.yaml`. It includes a `status_summary` block with counts and percentages for every status value, including zero-count states.

## Testing

Run all unit tests:

```sh
make test
```

Run static Python-vs-Go compatibility checks that do not require cloud credentials:

```sh
go run ./tools/compat-check
```

Run live lifecycle tests against a configured cloud:

```sh
make lifecycle CLOUD=mycloud
make lifecycle CLOUD=mycloud SUITE=server
make lifecycle CLOUD=mycloud SUITE=all
```

Lifecycle tests create uniquely named resources, clean up resources they create, and retain failure diagnostics under `compat/lifecycle-diagnostics/`.

## Project Documentation

Contributor and agent workflow rules live in [AGENTS.md](AGENTS.md).

The compatibility plan is [COMPATABILITY.md](COMPATABILITY.md). The project diary is [DIARY.md](DIARY.md).

## Command Matrix Status

Current generated command-matrix status, from `compat/matrix.yaml`:

| Status | Count | Percent |
| --- | ---: | ---: |
| `unknown` | 0 | 0.00% |
| `sdk-covered` | 0 | 0.00% |
| `shim-needed` | 0 | 0.00% |
| `implemented` | 277 | 46.63% |
| `golden-matched` | 166 | 27.95% |
| `cloud-verified` | 151 | 25.42% |
| `blocked` | 0 | 0.00% |
| `local-client-needed` | 0 | 0.00% |
