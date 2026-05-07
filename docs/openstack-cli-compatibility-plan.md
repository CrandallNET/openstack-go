# golang-osc OpenStack CLI Compatibility Plan

Status: living plan and progress tracker, with an initial gap catalog, ordered workplan, and cloud test matrix.

This plan targets a Go `openstack` CLI built on [Gophercloud](https://github.com/gophercloud/gophercloud) that can replace `python-openstackclient` command-for-command where coverage exists. The default behavior must match Python OpenStackClient, including command names, flag names, exit behavior, stdout/stderr behavior, and default output. The new enhanced human output is opt-in via `--format=pretty`, `--pretty`, or the Go-only `OS_PRETTY=1` environment default; the two flags are synonymous, and an explicit `--format` overrides the environment default.

This document is the active execution tracker. It should show what needs to be done, the dependency order, current progress, and known blockers. Decisions, experiments, rejected alternatives, and research notes belong in the [plan diary](openstack-cli-plan-diary.md).

The current local oracle is `/Users/ken/.local/bin/openstack`, which reports `openstack 9.0.0`. The current online docs also expose a development command list for OpenStack Command Line Client `9.1.0.dev71`; that is useful for drift tracking but should not silently replace the pinned local oracle. PyPI lists `python-openstackclient 9.0.0` as released on February 17, 2026, so the first generated catalog should be pinned to `9.0.0`, with an explicit upgrade process for later versions.

Sources used for this initial plan:

* [python-openstackclient README](https://github.com/openstack/python-openstackclient), which says OSC covers Compute, Identity, Image, Network, Object Store, and Block Storage, with additional service APIs provided by plugins.
* [OpenStackClient command list](https://docs.openstack.org/python-openstackclient/latest/cli/command-list.html), currently rendered as `9.1.0.dev71`.
* [OpenStackClient Human Interface Guide](https://static.opendev.org/docs/python-openstackclient/latest/contributor/humaninterfaceguide.html), which defines the command grammar, global option behavior, option naming, and default output expectations.
* [OpenStackClient configuration docs](https://docs.openstack.org/python-openstackclient/latest/configuration/index.html), especially `clouds.yaml`, environment variable, and global option precedence.
* [Cliff demo app docs](https://docs.openstack.org/cliff/latest/user/demoapp.html), for `--max-width`, `--fit-width`, `CLIFF_MAX_TERM_WIDTH`, and `CLIFF_FIT_WIDTH` behavior.
* [Cliff table formatter source](https://github.com/openstack/cliff/blob/master/cliff/formatters/table.py), for the table-width fitting algorithm used by Python OSC's default table output.
* [python-openstackclient on PyPI](https://pypi.org/project/python-openstackclient/), for current published version context.
* [Gophercloud README](https://github.com/gophercloud/gophercloud), for supported services and authentication model.
* [Gophercloud v2.12.0 on pkg.go.dev](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2), for module version and package index.
* Gophercloud package docs for [Compute v2](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2), [Block Storage v3](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3), [Identity v3](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3), [Networking packages](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks), [Object Storage v1](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1), and [Placement v1](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders).

## Compatibility Target

Compatibility means more than reaching the same API endpoint. A command is compatible only when all externally visible behavior matches the Python client for the pinned target version:

* The command path exists with the same words in the same order.
* The same global options, command options, aliases, deprecated forms, and mutually exclusive groups parse the same way.
* `--help`, `help <command>`, shell completion, and `command list` expose the same command surface for the active API versions and installed plugins.
* Name-or-ID resolution, project/domain scoping, default filters, pagination, sorting, microversion selection, and wait behavior match.
* Default output matches the Python CLI column names, column order, value formatting, table shape, and empty-output behavior.
* Machine formats match where OSC supports them: `table`, `json`, `yaml`, `csv`, `value`, and `shell`, including `--column`, `--quote`, `--noindent`, `--prefix`, `--max-width`, `--fit-width`, `--print-empty`, and sort options.
* Exit status, stderr messages, and partial-failure behavior match closely enough for scripts to treat the Go binary as a drop-in replacement.

The enhanced `pretty` output is intentionally not a compatibility surface. It should be deterministic, readable, and useful for operators, but scripts must continue to use the OSC-compatible default or existing machine formats.

## Initial Command Inventory

The local `openstack 9.0.0` binary currently exposes 594 commands through `openstack command list -f json`. That local command list includes plugin-provided Placement and Tap-as-a-Service commands, which is broader than the core Python package described in the README. The plan tracks these commands instead of ignoring them, but labels plugin coverage separately.

| Command group | Local command count | Initial source of truth | Notes |
| --- | ---: | --- | --- |
| `openstack.cli` | 2 | local `openstack 9.0.0` | CLI metadata commands. |
| `openstack.common` | 11 | local `openstack 9.0.0` | Cross-service commands and local config display. |
| `openstack.compute.v2` | 96 | local `openstack 9.0.0`; OSC docs | Core scope. |
| `openstack.identity.v3` | 128 | local `openstack 9.0.0`; OSC docs | Core scope. |
| `openstack.image.v2` | 41 | local `openstack 9.0.0`; OSC docs | Core scope. |
| `openstack.network.v2` | 175 | local `openstack 9.0.0`; OSC docs | Core scope plus installed networking plugin extensions. |
| `openstack.object_store.v1` | 17 | local `openstack 9.0.0`; OSC docs | Core scope. |
| `openstack.placement.v1` | 31 | local `openstack 9.0.0`; plugin | Plugin scope, but Gophercloud has Placement v1 packages. |
| `openstack.volume.v3` | 93 | local `openstack 9.0.0`; OSC docs | Core scope. |

The first implementation task is to persist this generated inventory in the repository as machine-readable data. The Markdown table above is a seed, not the final compatibility matrix.

## Initial Gap Catalog

The biggest gap is architectural: Gophercloud is a Go SDK, not a command-line compatibility layer. It provides service clients, auth helpers, resource packages, pagination, and result extraction, but it does not provide OSC command parsing, help text, command discovery, output renderers, error text, name-or-ID lookup, shell completion, or Python client compatibility semantics. Those must be implemented in this project.

| Area | What Gophercloud provides | Gap to close |
| --- | --- | --- |
| CLI shell | No OSC-compatible command shell. | Build command registry, parser, help, completion, deprecation handling, and `command list`. The registry should be generated from the pinned Python oracle. |
| Output | No OSC renderers. | Implement OSC-compatible renderers first, then add enhanced `pretty`. Default output must remain Python-compatible. |
| Auth/config | Gophercloud supports `clouds.yaml`, environment-derived auth, `ProviderClient`, and per-service `ServiceClient` creation. | Reproduce OSC precedence and surface area: global auth options, `OS_*` environment variables, `OS_CLIENT_CONFIG_FILE`, `clouds.yaml`, `clouds-public.yaml`, prompting, TLS flags, `--os-interface`, `--timing`, `--debug`, and plugin-specific auth types. OSC 9.0.0 advertises many auth types locally; parity for OAuth2, OIDC, TOTP, MFA, tokenless auth, and Keystone-to-Keystone federation needs explicit validation. |
| Error behavior | Gophercloud returns Go errors and HTTP result errors. | Map errors to OSC-compatible stderr text, exit codes, partial-delete behavior, and traceback behavior under `--debug`. |
| Name-or-ID lookup | Gophercloud generally exposes list/get APIs. | Implement OSC's resource lookup behavior, including ambiguous names, domain/project-qualified lookups, and API-specific fallbacks. |
| Microversions | Gophercloud service clients support microversion fields for services that need them. | Match OSC's global `--os-*-api-version` flags, defaults, min/max negotiation, command-specific minimums, and error messages. |

### Compute v2

Gophercloud has documented Compute v2 packages for aggregates, attach interfaces, availability zones, diagnostics, extensions, flavors, hypervisors, instance actions, keypairs, limits, quota sets, remote consoles, security groups, server groups, servers, services, tags, usage, and volume attachments. This covers a large part of the local 96-command Compute surface. The current implementation uses server console-output actions for `console log show`, remote consoles for `console url show`, instance actions for `server event list/show`, volume attachments for `server volume list` and server attach/detach operations, server, attach-interface, volume-attachment, and tag helpers for most `server` lifecycle commands, usage for `usage list/show`, and narrow Nova REST shims for console auth token lookup, deprecated `compute agent list`, `host list/show`, server migration list/show/action gaps, deprecated or action-only server calls, and OSC-shaped server attachment behavior.

Known or likely gaps to validate are `compute agent create/delete/set`, `host set`, `aggregate cache image`, `server ssh` local SSH orchestration, and the exact admin/microversion behavior of evacuation, resize, shelve, unshelve, and migration commands. The `cloud6` server lifecycle now records paired Python-vs-Go write-output parity for the safe disposable server create/delete, set/unset, lock/unlock, reboot, stop/start, rebuild, pause/unpause, suspend/resume, rescue/unrescue, volume attach/detach/set/update, security-group add/remove, port add/remove, and network add/remove paths. Any missing SDK method should be implemented as a narrow raw REST shim first, with an upstream Gophercloud contribution considered after compatibility tests prove the behavior.

### Identity v3

Gophercloud has documented Identity v3 packages for application credentials and access rules, catalog, credentials, domains, EC2 credentials and tokens, endpoints, federation, groups, limits, OAuth1, inherited roles, policies, project endpoints, projects, regions, registered limits, roles, services, tokens, trusts, and users. That is strong coverage for the local 128-command Identity surface. The current implementation also uses the roles package for `role assignment list` and `implied role list`, with OSC-shaped extraction for system and inherited scope fields that the local typed Gophercloud assignment struct does not expose. Federation mapping reads use the Gophercloud federation package; identity-provider, federation-protocol, and service-provider reads are narrow OS-FEDERATION REST shims through the authenticated Identity client because no typed local Gophercloud helper exists in v2.12.0.

Gaps to validate are endpoint group commands, service-provider write commands, request/access token command details, remaining role-assignment edge cases, federation protocol/provider fixture coverage, application credential access-rule option parity, and OSC's domain-scoped name resolution. Authentication plugin parity is tracked separately because it affects every service.

### Image v2

Gophercloud has documented Image v2 packages for image data, image import, images, members, and tasks. That should cover core `image create/delete/list/save/set/show/stage/import`, import discovery, image members, and tasks. `image create` uses typed image metadata creation plus typed direct upload/stage/import calls for file data, `image save` and `image stage` use typed image data calls, `image set`/`image unset` use typed image patch and member helpers where available, and `image delete` uses typed image deletion for normal deletes. `image delete --store`, image activation actions, and image tag delete use narrow authenticated Glance REST calls because the local Gophercloud v2.12.0 module has no typed helpers for those operations. `image import info` uses Gophercloud's typed `imageimport` package; command-level `image import` uses a narrow authenticated Glance REST request because the typed helper does not cover stores, all-stores, remote import, or copy-image options. `image add project`/`image remove project` use the typed image member package.

The current local command surface includes cached image operations, stores, and extensive metadef namespace/object/property/resource-type commands. Those packages were not visible in the cited Gophercloud v2 package index, so they are initial SDK gaps until module inspection proves otherwise. `cached image clear/delete/list/queue`, `image metadef namespace create/delete/list/set/show`, `image metadef object create/delete/list/show/update`, `image metadef object property show`, `image metadef property create/delete/list/set/show`, `image metadef resource type list`, and `image metadef resource type association create/delete/list` now use narrow Glance REST operations because no typed local Gophercloud helpers exist in v2.12.0. Plan for raw REST shims for stores and any import workflow details not covered by `imageimport`.

The `cloud6` image lifecycle now records paired Python-vs-Go write-output parity for `image create/delete/set/unset/stage/import`, `image add project`, `image remove project`, and Glance metadef namespace/object/property/resource-type association create, update, set, and delete paths where those commands mutate only test-created images and metadata definitions.

### Network v2

Gophercloud documents core Networking v2 packages for networks, ports, and subnets, plus extension packages for agents, address scopes, floating IPs, port forwarding, routers, security groups and rules, network IP availability, RBAC policies, subnet pools, segments, QoS, trunks, DNS, external networks, extra routes, port binding, BGP, BGPVPN, FWaaS, VPNaaS, VLAN transparency, and Tap-as-a-Service tap mirrors. This covers much of the local 175-command Network surface. The current implementation also uses a narrow Neutron `service-providers` REST read for `network service provider list`, because no typed local Gophercloud helper exists in v2.12.0, implements `floating ip pool list` as the Python-observed Neutron compatibility error path, and implements `floating ip port forwarding` commands through Gophercloud with raw Neutron list extraction for range fields.

Gaps to validate are default security group rules, local IPs and local IP associations, network flavor and flavor profile commands, L3 conntrack helpers, network meters and meter rules, segment ranges, and Tap-as-a-Service tap flow and tap service commands. Tap mirror support is visible in Gophercloud, but tap flow and tap service support were not visible in the cited package search, so treat those as gap candidates. Floating IP port forwarding still needs a live cloud fixture because `cloud6` returns Neutron `404` for the `floatingips/{id}/port_forwardings` endpoint in both Python OSC and the Go CLI. QoS rule commands are implemented, but `minimum-packet-rate` remains a documented Gophercloud package gap in v2.12.0 and is handled by a narrow authenticated Neutron request. Trunk and subport commands are implemented, but need a cloud exposing the trunk extension for lifecycle verification.

The `cloud6` network lifecycle now records paired Python-vs-Go write-output parity for safe disposable `address group`, `address scope`, `network`, `subnet`, `port`, `router`, `router add/remove port`, `router add/remove subnet`, `router add/remove route`, `security group`, and `security group rule` mutation paths. The suite normalizes only paired resource names/IDs and cloud-generated values, such as Neutron revision numbers, provider segmentation IDs, standard attribute IDs, generated IPs, MAC addresses, UUIDs, and timestamps.

### Object Store v1

Gophercloud documents Object Storage v1 packages for accounts, containers, and objects. That maps cleanly to the local 17-command object store surface at the resource level.

The remaining work is broader compatibility behavior: large object behavior if OSC handles it implicitly, progress or quiet behavior, more metadata edge cases, and alternate output/flag coverage. The `flex-dfw` object-store lifecycle now records paired Python-vs-Go write-output parity for `container create/delete/set/unset` and `object create/delete/save/set/unset` with disposable containers and objects. `cloud6` is not treated as a product gap for Object Store because it currently lacks a usable object-store endpoint for this suite.

### Placement v1

The local CLI includes 31 Placement commands. Gophercloud has Placement v1 service creation and documented packages for allocation candidates, resource classes, resource providers, and traits. Resource provider docs include aggregates, inventories, allocations, traits, and usage operations.

Gaps to validate are consumer allocation commands, `resource usage show`, exact microversion defaults, generation-conflict errors, inventory class command behavior, and the local `osc-placement` command text. This should be treated as plugin scope, even though the SDK coverage appears good.

### Volume v3

Gophercloud has documented Block Storage v3 packages for attachments, availability zones, backups, limits, manageable volumes, QoS, quota sets, scheduler stats, services, snapshots, transfers, volumes, and volume types. That covers much of the local 93-command Volume surface.

The current implementation covers every command exposed by the pinned local `openstack.volume.v3` catalog. It uses Gophercloud's Block Storage [attachments](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments), [backups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups), [QoS](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos), [scheduler stats](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/schedulerstats), [services](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/services), [snapshots](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots), [transfers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers), [volumes](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes), and [volume types](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes) packages where those packages preserve the needed command behavior. The remaining Volume v3 surface uses narrow authenticated Cinder REST requests through Gophercloud service clients: cleanup workers, clusters, log-level set/get, resource filters, manageable volume and snapshot lists, consistency groups, consistency group snapshots, volume groups, volume group snapshots, volume group types, host freeze/thaw, messages, migration, revert, backend capabilities, volume service set, remote-source manage calls, and group/type replication actions.

Remaining gaps are no longer command stubs; they are validation gaps or cloud-fixture blockers. The `cloud6` Volume lifecycle suite now validates disposable volume, snapshot, transfer request, QoS, volume group type, volume group, consistency group, and attachment paths with unique names, LIFO cleanup, wait loops, retained diagnostics, and recorded fixture choices. Attachment validation uses a disposable server and disposable volume, then exercises attachment create/show/list/set/complete/delete where the cloud accepts the step. Live Python-vs-Go parity now covers the core Cinder read expansion for resource filters, manageable volume and snapshot lists, backend capabilities, log-level list, and volume summary in both default table and JSON output. The blocked or intentionally unpromoted Volume commands are backup mutations because `cinder-backup` is down on `cloud6`, volume type mutations because both Python OSC and the Go CLI hit a Cinder HTTP 500 deleting a test-created type while `__DEFAULT__` is missing, group snapshot mutations until a valid populated group fixture is available, manage-existing remote-source calls until backend-local unmanaged storage fixtures exist, and real service/backend/admin mutations such as cleanup, host set, service set, log-level set, message delete, migration, and group failover because they are not scoped to test-created resources.

### Common and Cross-Service Commands

`availability zone list`, `extension list/show`, `limits show`, `quota *`, `versions show`, and `project cleanup` are not single-resource wrappers. They aggregate multiple services, depend on active API versions, and often have service-specific option behavior. Implement these as first-class compatibility commands using Gophercloud clients underneath, not as thin SDK package calls. `quota set/delete` now have Compute, Volume, and Network implementations through authenticated service clients and narrow quota REST requests, including Compute and Volume quota-class updates. Project-scoped quota mutation is live-verified on `cloud6` with a dedicated test project, per-service resets, aggregate `quota show -f json` Python-oracle parity before and after reset, and test project cleanup. Quota class/default mutation remains unpromoted because it changes default quota-class state rather than only a test-owned project.

`configuration show`, `module list`, `command list`, `complete`, `--help`, and `--version` are CLI-local. They should not call OpenStack APIs except where Python does.

## Parser Decision

Use [Cobra](https://pkg.go.dev/github.com/spf13/cobra) with [pflag](https://github.com/spf13/pflag) for command parsing and dispatch. The decision record, alternatives considered, and local parser experiments live in the [plan diary](openstack-cli-plan-diary.md).

This is a compatibility substrate, not a license to accept Cobra defaults. The project should generate a Cobra command tree from the pinned OSC catalog, attach root persistent flags for global options, attach command-local flags from generated command metadata, and keep command implementations independent of Cobra internals. Custom compatibility layers must own help text, completion output, error formatting, command sorting, suggestions, global option placement, and `openstack complete` behavior until Python-oracle tests prove the behavior matches.

## Plugin Framework Decision

Use Caddy's module system as the existing plugin framework for CLI plugin-scope commands and service-scoped extras plugins. Caddy modules register themselves when their packages are imported, expose stable module IDs and constructors, and are compiled into a single Go binary. This matches the project's self-contained, cross-platform requirement better than subprocess plugins, shared-object plugins, or runtime-loaded external plugin binaries.

The CLI should define OpenStack-specific module namespaces, such as `openstack.commands.core`, `openstack.commands.plugins`, and `openstack.commands.extras`. Plugin packages should register command providers in those namespaces. The command registry generator should load registered modules, assert them to project-defined command-provider interfaces, and add their command metadata and handlers to the Cobra/pflag tree.

Do not use Caddy's server/runtime behavior for OpenStack CLI execution. Use only the module registration and loading model unless a later decision explicitly expands the scope. The decision record and rejected alternatives are in the [plan diary](openstack-cli-plan-diary.md).

## Decision And Question Register

This register keeps implementation-defining decisions reviewable. Status values are `decided`, `assumed`, `pending-user`, `deferred`, and `rejected`. `pending-user` items should be answered before approving full implementation. `assumed` items are current defaults that can be changed if review finds a better path.

### User Questions

| ID | Status | Question | Answer or default | Why this matters |
| --- | --- | --- | --- | --- |
| Q-001 | `decided` | Should the canonical compatibility target be the local Python OSC `/Users/ken/.local/bin/openstack`, currently reporting `openstack 9.0.0`? | Yes. Pin the first catalog to local OSC `9.0.0`. | Controls command catalog, help snapshots, output goldens, drift policy, and what counts as compatible. |
| Q-002 | `decided` | Should the Go CLI include the locally installed plugin command surface, especially Placement and Tap-as-a-Service, when Gophercloud or raw REST shims can support it? | Yes. Include plugin-scope commands, label them separately from core OSC, and implement them through a modular CLI plugin system using an existing plugin framework. | Controls command list size, test matrix scope, plugin architecture, and whether plugin APIs are first-class project work. |
| Q-003 | `decided` | For Python commands without public Gophercloud package coverage, should we implement raw REST shims through Gophercloud's authenticated service client? | Yes, when needed for compatibility and testable. Put those commands into service-scoped extras plugins, such as `nova-extras` for Nova gaps and `neutron-extras` for Neutron gaps. | Controls whether the project stops at existing SDK wrappers or reaches Python CLI equivalence where API support exists, while keeping SDK gaps modular. |
| Q-004 | `decided` | During partial implementation, should `openstack command list` mirror Python even for commands not implemented yet, or show only implemented commands? | Mirror Python output, but append `(Not Implemented Yet)` beside commands that are not implemented. Generate command stubs for unimplemented commands; when called, the stub prints `This command is not yet implemented`. | Controls replacement behavior for scripts that inspect command availability and for users who call parsed but unfinished commands. |
| Q-005 | `decided` | Should default output be treated as strict golden behavior, including column order, table shape, empty output, stderr text, and exit status? | Yes where practical, with documented exceptions. | Controls renderer design and test strictness. |
| Q-006 | `decided` | Should `--help`, `help <command>`, and completion output be compatibility surfaces? | Yes. | Controls how much Cobra default behavior must be replaced. |
| Q-007 | `decided` | Should enhanced `pretty` output use color when stdout is a TTY? | Yes, with non-TTY and `NO_COLOR` disabling color. | Controls pretty renderer dependencies and terminal behavior. |
| Q-008 | `decided` | Which auth methods must be supported in the first production-capable milestone? | Password, application credential, token, `clouds.yaml`, and the default OpenStack `OS_*` environment variables typically loaded from an RC file; track the rest. | Controls auth implementation depth and live test readiness. |
| Q-009 | `decided` | May tests read the provided `clouds.yaml` from the standard OpenStack config location or configured path? | Yes. Use the local `clouds.yaml` for tests, never commit or log secrets, and make normal Go CLI config discovery follow the same XDG/config precedence used by Python OSC/openstacksdk. | Controls whether live cloud testing can proceed without synthetic credentials and keeps normal config lookup compatible with Python behavior. |
| Q-010 | `decided` | May the implementation ever shell out to Python/OpenStackClient in production behavior? | No. Python CLI is only a reference/oracle for generation and tests. The Go CLI must be self-contained and must not shell out to the OS for production behavior, preserving macOS, Linux, and Windows support. | Controls architecture, runtime dependency guarantees, and cross-platform support. |
| Q-011 | `decided` | May read-only live tests run against `cloud6`, `flex-sjc`, `flex-dfw`, and `flex-iad`? | Yes. Create a structured cloud test capability config documenting which clouds are usable for which tests so future clouds can be added without redesigning the test harness. | Controls cloud discovery, non-destructive compatibility verification, and extensibility of live test coverage. |
| Q-012 | `decided` | May destructive and admin lifecycle tests run on `cloud6`? | Yes, with unique test prefixes such as `golang-osc-test-UUID`, cleanup, and retained diagnostics on failure. | Controls whether admin and write behavior can be fully verified. |
| Q-013 | `decided` | Should remote flex clouds remain read-only unless explicitly opted in for a run? | No. Project-level read/write tests may run on flex clouds, but tests must only delete resources they created themselves and must respect existing networks, images, servers, keys, and other resources. Admin-level tests are not available on flex clouds. | Protects shared remote environments while allowing project-scope lifecycle coverage. |
| Q-014 | `decided` | Is there a disposable project or tenant on `cloud6` for destructive tests, or should tests create one when policy allows? | Use admin-level commands on `cloud6` to create or use a dedicated test project named `golang-osc-testing` for destructive and CRUD-level test isolation. | Controls cleanup isolation and test safety. |
| Q-015 | `decided` | Are there preferred fixture resources for image, flavor, network, volume type, external network, and keypair tests? | Discover safe defaults dynamically and record them before lifecycle tests. Before starting a test, query the target cloud for currently available images, flavors, networks, volume types, external networks, and other required fixtures; use only options available on that specific cloud for that test. | Controls reliable server, volume, floating IP, and image lifecycle tests in changing cloud environments. |
| Q-016 | `decided` | What Go module path should `go.mod` use? | `github.com/crandallnet/golang-osc`. | Controls import paths and generated package names. |
| Q-017 | `decided` | What local binary name and path should be used? | Build `bin/openstack`; do not replace the existing Python `openstack`. | Protects the oracle binary and makes comparisons explicit. |
| Q-018 | `decided` | What minimum Go version should be supported? | Use the current local Go toolchain for now, then lower only if needed. | Controls dependency versions, CI setup, and language features. |
| Q-019 | `decided` | Are small dependencies beyond Gophercloud, Cobra, and pflag acceptable for YAML, table rendering, color, and golden-test helpers? | Yes. Prefer established, popular, maintained Go libraries when available. Document every dependency choice and its reasoning in the diary so it can be revisited later. | Controls build footprint, maintainability, and whether compatibility helpers are hand-rolled. |
| Q-020 | `decided` | Should generated compatibility artifacts be committed under `compat/`? | Yes. | Controls reviewability, drift reports, and reproducibility of generated command data. |
| Q-021 | `decided` | Which existing Go plugin framework should the CLI use for plugin-scope commands? | Use Caddy's module system for statically linked, in-process CLI plugins. Do not invent a custom plugin system. Documented rationale and rejected alternatives are in the diary. | Controls binary architecture, plugin distribution, command registration, versioning, and test isolation. |

### Current Assumptions And Decisions

| ID | Status | Decision or assumption | Review path |
| --- | --- | --- | --- |
| A-001 | `decided` | Use Gophercloud as the OpenStack SDK foundation. | Reopen only if Gophercloud cannot support required authentication or service-client behavior. |
| A-002 | `decided` | Use Cobra with pflag as the parser and dispatch substrate. | Alternatives and parser experiments are recorded in the diary. |
| A-003 | `decided` | Pin the first compatibility catalog to local OSC `9.0.0`, not online development docs. | Answered Q-001. |
| A-004 | `decided` | Keep Python/OpenStackClient out of production execution paths. | Answered Q-010. |
| A-005 | `decided` | Treat default OSC output, help, completion, stderr, and exit status as compatibility surfaces. | Answered Q-005 and Q-006. |
| A-006 | `decided` | Implement raw REST shims when public Gophercloud packages are missing but the OpenStack API is available and testable. | Answered Q-003. |
| A-007 | `decided` | Track locally installed plugin commands separately from core OSC commands instead of hiding them. | Answered Q-002. |
| A-013 | `decided` | Plugin-scope commands should be implemented through an existing plugin framework, not a hand-rolled plugin mechanism. | Answered Q-002; select the specific framework in Q-021. |
| A-014 | `decided` | Raw REST compatibility shims should live in service-scoped extras plugins, such as `nova-extras`, `neutron-extras`, `glance-extras`, `cinder-extras`, `keystone-extras`, `swift-extras`, and `placement-extras`. | Answered Q-003. |
| A-015 | `decided` | Parser generation should create command stubs for unimplemented commands. `openstack command list` should mirror Python output and mark unfinished commands with `(Not Implemented Yet)`. | Answered Q-004. |
| A-016 | `decided` | Enhanced `pretty` output may use color for TTY output, but must disable color for non-TTY output and when `NO_COLOR` is set. | Answered Q-007. |
| A-017 | `decided` | The first auth milestone must support password auth, application credentials, token auth, `clouds.yaml`, and standard OpenStack RC-file `OS_*` environment variables. | Answered Q-008. |
| A-018 | `decided` | Live tests may use the local `clouds.yaml`; normal CLI config discovery must follow Python OSC/openstacksdk XDG/config precedence. | Answered Q-009. |
| A-019 | `decided` | Production CLI behavior must be self-contained Go code and cross-platform across macOS, Linux, and Windows; it must not shell out to the OS for Python/OpenStackClient behavior. | Answered Q-010. |
| A-020 | `decided` | Read-only live tests may run against `cloud6`, `flex-sjc`, `flex-dfw`, and `flex-iad`, governed by a structured cloud test capability config that can include future clouds. | Answered Q-011. |
| A-021 | `decided` | Destructive and admin lifecycle tests may run on `cloud6` using unique names such as `golang-osc-test-UUID`, cleanup, and retained diagnostics on failure. | Answered Q-012 and updated by the project rename. |
| A-022 | `decided` | `cloud6` admin tests may create or use a dedicated project named `golang-osc-testing` for isolated CRUD/destructive test coverage. | Answered Q-014 and updated by the project rename. |
| A-023 | `decided` | Lifecycle tests must dynamically discover fixture resources per cloud immediately before running and record the selected values for diagnostics. | Answered Q-015. |
| A-024 | `decided` | The Go module path is `github.com/crandallnet/golang-osc`. | Answered Q-016 and updated by the project rename. |
| A-025 | `decided` | Use the current local Go toolchain as the initial Go version baseline, lowering it later only if needed. | Answered Q-018. |
| A-026 | `decided` | Prefer established, popular, maintained Go libraries for supporting functionality, and record dependency choices and rationale in the diary. | Answered Q-019. |
| A-027 | `decided` | Use Caddy's module system as the existing plugin framework for statically linked, in-process CLI plugins. | Answered Q-021. |
| A-028 | `decided` | Table output should mirror Cliff width behavior: `--max-width` and `CLIFF_MAX_TERM_WIDTH` set an explicit bound, `--fit-width` and `CLIFF_FIT_WIDTH=1` enable fitting, and terminal auto-fit applies only when width can be detected. | User requested `CLIFF_FIT_WIDTH=1` behavior; sources are linked above. |
| A-029 | `decided` | Command families are not finished or `done` until output parity testing has run against the Python OSC oracle, parity defects found by that testing have been fixed or explicitly documented as accepted exceptions, and validation has passed against mocked or live OpenStack endpoints as appropriate. Implementation coverage and command-list parity may be recorded as `implemented`, but not as completed compatibility work. | User clarified the project definition of done on 2026-05-05. |
| A-030 | `decided` | Routine Volume lifecycle suites may not mutate real Cinder service, backend, logging, migration, failover, or existing-message state unless a future test fixture scopes the operation to resources created by the suite or the user explicitly approves the operation for that run. | This keeps the Volume lifecycle suite aligned with the existing rule to delete or mutate only resources the test created. |
| A-031 | `decided` | Pretty OS image colors shown by `make os-test` should be brand-first when the source color remains legible in the Fancy table. `make os-test` reports contrast for review, and tests enforce a low dark-background legibility guard rather than a WCAG AA-style threshold, so source colors such as RHEL red, AlmaLinux blue, CentOS Stream purple, Debian red, Oracle Linux red, and SUSE green remain usable when they read cleanly. | User clarified that brand-centric colors are preferred as long as they are legible; contrast is a review aid and safety check, not the only selection rule. |
| A-032 | `decided` | Pretty OS image color definitions live in `internal/cli/pretty_os_colors.json` and are embedded into the binary at compile time. Users may edit that file before building to change OS display names, colors, sample values, match keywords, source URLs, the contrast-report background, and the minimum legibility guard. Runtime user config for this palette is deferred. | User requested a user-editable pre-compilation source file for OS colors and keywords. |
| A-008 | `assumed` | Use static and mocked tests as mandatory gates before live cloud tests. | Reopen only if test runtime becomes impractical. |
| A-009 | `decided` | Run project-level read/write tests on remote flex clouds when safe, but delete only test-created resources and never run admin-level tests there. | Answered Q-013. |
| A-010 | `decided` | Use `cloud6` for admin and destructive tests where services exist, with a dedicated `golang-osc-testing` project for isolated CRUD/destructive coverage. | Answered Q-012 and Q-014; project name updated by the rename. |
| A-011 | `decided` | Commit generated compatibility data under `compat/`. | Answered Q-020. |
| A-012 | `decided` | Build the Go binary at `bin/openstack` and preserve the Python `openstack` as the oracle. | Answered Q-017. |

## Implementation Plan

The repository should start with compatibility data, not hand-written commands. Generate and commit a pinned command catalog from the Python oracle:

* `compat/osc/9.0.0/commands.json` from `openstack command list -f json`.
* `compat/osc/9.0.0/completion.bash` from `openstack complete`.
* `compat/osc/9.0.0/help/<command>.txt` from `openstack help <command>` for every command.
* `compat/osc/9.0.0/global-help.txt` from `openstack --help`.
* `compat/gophercloud/v2.12.0/packages.json`, generated from `go list` once a module exists, plus a source URL or module checksum.
* `compat/matrix.yaml`, where each command has `status`, `service`, `api`, `sdk_package`, `shim`, `tests`, and `notes`.
* `compat/test-matrix.yaml`, where each test suite records required service, required role, risk level, allowed clouds, setup resources, cleanup behavior, and skip conditions.
* `compat/test-clouds.yaml`, where each configured test cloud records non-secret capability policy, allowed risk levels, destructive-test project, resource prefix, and cleanup policy.

Use the following status values in the matrix:

* `unknown`: not yet inspected.
* `sdk-covered`: Gophercloud has the needed API wrapper, but the CLI command is not implemented.
* `shim-needed`: no suitable Gophercloud wrapper is known; use raw REST through a service client.
* `implemented`: command exists in Go and has unit tests.
* `golden-matched`: command output and error behavior match the Python oracle for mocked or local non-auth cases.
* `cloud-verified`: command has passed against a real OpenStack cloud.
* `blocked`: source behavior is unclear, the API is not available in the test cloud, or the required role is unavailable.

Build the core before service commands. The core includes command registration, command parsing, global options, auth/config loading, service-client creation, renderer interfaces, error mapping, logging, debug output, timing output, shell completion, and a test harness that can run Python and Go commands side-by-side.

## Detailed Workplan

This workplan is ordered by dependency. Later phases should not start broad implementation until their prerequisite gate is complete, although small prototypes are acceptable when they answer a specific uncertainty and are recorded in the plan.

Progress states are `not-started`, `in-progress`, `done`, `blocked`, and `deferred`. A work item should move to `done` only when the named artifact exists and the validation described in the row has passed. For command families, `done` requires Python OSC output parity testing, fixes for discovered parity issues or documented accepted exceptions, and mocked or live endpoint validation. Command-list parity and implemented handlers are implementation coverage, not completion.

| Work item | Depends on | Status | Progress evidence | Next action |
| --- | --- | --- | --- | --- |
| Maintain plan and diary split | None | `done` | This plan is the active tracker; the diary records decisions and experiments. | Keep both updated as work lands. |
| Pin Python OSC oracle | None | `done` | `compat/osc/9.0.0/metadata.json` records `/Users/ken/.local/bin/openstack`, `openstack 9.0.0`, command counts, and the sanitized generation commands. | Reopen only for explicit oracle upgrades or regeneration drift. |
| Generate OSC command catalog | Python OSC oracle | `done` | `tools/osc-catalog` generated `commands.json`, `global-help.txt`, `completion.bash`, `metadata.json`, and 594 per-command help snapshots under `compat/osc/9.0.0`. | Use the catalog to seed command registry generation and compatibility matrices. |
| Seed command compatibility matrix | OSC command catalog | `done` | `tools/matrix` generated `compat/matrix.yaml` with one row per local command, initial status, service/API mapping, plugin-scope markers, and the implemented `command list` row. | Fill SDK package coverage, shim requirements, command-specific tests, and status changes as service work lands. |
| Seed cloud test matrix | Cloud test matrix in this plan | `done` | `tools/matrix` generated `compat/test-matrix.yaml` and non-secret `compat/test-clouds.yaml` from the agreed cloud safety rules. | Keep the YAML files updated as live discovery adds service and fixture details. |
| Create Go module and package skeleton | None | `done` | `go.mod`, `cmd/openstack`, and `internal/cli` exist. `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and basic binary smoke checks pass with workspace-local Go caches. | Expand the package layout as the catalog generator, renderers, auth, and service clients land. |
| Generate Cobra/pflag command registry | OSC command catalog; Go module skeleton | `done` | The CLI embeds `compat/osc/9.0.0/commands.json` and builds nested Cobra commands for all 594 cataloged command paths. Unimplemented leaves are runnable stubs and representative registry tests pass. | Extend generated metadata with command-local flags, aliases, deprecated forms, and implementation ownership as the matrix lands. |
| Select CLI plugin framework | Dependency policy; plugin architecture decisions | `done` | Caddy's module system is selected for statically linked, in-process CLI plugins; rationale and rejected alternatives are documented in the diary. | Revisit only if dependency footprint or integration prototype shows it is unsuitable. |
| Implement parser compatibility layer | Cobra/pflag registry | `in-progress` | Initial root/global parsing exists, including table formatter width flags, auth flags, common formatter flags, command-local flags for many implemented commands, OSC snapshot leaf help, OSC snapshot root help, argparse-style invalid flag usage, and Cliff-style fuzzy invalid-command suggestions. `complete` emits the captured OSC completion script. Root help still cannot be live-compared byte-for-byte against Python OSC because the Python root auth option groups have been observed to appear in different orders between invocations. Complete command-local flag coverage, sorting, aliases, option placement, and more parser edge cases still need oracle-backed work. | Continue expanding parser cases beyond the current static gate, especially mutually exclusive flags, alias handling, global option placement, and all command-local flags. |
| Add parser and help parity tests | Parser compatibility layer | `in-progress` | `tools/compat-check` compares the pinned Python OSC oracle with `./bin/openstack` for static no-cloud cases covering completion, command-list output, representative leaf help, invalid command errors, and invalid flag errors. `make compat-static` runs the required subset, and `make compat-static-all` adds `--help` parity for every cataloged command. The known static gaps are nondeterministic Python root help ordering and intentionally different Go plugin module reporting. | Add more parser edge cases and default-output commands as compatibility behavior lands. |
| Implement OSC-compatible renderers | Go module skeleton | `in-progress` | Initial table renderer supports Cliff-style `--max-width`, `--fit-width`, `CLIFF_MAX_TERM_WIDTH`, and `CLIFF_FIT_WIDTH` behavior. Shared list/show renderers now cover `table`, `json`, `yaml`, `csv`, `value`, `shell` for show output, column selection, noindent, prefixes, and basic sorting. JSON list/show output preserves the command's declared column or field order instead of Go map ordering. The Go-only `pretty` path now uses Charm Bubbles table output, accepts `--compact` and `OS_COMPACT=1` to remove Pretty row separators, accepts `--no-compact` to restore expanded Pretty output when `OS_COMPACT=1` is set, marks the first rendered server network IP with a dot marker while aligning later IP labels under it, formats UUID-like resource reference columns such as port, router, network, project, user, server, image, flavor, and volume as UUIDs, groups `command list` command families such as `server` and `network` into one row with multiline subcommands, colorizes OS image names from a compile-time embedded editable JSON palette, reports OS color contrast through `make os-test`, and renders Pretty wait progress as padded action blocks with an action line above the animated progress bar. | Add oracle-backed golden tests and close remaining formatter gaps, including exact CSV quote modes, shell escaping, empty-output behavior, command-specific default columns, and command-level wait progress integration. |
| Implement auth/config resolution | Go module skeleton; renderer basics | `in-progress` | `token issue` uses Gophercloud `clouds.Parse` for `OS_CLOUD` and `--os-cloud`, falls back to standard `OS_*` environment auth, and accepts password, token, and application credential flags. `OS_CLOUD=cloud6 ./bin/openstack token issue -f json` passed a live smoke check. | Implement exact Python-compatible XDG/config discovery for `clouds.yaml` and `clouds-public.yaml`, `OS_CLIENT_CONFIG_FILE`, full auth precedence, TLS behavior, interface, region behavior, prompting, and remaining auth types. |
| Add live cloud discovery | Auth/config resolution | `in-progress` | `tools/cloud-discovery` and `make discover CLOUD=name[,name]` write non-secret JSON reports under `compat/live-clouds/` and print a terminal summary table with cloud status, service count, API probe status, fixture status, lifecycle eligibility counts, and report path. Reports include service catalog endpoints, API version details for Compute, Network, and Block Storage, extension probes, safe fixture candidates for images, flavors, networks, volume types, and roles where visible, and structured test eligibility with skip reasons. Reports were refreshed for `cloud6`, `flex-sjc`, `flex-dfw`, and `flex-iad`. | Add quota, external-network, server, volume, image import/store, and object-store fixture discovery before broader lifecycle suites. |
| Implement Gophercloud client factories | Auth/config resolution | `in-progress` | Shared provider setup and service-client helpers exist for Identity v3, Compute v2, Image v2, Network v2, Block Storage v3, Object Storage v1, and Placement v1. `token issue`, the implemented Identity read commands, and the first cross-service read commands use these helpers. Placement sets an explicit default microversion when `OS_PLACEMENT_API_VERSION` is absent. Compute honors `OS_COMPUTE_API_VERSION`, defaults to microversion `2.53` for general reads so Nova service and hypervisor IDs match the UUID-shaped Python OSC output observed on `cloud6`, and commands with higher minimum Compute microversions discover and use the service's supported maximum when no explicit version is set. Image honors `OS_IMAGE_API_VERSION` for Glance commands that require an explicit Image API microversion, such as store details. Block Storage honors `OS_VOLUME_API_VERSION`, and commands with minimum Cinder microversions now discover and use the service's supported maximum when no explicit version is set. | Add endpoint selection parity, interface/region edge cases, debug logging, timing, complete microversion flag parsing, and remaining service-specific client setup details. |
| Implement lookup and error compatibility | Client factories; parser tests | `in-progress` | Typed lookup errors now distinguish not-found and ambiguous matches, and shared helpers exist for partial failures and normalized OpenStack HTTP error messages. Existing name-or-ID lookup paths still need to adopt resource-specific typed errors and oracle-backed stderr text. | Thread the helpers through service lookups, delete paths, HTTP error mapping, partial failure reporting, and `--debug` behavior. |
| Implement CLI-local commands | Parser layer; renderers | `in-progress` | `--version`, root `--help`, `help`, `complete`, catalog-backed `command list`, Caddy-backed `module list`, and initial `configuration show` are wired. `configuration show` resolves the same local cloud/auth inputs as the Go CLI, masks secrets by default, and passed a live `cloud6` smoke check. | Add oracle-backed parity for the full OpenStackSDK configuration dictionary, including defaults not currently emitted by the Go CLI. |
| Implement Identity read bootstrap | Auth/config; client factories; renderers | `in-progress` | `token issue`, `catalog list/show`, `domain list/show`, `endpoint list/show`, `group list/show`, `project list/show`, `region list/show`, `role list/show`, `role assignment list`, `implied role list`, `mapping list/show`, `identity provider list/show`, `federation protocol list/show`, `service list/show`, `service provider list/show`, and `user list/show` are implemented through Gophercloud or narrow authenticated Identity REST reads. Available fixtures passed live JSON smoke checks on `cloud6`, and `compat/matrix.yaml` records current command status. | Add oracle parity tests, complete remaining filters and domain-aware lookup semantics, then continue with Identity write/admin commands. |
| Roll out service read commands | Identity bootstrap; lookup/error helpers | `in-progress` | Compute, Image, Network, Volume, Object Store, Placement, and Common now have initial read slices. `tools/compat-check` supports live fixture placeholders such as `{server}`, `{volume}`, `{network}`, `{network_agent}`, `{volume_message}`, and related read fixtures so show commands can use IDs discovered from the Python oracle at run time. Static and live oracle parity has promoted 57 matrix rows to `golden-matched`, including `command list` and the `cloud6` default table and JSON read suite for `aggregate list`, `compute service list`, `flavor list/show`, `hypervisor list/show`, `hypervisor stats show`, `image list/show`, `ip availability list/show`, `keypair list/show`, `network list/show`, `network agent list/show`, `port list/show`, `project list/show`, `router list/show`, `security group list/show`, `security group rule list/show`, `server list/show`, `server group list`, `subnet list/show`, `subnet pool list`, `volume list/show`, `volume attachment list/show`, `volume backend capability show`, `volume backend pool list`, `volume backup list`, `volume group list`, `volume group type list`, `volume message list/show`, `volume qos list`, `volume service list`, `volume snapshot list`, `volume transfer request list`, `volume type list/show`, `volume summary`, `block storage log level list`, `block storage resource filter list/show`, and `block storage volume/snapshot manageable list`. The current expanded `cloud6` live suites pass for both default table output and JSON output with only the documented static known gaps for root help ordering and Go module reporting; `aggregate show`, `server group show`, `volume snapshot show`, `subnet pool show`, and `volume qos show` were skipped because `cloud6` had no matching fixture rows during the run. | Keep promoting the remaining implemented and cloud-verified commands through fixture-aware live comparisons, including command flags, alternate formats, write lifecycle output, admin-only commands, and remote service breadth smoke on the flex clouds. |
| Implement write lifecycle framework | Service read commands; lookup/error helpers | `in-progress` | A lifecycle helper now creates unique `golang-osc-test-*` resource names, records selected fixtures, runs cleanup callbacks in LIFO order, writes retained JSON diagnostics, and supports suites selected with `--suite`. `tools/lifecycle-smoke` and `make lifecycle CLOUD=name SUITE=keypair|server|volume|quota|image|network|object|all` now cover keypair, Compute server, Volume v3, quota, Image v2, Network v2, and Object Store disposable lifecycles. `make help` lists the available lifecycle suites. The keypair, server, Volume, quota, image, and network suites passed on `cloud6`; the object-store suite passed on `flex-dfw`. The server, Volume attachment, quota, image, network, and object-store suites include paired Python-vs-Go write-output parity for their safe disposable mutation paths. Successful diagnostics can be retained with `--keep-success`, and `compat/lifecycle-diagnostics/` is ignored to avoid committing cloud details. | Extend the smoke runner to remaining group snapshot, floating IP, Network extension, Volume backup/type/manage-existing, and admin-scoped fixtures, adding idempotent cleanup wrappers and per-command failure artifacts as each suite lands. |
| Implement service write commands | Write lifecycle framework | `in-progress` | Low-risk Image writes now include `image create`, `image delete`, `image set`, `image unset`, `image stage`, `image import`, `image add project`, `image remove project`, Glance cache mutations, and Glance metadef namespace/object/property/resource-type lifecycle commands. The `cloud6` image lifecycle passed paired Python-vs-Go write-output parity for image create/delete/set/unset/stage/import, image membership add/remove, and the Glance metadef mutation paths that use test-created namespaces, objects, properties, and resource-type associations. Compute writes now include `keypair create/delete`, `server group create/delete`, and the API-backed `server` namespace except `server ssh`: create/delete/set/unset, lifecycle actions, rebuild/rescue/resize/migrate/evacuate/shelve/unshelve, server image and backup creation, network/port/fixed-IP/floating-IP/security-group/volume attach and detach, migration actions, and server volume set/update. The `cloud6` server lifecycle suite passed for test-created server create/delete, server group create/delete/show, event list/show, set/unset, lock/unlock, reboot, stop/start, rebuild, pause/unpause, suspend/resume, rescue/unrescue, volume attach/detach/list/set/update, security-group add/remove, port add/remove, and network add/remove where a safe alternate network fixture exists. The same suite records structured skips for fixed/floating IP actions, migration actions, evacuation, crash dump, resize completion when the cloud never reaches `VERIFY_RESIZE`, server image/backup completion when the cloud never returns the image to `active`, and offloaded-unshelve completion when the cloud accepts unshelve but leaves the server offloaded. Network writes now include `address group create/delete/set/unset`, `address scope create/delete/set`, `floating ip create/delete/set/unset`, `floating ip port forwarding create/delete/list/set/show`, `network create/delete/set/unset`, `network qos policy create/delete/set`, `network qos rule create/delete/list/set/show`, `network rbac create/delete/set`, `network segment create/delete/set`, `network trunk create/delete/set/unset`, `port create/delete/set/unset`, `router add/remove gateway`, `router add/remove port`, `router add/remove route`, `router add/remove subnet`, `router create/delete/set/unset`, `security group create/delete/set/unset`, `security group rule create/delete`, `subnet create/delete/set/unset`, and `subnet pool create/delete/set/unset`, including project lookup, stateful/stateless selection, standard tags, address-scope association, floating IP allocation, port lifecycle, router interface attachment, router extra-route actions, QoS policy false-boolean request handling, QoS rule subtype validation, RBAC target-project handling, segment extension fields, trunk subport parsing, port-forwarding range parsing, and Python-compatible raw show output. The `cloud6` network lifecycle passed paired Python-vs-Go write-output parity for the safe disposable address group, address scope, network, subnet, port, router, router interface, router route, security group, and security group rule mutation paths; port forwarding, QoS policy, QoS rule, segment, and trunk commands are implemented but still need a cloud exposing those endpoints because `cloud6` returns the same Neutron `404` for Python OSC and the Go CLI. Router gateway add/remove are implemented and match Python's `cloud6` extension-gated error, but successful gateway action verification needs a cloud exposing `external-gateway-multihoming`. Object Store writes now include `container create/delete/set/unset` and `object create/delete/save/set/unset`, verified with paired Python-vs-Go write-output parity and disposable containers and objects on `flex-dfw` because `cloud6` currently has no usable object-store endpoint. Volume writes now have implementation coverage for all commands in the pinned `openstack.volume.v3` catalog and disposable lifecycle validation on `cloud6` for `volume create/delete/set/unset`, `volume snapshot create/delete/set/unset`, `volume revert`, `volume transfer request create/delete/accept`, `volume qos create/delete/set/unset`, `volume group type create/delete/set`, `volume group create/delete/set`, `consistency group create/delete/set`, `volume attachment create/show/list/set/complete/delete`, and the associated show/list reads. The suite records explicit skips for commands that are implemented but not safe or available to validate on `cloud6`: backup mutations while `cinder-backup` is down, volume type mutations while test-created type deletion hits the missing `__DEFAULT__` Cinder failure, group snapshot mutations until a valid source group fixture exists, manage-existing remote-source calls until backend-local unmanaged storage fixtures exist, and real service/backend/admin mutations. Common admin writes now include `quota set` and `quota delete` support for Compute, Volume, and Network quotas, with project-scoped quota mutation live-verified on `cloud6` through a dedicated test project and Python-oracle aggregate `quota show -f json` comparisons before and after reset. Compute and Volume quota-class updates for `--class` and `--default` are implemented but intentionally unpromoted because they change default quota-class state rather than only a test-owned project. | Add automated setup/teardown fixtures for remaining blocked Compute, Volume, Image, Network extension, and Object Store edge fixtures, then continue with command-level parity for less common flags and current-project image membership without `--project`. |
| Close Gophercloud SDK gaps with shims | Command matrix; service implementations | `in-progress` | Narrow shims now cover Neutron availability zones, Neutron QoS `minimum-packet-rate` rule operations, OSC-shaped quota aggregation, quota reset/update requests, service-catalog version discovery, Identity federation/provider REST reads, Neutron service-provider reads, Nova console auth token lookup, Nova deprecated `os-agents` and `os-hosts` reads, Nova migration list reads, Glance cache operations, Glance image activation, Glance image import, Glance image multi-store delete, Glance image tag delete, Glance metadef namespace/object/property/resource-type read and write operations, Cinder volume summary, Cinder cleanup workers, Cinder clusters, Cinder consistency groups, Cinder consistency group snapshots, Cinder groups, Cinder group snapshots, Cinder group types, Cinder manageable volume and snapshot lists, Cinder backend capabilities, Cinder resource filters, Cinder log-level reads and writes, Cinder host freeze/thaw, Cinder messages, Cinder volume migration, Cinder volume revert, and Cinder service enable/disable calls. They are implemented through authenticated Gophercloud clients where typed packages are missing or do not preserve the Python output shape. The matrix marks these package gaps explicitly. | Move shims behind service-scoped extras packages/plugins as that plugin boundary is built, and add mocked HTTP tests for each shim endpoint before expanding beyond read-only calls. |
| Add OSC version drift workflow | Generated catalogs | `not-started` | Version policy is documented only. | Add regeneration and diff workflow for command, option, help, output, and SDK coverage drift. |

Latest write-output parity evidence, 2026-05-06: the current equivalent invocations are `make lifecycle CLOUD=cloud6 SUITE=server`, `make lifecycle CLOUD=cloud6 SUITE=volume`, `make lifecycle CLOUD=cloud6 SUITE=quota`, `make lifecycle CLOUD=cloud6 SUITE=image`, `make lifecycle CLOUD=cloud6 SUITE=network`, and `make lifecycle CLOUD=flex-dfw SUITE=object`. These suites passed before the Make target consolidation after adding paired Python-vs-Go output comparisons to the disposable lifecycle runner. The server suite compares safe server lifecycle write output for create/delete, set/unset, lock/unlock, reboot, stop/start, rebuild, pause/unpause, suspend/resume, rescue/unrescue, volume attach/detach/set/update, security-group add/remove, port add/remove, and network add/remove. The Volume suite compares `volume attachment create/set/complete/delete`, and the quota suite compares project-scoped `quota set` and `quota delete` output while preserving the existing `quota show -f json` oracle checks before and after reset. The image suite compares disposable image create/delete/set/unset/stage/import, image membership, and Glance metadef mutations. The network suite compares disposable address group, address scope, network, subnet, port, router, router interface, router route, security group, and security group rule mutations. The object-store suite compares disposable container and object mutations on `flex-dfw`, where Swift is available.

| Gate | Required completed work items | Unlocks |
| --- | --- | --- |
| Oracle catalog gate | Pin Python OSC oracle; generate OSC command catalog. | Parser generation, help parity tests, command matrix seeding, and command drift reports. |
| Repository skeleton gate | Create Go module and package skeleton. | Buildable CLI stub, generated command registry, and CI jobs. |
| Parser gate | Generate Cobra/pflag command registry; select CLI plugin framework; implement parser compatibility layer; add parser and help parity tests. | CLI-local commands, parse-only compatibility tests, service command stubs, and plugin command registration. |
| Renderer gate | Implement OSC-compatible renderers. | Any command that prints structured output. |
| Auth/config gate | Implement auth/config resolution; add live cloud discovery. | Live cloud tests, `token issue`, service catalog tests, and per-service client creation. |
| Client factory gate | Implement Gophercloud client factories. | Service read commands and cloud matrix execution. |
| Lookup/error gate | Implement lookup and error compatibility. | Write commands, delete commands, wait commands, and negative compatibility tests. |
| Service read gate | Implement Identity read bootstrap; roll out service read commands for at least one service. | Service write paths and repeated service rollout. |
| Write lifecycle gate | Implement write lifecycle framework. | Cloud lifecycle tests and destructive/admin command coverage. |
| Shim gate | Close Gophercloud SDK gaps with shims as needed. | Image metadef/cache, Network extension gaps, Volume admin/group gaps, and other uncovered APIs. |
| Drift gate | Add OSC version drift workflow. | Controlled upgrades from OSC `9.0.0` to later versions. |

### Foundation Phase

Create the repository skeleton and compatibility data first. This phase should add the Go module, the `cmd/openstack` entry point, internal package boundaries, `compat/osc/9.0.0`, `compat/gophercloud/v2.12.0`, `compat/matrix.yaml`, and `compat/test-matrix.yaml`.

The catalog generator should capture command inventory, help text, completion output, the Python OSC version, the Python executable path, and the environment used to generate the files. It should redact secrets and avoid storing live `clouds.yaml` content. The generated command catalog becomes the source for the parser, the matrix, and the initial documentation, not an incidental report.

### Parser And CLI Phase

Generate the command tree from the pinned catalog. Use Cobra and pflag for nested commands and flags, but keep compatibility behavior in local packages. The generated registry should distinguish canonical command paths, real aliases, deprecated forms, intermediate nodes, plugin commands, hidden or help-only behavior, implemented commands, and unimplemented stubs.

Parser tests should run before any service code exists. They should compare `--help`, `help <command>`, missing command errors, unknown flag errors, mutually exclusive flag behavior, option placement, `command list`, and `complete` against the Python oracle. During partial implementation, `command list` should mirror Python output while marking unfinished commands with `(Not Implemented Yet)`, and unimplemented command stubs should print `This command is not yet implemented`. These tests are static and should not need cloud credentials.

### Renderer Phase

Implement output renderers before implementing broad service coverage. Every service command will depend on the same formatter behavior, so defects here would multiply quickly.

The first renderer tests should be golden tests using fixed rows and nested values. After that, each implemented command should add Python-vs-Go output comparisons for default output and each machine format OSC supports for that command. The enhanced `pretty` renderer should have its own tests, but it should never become the default compatibility output.

### Auth And Client Phase

Implement authentication and configuration resolution before service commands. This phase should validate `clouds.yaml`, `clouds-public.yaml`, `OS_CLIENT_CONFIG_FILE`, `OS_CLOUD`, `OS_*` environment variables, global auth flags, TLS flags, `--os-interface`, region selection, and endpoint selection against OSC behavior. The cited OSC configuration docs are the starting point, but local oracle tests decide behavior for the pinned version.

After auth works, add service-client factories for Identity, Compute, Image, Network, Object Store, Placement, and Volume. Each client factory should expose the selected endpoint, interface, region, API version, microversion, and debug/timing hooks in a way the compatibility harness can verify without leaking tokens.

### CLI-Local Phase

Implement commands that do not need OpenStack service APIs: `--version`, `--help`, `help`, `command list`, `complete`, `module list`, and as much of `configuration show` as OSC exposes without making service calls. These commands prove parser generation, output formatting, and local compatibility behavior before service-specific behavior enters the codebase.

### Identity Bootstrap Phase

Implement Identity read commands next because every cloud test depends on a working token and service catalog. Start with `token issue`, `catalog list`, `catalog show`, `project list/show`, `domain list/show`, `user list/show`, `group list/show`, `role list/show`, `service list/show`, and `endpoint list/show`.

Admin-only Identity create/set/delete commands should be tested against `cloud6` first. Remote flex clouds should be used for non-admin read behavior and token/catalog verification only unless a later test account explicitly grants the needed role.

### Service Readout Phase

Roll out read-only service commands in this order: Compute, Network, Volume, Image, Object Store, and Placement. This order exercises high-value operator workflows early while keeping risk low. Placement is plugin scope, but it is visible in the local command inventory and Gophercloud has visible Placement packages, so it should be tracked rather than deferred indefinitely.

Each service should follow the same pattern: fill the command matrix, map commands to Gophercloud packages or shim candidates, implement list/show first, add output parity tests, run static and mocked HTTP tests, then run eligible cloud tests from the matrix.

### Write And Lifecycle Phase

After read paths and lookup behavior are stable, add write paths service by service. Start with low-risk lifecycle resources that can be created with unique names, verified, and deleted cleanly. Every lifecycle test must record setup, teardown, idempotent cleanup, timeout behavior, and what diagnostic artifacts to keep if cleanup fails.

Remote flex clouds may run project-scoped read/write lifecycle tests when the suite creates uniquely named resources, deletes only those resources, and records cleanup diagnostics. Admin lifecycle and destructive tests should run on `cloud6` only unless the user later provides a remote admin test cloud.

### Shim And Gap Closure Phase

When Gophercloud lacks an API wrapper, add a narrow raw REST shim through the authenticated Gophercloud service client. Do not hide these shims in command code. Each shim should have a package-level owner, request and response structs, microversion notes, mocked HTTP tests, and a matrix entry pointing to the exact OSC command that needs it.

The first expected shim backlogs are Image cache/metadef/store commands, Network extension gaps, Volume admin/group gaps, and some Compute admin gaps. This list is not final; it must be revised after direct module inventory and command-by-command inspection.

### Drift And Release Phase

Add a catalog regeneration workflow after the first usable command set exists. The workflow should compare the pinned local OSC `9.0.0` catalog against a newer Python OSC in a controlled environment and report added commands, removed commands, option changes, help changes, output changes, and Gophercloud coverage changes. Upgrades should be explicit; the project should not silently chase development docs.

## Output Plan

The default renderer must match OSC, even though OSC's own guide describes the default as pretty-printed with Python `prettytable`. In this project, call that renderer `table` or `osc-table` internally to avoid confusing it with the new enhanced `pretty` format.

`--format=pretty` and `--pretty` must set the same output mode. `OS_PRETTY=1` should make `pretty` the default output mode for Go CLI invocations, but explicit `--format` values such as `-f json` must override that environment default. `--pretty` should be parsed as a global convenience flag, but it should apply only to commands that produce structured output. Commands whose Python output is intentionally raw text, such as console logs or object saves, should remain compatible by default and only use enhanced output where it makes sense and tests define the behavior.

The enhanced pretty renderer should be stable enough for screenshots and operator use, but it is not a script API. It can use aligned sections, labels, status coloring when stdout is a TTY, condensed nested fields, service-specific summaries, and progress bars for wait-capable operations. It must degrade to plain text when color is disabled or stdout is not a TTY. Charm Bubbles and Lip Gloss are restricted to this `pretty` path; the default OSC-compatible renderer must not depend on them for behavior.

## Test Strategy

The compatibility harness should run every command in three layers.

Static tests do not require a cloud. They compare command discovery, help text, option parsing, invalid-option errors, renderer behavior, and `--format` support against the Python oracle.

Mocked HTTP tests use captured Python request/response fixtures or service-specific fake servers. They verify URL paths, query strings, request bodies, microversion headers, response extraction, name-or-ID lookup, output columns, and error mapping.

Cloud tests run against configured real OpenStack clouds when credentials are available. These tests should be tagged by service and risk level. Destructive tests must create resources with unique names, clean up after themselves, and preserve partial-failure evidence in logs.

A command should not be marked `cloud-verified` unless both the Python and Go command were run against the same cloud state or the test fixture proves that state differences are irrelevant.

Use the provided clouds deliberately:

* `cloud6`: local cloud with full admin access. Use it for admin-only behavior, destructive lifecycle tests, quota/admin commands, identity administration, and fast iteration. Do not treat missing services on `cloud6` as product gaps.
* `flex-sjc`, `flex-dfw`, and `flex-iad`: remote clouds with broader service coverage but no admin-level access. Use them for non-admin service coverage, read compatibility, project-scoped lifecycle tests where cleanup is proven, and plugin/service availability checks. Do not use them for admin-only negative conclusions.

The matrix should record cloud verification per cloud, not as a single boolean. A command can be `cloud-verified: cloud6` for admin behavior, `cloud-verified: flex-sjc` for broader service behavior, or both. If a service exists only on remote flex clouds but a command requires admin rights, mark the command `blocked` with the exact reason rather than treating it as unsupported.

### Cloud Capability Baseline

The cloud descriptions below come from the provided testing notes, not from a completed service discovery pass. Before running broad live tests, add a discovery job that records each cloud's service catalog, regions, endpoint interfaces, supported API versions, and the role visible to the configured credential.

| Cloud | Known access profile | Primary use | Default live-test risk | Caveat |
| --- | --- | --- | --- | --- |
| `cloud6` | Local cloud, full admin access. | Admin behavior, destructive lifecycle tests, identity administration, quotas, fast iteration, negative permission-independent tests. | Read, write-cleanup, destructive, and admin-write, subject to service availability. | It does not have all services, so missing service results are environment gaps, not product gaps. |
| `flex-sjc` | Remote cloud, more services, no admin-level access. | Read service coverage, non-admin behavior, service availability checks, project-scoped lifecycle tests with proven cleanup. | Read and write-cleanup for test-created resources only. | Admin failures are expected and should not be marked as command incompatibility. |
| `flex-dfw` | Remote cloud, more services, no admin-level access. | Same as `flex-sjc`, with regional drift coverage. | Read and write-cleanup for test-created resources only. | Admin failures are expected and should not be marked as command incompatibility. |
| `flex-iad` | Remote cloud, more services, no admin-level access. | Same as `flex-sjc`, with regional drift coverage. | Read and write-cleanup for test-created resources only. | Admin failures are expected and should not be marked as command incompatibility. |

### Test Matrix

Use `Yes` when a suite should run by default, `Conditional` when it depends on service discovery, quota, role, or fixture availability, `Opt-in` when the suite is disabled by default because it may create remote resources, and `No` when the cloud should not be used for that suite.

| Test suite | Static or mocked | `cloud6` | `flex-sjc` | `flex-dfw` | `flex-iad` | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Command catalog generation | Yes | No | No | No | No | Uses local Python OSC only: `command list`, global help, command help, and completion snapshots. |
| Parser and help parity | Yes | No | No | No | No | Covers command paths, aliases, global option placement, invalid commands, invalid flags, and help text. |
| Renderer parity | Yes | No | No | No | No | Covers OSC formats and the Go-only `--format=pretty` and `--pretty` aliases. |
| Auth/config precedence without API calls | Yes | Conditional | Conditional | Conditional | Conditional | Tests config resolution and redaction. It may use cloud names but must not persist secrets. |
| Token issue and service catalog | Mocked | Yes | Yes | Yes | Yes | First live gate for every configured cloud. Do not start service tests until this passes. |
| Service discovery | Mocked | Yes | Yes | Yes | Yes | Records catalog services, regions, interfaces, API versions, and extension lists where available. |
| CLI-local commands | Yes | Conditional | Conditional | Conditional | Conditional | `--version`, `--help`, `help`, `command list`, and `complete` are static. `configuration show` may use a selected cloud config. |
| Identity non-admin read | Mocked | Yes | Conditional | Conditional | Conditional | `token issue` and catalog are expected. Project, domain, user, group, and role list behavior depends on policy. |
| Identity admin read/write | Mocked | Yes | No | No | No | Includes users, projects, domains, groups, roles, assignments, endpoints, services, policies, limits, trusts, and application credentials where admin policy allows. |
| Compute read | Mocked | Conditional | Conditional | Conditional | Conditional | Run where Nova exists. Covers servers, flavors, keypairs, images-as-seen-by-compute, aggregates, hypervisors, services, limits, and usage as policy allows. |
| Compute project lifecycle | Mocked | Conditional | Conditional | Conditional | Conditional | Creates and deletes only test-created project-owned servers, keypairs, and server groups. Query each target cloud for current image, flavor, network, quota, and cleanup options before running. Existing resources must not be deleted. |
| Compute admin operations | Mocked | Conditional | No | No | No | Includes hosts, services, migrations, evacuate, shelve/unshelve admin behavior, lock/unlock policy, and quota-like admin paths. |
| Network read | Mocked | Conditional | Conditional | Conditional | Conditional | Run where Neutron exists. Covers networks, subnets, ports, routers, floating IPs, security groups, QoS, trunks, RBAC, extensions, and provider fields as policy allows. |
| Network project lifecycle | Mocked | Conditional | Conditional | Conditional | Conditional | Creates and deletes only test-created project-owned networks, subnets, routers, ports, security groups, floating IPs, QoS policies, and trunks when quotas allow. Query each target cloud for current external networks, quotas, and extension availability before running. Existing resources must not be deleted. |
| Network admin and extension writes | Mocked | Conditional | No | No | No | Includes agents, provider networks, address scopes when admin-only, default security group rules, flavors, service profiles, segment ranges, and Tap-as-a-Service admin paths. |
| Volume read | Mocked | Conditional | Conditional | Conditional | Conditional | Run where Cinder v3 exists. Covers volumes, snapshots, backups, types, transfers, attachments, pools, services, quotas, and limits as policy allows. |
| Volume project lifecycle | Mocked | Conditional | Conditional | Conditional | Conditional | Creates and deletes only test-created project-owned volumes, snapshots, backups, and transfers when quota and backend policy allow. Query each target cloud for current volume types, quotas, and backend policy before running. Existing resources must not be deleted. |
| Volume admin operations | Mocked | Conditional | No | No | No | Includes services, clusters, host operations, cleanup, manageable volumes/snapshots, groups, group snapshots, group types, messages, and resource filters. |
| Image read | Mocked | Conditional | Conditional | Conditional | Conditional | Run where Glance v2 exists. Covers images, members, tasks, stores, and import capabilities as policy allows. |
| Image project lifecycle | Mocked | Conditional | Conditional | Conditional | Conditional | Creates, uploads/stages/imports, updates, members, and deletes only test-created project images when policy and storage impact are acceptable. Query each target cloud for current image import/store capabilities before running. Existing images must not be deleted. |
| Image admin/cache/metadef | Mocked | Conditional | No | No | No | Includes image cache, metadef namespaces, objects, properties, resource types, protected images, and store administration where Glance exposes them. |
| Object Store read/lifecycle | Mocked | Conditional | Conditional | Conditional | Conditional | Run where Swift exists. Project-owned container and object lifecycle can be safe on remote clouds if quotas and cleanup are verified. |
| Placement read | Mocked | Conditional | Conditional | Conditional | Conditional | Run where Placement exists. Covers resource providers, traits, resource classes, allocation candidates, inventories, usages, and allocations as policy allows. |
| Placement admin writes | Mocked | Conditional | No | No | No | Includes resource class, trait, inventory, aggregate, and allocation mutation behavior that normally requires admin. |
| Common read commands | Mocked | Conditional | Conditional | Conditional | Conditional | Covers `availability zone list`, `extension list/show`, `limits show`, `quota show`, and `versions show` where backing services exist. |
| Common admin/destructive commands | Mocked | Conditional | No | No | No | Covers `quota set/delete` and `project cleanup`. `project cleanup` must use the dedicated `golang-osc-testing` project on `cloud6`. |
| Name-or-ID lookup and ambiguity | Yes | Conditional | Conditional | Conditional | Conditional | Ambiguity tests often need duplicate names. Prefer mocked tests, then create duplicate test-owned names only; existing resources must not be modified or deleted. |
| Microversion negotiation | Mocked | Conditional | Conditional | Conditional | Conditional | Static and mocked tests are mandatory. Live tests run only where the service exposes the relevant microversion range. |
| Wait, async, and timeout behavior | Mocked | Conditional | Conditional | Conditional | Conditional | Server, volume, image, stack-like, and delete wait tests can consume resources. Remote variants may run only on test-created resources after cleanup policy is proven. |
| Error and negative behavior | Mocked | Conditional | Conditional | Conditional | Conditional | Prefer mocked HTTP for exact errors. Live negative tests should avoid intentional permission spam on shared remote clouds. |
| Raw REST shims | Mocked | Conditional | Conditional | Conditional | Conditional | Mocked tests are required before live tests. Live eligibility follows the service, role, and risk of the command using the shim. |
| Remote service breadth smoke | No | No | Yes | Yes | Yes | Read-only suite that finds services missing on `cloud6` and records remote-only coverage opportunities. |
| Admin coverage smoke | No | Yes | No | No | No | Read/admin-write suite proving cloud6-only admin coverage and documenting missing services separately. |
| OSC version drift | Yes | No | No | No | No | Compares regenerated Python catalogs; does not require OpenStack credentials. |

### Test Matrix Maintenance Rules

Every test suite should carry tags for `service`, `command`, `risk`, `role`, `cloud`, and `oracle`. Suggested risk values are `static`, `mocked-http`, `read`, `write-cleanup`, `destructive`, and `admin-write`. Suggested role values are `none`, `project`, and `admin`.

Live tests should run the Python and Go commands against the same cloud and, when possible, the same resource IDs. If state can change between runs, the test should use a setup fixture that creates its own resources, records IDs, runs both clients, and then cleans up.

Remote flex project-level writes are allowed when the test creates its own resources and cleans up only those resources. Tests must not delete or mutate pre-existing networks, images, servers, keys, volumes, or other resources. Admin-level tests remain unavailable on flex clouds unless a future credential explicitly grants admin access.

Skip reasons should be structured. Use exact reasons such as `service-missing`, `endpoint-missing`, `extension-missing`, `role-missing`, `quota-missing`, `policy-denied`, `unsafe-remote-write`, `fixture-missing`, or `oracle-unclear`. Skips caused by cloud capability should not lower a command's compatibility status unless the command cannot be tested anywhere.

Cloud verification should be per command, per format, and per behavior class. For example, a command can be verified for `default-output` and `json-output` on `flex-sjc`, but still need `admin-error-behavior` on `cloud6`.

## Gophercloud Gap Policy

Use public Gophercloud packages when they cover the API. When they do not, implement a narrow raw REST shim in this project using Gophercloud's authenticated `ServiceClient`; keep request/response structs close to the API, keep compatibility mapping in the command layer, and add tests that make later upstreaming straightforward.

Do not shell out to Python for production behavior. The Python client is the oracle for tests and catalog generation only.

When a gap is found, record it in `compat/matrix.yaml` with the exact Python command, missing Gophercloud package or method, OpenStack API endpoint, microversion requirement if known, test fixture, and whether the shim should be upstreamed.

## Open Questions

Open questions are tracked in the Decision And Question Register above. The implementation-blocking items are Q-001 through Q-020. After those are answered, update the register, move accepted defaults from `assumed` to `decided`, and ask for approval to proceed with implementation.

## First Iteration Checklist

* Create `go.mod`, the `openstack` command entry point, and the internal package layout.
* Add the catalog generator for Python command inventory, completion, and help snapshots.
* Add the Gophercloud package inventory generator after the module exists.
* Add `compat/matrix.yaml` seeded from the 594 local commands, all initially `unknown`.
* Add `compat/test-matrix.yaml` seeded from the matrix in this document.
* Add a live cloud discovery command that records service catalog, regions, endpoint interfaces, API versions, and skip reasons for `cloud6`, `flex-sjc`, `flex-dfw`, and `flex-iad`.
* Implement the renderer contract and golden tests for table, JSON, YAML, CSV, value, shell, and enhanced pretty output.
* Implement CLI-local commands and global option parsing.
* Implement auth/config loading to the point where `token issue`, `catalog list`, and `configuration show` can be tested.
* Fill the matrix for Identity, Compute, Network, Volume, Image, Object Store, and Placement in that order.
