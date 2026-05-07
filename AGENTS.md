# AGENTS.md

This repository is for building `golang-osc`, a Go `openstack` CLI intended to be compatible with `python-openstackclient` where API coverage exists through Gophercloud or documented extras plugins.

## Working Rule

Do not make things up. Cite and link to sources for external facts, and say when something is unclear. Prefer clear, concise language. Use paragraphs for explanation and bullets for concise lists.

## Source Of Truth

Use [docs/openstack-cli-compatibility-plan.md](docs/openstack-cli-compatibility-plan.md) as the active execution tracker. It should always show:

* what needs to be done,
* dependency order,
* current status,
* progress evidence,
* blockers,
* test coverage expectations,
* open questions, and
* accepted decisions.

Use [docs/openstack-cli-plan-diary.md](docs/openstack-cli-plan-diary.md) as the project history. It should record decisions, experiments, dependency choices, rejected alternatives, source links, observed behavior, and reasoning that may need to be reviewed later.

## Decision Process

Implementation-defining decisions belong in the plan's Decision And Question Register. Use stable IDs such as `Q-021` or `A-027` when adding or changing decisions.

When a decision is still unresolved, mark it as `pending-user`, `assumed`, `deferred`, or `blocked`. When the user answers or evidence settles the matter, update the status to `decided` and record the answer.

When a decision depends on research or an experiment, record the research or experiment in the diary, then update the plan with the outcome. The plan should contain the current path forward; the diary should contain how that path was chosen.

Dependency choices must be documented in the diary. Prefer established, popular, maintained Go libraries when available, and explain why a dependency was selected.

## Progress Tracking

Update the plan's work table when work starts, completes, blocks, or changes scope. A work item should move to `done` only when the named artifact exists and the validation described in the row has passed. For command compatibility work, `done` requires output parity testing against the Python OSC oracle, any fixes found by that testing, and validation against mocked or live OpenStack endpoints as appropriate. Implementation coverage alone is not finished work.

Do not leave progress only in chat. If a task changes the project state, update the plan, the diary, or both.

## Compatibility Rules

The first Python oracle is `/Users/ken/.local/bin/openstack`, recorded as `openstack 9.0.0`. Generated compatibility artifacts should be committed under `compat/` once generators exist.

Default output, help text, completion output, stderr text, and exit status are compatibility surfaces. Exact compatibility should be tested where practical, and exceptions should be documented.

The Go CLI must not shell out to Python/OpenStackClient in production behavior. Python/OpenStackClient may be used only as an oracle for catalog generation, tests, and behavior comparison.

`server ssh` is a documented compatibility exception to the normal API-backed command rule, not an exception to the self-contained runtime rule. It belongs in the `nova-extras` plugin and must use the pure Go SSH client path; do not shell out to `ssh`, Python/OpenStackClient, or another OS command in production behavior.

Build the local Go CLI as `bin/openstack` and preserve the Python `openstack` binary as the oracle.

## Plugin Rules

Core Gophercloud-backed commands should stay separate from plugin-scope commands and raw REST gap coverage.

Plugin-scope commands must use Caddy's module system as the selected existing plugin framework. Use Caddy's module registration and loading model for statically linked, in-process CLI plugins. Do not use Caddy's server runtime behavior unless the plan and diary are updated with a new decision.

Raw REST compatibility shims for missing Gophercloud package coverage should live in service-scoped extras plugins, such as `nova-extras`, `neutron-extras`, `glance-extras`, `cinder-extras`, `keystone-extras`, `swift-extras`, and `placement-extras`.

## Test Safety

Do not commit `clouds.yaml`, tokens, passwords, application credentials, unsanitized debug logs, or sensitive cloud response data.

The local `clouds.yaml` may be used for live tests. Normal CLI config discovery must follow the XDG/config precedence described by the [OpenStackClient configuration docs](https://docs.openstack.org/python-openstackclient/latest/configuration/index.html).

Project-level read/write tests may run on flex clouds, but tests must only delete or mutate resources they created themselves. Existing networks, images, servers, keys, volumes, and other resources must be respected.

Admin and destructive tests may run on `cloud6`, using unique names such as `golang-osc-test-UUID`, cleanup, retained diagnostics on failure, and the dedicated `golang-osc-testing` project where applicable.

Before lifecycle tests run, dynamically query the target cloud for currently available fixtures such as images, flavors, networks, volume types, external networks, quotas, and extensions. Record selected fixture values for diagnostics.
