# OpenStack CLI Plan Diary

This diary records decisions and experiments worked on during the project. It is project history, not the active todo list. The living plan tracks current work items, dependencies, progress, and blockers.

Use diary entries for choices made, alternatives considered, local experiments, source links, observed command behavior, and follow-up questions raised by the work. When a decision changes the execution path, update the living plan as well.

## 2026-05-02: Project Rename To golang-osc

Decision: rename the project to `golang-osc`.

Work done: updated the README, AGENTS guidance, living plan, and diary references so the project identity is `golang-osc`. The planned Go module path is now `github.com/crandallnet/golang-osc`, because that is the intended long-term public import path. The built compatibility binary remains `bin/openstack` because the product goal is still a drop-in replacement for the Python `openstack` command.

Related test naming update: lifecycle resources should use prefixes such as `golang-osc-test-UUID`, and the cloud6 dedicated destructive-test project should be `golang-osc-testing`.

## 2026-05-02: Initial Go CLI Skeleton

Work done: added the first buildable Go module at `github.com/crandallnet/golang-osc`, with `cmd/openstack`, `internal/cli`, a Cobra/pflag root command, global flag parsing skeleton, basic CLI-local command wiring, and unit tests.

Dependency note: this slice uses only Cobra and pflag because they are the selected parser substrate and are immediately exercised by code and tests. Caddy, Gophercloud, renderer libraries, and YAML libraries remain planned dependencies, but they will be added only in the slice that first uses them.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, `./bin/openstack --version`, and `./bin/openstack module list` passed with workspace-local Go caches.

## 2026-05-02: OSC 9.0.0 Catalog Generation

Work done: added `tools/osc-catalog`, a Go generator that uses the pinned local Python OpenStackClient oracle for compatibility artifacts. It writes `commands.json`, `global-help.txt`, `completion.bash`, `metadata.json`, and per-command help snapshots under `compat/osc/9.0.0`.

Result: generated artifacts record `openstack 9.0.0`, nine command groups, 594 commands, and 594 help snapshots. The generator records the oracle path and generation commands, but it does not read or store `clouds.yaml`, tokens, passwords, application credentials, or debug logs.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and the catalog generator run passed with workspace-local Go caches.

## 2026-05-02: Embedded Command Registry And Stubs

Work done: added an embedded catalog package and wired the CLI to build nested Cobra commands from the OSC 9.0.0 command catalog. All cataloged command paths now exist in the Go CLI as either implemented local commands or generated not-implemented stubs.

Behavior added: generated stubs print `This command is not yet implemented`, command-specific help can be served from captured OSC help snapshots, `openstack complete` emits the captured OSC completion script, and `openstack command list -f json` uses the embedded catalog while marking unfinished commands with `(Not Implemented Yet)`.

Known limitation: this is not full parser parity yet. Command-local flags are not generated as pflag definitions, default table output is a minimal placeholder, and root help/error formatting still needs oracle-backed compatibility work.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, `./bin/openstack command list -f json --group openstack.cli`, `./bin/openstack server list --long`, `./bin/openstack server list --help`, and `./bin/openstack complete` passed with workspace-local Go caches.

## 2026-05-02: Structured Compatibility And Test Matrices

Work done: added `tools/compat-matrix`, which generates `compat/matrix.yaml`, `compat/test-matrix.yaml`, and `compat/test-clouds.yaml`.

Result: the command matrix has one row for each of the 594 OSC commands, including service/API mapping, initial status, plugin-scope markers for Placement and Tap commands, and an implemented row for `command list`. The test matrix records static, mocked, read, write-cleanup, destructive, and admin-write suites. The cloud config records non-secret policy for `cloud6`, `flex-sjc`, `flex-dfw`, and `flex-iad`, including resource prefixes and cleanup rules.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and `go run ./tools/compat-matrix` passed with workspace-local Go caches.

## 2026-05-02: Caddy Plugin Registry Prototype

Work done: added `internal/cliplugin` as the project-owned command-provider interface on top of Caddy modules, plus a built-in local command provider registered as `openstack.commands.core.local`.

Behavior added: `module list` now uses the Caddy-backed registry and reports the Go CLI version, OSC compatibility target, and loaded command-provider module IDs. `command list` now marks both `command list` and `module list` as implemented.

