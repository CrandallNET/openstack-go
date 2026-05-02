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

Plugin-scope commands and service-scoped extras plugins should use [Caddy's module system](https://caddyserver.com/docs/extending-caddy) as the plugin framework. The project should use Caddy's module registration and loading model for statically linked, in-process CLI plugins, not Caddy's server runtime behavior.

## Compatibility Artifacts

Generated compatibility artifacts should be committed under `compat/` once generators exist. Expected artifacts include:

* `compat/osc/9.0.0/commands.json`
* `compat/osc/9.0.0/global-help.txt`
* `compat/osc/9.0.0/help/...`
* `compat/osc/9.0.0/completion.bash`
* `compat/osc/9.0.0/metadata.json`
* `compat/matrix.yaml`
* `compat/test-matrix.yaml`
* `compat/gophercloud/<version>/packages.json`

Do not commit `clouds.yaml`, tokens, passwords, application credentials, unsanitized debug logs, or sensitive cloud response data.

## Testing Clouds

The local `clouds.yaml` is available for live testing. Normal CLI configuration discovery must follow the same XDG/config precedence described by the [OpenStackClient configuration docs](https://docs.openstack.org/python-openstackclient/latest/configuration/index.html).

Known test clouds:

* `cloud6`: local cloud with full admin access, but not all services.
* `flex-sjc`, `flex-dfw`, and `flex-iad`: remote clouds with broader service coverage, but no admin-level access.

Live tests should use a structured cloud capability config so additional clouds can be added later. Project-level read/write tests may run on flex clouds, but tests must only delete or mutate resources they created themselves. Admin and destructive tests may run on `cloud6`, using unique names such as `golang-osc-test-UUID` and a dedicated project named `golang-osc-testing` where applicable.

## Implementation Status

The repository now contains the initial Go module, `cmd/openstack` entry point, Cobra/pflag root command, global flag parsing skeleton, and CLI-local command stubs. See the plan for the current progress table and next work items.
