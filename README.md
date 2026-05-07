# golang-osc

`golang-osc` is a planned Go implementation of an `openstack` CLI built on [Gophercloud](https://github.com/gophercloud/gophercloud). The goal is command compatibility with `python-openstackclient` where the OpenStack API is supported through Gophercloud or documented service-scoped extras plugins.

The first compatibility oracle is the local Python OpenStackClient binary at `/Users/ken/.local/bin/openstack`, currently recorded as `openstack 9.0.0`. The Go binary should be built as `bin/openstack` so the Python CLI remains available as the reference oracle.

## Project Documents

The active plan is [docs/openstack-cli-compatibility-plan.md](docs/openstack-cli-compatibility-plan.md). It tracks scope, dependencies, work items, status, blockers, the test matrix, and the decision register.

The project diary is [docs/openstack-cli-plan-diary.md](docs/openstack-cli-plan-diary.md). It records decisions, experiments, dependency choices, rejected alternatives, and the reasoning behind those choices.

[AGENTS.md](AGENTS.md) documents how agents should work in this repository, including how to make decisions, update progress, and use the diary.

## Current Setup

The planned Go module path is:

```text
github.com/crandallnet/golang-osc
```

The current git remote may be a staging remote, but the Go module path should remain `github.com/crandallnet/golang-osc` because that is the intended long-term public import path.

The CLI parser decision is [Cobra](https://pkg.go.dev/github.com/spf13/cobra) with [pflag](https://github.com/spf13/pflag). Cobra and pflag are a parsing substrate only; compatibility behavior such as help text, completion, error text, command sorting, command stubs, and global option placement must be owned by this project and tested against the Python oracle.

Core OpenStack API access should use Gophercloud. Python/OpenStackClient must not be used in production execution paths. It may be used only as a reference for catalog generation, compatibility tests, and behavior comparison.

Plugin-scope commands and service-scoped extras plugins use [Caddy's module system](https://caddyserver.com/docs/extending-caddy) as the plugin framework. The project uses Caddy's module registration and loading model for statically linked, in-process CLI plugins, not Caddy's server runtime behavior.

## Build And Test

Use the top-level `Makefile` for the common build and test workflow:

```sh
make help
make test
make build
make smoke
make os-test
```

The `Makefile` defaults to workspace-local Go caches so builds and tests do not write into user-level cache directories from constrained environments:

```sh
export GOCACHE="$PWD/.cache/go-build"
export GOMODCACHE="$PWD/.cache/gomod"
```

Run the unit test suite:

```sh
go test ./...
```

Build the local compatibility binary:

```sh
go build -o bin/openstack ./cmd/openstack
```

Run basic smoke checks:

```sh
./bin/openstack --version
./bin/openstack command list -f json --group openstack.cli
./bin/openstack server list --help
./bin/openstack module list --max-width 52
```

Run static Python-vs-Go compatibility checks that do not require cloud credentials:

```sh
make compat-static
```

This compares the pinned Python OSC oracle with `./bin/openstack` for required parser, help, completion, and local-output cases. It also reports known gaps, such as nondeterministic Python OSC root help and intentionally different Go plugin module reporting, without failing the target. Use `make compat-static-all` to additionally compare `--help` output for every cataloged command.

To compare selected live read commands against the same cloud state:

```sh
go run ./tools/compat-check --live-cloud cloud6 --live-command "flavor list,image list,network list"
```

Live command checks support fixture placeholders resolved from the Python oracle before comparison. For example, `server show {server}` uses the first ID from `openstack server list -f json` on the selected cloud. If a fixture is unavailable, the case is reported as `SKIP` with a reason.

Discover non-secret live cloud capabilities and fixture candidates:

```sh
make discover CLOUD=cloud6
make discover CLOUD=cloud6,flex-sjc,flex-dfw,flex-iad
```

Discovery prints a summary table to the terminal and writes JSON reports under `compat/live-clouds/`. These reports intentionally omit secrets and should be refreshed before lifecycle tests because cloud state can change.

Run live write lifecycle tests:

```sh
make lifecycle CLOUD=cloud6
make lifecycle CLOUD=cloud6 SUITE=server
make lifecycle CLOUD=cloud6 SUITE=all
```

The default lifecycle suite is `keypair`. Available suites are `keypair`, `server`, `volume`, `quota`, `image`, `network`, `object`, and `all`; `make help` prints the current list. Lifecycle tests create uniquely named `golang-osc-test-*` resources, compare Go CLI behavior against the Python OSC oracle where supported, clean up resources they created, and retain failure diagnostics under `compat/lifecycle-diagnostics/`. Successful runs print concise pass lines and do not retain diagnostics unless `tools/lifecycle-smoke --keep-success` is used.

Regenerate compatibility artifacts after changing the local Python OSC oracle or matrix generator:

```sh
make matrix
go run ./tools/osc-catalog
go run ./tools/matrix
```

The matrix generator writes `compat/matrix.yaml`, `compat/test-matrix.yaml`, and `compat/test-clouds.yaml` by default, and prints a summary with command status counts and the files it wrote. The default summary is terminal-oriented; use `go run ./tools/matrix --report-format readme` for Markdown suitable for README updates. Artifact paths can be overridden with the generator flags shown by `go run ./tools/matrix --help`.

To generate a Markdown command compatibility table that compares the pinned Python OSC command catalog with the current Go CLI status:

```sh
make report
go run ./tools/matrix --report command-status --report-format readme
```

The command-status report uses these conservative statuses: `compatible`, `partially compatible`, `implemented`, and `partially implemented`. It also marks each command source as `built-in` or `plugin`.

Use `--report-output <path>` to write the selected report to a file instead of stdout.

Display the supported Fancy operating-system image colors:

```sh
make os-test
```

## Compatibility Artifacts

Generated compatibility artifacts should be committed under `compat/` once generators exist. Expected artifacts include:

* `compat/osc/9.0.0/commands.json`
* `compat/osc/9.0.0/global-help.txt`
* `compat/osc/9.0.0/help/...`
* `compat/osc/9.0.0/completion.bash`
* `compat/osc/9.0.0/metadata.json`
* `compat/matrix.yaml`
* `compat/test-matrix.yaml`
* `compat/test-clouds.yaml`
* `compat/gophercloud/<version>/packages.json`

Do not commit `clouds.yaml`, tokens, passwords, application credentials, unsanitized debug logs, or sensitive cloud response data. Commands such as `token issue` print live tokens, so do not paste their raw output into docs, commits, issues, or chat.

## Testing Clouds

The local `clouds.yaml` is available for live testing. Normal CLI configuration discovery must follow the same XDG/config precedence described by the [OpenStackClient configuration docs](https://docs.openstack.org/python-openstackclient/latest/configuration/index.html).

Known test clouds:

* `cloud6`: local cloud with full admin access, but not all services.
* `flex-sjc`, `flex-dfw`, and `flex-iad`: remote clouds with broader service coverage, but no admin-level access.

Live tests should use a structured cloud capability config so additional clouds can be added later. Project-level read/write tests may run on flex clouds, but tests must only delete or mutate resources they created themselves. Admin and destructive tests may run on `cloud6`, using unique names such as `golang-osc-test-UUID` and a dedicated project named `golang-osc-testing` where applicable.

## Implementation Status

The repository now contains the initial Go module, `cmd/openstack` entry point, Cobra/pflag root command, global flag parsing skeleton, Caddy-backed command-provider registry, embedded OSC 9.0.0 compatibility catalog, generated command stubs, captured help/completion snapshots, catalog-backed `command list`, plugin-backed `module list`, a Cliff-style width-aware table renderer, and the first Gophercloud-backed read command slices.

Implemented live Identity commands currently include `token issue`, `catalog list/show`, `domain list/show`, `endpoint list/show`, `group list/show`, `project list/show`, `region list/show`, `role list/show`, `role assignment list`, `implied role list`, `mapping list/show`, `identity provider list/show`, `federation protocol list/show`, `service list/show`, `service provider list/show`, `user list/show`, and initial read support for access rules, application credentials, credentials, EC2 credentials, Keystone limits, policies, registered limits, and trusts. Non-Identity coverage now includes initial Compute, Image, Network, Volume, Object Store, Placement, and Common slices, including server/flavor/image/network/volume reads, API-backed server lifecycle and attachment commands except `server ssh`, aggregate reads, compute service reads, hypervisor reads, console log and URL reads, server event reads, server volume attachment reads, compute usage reads, image member/task/store reads, router reads, security group reads, network agent and RBAC reads, address group/address scope/subnet pool reads, IP availability reads, network service-provider reads, initial QoS policy/rule-type, segment, and trunk reads, subnet/port/floating IP/floating IP pool/keypair/server group reads, Cinder cluster, group, group snapshot, group type, resource filter, log-level, and message reads, volume attachment/QoS/backup/service/backend pool/transfer reads, volume type/snapshot/summary reads, container/object/account reads, allocation candidate reads, resource class reads, resource provider reads, trait reads, availability zone reads, extension reads, limits reads, quota reads, and version discovery. Several show and write commands are implemented, but still need live fixtures, golden Python OSC comparisons, or a cloud exposing the relevant extension. The table renderer accepts `--max-width`, `--fit-width`, `CLIFF_MAX_TERM_WIDTH`, and `CLIFF_FIT_WIDTH=1`, and JSON list/show output now preserves the command column/field order instead of Go map ordering. The enhanced human output remains opt-in through `--format=pretty`, `--pretty`, or `OS_PRETTY=1`, uses Charm Bubbles/Lip Gloss for pretty tables and progress helpers, and keeps ANSI color disabled for non-TTY output and `NO_COLOR`.