Dependency note: adding `github.com/caddyserver/caddy/v2 v2.11.2` also pulled in a large transitive dependency set because the module registry lives in Caddy's root package. This matches the accepted Caddy decision, but the footprint should remain reviewable if it becomes a release concern.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, `./bin/openstack module list -f json`, `./bin/openstack command list -f json --group openstack.cli`, and `./bin/openstack configuration show` passed with workspace-local Go caches.

## 2026-05-02: Gophercloud Auth Bootstrap, Token Issue, And Cliff Width

Work done: added the first Gophercloud-backed command, `token issue`. It resolves auth through Gophercloud's `clouds.Parse` path when `OS_CLOUD` or `--os-cloud` is present, falls back to standard `OS_*` environment auth, creates a provider client, extracts Identity v3 token data, and renders `table`, `json`, `value`, and initial `pretty` output.

Renderer work: replaced fixed-width truncating tables with a width-aware table renderer. The renderer implements the same control surface as Cliff table output: `--max-width`, `CLIFF_MAX_TERM_WIDTH`, `--fit-width`, and `CLIFF_FIT_WIDTH=1`. It uses terminal width only when it can be detected, and it preserves the Cliff behavior that `--max-width` is an explicit bound.

Dependency note: this slice added [Gophercloud v2.12.0](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2), [Gophercloud `config`](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/config), [Gophercloud `clouds`](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/config/clouds), and [Gophercloud Identity v3 tokens](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens) as the OpenStack SDK foundation for the first live command. It also made [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) direct so terminal width detection is portable and does not depend on shelling out.

Sources consulted: [Cliff demo app docs](https://docs.openstack.org/cliff/latest/user/demoapp.html) document `--max-width`, `--fit-width`, `CLIFF_MAX_TERM_WIDTH`, and `CLIFF_FIT_WIDTH`; [Cliff table formatter source](https://github.com/openstack/cliff/blob/master/cliff/formatters/table.py) shows the fitting algorithm used by the Python table formatter.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, `./bin/openstack module list --max-width 52`, and `OS_CLOUD=cloud6 ./bin/openstack token issue -f json` passed with workspace-local Go caches. The live token value must not be copied into docs, commits, or issue comments.

## 2026-05-02: Identity Read Command Expansion

Work done: added a shared Gophercloud provider/Identity v3 client factory, generic list/show renderers, formatter flags for `--column`, `--noindent`, `--prefix`, `--quote`, and basic sort-column handling, plus Gophercloud-backed Identity read commands.

Commands implemented in this slice: `catalog list`, `catalog show`, `domain list`, `domain show`, `endpoint list`, `endpoint show`, `group list`, `group show`, `project list`, `project show`, `region list`, `region show`, `role list`, `role show`, `service list`, `service show`, `user list`, and `user show`.

Compatibility note: these commands are functional and use Gophercloud SDK packages directly, but they are not yet complete OSC parity. Known remaining gaps include full domain-aware name resolution, every command-local filter, exact stderr text, all formatter edge cases, output key ordering, and Python-oracle golden tests for every command/flag combination.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and live `cloud6` JSON smoke checks passed for the available Identity list/show fixtures. `group show` is implemented, but `cloud6` currently has no groups to use as a live show fixture.

## 2026-05-02: Core Service Read Command Bootstrap

Work done: added shared service-client helpers for Compute v2, Image v2, Network v2, and Block Storage v3. Added Gophercloud-backed read implementations for `server list`, `server show`, `flavor list`, `flavor show`, `image list`, `image show`, `network list`, `network show`, `volume list`, and `volume show`.

Compatibility note: this slice focuses on functional read coverage and command-surface replacement. It resolves server list flavor and image names for the default columns, but exact `show` payload parity, every list filter, microversion-specific fields, extension fields, and OSC error text still need golden tests and follow-up work.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and live `cloud6` JSON smoke checks passed for the implemented commands.

## 2026-05-02: Adjacent Service Read Expansion

Work done: added Gophercloud-backed read implementations for `subnet list/show`, `port list/show`, `floating ip list/show`, `keypair list/show`, `server group list/show`, `volume type list/show`, and `volume snapshot list/show`.

Compatibility note: this slice keeps the same pragmatic boundary as the prior read slice. The commands are functional, parse the most important OSC list filters from the captured help snapshots, and use SDK-backed name-or-ID fallbacks where the needed list/get APIs exist. Exact OSC parity still needs oracle tests for command-local flags, output columns, stderr, ambiguity behavior, repeated filter flags, microversion fields, and project/user/domain-qualified lookups. `server group show` and `volume snapshot show` are implemented but still need live fixtures because `cloud6` had no server groups or volume snapshots during this run.

Sources consulted: Gophercloud package docs and local module sources for [subnets](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets), [ports](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports), [floating IPs](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips), [keypairs](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs), [server groups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups), [volume types](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes), and [volume snapshots](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and live `cloud6` JSON smoke checks passed for `subnet list/show`, `port list/show`, `floating ip list/show`, `keypair list/show`, `server group list`, `volume type list/show`, and `volume snapshot list`.

## 2026-05-02: Object Store And Placement Read Expansion

Work done: added Object Storage v1 client support and Gophercloud-backed implementations for `container list/show`, `object list/show`, and `object store account show`. Added Placement v1 client support and Gophercloud-backed implementations for `allocation candidate list`, `resource class list/show`, `resource provider list/show`, `resource provider aggregate list`, `resource provider allocation show`, `resource provider inventory list/show`, `resource provider trait list`, `resource provider usage show`, and `trait list/show`.

Compatibility note: Placement commands need microversion headers for several endpoints. The client now honors `OS_PLACEMENT_API_VERSION` and defaults Placement to microversion `1.39` when no explicit version is provided, matching the behavior observed from the Python placement plugin on `cloud6` for these read commands. This is a useful functional default, but exact microversion negotiation and global `--os-placement-api-version` parsing still need oracle-backed tests. Object Store commands were not available on `cloud6`, so their live smoke checks used `flex-sjc`.

Sources consulted: Gophercloud package docs and local module sources for [Object Storage accounts](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/accounts), [containers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers), [objects](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects), [Placement allocation candidates](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/placement/v1/allocationcandidates), [resource classes](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceclasses), [resource providers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders), and [traits](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/placement/v1/traits).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, live `cloud6` Placement JSON smoke checks, and live `flex-sjc` Object Store JSON smoke checks passed. The main command now prints returned errors to stderr before exiting non-zero, which made live endpoint and microversion failures diagnosable.

## 2026-05-02: Pre-Implementation Question Register

Work done: reviewed the living plan for implementation-defining choices and added a decision and question register to the plan. The register separates questions that require user answers from assumptions that can be reviewed and reopened later.

Why it matters: the Go CLI implementation has several choices that affect command compatibility, test safety, generated artifacts, auth behavior, and SDK gap handling. Recording them with stable IDs makes later review easier and keeps alternatives visible without burying the active plan in discussion.

Open follow-up: get user answers for Q-001 through Q-021 in the living plan, update the register with the accepted decisions, then ask for approval to proceed with implementation.

Update: Q-001 was answered yes, so the local Python OSC `9.0.0` binary is the canonical first oracle. Q-002 was answered yes with an added architecture constraint: plugin-scope commands should be modularized through an existing CLI plugin framework. The specific plugin framework is now tracked as Q-021.

Update: Q-003 was answered yes with an added modularity rule. Raw REST shims for commands not covered by public Gophercloud packages should live in service-scoped extras plugins, such as `nova-extras` for Nova gaps and `neutron-extras` for Neutron gaps, instead of being mixed into the core command packages.

Update: Q-004 was answered with partial-implementation behavior. `openstack command list` should mirror Python output, but add `(Not Implemented Yet)` beside unfinished commands. The generated command registry should still include stubs for unimplemented commands, and calling one should print `This command is not yet implemented`.

Update: Q-005 accepted the default. Default output should be treated as strict golden behavior where practical, including column order, table shape, empty output, stderr text, and exit status, with exceptions documented when exact compatibility is not practical.

Update: Q-006 was answered yes. `--help`, `help <command>`, and completion output are compatibility surfaces, so Cobra defaults must be replaced or adapted until oracle tests prove they match.

Update: Q-007 was answered yes. Enhanced `pretty` output may use color for TTY output, but it must disable color for non-TTY output and when `NO_COLOR` is set.

Update: Q-008 was answered as the default auth set plus standard OpenStack `OS_*` environment variables from an RC file. The first production-capable auth milestone should support password auth, application credentials, token auth, `clouds.yaml`, and RC-file environment variables.

Update: Q-009 was answered with separate test and production behavior. Live tests may use the local `clouds.yaml`, but normal Go CLI config discovery should follow the same XDG/config precedence used by Python OSC/openstacksdk rather than using a Go-specific lookup order.

Update: Q-010 was answered no. Python/OpenStackClient may only be used as a reference/oracle for generation and tests. Production behavior in the Go CLI must be self-contained, must not shell out to the OS for Python behavior, and must preserve macOS, Linux, and Windows support.

Update: Q-011 was answered yes with a test-harness requirement. Read-only live tests may run against `cloud6`, `flex-sjc`, `flex-dfw`, and `flex-iad`, and the project should add a structured cloud test capability config that documents which clouds can run which tests so future clouds can be added later.

Update: Q-012 was answered yes. Destructive and admin lifecycle tests may run on `cloud6` using unique test prefixes such as `golang-osc-test-UUID`, cleanup, and retained diagnostics on failure.

Update: Q-013 changed the default. Remote flex clouds are not read-only for all tests; project-level read/write tests may run there. The safety rule is that tests may only delete or mutate resources they create themselves and must respect existing networks, images, servers, keys, volumes, and other resources. Admin-level tests are unavailable on flex clouds because the credentials do not have admin access.

Update: Q-014 was answered with a dedicated test project. `cloud6` admin-level commands may create or use a project named `golang-osc-testing` for CRUD-level and destructive test isolation.

Update: Q-015 accepted dynamic fixture discovery. Before lifecycle tests run, the harness should query the target cloud for currently available images, flavors, networks, volume types, external networks, quotas, and other required fixtures, then record the selected values. Static fixture names should not be assumed because cloud availability can change.

Update: Q-016 originally set a GitHub-hosted Go module path. It now uses `github.com/crandallnet/golang-osc`, even if the current git remote is temporarily hosted elsewhere.

Update: Q-017 accepted the default local binary path. Build the Go CLI as `bin/openstack` and preserve the existing Python `openstack` binary as the oracle.

Update: Q-018 accepted the default. Use the current local Go toolchain as the initial version baseline, and lower it later only if needed.

Update: Q-019 was answered yes with a dependency policy. Prefer established, popular, maintained Go libraries when available. Document each dependency choice and the reasoning in this diary so choices can be revisited later.

Update: Q-020 was answered yes. Generated compatibility artifacts should be committed under `compat/`, while secrets, live credentials, unsanitized logs, and sensitive cloud data must not be committed.

Update: Q-021 accepted the default process. Before implementing plugin-scope commands, do a focused review of existing Go plugin frameworks, propose a short list, choose one, and document the decision and reasoning in this diary. Do not invent a custom plugin system.

## 2026-05-02: Plugin Framework Decision

Decision: use Caddy's module system as the existing plugin framework for plugin-scope commands and service-scoped extras plugins.

Why this fits: this CLI must remain self-contained, must not shell out for production behavior, and must support macOS, Linux, and Windows. Caddy's module system registers modules when packages are imported, gives each module a stable ID and constructor, and compiles modules into the Go binary. Caddy's own architecture documentation describes a single, self-contained static binary model with modular extension points, which aligns with this project's production constraints.

Sources consulted:

* [Caddy Extending Caddy](https://caddyserver.com/docs/extending-caddy), which documents module registration with `caddy.RegisterModule`, module IDs, namespaces, host modules, guest modules, and module loading.
* [Caddy Architecture](https://caddyserver.com/docs/architecture), which describes Caddy's self-contained static binary and modular architecture.
* [Go standard library `plugin`](https://pkg.go.dev/plugin), whose warnings say Go plugins are currently supported only on Linux, FreeBSD, and macOS and call out deployment, initialization, security, and toolchain/version coupling drawbacks.
* [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin), which is mature and widely used, but launches subprocesses and communicates over RPC.
* [knqyf263/go-plugin](https://github.com/knqyf263/go-plugin), which uses WebAssembly and is portable, but would add a generated protobuf/Wasm plugin ABI and complicate direct Gophercloud command registration.
* [go-plugger static plugin example](https://pkg.go.dev/github.com/thediveo/go-plugger/examples/staticplugin), which shows static plugin registration through imported packages.
* [pluggo](https://pkg.go.dev/github.com/CAFxX/pluggo), which is a compile-time, in-process plugin framework but is old, untagged, and lacks modern module metadata.
* [plugGo](https://pkg.go.dev/github.com/seencxy/plugGo), which supports compile-time static linking and auto-registration, but is young, has no known importers on pkg.go.dev at review time, and is oriented around application boot entries rather than CLI command providers.

Comparison:

| Option | Decision | Reason |
| --- | --- | --- |
| Caddy modules | Selected. | Existing, maintained, static/in-process module framework with IDs, namespaces, constructors, and compile-time inclusion. Satisfies cross-platform and no-production-shell-out constraints. |
| HashiCorp go-plugin | Rejected for production CLI plugins. | Mature and widely used, but it launches plugin subprocesses over RPC, which conflicts with the self-contained/no-shell-out production rule. |
| Go `plugin` package | Rejected. | Not portable to Windows and has documented deployment, security, and toolchain/version coupling drawbacks. |
| Wasm go-plugin | Deferred. | Portable and sandboxed, but overkill for first-party command modules and would complicate Gophercloud access behind host functions or generated ABI boundaries. |
| go-plugger | Rejected for now. | Static plugin registration model fits, but the project appears older and less broadly maintained than Caddy. |
| pluggo | Rejected. | Simple compile-time registry, but old, untagged, and too low-level for the dependency policy. |
| plugGo | Rejected for now. | Recent and static, but young, low adoption, and centered on lifecycle/config boot entries rather than CLI command registration. |

Implementation direction: define OpenStack-specific Caddy module namespaces, such as `openstack.commands.core`, `openstack.commands.plugins`, and `openstack.commands.extras`. Each module should implement a project-owned command-provider interface. The CLI should load registered modules, assert them to that interface, and add their command metadata and handlers to the generated Cobra/pflag tree. Use only Caddy's module registration/loading model unless a later decision expands the scope; do not use Caddy's server runtime for CLI behavior.

## 2026-05-02: Command Parsing Decision And CLI Experiments

Decision: use [Cobra](https://pkg.go.dev/github.com/spf13/cobra) with [pflag](https://github.com/spf13/pflag) as the parsing and dispatch substrate for the Go `openstack` CLI.

The parser was selected for compatibility and catalog generation, not for aesthetics. OSC has hundreds of multi-word commands, shared global options, command-local formatter options, deprecated aliases, generated completion, and help text that users and scripts may depend on. The local `openstack 9.0.0` binary also accepts some global options after the command path in help-mode tests, so exact flag placement must be verified by oracle tests rather than assumed from any Go library defaults.

Sources consulted:

* [Cobra](https://pkg.go.dev/github.com/spf13/cobra)
* [Cobra shell completion](https://cobra.dev/docs/how-to-guides/shell-completion/)
* [pflag](https://github.com/spf13/pflag)
* [Viper](https://github.com/spf13/viper)
* [urfave/cli v3](https://cli.urfave.org/)
* [Kong](https://github.com/alecthomas/kong)
* [go-flags](https://github.com/jessevdk/go-flags)
* [ff](https://pkg.go.dev/github.com/peterbourgon/ff)
* [Kingpin](https://github.com/alecthomas/kingpin)

Local experiments:

| Experiment | Observation | Impact |
| --- | --- | --- |
| `openstack --os-cloud cloud6 server list --help` | The Python CLI printed help for `server list`. | Global option placement cannot be assumed from Cobra defaults; it needs oracle tests. |
| `openstack server list --os-cloud cloud6 --help` | The Python CLI also printed help for `server list`. | Some global options can appear after the command path in at least help-mode behavior. |
| `openstack server list --format pretty --help` | The Python CLI rejected `pretty` as an unrecognized formatter in this context. | `--format=pretty` and `--pretty` are Go-only extensions and must not change default OSC-compatible behavior. |

| Tool | Fit for this CLI | Useful features | Compatibility risks |
| --- | --- | --- | --- |
| [Cobra](https://pkg.go.dev/github.com/spf13/cobra) plus [pflag](https://github.com/spf13/pflag) | Selected. | Nested subcommands, persistent flags, local flags, aliases, custom help/usage templates, flag groups, required/mutually exclusive flag annotations, custom completion hooks, shell completion, and broad ecosystem use. | Default help, completion, suggestions, usage errors, and command sorting will not match OSC. Prefix/case-insensitive matching must stay disabled unless the Python oracle proves otherwise. |
| [pflag](https://github.com/spf13/pflag) alone | Good low-level flag parser, not enough by itself. | GNU/POSIX-style long flags, shorthand flags, hidden flags, repeated values, independent flag sets, and disabled flag sorting. | The project would need to build the entire command tree, help system, completion, and dispatch layer directly. |
| [urfave/cli v3](https://cli.urfave.org/) | Viable, but less attractive for strict OSC parity. | Commands/subcommands, aliases, prefix match support, dynamic shell completion for bash/zsh/fish/powershell, no dependencies except the standard library, and docs generation via `urfave/cli-docs`. | Prefix matching and permissive help are useful for many CLIs but risky for a drop-in replacement. Cobra/pflag has more direct hooks for generated command trees. |
| [Kong](https://github.com/alecthomas/kong) | Strong parser for type-driven CLIs, weaker for a generated OSC clone. | Struct-tag command definitions, nested commands, hooks, custom decoders, dynamic commands, validation, and configurable help. | OSC commands should be generated from an oracle catalog. A reflection-first, struct-first model can fight that shape unless the project writes a custom dynamic layer. |
| [go-flags](https://github.com/jessevdk/go-flags) | Good lower-level option parser, not the best command framework here. | Long/short options, optional values, option groups, generated help, env defaults, repeated values, maps, callbacks, unknown-option handling, and nested option namespaces. | Reflection tags are less natural for 594 generated commands, and command/help/completion parity would still be mostly custom work. |
| [ff / ffcli](https://pkg.go.dev/github.com/peterbourgon/ff) | Too small for this project as the main framework. | Simple flag/config/env population around Go `flag.FlagSet`; useful ideas for explicit precedence. | OSC needs a large nested command registry, completion, help, deprecation handling, and formatter-aware flags. |
| [Viper](https://github.com/spf13/viper) | Maybe useful for non-OpenStack local config, but not for auth as the primary mechanism. | Config files, environment variables, pflag binding, defaults, and precedence support. | OpenStack config has specific `clouds.yaml`, `clouds-public.yaml`, `OS_CLIENT_CONFIG_FILE`, profile merge, and auth precedence semantics. Viper's generic merge model could create subtle incompatibilities. Prefer Gophercloud `clouds` parsing plus explicit OSC compatibility code. |
| [Kingpin](https://github.com/alecthomas/kingpin) | Do not choose for new work. | Mature command/flag parser. | The maintainer says it is "CONTRIBUTIONS ONLY" and that he now uses Kong. That is not a good foundation for a long-lived compatibility CLI. |

Implementation consequences:

* Build one generated registry keyed by canonical command string, such as `server list` or `volume transfer request accept`.
* Represent OSC multi-word commands as Cobra nested commands. Intermediate nodes are non-runnable unless the Python oracle exposes them as runnable.
* Use Cobra aliases only for real Python aliases and deprecated forms recorded in the catalog.
* Disable command sorting and any fuzzy, prefix, or case-insensitive matching unless the oracle says the Python CLI does the same.
* Override help and usage templates, and snapshot-test them against `openstack --help` and `openstack help <command>`.
* Override flag errors and command errors so stderr and exit codes match OSC.
* Prefer the existing Python-style `openstack complete` behavior over Cobra's default completion command. Cobra's completion hooks may still be useful internally, but the public command must match OSC.
* Keep auth/config resolution outside Viper unless a later proof shows it can exactly reproduce OpenStackClient semantics.
