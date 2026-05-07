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

Work done: added `tools/matrix`, which generates `compat/matrix.yaml`, `compat/test-matrix.yaml`, and `compat/test-clouds.yaml`.

Result: the command matrix has one row for each of the 594 OSC commands, including service/API mapping, initial status, plugin-scope markers for Placement and Tap commands, and an implemented row for `command list`. The test matrix records static, mocked, read, write-cleanup, destructive, and admin-write suites. The cloud config records non-secret policy for `cloud6`, `flex-sjc`, `flex-dfw`, and `flex-iad`, including resource prefixes and cleanup rules.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and `go run ./tools/matrix` passed with workspace-local Go caches.

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

## 2026-05-02: Common Read Command Expansion

Work done: added Common command implementations for `availability zone list`, `extension list`, `extension show`, and `limits show`. Availability-zone output now combines Nova, Cinder, and Neutron rows to match the default Python OSC behavior observed on `cloud6`. Limits output requires `--absolute` or `--rate`, matching Python OSC, and normalizes absolute-limit names to the OSC JSON names observed from `openstack limits show --absolute -f json`.

Compatibility note: Nova and Cinder availability zones, extensions, and limits use Gophercloud packages directly. Neutron availability zones are implemented as a narrow raw REST read through the authenticated Gophercloud Network v2 service client because no dedicated Neutron availability-zone package was found in the local Gophercloud v2.12.0 module inventory. That shim is intentionally recorded in the matrix and should move behind the service-scoped extras boundary when that plugin layer is ready. `quota list/show` and `versions show` remain unimplemented because they need broader service aggregation, project/domain policy handling, and oracle tests before they should replace their stubs.

Sources consulted: Gophercloud package docs and local module sources for [Compute availability zones](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/availabilityzones), [Block Storage availability zones](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/availabilityzones), [common extensions](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/common/extensions), [Compute limits](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/limits), and [Block Storage limits](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, `go run ./tools/matrix`, and live `cloud6` JSON smoke checks passed for `availability zone list`, `availability zone list --long`, `extension list --network`, `extension show router`, `limits show --absolute`, and `limits show --rate`.

## 2026-05-02: Common Quota Read Expansion

Work done: added read-only implementations for `quota show` and `quota list`. `quota show` resolves the current project from the project-scoped Keystone token when no project argument is provided, aggregates Compute, Volume, and Network quota rows by default, supports `--compute`, `--volume`, `--network`, `--all`, `--usage`, and the Volume default-quota path, and keeps `quota list` behind the same required `--compute`, `--volume`, or `--network` parser rule as Python OSC.

Compatibility note: Gophercloud has typed quota packages for Compute, Network, and Block Storage, but OSC-shaped output includes service-specific extra quota keys, null compatibility rows, and aggregate service ordering. The implementation therefore uses authenticated Gophercloud service clients for narrow raw quota reads and records `quota list/show` as shim-backed in the matrix. This is functional coverage, not exact parity; output key order, all admin list behavior, domain-qualified project lookup, quota class behavior, and full golden tests remain open.

Sources consulted: Gophercloud package docs and local module sources for [Compute quota sets](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/quotasets), [Network quotas](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/quotas), and [Block Storage quota sets](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/quotasets).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, live `cloud6` JSON smoke checks passed for `quota show`, `quota show --compute`, `quota show --usage`, and `quota list --compute`, and the no-service `quota list` parser error matched the required behavior shape.

## 2026-05-02: Common Versions Read Expansion

Work done: added `versions show` using the Keystone service catalog from the current token, endpoint/interface/region/service/status filtering, service root version discovery through the authenticated Gophercloud provider client, and conservative catalog-derived fallback rows when a service does not expose a parseable version document.

Compatibility note: this is shim-backed because Gophercloud exposes service-specific API version helpers for some services, but not a single OSC-shaped aggregate `versions show` API. The implementation matches the observed `cloud6` Compute and Image filter outputs closely, normalizes Keystone `stable` status to OSC's `CURRENT`, and keeps unavailable services visible with fallback rows. Exact service-types-authority alias handling, full endpoint ordering, interface precedence, and exhaustive version document shapes still need oracle-backed tests.

Sources consulted: local Gophercloud module sources for service-specific API version packages, including [Compute API versions](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/apiversions), [Block Storage API versions](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/apiversions), and [Network API versions](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/apiversions), plus the Identity v3 token service catalog type in [Gophercloud tokens](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and live `cloud6` JSON smoke checks passed for `versions show`, `versions show --service compute`, `versions show --service image --status CURRENT`, and `versions show --service identity`.

## 2026-05-02: Compute Admin Read Expansion

Work done: added Gophercloud-backed read implementations for `aggregate list/show`, `compute service list`, `hypervisor list/show`, and `hypervisor stats show`. `compute service list --long` includes the Python-observed `Disabled Reason` and `Forced Down` fields. `hypervisor list --long` currently keeps the Python-observed `cloud6` column shape, which matched the default output even though the help snapshot advertises `--long`.

Compatibility note: Nova service and hypervisor IDs changed from integer-shaped to UUID-shaped when the Compute microversion was set high enough. The Compute client now honors `OS_COMPUTE_API_VERSION` and defaults to microversion `2.53` when no explicit value is present, because Gophercloud's service and hypervisor result structs document that Compute returns service IDs as strings starting with microversion `2.53`. This matches the Python OSC JSON output observed on `cloud6` for `compute service list`, `hypervisor list`, and `server list`. Exact global flag parsing, microversion negotiation, and older-cloud behavior still need oracle-backed tests.

Sources consulted: Gophercloud package docs and local module sources for [Compute aggregates](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates), [Compute services](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/services), and [Compute hypervisors](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and live `cloud6` JSON smoke checks passed for `aggregate list`, `compute service list`, `compute service list --long`, `hypervisor list`, `hypervisor list --long`, `hypervisor show`, `hypervisor stats show`, and `server list` after the Compute microversion default changed. `aggregate show` is implemented but still needs a live fixture because `cloud6` returned no aggregates during this run.

## 2026-05-02: Image Member, Task, And Store Read Expansion

Work done: added Image v2 read implementations for `image member list`, `image member get`, `image task list`, `image task show`, and `image stores list`. Member and task commands use Gophercloud packages. Store discovery uses a narrow raw REST read through the authenticated Image v2 service client because no dedicated Gophercloud store-discovery package exists in v2.12.0.

Compatibility note: `image stores list --detail` requires an Image API microversion that exposes detailed store discovery. The Image client now honors `OS_IMAGE_API_VERSION`; no default is set yet because the safe default needs broader cloud testing. The live `cloud6` check used `OS_IMAGE_API_VERSION=2.15`, matching the Python help text requirement for `--detail`.

Sources consulted: Gophercloud package docs and local module sources for [Image members](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/image/v2/members) and [Image tasks](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/image/v2/tasks), plus the OpenStack Image API reference for [store discovery and detailed store discovery](https://docs.openstack.org/api-ref/image/v2/index.html#image-service-info-discovery).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and live `cloud6` JSON smoke checks passed for `image member list`, `image task list`, `image stores list`, and `OS_IMAGE_API_VERSION=2.15 image stores list --detail`. `image member get` and `image task show` are implemented but still need live fixtures because the tested image had no members and `cloud6` returned no image tasks during this run.

## 2026-05-02: Network Router And Security Group Read Expansion

Work done: added Network v2 read implementations for `router list/show`, `security group list/show`, and `security group rule list/show`. These commands use Gophercloud's Neutron router, security group, and security group rule packages.

Compatibility note: the implementation now matches the Python-observed JSON column shape for the default `cloud6` list outputs, including router `Distributed` and `HA` columns and security-group-rule `Remote Address Group` output. Exact parity still needs oracle golden tests for project-name resolution, shared security group filtering, router HA data, security group rule port-range edge cases, and full show output formatting.

Sources consulted: Gophercloud package docs and local module sources for [Neutron routers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers), [security groups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups), and [security group rules](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and live `cloud6` JSON smoke checks passed for `router list/show`, `security group list/show`, and `security group rule list/show`.

## 2026-05-02: Volume Admin And Backup Read Expansion

Work done: added Block Storage v3 read implementations for `volume backup list/show`, `volume service list`, `volume backend pool list`, and `volume transfer request list/show`. These commands use Gophercloud's Cinder backups, services, scheduler stats, and transfers packages.

Compatibility note: `volume service list` includes the Python-observed `Cluster` and `Backend State` columns by default on `cloud6`, and `--long` adds `Disabled Reason`. `volume backend pool list --long` exposes the scheduler pool capabilities map. Exact parity still needs oracle golden tests for Cinder microversion-dependent fields, project-name resolution, volume-name filtering for backup lists, and show output when backup and transfer fixtures exist.

Sources consulted: Gophercloud package docs and local module sources for [Block Storage backups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups), [services](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/services), [scheduler stats](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/schedulerstats), and [transfers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and live `cloud6` JSON smoke checks passed for `volume backup list`, `volume service list`, `volume service list --long`, `volume backend pool list`, `volume backend pool list --long`, and `volume transfer request list`. `volume backup show` and `volume transfer request show` are implemented but still need live fixtures because the list commands returned no rows during this run.

## 2026-05-03: Volume Attachment, QoS, Summary, And Ordered JSON

Work done: added Block Storage v3 read implementations for `volume attachment list/show`, `volume qos list/show`, and `volume summary`. Attachments and QoS use Gophercloud packages. `volume summary` uses a narrow authenticated Cinder REST read through the Gophercloud service client because the local Gophercloud v2.12.0 module has no typed summary helper.

Renderer work: JSON list and show output now preserves command column and field order instead of serializing Go maps. This closes visible mismatches for `volume attachment list -f json`, `volume attachment show -f json`, and `volume summary -f json`, and should reduce future golden-test churn for all implemented commands.

Microversion work: the Block Storage client now honors `OS_VOLUME_API_VERSION`. Commands with minimum Cinder microversions discover supported service microversions and use the service's maximum when no explicit version is set. This was needed because Python OSC/openstacksdk used Cinder 3.71 for attachment reads on `cloud6`, while `volume summary` needs at least 3.12 and includes `metadata` only at 3.36 or later.

Sources consulted:

* [Gophercloud Block Storage attachments](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments), whose package note says the attachment API requires Cinder 3.27 minimum.
* [Gophercloud Block Storage QoS](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos), for QoS specs and associations.
* [Gophercloud OpenStack utils](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/utils), for service version and supported microversion discovery.
* [OpenStack Block Storage API v3](https://docs.openstack.org/api-ref/block-storage/v3/), which documents attachments as new in microversion 3.27 and volume summary as available since 3.12, with `metadata` new in 3.36.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v3/volume_attachment.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v2/qos_specs.py`, and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v3/volume.py`, used only as the pinned local oracle implementation source.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and `go run ./tools/matrix` passed. Live `cloud6` checks matched Python OSC for `volume attachment list -f json`, `volume attachment show <existing-attachment> -f json`, `volume qos list --print-empty`, and `volume summary -f json`. A temporary QoS spec named `golang-osc-test-64fb7c64-bee7-47ed-8dba-65b585469eb5` was created with Python OSC to provide a show fixture; Python and Go matched for `volume qos show -f json` and `volume qos list -f json`, and the temporary QoS spec was deleted successfully.

## 2026-05-03: Compute Events, Attachments, And Usage Read Expansion

Work done: added Compute v2 read implementations for `server event list`, `server event show`, `server volume list`, `usage list`, and `usage show`. Server events use Gophercloud's instance actions package, server volume attachment reads use Gophercloud's Nova volume attachments package with an extended extraction struct for the `attachment_id` and `bdm_uuid` fields observed from Python OSC, and usage reads use Gophercloud's tenant usage package.

Compatibility note: Python OSC/openstacksdk requested Compute microversion 2.89 for `server volume list` on `cloud6` so that `Attachment ID` and `BlockDeviceMapping UUID` appear instead of the older `ID` column. The Go command now discovers supported Compute microversions and uses the service maximum when no explicit `OS_COMPUTE_API_VERSION` is set for commands that need a higher minimum. If the user sets `OS_COMPUTE_API_VERSION`, the command honors it and returns a compatibility error when it is too low for the requested behavior. `usage show` also preserves Python's JSON detail shape for usage server rows, including the duplicate `name` key emitted by the local OSC oracle.

Sources consulted:

* [Gophercloud Compute instance actions](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/instanceactions), for server event list and show.
* [Gophercloud Compute volume attachments](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach), for Nova server volume attachment reads.
* [Gophercloud Compute usage](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/usage), for tenant usage list and show.
* [Gophercloud OpenStack utils](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/utils), for supported Compute microversion discovery.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/server_event.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/server_volume.py`, and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/usage.py`, used only as the pinned local oracle implementation source.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and `go run ./tools/matrix` passed with workspace-local Go caches. Live `cloud6` JSON checks matched Python OSC for `server event list`, `server event list --long`, `server event show`, `server volume list`, `usage list --start 2026-05-01 --end 2026-05-03`, and `usage show --start 2026-05-01 --end 2026-05-02`; `usage show` has dynamic `uptime` values, so only stable structure and non-time-varying fields should be used for golden tests.

## 2026-05-03: Compute Console Read Expansion

Work done: added `console log show` and `console url show`. Console log reads use Gophercloud's server console-output action and write raw log text, matching Python OSC's command behavior instead of routing through the structured formatter. Console URL reads use Gophercloud's remote console package and preserve the Python-observed `protocol`, `type`, and `url` field order for show output.

Compatibility note: `console url show` uses Compute microversion discovery with a minimum of 2.6 when no explicit `OS_COMPUTE_API_VERSION` is set. The `console connection show` command remains a stub because it needs a separate Nova console-token lookup path that is not exposed by the local Gophercloud `remoteconsoles` package.

Sources consulted:

* [Gophercloud Compute servers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers), for console-output actions.
* [Gophercloud Compute remote consoles](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/remoteconsoles), for remote console URL creation.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/console.py`, used only as the pinned local oracle implementation source.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and `go run ./tools/matrix` passed with workspace-local Go caches. Live `cloud6` checks matched Python OSC for `console log show --lines 5 rocky` and the stable JSON fields of `console url show rocky -f json`; the URL token itself is expected to differ between calls.

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

## 2026-05-03: Neutron Extension Read Slice

Decision: add the next read-only Network slice through Gophercloud packages where the SDK has first-class support, and use a narrow raw GET only where the typed SDK loses OSC-visible nullability.

Implemented commands: `address group list/show`, `address scope list/show`, `subnet pool list/show`, `network agent list/show`, `network rbac list/show`, `network segment list/show`, `network trunk list/show`, `network qos policy list/show`, and `network qos rule type list/show`.

The `network agent show` command uses Gophercloud's authenticated `ServiceClient` directly for `GET /v2.0/agents/{id}` because the typed `agents.Agent.ResourcesSynced` field is a Go `bool`, and the `cloud6` API returned JSON `null` for `resources_synced`. Python OSC showed `resources_synced: null`, so preserving the raw nullable value is the closer compatibility behavior.

Live observations on `cloud6`: `address group list`, `address scope list`, and `subnet pool list` succeeded with empty tables matching the Python OSC default columns. `network agent list/show` and `network rbac list/show` succeeded against real resources. `network segment list`, `network trunk list`, `network qos policy list`, and `network qos rule type list` returned 404 on both Python OSC and the Go CLI because those Neutron extensions are not exposed on `cloud6`; they remain implemented but need a flex cloud or another cloud exposing the extension before they can be marked cloud-verified.

Sources consulted:

* [Gophercloud address groups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups)
* [Gophercloud address scopes](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes)
* [Gophercloud subnet pools](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools)
* [Gophercloud agents](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/agents)
* [Gophercloud RBAC policies](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies)
* [Gophercloud segments](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments)
* [Gophercloud trunks](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks)
* [Gophercloud QoS policies](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies)
* [Gophercloud QoS rule types](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/ruletypes)
* Local OSC oracle snapshots in `compat/osc/9.0.0/help/address/group/list.txt`, `compat/osc/9.0.0/help/address/scope/list.txt`, `compat/osc/9.0.0/help/subnet/pool/list.txt`, `compat/osc/9.0.0/help/network/agent/list.txt`, `compat/osc/9.0.0/help/network/rbac/list.txt`, `compat/osc/9.0.0/help/network/segment/list.txt`, `compat/osc/9.0.0/help/network/trunk/list.txt`, and `compat/osc/9.0.0/help/network/qos/rule/type/list.txt`.

## 2026-05-03: Keystone Read Slice

Decision: expand Identity v3 read coverage with Gophercloud-backed list/show commands for Keystone resources that are low-risk and directly supported by the SDK. The implemented list commands were live-smoked on `cloud6`; most returned empty tables, which is still useful because the Python oracle exposes the expected default columns for empty output.

Implemented commands: `access rule list/show`, `application credential list/show`, `credential list/show`, `ec2 credentials list/show`, `limit list/show`, `policy list/show`, `registered limit list/show`, and `trust list/show`.

The user-scoped commands use the authenticated token's user ID when `--user` is not supplied. That matches the OSC command shape for application credentials, access rules, and EC2 credentials while keeping production behavior self-contained in Go. Show commands for these resources still need live fixtures before being marked cloud-verified.

Sources consulted:

* [Gophercloud application credentials](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials)
* [Gophercloud credentials](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/credentials)
* [Gophercloud EC2 credentials](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/ec2credentials)
* [Gophercloud limits](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/limits)
* [Gophercloud policies](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/policies)
* [Gophercloud registered limits](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/registeredlimits)
* [Gophercloud trusts](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/trusts)
* Local OSC oracle snapshots in `compat/osc/9.0.0/help/access/rule/list.txt`, `compat/osc/9.0.0/help/application/credential/list.txt`, `compat/osc/9.0.0/help/credential/list.txt`, `compat/osc/9.0.0/help/ec2/credentials/list.txt`, `compat/osc/9.0.0/help/limit/list.txt`, `compat/osc/9.0.0/help/policy/list.txt`, `compat/osc/9.0.0/help/registered/limit/list.txt`, and `compat/osc/9.0.0/help/trust/list.txt`.

## 2026-05-03: Keystone Role Assignment Read Expansion

Work done: added `role assignment list` through Gophercloud's Identity v3 roles package. The command supports `--effective`, `--role`, `--role-domain`, `--names`, `--user`, `--user-domain`, `--group`, `--group-domain`, `--domain`, `--project`, `--project-domain`, `--system`, `--inherited`, `--auth-user`, and `--auth-project`.

Compatibility note: Gophercloud's typed `roles.RoleAssignment` result does not expose Keystone's `scope.system` and `scope.OS-INHERIT:inherited_to` response fields in the local v2.12.0 module. The command still uses Gophercloud's role-assignment pager and query builder, but extracts into an OSC-shaped local struct so the `System` and `Inherited` columns match Python OSC instead of silently dropping data.

Live observations on `cloud6`: default JSON output matched Python OSC for the observed role assignments, including the system-scoped `System: all` row. `--names`, `--auth-user`, `--auth-project`, `--system all`, and `--role admin --names` also matched Python OSC in live smoke checks.

Sources consulted:

* [Gophercloud Identity roles](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/roles)
* Local Gophercloud source file `.cache/gomod/github.com/gophercloud/gophercloud/v2@v2.12.0/openstack/identity/v3/roles/results.go`, which shows the typed assignment struct fields available in the local module.
* Local OSC oracle snapshot `compat/osc/9.0.0/help/role/assignment/list.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/identity/v3/role_assignment.py`, used only as the pinned local oracle implementation source.
* Local openstacksdk source for `openstack.identity.v3.role_assignment.RoleAssignment`, which maps `scope_system` to `scope.system` and `inherited_to` to `scope.OS-INHERIT:inherited_to`.

## 2026-05-03: Keystone Federation And Provider Read Expansion

Work done: added read implementations for `mapping list`, `mapping show`, `identity provider list`, `identity provider show`, `federation protocol list`, `federation protocol show`, `service provider list`, `service provider show`, and `implied role list`.

Implementation note: `mapping list/show` uses Gophercloud's Identity v3 federation package, and `implied role list` uses Gophercloud's Identity v3 roles package. Identity-provider, federation-protocol, and service-provider reads use narrow OS-FEDERATION REST reads through the authenticated Gophercloud Identity service client because no typed local Gophercloud helper exists in v2.12.0.

Live observations on `cloud6`: Python OSC and the Go CLI both returned empty JSON lists for `mapping list`, `identity provider list`, and `service provider list`. `implied role list` matched Python OSC against the three role-inference rows exposed by Keystone. The show commands and federation protocol list/show are implemented but still need fixtures because `cloud6` currently has no mappings, identity providers, service providers, or federation protocols.

Sources consulted:

* [Gophercloud Identity federation](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/federation)
* [Gophercloud Identity roles](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/identity/v3/roles)
* Local OSC oracle snapshots in `compat/osc/9.0.0/help/mapping/list.txt`, `compat/osc/9.0.0/help/mapping/show.txt`, `compat/osc/9.0.0/help/identity/provider/list.txt`, `compat/osc/9.0.0/help/identity/provider/show.txt`, `compat/osc/9.0.0/help/federation/protocol/list.txt`, `compat/osc/9.0.0/help/federation/protocol/show.txt`, and `compat/osc/9.0.0/help/implied/role/list.txt`.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/identity/v3/mapping.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/identity/v3/identity_provider.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/identity/v3/federation_protocol.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/identity/v3/service_provider.py`, and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/identity/v3/implied_role.py`, used only as pinned local oracle implementation sources.

## 2026-05-03: Neutron IP Availability And Service Provider Reads

Work done: added `ip availability list`, `ip availability show`, `network service provider list`, and the Neutron compatibility behavior for `floating ip pool list`.

Implementation note: IP availability uses Gophercloud's `networkipavailabilities` package and converts Gophercloud's string-preserved IP counts back to JSON numbers to match Python OSC output. `network service provider list` uses a narrow authenticated Neutron `service-providers` REST read because no typed local Gophercloud helper exists in v2.12.0. `floating ip pool list` returns the Python-observed Neutron error, `Floating ip pool operations are only available for Compute v2 network.`, because that command is only available on the legacy Nova-network path in Python OSC.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `ip availability list -f json`, `ip availability list --ip-version 4 -f json`, `ip availability show os6-lan -f json`, `network service provider list -f json`, and the `floating ip pool list -f json` error text.

Sources consulted:

* [Gophercloud network IP availabilities](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networkipavailabilities)
* Local OSC oracle snapshots in `compat/osc/9.0.0/help/ip/availability/list.txt`, `compat/osc/9.0.0/help/ip/availability/show.txt`, `compat/osc/9.0.0/help/network/service/provider/list.txt`, and `compat/osc/9.0.0/help/floating/ip/pool/list.txt`.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/ip_availability.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/network_service_provider.py`, and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/floating_ip_pool.py`, used only as pinned local oracle implementation sources.

## 2026-05-03: Cinder Resource Filter And Log-Level Reads

Work done: added `block storage resource filter list`, `block storage resource filter show`, and `block storage log level list`.

Implementation note: the local Gophercloud v2.12.0 module has no typed package for these Cinder command surfaces. The commands therefore use narrow authenticated Block Storage REST calls through Gophercloud's service client. Resource filters call `GET /resource_filters` with Cinder microversion 3.33, and log-level reads call `PUT /os-services/get-log` with Cinder microversion 3.32 and accept the Cinder-observed HTTP 200 response.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `block storage resource filter list -f json`, `block storage resource filter show volume -f json`, and `block storage log level list --service cinder-api --log-prefix cinder.api -f json`.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/block/storage/resource/filter/list.txt`, `compat/osc/9.0.0/help/block/storage/resource/filter/show.txt`, and `compat/osc/9.0.0/help/block/storage/log/level/list.txt`.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v3/block_storage_resource_filter.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v3/block_storage_log_level.py`, used only as pinned local oracle implementation sources.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/block_storage/v3/resource_filter.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/block_storage/v3/service.py`, which document the resource paths and microversion requirements used by the Python command implementation.

## 2026-05-03: Cinder Legacy-Client Read Expansion

Work done: added `volume message list`, `volume message show`, `block storage cluster list`, and `block storage cluster show`.

Implementation note: these Python OSC commands use the older cinderclient manager path, not the OpenStackSDK volume proxy path used by the previous Cinder slice. Live Python checks showed that `volume message list` fails by default unless `OS_VOLUME_API_VERSION` is at least 3.3, and `block storage cluster list` fails by default unless `OS_VOLUME_API_VERSION` is at least 3.7. The Go commands preserve that default error behavior with an explicit-microversion helper instead of auto-negotiating to the service maximum.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `OS_VOLUME_API_VERSION=3.3 volume message list -f json`, `OS_VOLUME_API_VERSION=3.3 volume message show <message-id> -f json`, `OS_VOLUME_API_VERSION=3.7 block storage cluster list -f json`, and the default microversion errors for `volume message list` and `block storage cluster list`. `block storage cluster show` is implemented but still needs a cluster fixture because `cloud6` returned an empty cluster list.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/volume/message/list.txt`, `compat/osc/9.0.0/help/volume/message/show.txt`, `compat/osc/9.0.0/help/block/storage/cluster/list.txt`, and `compat/osc/9.0.0/help/block/storage/cluster/show.txt`.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v3/volume_message.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v3/block_storage_cluster.py`, used only as pinned local oracle implementation sources.
* Local cinderclient source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/cinderclient/v3/messages.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/cinderclient/v3/clusters.py`, which document the resource paths and cinderclient version gates used by the Python command implementation.

## 2026-05-03: Cinder Group Read Expansion

Work done: added `volume group list`, `volume group show`, `volume group snapshot list`, `volume group snapshot show`, `volume group type list`, and `volume group type show`.

Implementation note: `volume group list/show` and `volume group type list/show` preserve the local Python OSC cinderclient behavior that requires an explicit `OS_VOLUME_API_VERSION` of 3.13 and 3.11, respectively. `volume group snapshot list/show` follows the OpenStackSDK-backed Python command path and uses the same auto-negotiating Cinder microversion helper as the SDK-backed Cinder reads. `volume group type show` intentionally implements the intended Cinder API lookup even though the pinned local Python OSC command currently raises `'Namespace' object has no attribute 'group'`; it is therefore implemented but not marked cloud-verified in the matrix.

Renderer note: the shared JSON renderer now disables Go's default HTML escaping so values such as `<is> True` match Python JSON output instead of becoming `\u003cis\u003e True`.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `OS_VOLUME_API_VERSION=3.11 volume group type list -f json`, `OS_VOLUME_API_VERSION=3.13 volume group list -f json`, `volume group snapshot list -f json`, and the default microversion errors for `volume group type list` and `volume group list`. Both Python OSC and the Go CLI return an error for `OS_VOLUME_API_VERSION=3.11 volume group type list --default -f json` because cloud6 has no configured default group type, but the exact HTTP error text still needs normalization.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/volume/group/list.txt`, `compat/osc/9.0.0/help/volume/group/show.txt`, `compat/osc/9.0.0/help/volume/group/snapshot/list.txt`, `compat/osc/9.0.0/help/volume/group/snapshot/show.txt`, `compat/osc/9.0.0/help/volume/group/type/list.txt`, and `compat/osc/9.0.0/help/volume/group/type/show.txt`.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v3/volume_group.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v3/volume_group_snapshot.py`, and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/v3/volume_group_type.py`, used only as pinned local oracle implementation sources.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/block_storage/v3/group.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/block_storage/v3/group_snapshot.py`, and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/block_storage/v3/group_type.py`, which document the REST paths used by the SDK path.
* Local cinderclient source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/cinderclient/v3/groups.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/cinderclient/v3/group_types.py`, which document the cinderclient version gates and query construction used by the Python command implementation.

## 2026-05-03: Deprecated Nova Host Reads

Work done: added `host list` and `host show`.

Implementation note: Python OSC implements these deprecated commands as raw Compute API calls because OpenStackSDK intentionally does not support the deprecated host API. The Go CLI follows that shape through Gophercloud's authenticated `ServiceClient`, calls `GET /os-hosts` and `GET /os-hosts/<host>` at Compute microversion 2.1, filters `host list --zone` client-side, and emits the same deprecation warning text observed from the local Python OSC oracle.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `host list -f json`, `host list --zone nova -f json`, and `host show dell6.crandall.haus -f json`.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/host/list.txt` and `compat/osc/9.0.0/help/host/show.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/host.py`, used only as the pinned local oracle implementation source.

## 2026-05-03: Deprecated Nova Compute Agent Read

Work done: added `compute agent list`.

Implementation note: Python OSC implements `compute agent list` as a raw Compute API call because OpenStackSDK intentionally does not support the deprecated agent API. The command calls `GET /os-agents` at Compute microversion 2.1, includes the optional `--hypervisor` query parameter, and uses the OSC columns `Agent ID`, `Hypervisor`, `OS`, `Architecture`, `Version`, `Md5Hash`, and `URL`. The Go CLI also added a small HTTP error formatter for these raw Compute shims so removed Nova APIs return Python-style `HttpException` text instead of Gophercloud's default response-code error.

Live observations on `cloud6`: Python OSC and the Go CLI matched for the removed API error path on `compute agent list -f json` and `compute agent list --hypervisor xen -f json`. `cloud6` returns HTTP 410 for the base list and HTTP 500 with Nova's `KeyError` message for the filtered list, so success output still needs a XenAPI-capable or older fixture cloud before golden success rows can be recorded.

Sources consulted:

* Local OSC oracle snapshot in `compat/osc/9.0.0/help/compute/agent/list.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/agent.py`, used only as the pinned local oracle implementation source.

## 2026-05-03: Nova Server Migration List Read

Work done: added `server migration list`.

Implementation note: the local Gophercloud v2.12.0 module has no typed Compute migration package, while Python OSC uses OpenStackSDK's `Migration` resource at `/os-migrations`. The Go CLI uses a narrow authenticated Nova REST read, mirrors the Python query names, resolves `--server`, `--project`, and `--user` filters before querying, maps `--type cold-migration` to Nova's `migration` value, and caps automatic Compute microversion discovery to 2.80 because the pinned OpenStackSDK migration resource declares `_max_microversion = '2.80'`. The output column set follows the Python source's microversion-dependent columns.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `server migration list -f json`, `server migration list --limit 1 -f json`, and `server migration list --project admin -f json`. The cloud currently returns an empty migration list, so non-empty row golden fixtures are still needed.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/server/migration/list.txt` and `compat/osc/9.0.0/help/server/migration/show.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/server_migration.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/compute/v2/migration.py`, which documents the `/os-migrations` path, query mapping, response fields, and 2.80 resource microversion cap.

## 2026-05-03: Nova Console Connection Read

Work done: added `console connection show`.

Implementation note: Python OSC calls OpenStackSDK's `validate_console_auth_token`, which maps to the `ConsoleAuthToken` resource at `/os-console-auth-tokens`. The local Gophercloud v2.12.0 module has remote-console creation helpers but no typed console-auth-token lookup helper, so the Go CLI uses a narrow authenticated Nova REST read, caps automatic Compute microversion discovery to 2.99 to match the pinned SDK resource, and renders the SDK-observed fields `host`, `instance_uuid`, `internal_access_path`, `port`, and `tls_port`. The not-found path uses a Python-style resource lookup error instead of the generic `HttpException` text.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `console connection show invalid-token -f json`. A successful row still needs a valid, live console token fixture because these tokens are short-lived.

Sources consulted:

* Local OSC oracle snapshot in `compat/osc/9.0.0/help/console/connection/show.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/console_connection.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/compute/v2/console_auth_token.py`, which documents the `/os-console-auth-tokens` path, fields, and 2.99 resource microversion cap.

## 2026-05-03: Glance Cached Image List Read

Work done: added `cached image list`.

Implementation note: Python OSC calls OpenStackSDK's image cache resource at `/cache` and formats the response into normal list rows. The local Gophercloud v2.12.0 module has no typed image cache helper, so the Go CLI uses a narrow authenticated Glance REST read, caps the Image API microversion to 2.14 to match the pinned SDK resource, converts cached image epoch timestamps to UTC ISO strings, and adds queued-image rows with Python's `N/A` placeholders.

Live observations on `cloud6`: Python OSC and the Go CLI matched for the unsupported-cache error path on `cached image list -f json`. `cloud6` returns `404 Not Found: Caching via API is not supported at this site.`, so non-empty cache and queue rows still need a cloud with Glance API caching enabled.

Sources consulted:

* Local OSC oracle snapshot in `compat/osc/9.0.0/help/cached/image/list.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/cache.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/cache.py`, which documents the `/cache` path, response fields, and 2.14 resource microversion cap.

## 2026-05-03: Glance Metadef Namespace Reads

Work done: added `image metadef namespace list` and `image metadef namespace show`.

Implementation note: Python OSC uses OpenStackSDK metadef namespace resources at `/metadefs/namespaces`. The local Gophercloud v2.12.0 module has no typed metadef namespace helper, so the Go CLI uses narrow authenticated Glance REST reads. List follows Glance pagination and supports the Python filters `--resource-types` and `--visibility`. Show intentionally formats the SDK-shaped resource rather than the raw Glance response, because the SDK drops raw `properties` and `schema` keys before Python OSC formats namespace show output.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `image metadef namespace list -f json`, `image metadef namespace list --visibility private -f json`, and `image metadef namespace show OS::OperatingSystem -f json`.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/image/metadef/namespace/list.txt` and `compat/osc/9.0.0/help/image/metadef/namespace/show.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/metadef_namespaces.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/metadef_namespace.py`, which documents the `/metadefs/namespaces` path, query mapping, and SDK resource fields.

## 2026-05-03: Glance Metadef Resource Type Reads

Work done: added `image metadef resource type list` and `image metadef resource type association list`.

Implementation note: Python OSC uses OpenStackSDK metadef resource type resources at `/metadefs/resource_types` and `/metadefs/namespaces/{namespace}/resource_types`. The local Gophercloud v2.12.0 module has no typed metadef resource type helper, so the Go CLI uses narrow authenticated Glance REST reads and preserves the Python `Name` column shape.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `image metadef resource type list -f json` and `image metadef resource type association list OS::OperatingSystem -f json`.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/image/metadef/resource/type/list.txt` and `compat/osc/9.0.0/help/image/metadef/resource/type/association/list.txt`.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/metadef_resource_types.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/metadef_resource_type_association.py`, used only as pinned local oracle implementation sources.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/metadef_resource_type.py`, which documents the resource type and association paths and response fields.

## 2026-05-03: Glance Metadef Object and Property Reads

Work done: added `image metadef object list`, `image metadef object show`, `image metadef object property show`, `image metadef property list`, and `image metadef property show`.

Implementation note: Python OSC uses OpenStackSDK metadef object and property resources at `/metadefs/namespaces/{namespace}/objects` and `/metadefs/namespaces/{namespace}/properties`. The local Gophercloud v2.12.0 module has no typed metadef object or property helper, so the Go CLI uses narrow authenticated Glance REST reads. The implementation preserves ordered JSON for nested metadef property dictionaries because Python OSC's JSON output reflects the order returned by Glance and OpenStackSDK.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `image metadef object list OS::Software::WebServers -f json`, `image metadef object show OS::Software::WebServers "Apache HTTP Server" -f json`, `image metadef property list OS::OperatingSystem -f json`, `image metadef property show OS::OperatingSystem os_distro -f json`, and `image metadef object property show OS::Software::WebServers "Apache HTTP Server" sw_webserver_apache_http_port -f json`.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/image/metadef/object/list.txt`, `compat/osc/9.0.0/help/image/metadef/object/show.txt`, `compat/osc/9.0.0/help/image/metadef/object/property/show.txt`, `compat/osc/9.0.0/help/image/metadef/property/list.txt`, and `compat/osc/9.0.0/help/image/metadef/property/show.txt`.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/metadef_objects.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/metadef_properties.py`, used only as pinned local oracle implementation sources.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/metadef_object.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/metadef_property.py`, which document the object and property paths and SDK-shaped fields.

## 2026-05-03: Glance Metadef Namespace Lifecycle and Resource Type Association Writes

Work done: added `image metadef namespace create`, `image metadef namespace set`, `image metadef namespace delete`, `image metadef resource type association create`, and `image metadef resource type association delete`.

Implementation note: Python OSC routes these operations through OpenStackSDK metadef namespace and resource type association resources. The local Gophercloud v2.12.0 module has no typed helpers for these resources, so the Go CLI uses authenticated Glance REST calls. Namespace update explicitly accepts Glance's `200 OK` response for PUT. Resource type association create omits the namespace from the JSON body because Python passes it to the SDK as a URI field, and Glance rejects it as an additional body property.

Live observations on `cloud6`: Python OSC was used as the output oracle for namespace create/set/show, resource type association create/list/delete, and namespace cleanup with a disposable namespace. The Go CLI completed the same lifecycle with `gocli-test-ns-go-20260503a`, including association create/delete and final namespace delete. A parallel Python cleanup attempt showed that dependent delete operations should not be run concurrently; the test matrix should keep association cleanup before namespace cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/image/metadef/namespace/create.txt`, `compat/osc/9.0.0/help/image/metadef/namespace/set.txt`, `compat/osc/9.0.0/help/image/metadef/namespace/delete.txt`, `compat/osc/9.0.0/help/image/metadef/resource/type/association/create.txt`, and `compat/osc/9.0.0/help/image/metadef/resource/type/association/delete.txt`.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/metadef_namespaces.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/metadef_resource_type_association.py`, used only as pinned local oracle implementation sources.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/metadef_namespace.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/metadef_resource_type.py`, which document the namespace and resource type association paths and fields.

## 2026-05-03: Glance Metadef Object and Property Writes

Work done: added `image metadef object create`, `image metadef object update`, `image metadef object delete`, `image metadef property create`, `image metadef property set`, and `image metadef property delete`.

Implementation note: Python OSC uses OpenStackSDK metadef object and property resources for these operations. The local Gophercloud v2.12.0 module has no typed helpers, so the Go CLI uses authenticated Glance REST calls. Property set first fetches the current property and merges updates into the full property body, matching the Python OSC behavior that avoids resetting omitted attributes.

Live observations on `cloud6`: Python OSC was used as the output oracle for object create/update/show/delete and property create/set/show/delete with a disposable namespace. The Go CLI completed the same lifecycle with `gocli-test-ns-go-20260503b`, including final object, property, and namespace cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/image/metadef/object/create.txt`, `compat/osc/9.0.0/help/image/metadef/object/update.txt`, `compat/osc/9.0.0/help/image/metadef/object/delete.txt`, `compat/osc/9.0.0/help/image/metadef/property/create.txt`, `compat/osc/9.0.0/help/image/metadef/property/set.txt`, and `compat/osc/9.0.0/help/image/metadef/property/delete.txt`.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/metadef_objects.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/metadef_properties.py`, used only as pinned local oracle implementation sources.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/metadef_object.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/metadef_property.py`, which document the object and property paths and fields.

## 2026-05-03: Glance Cache Mutations

Work done: added `cached image clear`, `cached image delete`, and `cached image queue`.

Implementation note: Python OSC uses OpenStackSDK's cache resource at `/cache` with Image API microversion capped to 2.14. The local Gophercloud v2.12.0 module has no typed cache helper, so the Go CLI uses narrow authenticated Glance REST calls. `cached image delete` intentionally ignores a missing cache entry, matching OpenStackSDK's default `ignore_missing=True` behavior.

Live observations on `cloud6`: Python OSC and the Go CLI matched the unsupported-cache queue error path for `cached image queue da8beb8e-7301-49a3-b952-ebde206f9a0b`, and both returned no output for `cached image delete da8beb8e-7301-49a3-b952-ebde206f9a0b`. `cached image clear` matched the Python command-level failure message on this cloud, where Glance API caching is not enabled.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/cached/image/clear.txt`, `compat/osc/9.0.0/help/cached/image/delete.txt`, and `compat/osc/9.0.0/help/cached/image/queue.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/cache.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/cache.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/_proxy.py`, which document the `/cache` path, 2.14 resource microversion cap, and delete `ignore_missing` default.

## 2026-05-03: Image Import Info

Work done: added `image import info`.

Implementation note: Python OSC calls OpenStackSDK import service info at `/info/import` and renders the `import-methods.value` list under the `import-methods` field. Gophercloud v2.12.0 has a typed `imageimport` package for this path, so the Go CLI uses that package instead of a raw REST shim.

Live observations on `cloud6`: Python OSC and the Go CLI matched for `image import info -f json`, returning `glance-direct`, `web-download`, and `copy-image`.

Sources consulted:

* Local OSC oracle snapshot in `compat/osc/9.0.0/help/image/import/info.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/info.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/service_info.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/_proxy.py`, which document the `/info/import` path and import methods field.
* Local Gophercloud source package `github.com/gophercloud/gophercloud/v2/openstack/image/v2/imageimport`, which provides the typed `Get` call used by the Go CLI.

## 2026-05-03: Image Project Membership Commands

Work done: added `image add project` and `image remove project`.

Implementation note: Python OSC resolves the target project through Identity, resolves the target image through Glance, and then creates or deletes an image member. Gophercloud v2.12.0 has typed Image v2 member helpers, so the Go CLI uses `members.Create` and `members.Delete`.

Live observations on `cloud6`: `cloud6` currently has no private image fixture returned by `openstack image list --private -f json`, so successful share/unshare lifecycle verification is deferred until the image lifecycle test fixture exists. The commands are marked implemented, not cloud-verified, in `compat/matrix.yaml`.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/image/add/project.txt` and `compat/osc/9.0.0/help/image/remove/project.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/image.py`, used only as the pinned local oracle implementation source.
* Local Gophercloud source package `github.com/gophercloud/gophercloud/v2/openstack/image/v2/members`, which provides the typed create and delete calls used by the Go CLI.

## 2026-05-03: Image Create And Delete Lifecycle

Work done: added `image create` and `image delete`.

Implementation note: `image create` uses Gophercloud's typed Image v2 metadata create API, typed image data upload/stage APIs for file data, and the typed image import request when `--import` is selected with upload data. The implementation mirrors Python OSC's default `container_format=bare` and `disk_format=raw`, repeated `--property` and `--tag` handling, visibility/protected flags, project owner lookup, and OpenStackSDK's `owner_specified.openstack.*` metadata defaults. `image create --volume` is wired through Gophercloud's Block Storage v3 `volumes.UploadImage` helper, but still needs a live volume fixture before it can be marked cloud-verified.

Implementation note: normal `image delete` uses Gophercloud's typed Image v2 delete API. `image delete --store` uses a narrow authenticated Glance REST delete at `/stores/{store}/{image_id}` because the local Gophercloud v2.12.0 module has no typed multi-store delete helper. Python OSC currently reports `Multi Backend support not enabled.` for a nonexistent image delete on `cloud6`; the Go CLI mirrors that observed oracle behavior for this command.

Live observations on `cloud6`: Python OSC was used as the output oracle for metadata shape and direct zero-byte upload. The Go CLI created and deleted disposable images `gocli-test-go-20260503-image-tty-001`, `gocli-test-go-20260503-image-stdin-001`, and `gocli-test-go-20260503-image-file-002`, covering TTY metadata-only queued create, non-interactive stdin zero-byte upload, direct `--file` upload, properties/tags, and cleanup. The Go CLI also matched the Python OSC nonexistent-image delete message for `does-not-exist-gocli-test-20260503`.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/image/create.txt` and `compat/osc/9.0.0/help/image/delete.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/image.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/_proxy.py`, which documents the `create_image` owner-specified metadata defaults and `delete_image(..., store=...)` behavior.
* Gophercloud package docs for [Image v2 images](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/image/v2/images), [Image v2 image data](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata), [Image v2 import](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/image/v2/imageimport), and [Block Storage v3 volumes](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes).

## 2026-05-03: Image Save, Stage, And Import

Work done: added `image save`, `image stage`, and command-level `image import`.

Implementation note: `image save` uses Gophercloud's typed Image v2 download helper and honors `--file` plus the OSC `--chunk-size` buffer option. `image stage` uses Gophercloud's typed staging helper and mirrors Python OSC's file-or-stdin behavior and queued-state precondition. `image import` validates import capability and method-specific options against the Python OSC command logic, then uses an authenticated Glance REST request at `/v2/images/{image_id}/import`. The local Gophercloud v2.12.0 `imageimport.Create` helper covers the simple method/URI body, but not the full OSC command surface for stores, all-stores, remote import, and copy-image.

Live observations on `cloud6`: the Go CLI created and deleted disposable images `gocli-test-go-20260503-image-save-001` and `gocli-test-go-20260503-image-import-001`. The first image verified `image save --file` against an active zero-byte image. The second image verified TTY queued creation, `image stage --file`, status transition to `uploading`, `image import -f json`, and cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/image/save.txt`, `compat/osc/9.0.0/help/image/stage.txt`, and `compat/osc/9.0.0/help/image/import.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/image.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/image.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/_proxy.py`, which document image stage and import request bodies.
* Gophercloud package docs for [Image v2 image data](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata) and [Image v2 import](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/image/v2/imageimport).

## 2026-05-03: Image Set And Unset

Work done: added `image set` and `image unset`.

Implementation note: `image set` uses Gophercloud Image v2 JSON Patch helpers for name, min disk, min RAM, protected, visibility, hidden, tags, and arbitrary properties. It uses Gophercloud Image v2 member update for explicit `--project` membership state changes. Image activation and reactivation use narrow authenticated Glance REST action calls because the local Gophercloud v2.12.0 module has no typed helper for `/actions/deactivate` or `/actions/reactivate`. The Python OSC shortcut that updates the current project's image membership without `--project` is not guessed yet; the Go command currently requires `--project` for membership changes until auth-ref project lookup is wired.

Implementation note: `image unset` removes properties through Image v2 JSON Patch and removes tags through a narrow authenticated Glance REST `DELETE /v2/images/{image_id}/tags/{tag}` call because the local Gophercloud module has no typed tag-delete helper.

Live observations on `cloud6`: the Go CLI created and deleted disposable image `gocli-test-go-20260503-image-set-001`, set a new name, `min_ram`, a custom property, a tag, and the hidden flag, then unset the custom property and tag before cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/image/set.txt` and `compat/osc/9.0.0/help/image/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/image/v2/image.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/image.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/image/v2/_proxy.py`, which document image update, activation, tag, and member behavior.
* Gophercloud package docs for [Image v2 images](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/image/v2/images) and [Image v2 members](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/image/v2/members).

## 2026-05-03: Configuration Show

Work done: added initial `configuration show`.

Implementation note: Python OSC marks this command as not requiring authentication, reads the resolved OpenStackSDK config dictionary, flattens `auth.*` values, and redacts secret auth options unless `--unmask` is used. The Go CLI now resolves the same auth and endpoint options it uses for service clients through Gophercloud's `clouds.Parse`, falls back to explicit `OS_*`/flag values when no auth config is present, and renders sorted show output through the shared formatter. This is intentionally self-contained and does not call the Python CLI.

Compatibility note: this is not yet a full OpenStackSDK config dump. It covers the high-value auth, cloud, interface, region, and TLS verification fields, but omits OpenStackSDK defaults such as retry counts, vendor-agent settings, and some service-specific API version defaults until those have a Go-native config model.

Live observations on `cloud6`: `configuration show -f json` returned masked auth fields, cloud, public interface, RegionOne, password auth type, and TLS verification status. `configuration show --unmask -f value -c auth.password` returned the clear configured secret as expected; do not include that output in committed artifacts.

Sources consulted:

* Local OSC oracle snapshot in `compat/osc/9.0.0/help/configuration/show.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/common/configuration.py`, used only as the pinned local oracle implementation source.
* Local osc-lib source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/osc_lib/clientmanager.py`, which exposes `get_configuration()`.
* Gophercloud package docs for [clouds.yaml parsing](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/config/clouds).

## 2026-05-03: Quota Set And Delete

Work done: added initial `quota set` and `quota delete`.

Implementation note: `quota set` now builds the same Compute, Volume, and Network quota key groups used by Python OSC. Project quota updates use authenticated Compute, Block Storage, and Network service clients with raw `PUT` requests to `os-quota-sets` and `quotas` endpoints. Compute receives `force=true` only when quota values are present and `--force` is selected. Network receives `force=true` with `--force`, otherwise it receives `check_limit=true`, matching the Python OSC source behavior. `--volume-type` rewrites the Volume quotas that Python marks as volume-type-aware. `--class` and `--default` use Compute and Volume `os-quota-class-sets` requests; Network quota class values are ignored because Python OSC documents and implements quota classes as unsupported by Network.

Implementation note: `quota delete` resolves the target project through Identity and reverts selected service quotas with raw `DELETE` requests to Compute `os-quota-sets`, Volume `os-quota-sets`, and Network `quotas` endpoints. The parser accepts the Python-compatible `--all`, `--compute`, `--volume`, and `--network` selectors, and the suppressed Python compatibility flag `--check-limit` is accepted for `quota set`.

Live observations: no live quota mutation was run in this pass. Quota updates and resets can alter admin state, so the live suite needs a dedicated `golang-osc-testing` project, pre-test default capture, test-owned quota values, post-test reset, and retained diagnostics before these commands can be marked cloud-verified.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/quota/set.txt` and `compat/osc/9.0.0/help/quota/delete.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/common/quota.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/compute/v2/quota_class_set.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/block_storage/v3/quota_class_set.py`, which document the `os-quota-class-sets` base path.
* Gophercloud package docs for [Compute quota sets](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/quotasets), [Block Storage quota sets](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/quotasets), and [Network quotas](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/quotas).

## 2026-05-03: Keypair Create And Delete

Work done: added `keypair create` and `keypair delete`.

Implementation note: `keypair create` uses Gophercloud's Compute v2 keypair create helper. When `--public-key` is not supplied, the Go CLI generates an Ed25519 keypair locally, imports the generated public key into Nova, and either writes the private key to `--private-key` or prints it to stdout. This mirrors Python OSC's local key generation behavior and avoids depending on Nova-generated private keys. Normal show output hides `public_key` and includes the Python-observed SDK compatibility fields `created_at`, `id`, `is_deleted`, and `private_key`. `keypair delete` uses Gophercloud's typed delete helper and supports multiple key names in one invocation.

Live observations on `cloud6`: Python OSC created and deleted disposable keypair `gocli-test-keypair-python-20260503`. The Go CLI created and deleted disposable keypairs `gocli-test-keypair-go-20260503` and `gocli-test-keypair-go2-20260503`. The public-key create output was compared against Python OSC's JSON field shape, `keypair show -f json` was checked after creation, and Python/Go `keypair list -f json` matched after cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/keypair/create.txt` and `compat/osc/9.0.0/help/keypair/delete.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/keypair.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Compute keypairs](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs).
* Go package docs for [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh), used for OpenSSH public and private key encoding.

## 2026-05-03: Server Group Create And Delete

Work done: added `server group create` and `server group delete`, and tightened `server group list/show` output.

Implementation note: the Go CLI now negotiates Compute microversion up to 2.64 for server group commands when `OS_COMPUTE_API_VERSION` is not explicitly set. That matches the Python-observed `cloud6` output shape, where server groups expose singular `policy` plus `rules` instead of legacy `policies`. The create command uses Gophercloud's typed Server Groups helper and supports the OSC policy choices plus `--rule max_server_per_host=<n>` when the negotiated microversion supports rules. Delete resolves each argument by name or ID, then deletes by ID.

Live observations on `cloud6`: Python OSC created and deleted disposable server group `gocli-test-server-group-python-20260503`. The Go CLI created and deleted disposable server groups `gocli-test-server-group-go-20260503` and `gocli-test-server-group-go2-20260503`. Create and show JSON output matched the Python-observed 2.64 field shape, and Python/Go `server group list -f json` both returned an empty list after cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/server/group/create.txt` and `compat/osc/9.0.0/help/server/group/delete.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/server_group.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Compute server groups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups).

## 2026-05-03: Container Lifecycle Commands

Work done: added `container create`, `container delete`, `container set`, and `container unset`, and aligned `container show` with the Python-observed output shape.

Implementation note: the Go CLI uses Gophercloud's Object Storage v1 container helpers for create, delete, and metadata update. `container create --public` sets the Swift read ACL used for public listing and reads, and `--storage-policy` maps to `X-Storage-Policy`. `container delete --recursive` lists objects in the target container and deletes them before deleting the container. `container set/unset` map OSC properties to Swift container metadata headers. `container show` now reports Python-compatible `account`, string-valued `bytes_used` and `object_count`, `container`, optional `properties`, and `storage_policy`, rather than raw Swift transport headers.

Live observations: `cloud6` currently does not expose a usable public object-store endpoint, so the lifecycle smoke ran on `flex-sjc`. Python OSC created and deleted disposable container `gocli-test-container-python-20260503`. The Go CLI created, set metadata on, unset metadata from, showed, and deleted disposable containers `gocli-test-container-go-20260503` and `gocli-test-container-go3-20260503`. Python/Go prefix-filtered container lists both returned empty after cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/container/create.txt`, `compat/osc/9.0.0/help/container/delete.txt`, `compat/osc/9.0.0/help/container/set.txt`, and `compat/osc/9.0.0/help/container/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/object/v1/container.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Object Storage containers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers) and [Object Storage objects](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects).

## 2026-05-03: Object Lifecycle Commands

Work done: added `object create`, `object delete`, `object save`, `object set`, and `object unset`, and aligned `object show` with the Python-observed output shape.

Implementation note: `object create` uses Gophercloud's Object Storage v1 object upload helper and preserves Python OSC's default object naming behavior: the source filename string becomes the object name unless `--name` is supplied. `object save --file -` writes object bytes to stdout, and other saves write to the requested path or the object name. Object metadata set/unset uses Gophercloud's object update helper. `object show` now reports Python-compatible `account`, `container`, `content-length`, `content-type`, `etag`, `last-modified`, `object`, and optional `properties` fields.

Live observations: the lifecycle smoke ran on `flex-sjc`, using disposable container `gocli-test-object-go-20260503` and local fixture `/private/tmp/golang-osc-object-live.txt`. The Go CLI uploaded the object with the default full-path object name, uploaded a renamed copy with `--name renamed-object.txt`, set and unset object metadata, saved the renamed object to stdout, deleted both objects, deleted the container, and verified no `gocli-test-object` containers remained.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/object/create.txt`, `compat/osc/9.0.0/help/object/delete.txt`, `compat/osc/9.0.0/help/object/save.txt`, `compat/osc/9.0.0/help/object/set.txt`, and `compat/osc/9.0.0/help/object/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/object/v1/object.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Object Storage objects](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects).

## 2026-05-04: Security Group Create And Delete

Work done: added `security group create` and `security group delete`, and tightened `security group show` output.

Implementation note: `security group create` uses Gophercloud's Neutron security group create helper for the base resource. It mirrors Python OSC's default description behavior by using the group name when `--description` is not supplied, resolves `--project` through Identity when provided, supports `--stateful` and `--stateless`, accepts OSC-style `--extra-property`, and applies `--tag` through Neutron's standard tag subresource after create. The create output renders the raw Neutron create body with the local tag update applied, so tagged creates preserve Python OSC's create-time revision behavior. `security group show` now renders the raw Neutron security group body after Gophercloud lookup, including `is_shared` and the full nested rule dictionaries that Python OSC exposes.

Live observations on `cloud6`: Python OSC created and deleted disposable groups `gocli-test-sg-python-20260504` and `gocli-test-sg-tag-python-20260504`. The Go CLI created and deleted disposable groups `gocli-test-sg-go-20260504`, `gocli-test-sg-tag-go-20260504`, and `gocli-test-sg-tag-go2-20260504`. The tagged create path was checked against Python OSC's JSON behavior, the Go delete command deleted multiple groups in one invocation, and Python/Go `security group list -f json` matched after cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/security/group/create.txt` and `compat/osc/9.0.0/help/security/group/delete.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/security_group.py`, used only as the pinned local oracle implementation source.
* Local osc-lib tag helper source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/osc_lib/utils/tags.py`, used to mirror create-time tag behavior.
* Local OpenStackSDK resource sources `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/security_group.py` and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/common/tag.py`, which document the Neutron security-group path and tag subresource behavior.
* Gophercloud package docs for [Neutron security groups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups).

## 2026-05-04: Security Group Set And Unset

Work done: added `security group set` and `security group unset`.

Implementation note: `security group set` uses Gophercloud's Neutron security group update helper for name, description, statefulness, and OSC-style extra properties. It then reconciles tags through the same Neutron standard tag subresource used by create. The `--no-tag --tag ...` combination is intentionally supported because Python OSC treats it as "clear existing tags, then apply the requested tags." `security group unset` removes selected tags or clears all tags without failing when a requested tag is already absent, matching the osc-lib tag helper behavior.

Live observations on `cloud6`: the Go CLI created disposable group `gocli-test-sg-set-go-20260504`, renamed it to `gocli-test-sg-set-go-renamed-20260504`, changed its description, overwrote tags with `--no-tag --tag keep --tag newtag`, removed one tag with `security group unset --tag keep`, cleared the remaining tag with `--all-tag`, deleted the group, and verified only preexisting default security groups remained.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/security/group/set.txt` and `compat/osc/9.0.0/help/security/group/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/security_group.py`, used only as the pinned local oracle implementation source.
* Local osc-lib tag helper source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/osc_lib/utils/tags.py`, used to mirror set and unset tag behavior.
* Gophercloud package docs for [Neutron security groups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups).

## 2026-05-04: Security Group Rule Create And Delete

Work done: added `security group rule create` and `security group rule delete`, and tightened `security group rule show` output.

Implementation note: `security group rule create` uses Gophercloud's Neutron security group rule create helper, resolves the parent security group and optional remote security group or address group by name or ID, supports project ownership lookup, accepts OSC-style extra properties, implements OSC's default ingress direction, `any` protocol handling, derived ethertype, default remote CIDR, TCP/UDP destination port parsing, and ICMP type/code validation. Create and show output render the raw Neutron rule body with Python-compatible field names, including `ether_type`, `belongs_to_default_sg`, and `normalized_cidr` where Neutron returns them. `security group rule delete` deletes by rule ID and reports partial failures across multiple IDs.

Live observations on `cloud6`: the Go CLI created disposable groups `gocli-test-sgrule-go-20260504` and `gocli-test-sgrule-go2-20260504`, added disposable TCP/443 ingress rules, compared the rule create JSON shape against Python OSC's `gocli-test-sgrule-python-20260504` rule output, deleted the Go-created rule by ID, deleted the groups, and verified only preexisting default security groups remained.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/security/group/rule/create.txt` and `compat/osc/9.0.0/help/security/group/rule/delete.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/security_group_rule.py`, used only as the pinned local oracle implementation source.
* Local Python OSC network utility source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/utils.py`, used for protocol, ethertype, port-range, and remote-prefix behavior.
* Local osc-lib parser action source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/osc_lib/cli/parseractions.py`, used for `--dst-port` range parsing behavior.
* Gophercloud package docs for [Neutron security group rules](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules).

## 2026-05-04: Address Group Lifecycle Commands

Work done: added `address group create`, `address group delete`, `address group set`, and `address group unset`, and tightened `address group show` output.

Implementation note: address group create/update/delete uses Gophercloud's Neutron address group helpers. The Go CLI normalizes `--address` values with Go's `net/netip` to mirror Python OSC's `netaddr.IPNetwork` behavior, so plain IPv4 addresses become `/32` and plain IPv6 addresses become `/128`. Create and show output render the raw Neutron address group body, because Python OSC exposes `created_at`, `revision_number`, and `updated_at`, while Gophercloud's typed `AddressGroup` struct currently omits those fields.

Live observations on `cloud6`: the Go CLI created disposable groups `gocli-test-address-group-go-20260504` and `gocli-test-address-group-go2-20260504`, verified address normalization against Python OSC's `gocli-test-address-group-python-20260504` output, added an address with `address group set`, removed one with `address group unset`, renamed and updated the description, deleted the group, and verified `address group list -f json` returned an empty list after cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/address/group/create.txt`, `compat/osc/9.0.0/help/address/group/delete.txt`, `compat/osc/9.0.0/help/address/group/set.txt`, and `compat/osc/9.0.0/help/address/group/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/address_group.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron address groups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups).
* Go package docs for [net/netip](https://pkg.go.dev/net/netip), used for address and CIDR normalization.

## 2026-05-04: Address Scope Lifecycle Commands

Work done: added `address scope create`, `address scope delete`, and `address scope set`, and tightened `address scope show` output.

Implementation note: address scope create/update/delete uses Gophercloud's Neutron address scope helpers. Create uses a narrow custom request builder so `--no-share` can be sent explicitly when requested, and so OSC-style `--extra-property` values can be carried in the `address_scope` body. Create and show output render the raw Neutron address scope body when available, preserving the Python OSC field order observed for `id`, `ip_version`, `name`, `project_id`, and `shared`.

Live observations on `cloud6`: Python OSC created, showed, listed, and deleted disposable scope `gocli-test-address-scope-python-probe-20260504` to confirm JSON field shape. The Go CLI created shared scope `gocli-test-address-scope-go-20260504`, showed and listed it, then deleted it. Neutron rejected an attempted shared-to-private update with `AddressScopeUpdateError`, so the allowed mutation path was verified separately by creating private IPv6 scope `gocli-test-address-scope-go-private-20260504`, renaming it to `gocli-test-address-scope-go-renamed-20260504`, sharing it, showing the updated state, deleting it, and verifying all disposable scope names returned empty lists after cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/address/scope/create.txt`, `compat/osc/9.0.0/help/address/scope/delete.txt`, and `compat/osc/9.0.0/help/address/scope/set.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/address_scope.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron address scopes](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes).

## 2026-05-04: Subnet Pool Lifecycle Commands

Work done: added `subnet pool create`, `subnet pool delete`, `subnet pool set`, and `subnet pool unset`, and tightened `subnet pool show` output.

Implementation note: subnet pool create/update/delete uses Gophercloud's Neutron subnet pool helpers with custom request builders for fields where Python OSC can send false, null, and extra-property values. `subnet pool set --pool-prefix` mirrors Python OSC by extending the existing prefix list rather than replacing it, while Neutron may still merge adjacent prefixes in the returned resource. Tags are handled through the standard Neutron tag subresource, including Python's `--no-tag --tag ...` overwrite behavior on `set`. Raw show output normalizes Neutron's string-form prefix lengths back to numeric JSON fields, matching Python OSC's SDK-normalized output.

Live observations on `cloud6`: Python OSC created, showed, listed, and deleted `gocli-test-subnet-pool-python-probe-20260504` to confirm output field order and default-quota null behavior. The Go CLI created disposable address scope `gocli-test-subnet-pool-scope-20260504`, created subnet pool `gocli-test-subnet-pool-go-20260504` with an address-scope association and tag, renamed it, added a second prefix, removed the address-scope association, replaced tags with `--no-tag --tag second`, removed that tag with `subnet pool unset`, deleted the pool, deleted the scope, and verified all disposable names returned empty lists after cleanup. A second short Go create/delete check with `gocli-test-subnet-pool-go-normalize-20260504` verified numeric JSON output for prefix lengths after raw-response normalization.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/subnet/pool/create.txt`, `compat/osc/9.0.0/help/subnet/pool/delete.txt`, `compat/osc/9.0.0/help/subnet/pool/set.txt`, and `compat/osc/9.0.0/help/subnet/pool/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/subnet_pool.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron subnet pools](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools).

## 2026-05-04: Network Lifecycle Commands

Work done: added `network create`, `network delete`, `network set`, and `network unset`, tightened `network show` output, and made `network list --name`, `--project`, and `--status` use Neutron filters.

Implementation note: network create/update/delete uses Gophercloud's Neutron network helpers with custom request builders so extension-backed OSC flags can be passed in the `network` body. The implementation covers the common and extension fields exposed by Python OSC help for shared state, admin state, project ownership, description, MTU, availability-zone hints, port security, external/default network flags, QoS policy references, provider fields, DNS domain, VLAN transparency, QinQ, standard tags, and extra properties. `network unset --extra-property` follows Python OSC's unset command behavior by sending `null` for each named extra property. Create and show output render the raw Neutron network body with Python OSC field names, including mapped `is_vlan_qinq`, `is_vlan_transparent`, `provider:*`, `router:external`, address-scope, and port-security fields.

Live observations on `cloud6`: Python OSC created, showed, listed, and deleted `gocli-test-network-python-probe-20260504` to confirm output field order and extension-field defaults. The Go CLI created disposable network `gocli-test-network-go-20260504` with `--disable`, description, and tag, showed it, verified `network list --name` filtering, renamed it to `gocli-test-network-go-renamed-20260504`, enabled it, updated the description, overwrote the tag with `--no-tag --tag second`, removed that tag with `network unset`, deleted the network, and verified both disposable names returned empty lists after cleanup.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/network/create.txt`, `compat/osc/9.0.0/help/network/delete.txt`, `compat/osc/9.0.0/help/network/set.txt`, and `compat/osc/9.0.0/help/network/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/network.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/network.py`, used to map SDK attribute names to Neutron network JSON fields.
* Gophercloud package docs for [Neutron networks](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks).

## 2026-05-04: Subnet Lifecycle Commands

Work done: added `subnet create`, `subnet delete`, `subnet set`, and `subnet unset`, and tightened `subnet show` output.

Implementation note: subnet create/update/delete uses Gophercloud's Neutron subnet helpers with custom request builders for OSC-specific fields and extension values. The implementation resolves network, project, subnet pool, and network segment references, supports CIDR and subnet-pool allocation modes, DHCP and DNS publish flags, gateway handling, IPv6 modes, allocation pools, DNS nameservers, host routes, service types, standard tags, and extra properties. `subnet set` mirrors Python OSC's merge behavior for DNS nameservers, host routes, allocation pools, and service types, while `subnet unset` removes specific entries and reports an error if a requested value is absent. Raw show output follows Python OSC field order, and allocation-pool output uses ordered `start`, then `end` fields to match Python's JSON shape.

Live observations on `cloud6`: Python OSC created disposable network `gocli-test-subnet-python-net-20260504`, then created, showed, deleted subnet `gocli-test-subnet-python-probe-20260504`, and deleted the network to confirm field order. The Go CLI created disposable network `gocli-test-subnet-go-net-20260504`, created subnet `gocli-test-subnet-go-20260504` with allocation pool, DNS nameserver, host route, description, and tag, renamed it, appended additional allocation pool, DNS nameserver, and host route values, overwrote tags, removed the appended values and tag, deleted the subnet, deleted the network, and verified all disposable names returned empty lists after cleanup. A second short Go create/delete check with `gocli-test-subnet-order-20260504` verified nested allocation-pool JSON key order.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/subnet/create.txt`, `compat/osc/9.0.0/help/subnet/delete.txt`, `compat/osc/9.0.0/help/subnet/set.txt`, and `compat/osc/9.0.0/help/subnet/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/subnet.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron subnets](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets).

## 2026-05-04: Router Lifecycle Commands

Work done: added `router create`, `router delete`, `router set`, and `router unset`, and tightened `router show` output.

Implementation note: router create/update/delete uses Gophercloud's Neutron router helpers with custom request builders for OSC extension fields. The implementation covers admin state, distributed and HA flags, description, project ownership, availability-zone hints, tags, route mutation, external gateway body fields, SNAT, NDP proxy, QoS policy references, default-route BFD/ECMP flags, flavor ID passthrough, and extra properties. Raw show output follows Python OSC field order and adds `interfaces_info` for show output, including an empty list when no router interfaces exist.

Live observations on `cloud6`: Python OSC created, showed, and deleted `gocli-test-router-python-probe-20260504` to confirm output field order. The Go CLI created router `gocli-test-router-go-20260504` with `--disable`, description, and tag, showed it, renamed it to `gocli-test-router-go-renamed-20260504`, enabled it, updated the description, overwrote then removed its tag, deleted it, and verified both disposable names returned empty lists after cleanup. A second short Go create/show/delete check with `gocli-test-router-interfaces-20260504` verified that `router show -f json` emits `interfaces_info: []` for routers without interfaces.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/router/create.txt`, `compat/osc/9.0.0/help/router/delete.txt`, `compat/osc/9.0.0/help/router/set.txt`, and `compat/osc/9.0.0/help/router/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/router.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron routers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers).

## 2026-05-04: Router Interface Commands

Work done: added `router add port`, `router add subnet`, `router remove port`, and `router remove subnet`.

Implementation note: the four commands follow Python OSC's positional-only interface command shape. The Go CLI resolves the router and port or subnet by name or ID, then calls Gophercloud's Neutron router `AddInterface` and `RemoveInterface` helpers. The commands intentionally print no structured output on success, matching the Python command classes.

Live observations on `cloud6`: the Go CLI created disposable network, subnet, and router resources, added the subnet to the router, confirmed `router show -f json` included the router interface, removed that interface by port ID, added the subnet again, removed it by subnet name, and deleted all disposable resources. A second live check created a disposable Neutron port with Python OSC as a test fixture, then used the Go CLI to add and remove that port as a router interface, followed by cleanup and empty-list verification for the disposable router, network, subnet, and port prefixes.

Tooling update: `tools/matrix` now records the recently verified Neutron lifecycle write commands, including router interface commands, network lifecycle commands, subnet lifecycle commands, subnet pool lifecycle commands, and address scope lifecycle commands, so regenerated `compat/matrix.yaml` no longer shows those rows as `unknown`.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/router/add/port.txt`, `compat/osc/9.0.0/help/router/add/subnet.txt`, `compat/osc/9.0.0/help/router/remove/port.txt`, and `compat/osc/9.0.0/help/router/remove/subnet.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/router.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron routers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers).

## 2026-05-04: Port Lifecycle Commands

Work done: added `port create`, `port delete`, `port set`, and `port unset`, and changed `port show` to use Python-compatible raw Neutron field names where possible.

Implementation note: port create/update/delete uses Gophercloud's Neutron port helper methods with custom request builders for OSC extension fields such as `binding:*`, DNS fields, NUMA affinity, hints, trusted status, QoS policy IDs, port security, allowed address pairs, extra DHCP options, data plane status, device profile, hardware offload type, and standard tags. `port show` now maps Neutron's raw `security_groups` and `binding:*` fields to Python OSC's displayed `security_group_ids`, `binding_profile`, `binding_vif_details`, `binding_vif_type`, `binding_vnic_type`, and `binding_host_id` fields.

Live observations on `cloud6`: the Go CLI created disposable network and subnet resources, created disabled tagged port `gocli-test-port-*` with a fixed IP and no security groups, verified create JSON, renamed and enabled the port, replaced its tag, removed the tag with `port unset`, deleted the port, and verified cleanup. A follow-up router interface check used `port create` from the Go CLI instead of a Python fixture, attached the port to a disposable router, detached it, observed that Neutron deletes the interface port on `router remove port`, and cleaned up the remaining network, subnet, and router resources.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/port/create.txt`, `compat/osc/9.0.0/help/port/delete.txt`, `compat/osc/9.0.0/help/port/set.txt`, and `compat/osc/9.0.0/help/port/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/port.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/port.py`, used to map SDK attribute names to Neutron port JSON fields.
* Gophercloud package docs for [Neutron ports](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports).

## 2026-05-04: Floating IP Lifecycle Commands

Work done: added `floating ip create`, `floating ip delete`, `floating ip set`, and `floating ip unset`, and tightened `floating ip list/show` output.

Implementation note: floating IP create/update/delete uses Gophercloud's Neutron floating IP helpers with custom request builders for OSC extension fields, including `qos_policy_id`, DNS fields, tags, and extra properties. Show output now follows the Python OpenStackSDK sorted column behavior for floating IPs, including nullable fields such as `dns_domain`, `fixed_ip_address`, `port_id`, `qos_policy_id`, `router_id`, and `subnet_id`. List output now includes the default `Project` column and emits null fixed IP and port values for unassociated floating IPs, matching the Python OSC JSON shape checked on `cloud6`.

Live observations on `cloud6`: the Go CLI allocated a disposable floating IP from external network `os6-lan`, set the description and initial tag, showed it by allocated address, replaced the tag with `floating ip set --no-tag --tag second`, removed the tag with `floating ip unset`, deleted the floating IP, and verified list/show parity against Python OSC for an existing unassociated floating IP fixture.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/floating/ip/create.txt`, `compat/osc/9.0.0/help/floating/ip/delete.txt`, `compat/osc/9.0.0/help/floating/ip/set.txt`, and `compat/osc/9.0.0/help/floating/ip/unset.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/floating_ip.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/floating_ip.py`, used to map SDK attribute names to Neutron floating IP JSON fields.
* Gophercloud package docs for [Neutron floating IPs](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips).

## 2026-05-04: Floating IP Port Forwarding Commands

Work done: added `floating ip port forwarding create`, `floating ip port forwarding delete`, `floating ip port forwarding list`, `floating ip port forwarding set`, and `floating ip port forwarding show`.

Implementation note: create, update, get, and delete use Gophercloud's Neutron port forwarding package. The request builders are custom so Python OSC's `--extra-property` passthrough and `internal_port_range`/`external_port_range` fields can be sent even though the current Gophercloud struct only exposes single-port fields. List uses a raw Neutron JSON request through the authenticated service client so the Python list columns for `Internal Port Range` and `External Port Range` are preserved instead of being dropped by typed extraction.

Live observations on `cloud6`: the Go CLI created disposable network, subnet, router, port, and floating IP fixtures, but Neutron returned `404` for `POST /v2.0/floatingips/{id}/port_forwardings`. Cleanup removed the disposable resources. A read-only comparison against existing floating IP `06376810-3b16-4160-b1d7-6135571d2efd` showed Python OSC also returns `NotFoundException: 404` for `GET /v2.0/floatingips/{id}/port_forwardings` on the same cloud. The commands remain implemented but not cloud-verified until a test cloud exposes the port-forwarding endpoint.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/floating/ip/port/forwarding/create.txt`, `compat/osc/9.0.0/help/floating/ip/port/forwarding/delete.txt`, `compat/osc/9.0.0/help/floating/ip/port/forwarding/list.txt`, `compat/osc/9.0.0/help/floating/ip/port/forwarding/set.txt`, and `compat/osc/9.0.0/help/floating/ip/port/forwarding/show.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/floating_ip_port_forwarding.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron floating IP port forwarding](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/portforwarding).

## 2026-05-04: Router Gateway And Route Helper Commands

Work done: added `router add gateway`, `router remove gateway`, `router add route`, and `router remove route`.

Implementation note: route helpers call Neutron's `add_extraroutes` and `remove_extraroutes` router action endpoints through the authenticated Gophercloud service client, because no typed local Gophercloud helper exists in v2.12.0. Gateway helpers call Neutron's `add_external_gateways` and `remove_external_gateways` action endpoints and intentionally check for `external-gateway-multihoming` before resolving router or network arguments, matching Python OSC's command order.

Live observations on `cloud6`: the Go CLI created disposable network, subnet, and router resources, attached the subnet, added route `192.0.2.0/24` via the disposable subnet, removed it, repeated the remove for the documented missing-route success path, and cleaned up all disposable resources. `cloud6` advertises `extraroute` and `extraroute-atomic` but not `external-gateway-multihoming`; Python OSC and the Go CLI both return `The external-gateway-multihoming extension is not enabled at the Neutron side.` for gateway add/remove on that cloud.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/router/add/gateway.txt`, `compat/osc/9.0.0/help/router/remove/gateway.txt`, `compat/osc/9.0.0/help/router/add/route.txt`, and `compat/osc/9.0.0/help/router/remove/route.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/router.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/router.py`, used to confirm the Neutron action endpoint names.
* Gophercloud package docs for [Neutron routers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers).

## 2026-05-04: QoS Policy Lifecycle Commands

Work done: added `network qos policy create`, `network qos policy delete`, and `network qos policy set`, and changed QoS policy show/create rendering to preserve raw extension fields when available.

Implementation note: QoS policy create/update/delete uses Gophercloud's QoS policy package. Create and update use custom request builders so explicit `shared=false`, `is_default=false`, and Python OSC `--extra-property` values are retained in the Neutron `policy` body.

Live observations on `cloud6`: the Go CLI attempted a disposable QoS policy create with a unique `golang-osc-test-qos-*` name, but Neutron returned `404` for `POST /v2.0/qos/policies`. Python OSC also returns `NotFoundException: 404` for `GET /v2.0/qos/policies` on the same cloud, and the Go CLI returns the same endpoint-level 404 for list. No disposable QoS policy was created. The commands remain implemented but not cloud-verified until a test cloud exposes the QoS policy endpoint.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/network/qos/policy/create.txt`, `compat/osc/9.0.0/help/network/qos/policy/delete.txt`, and `compat/osc/9.0.0/help/network/qos/policy/set.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/network_qos_policy.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron QoS policies](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies).

## 2026-05-04: Network RBAC Lifecycle Commands

Work done: added `network rbac create`, `network rbac delete`, and `network rbac set`, and changed RBAC show/create rendering to preserve raw extension fields when available.

Implementation note: Network RBAC create/update/delete uses Gophercloud's RBAC policy package with custom request builders for owner project passthrough and Python OSC `--extra-property` values. Create resolves RBAC object names by object type across networks, QoS policies, security groups, address scopes, subnet pools, and address groups, and resolves target and owner projects through Identity v3.

Live observations on `cloud6`: the Go CLI created disposable network `golang-osc-test-rbac-*`, created an `access_as_shared` RBAC policy targeting all projects, verified show and long-list output, updated the RBAC policy to target the current token project, deleted the RBAC policy, deleted the network, and verified cleanup through the command success path.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/network/rbac/create.txt`, `compat/osc/9.0.0/help/network/rbac/delete.txt`, and `compat/osc/9.0.0/help/network/rbac/set.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/network_rbac.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron RBAC policies](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies).

## 2026-05-04: Network QoS Rule Commands

Work done: added `network qos rule create`, `network qos rule delete`, `network qos rule list`, `network qos rule set`, and `network qos rule show`.

Implementation note: QoS rule create/update/delete uses Gophercloud's QoS rule package for `bandwidth-limit`, `dscp-marking`, and `minimum-bandwidth`. Gophercloud v2.12.0 does not expose a typed `minimum-packet-rate` rule helper in the local module source, so that subtype uses a narrow authenticated Neutron request against `/qos/policies/{policy_id}/minimum_packet_rate_rules`. The OSC argument rules are mirrored from Python OSC: create requires the subtype's mandatory parameters, `--max-burst-kbits` is sent to Neutron as `max_burst_kbps`, `--any` is accepted only for `minimum-packet-rate`, and show output follows OpenStackSDK's per-resource column ordering.

Live observations on `cloud6`: no disposable QoS rule lifecycle was attempted because the prerequisite QoS policy collection returns Neutron `404` on `cloud6` for both Python OSC and the Go CLI. A negative smoke of `network qos rule list golang-osc-missing-policy -f json` reached the new Go command path and returned a Neutron `404`; Python OSC returned the same endpoint-level `404` while resolving the policy by name. The commands remain implemented but not cloud-verified until a test cloud exposes the QoS policy and rule endpoints.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/network/qos/rule/create.txt`, `compat/osc/9.0.0/help/network/qos/rule/delete.txt`, `compat/osc/9.0.0/help/network/qos/rule/list.txt`, `compat/osc/9.0.0/help/network/qos/rule/set.txt`, and `compat/osc/9.0.0/help/network/qos/rule/show.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/network_qos_rule.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK resource files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/qos_bandwidth_limit_rule.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/qos_dscp_marking_rule.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/qos_minimum_bandwidth_rule.py`, and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/qos_minimum_packet_rate_rule.py`, used to confirm resource keys and endpoint paths.
* Gophercloud package docs for [Neutron QoS rules](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/rules).

## 2026-05-04: Network Segment Lifecycle Commands

Work done: added `network segment create`, `network segment delete`, and `network segment set`, and changed segment show/create rendering to preserve raw extension fields when available.

Implementation note: segment create/update/delete uses Gophercloud's segment package. Create and update use custom request builders so Python OSC `--extra-property` values are retained in the Neutron `segment` body. Create resolves `--network` through the existing network lookup helper, validates the Python OSC network-type choices, and maps `--segment` to Neutron's `segmentation_id`.

Live observations on `cloud6`: no disposable segment lifecycle was attempted because the segment collection returns Neutron `404` on `cloud6` for both Python OSC and the Go CLI. The Go CLI and Python OSC both returned endpoint-level `404` for `network segment list -f json`. The commands remain implemented but not cloud-verified until a test cloud exposes the segment extension.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/network/segment/create.txt`, `compat/osc/9.0.0/help/network/segment/delete.txt`, and `compat/osc/9.0.0/help/network/segment/set.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/network_segment.py`, used only as the pinned local oracle implementation source.
* Local OpenStackSDK resource file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/network/v2/segment.py`, used to confirm resource keys and field names.
* Gophercloud package docs for [Neutron segments](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments).

## 2026-05-04: Network Trunk and Subport Commands

Work done: added `network trunk create`, `network trunk delete`, `network trunk set`, `network trunk unset`, and `network subport list`, and adjusted trunk list/show output to match the Python OSC default column surface more closely.

Implementation note: trunk create/update/delete and subport actions use Gophercloud's trunk package. Create, update, and add/remove subports use custom request builders so Python OSC's partial subport dictionaries are preserved instead of being coerced through Gophercloud's fixed `Subport` struct. Create defaults `admin_state_up` to true unless `--disable` is provided, resolves parent and subports through port lookup, resolves `--project` through Identity v3, and maps show output to Python OSC's `is_admin_state_up` field name.

Live observations on `cloud6`: no disposable trunk lifecycle was attempted because the trunk collection returns Neutron `404` on `cloud6` for both Python OSC and the Go CLI. The Go CLI and Python OSC both returned endpoint-level `404` for `network trunk list -f json`. The commands remain implemented but not cloud-verified until a test cloud exposes the trunk extension.

Sources consulted:

* Local OSC oracle snapshots in `compat/osc/9.0.0/help/network/trunk/create.txt`, `compat/osc/9.0.0/help/network/trunk/delete.txt`, `compat/osc/9.0.0/help/network/trunk/set.txt`, `compat/osc/9.0.0/help/network/trunk/unset.txt`, and `compat/osc/9.0.0/help/network/subport/list.txt`.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/network/v2/network_trunk.py`, used only as the pinned local oracle implementation source.
* Gophercloud package docs for [Neutron trunks](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks).

## 2026-05-04: Pretty Renderer With Charm Bubbles

Work done: replaced the ad hoc `--pretty` key/value list renderer with a Charm Bubbles table renderer for structured list and show output. Default `table` output still uses the local OSC-compatible renderer, so this dependency is only on the Go-only `--format=pretty` and `--pretty` path.

Decision: use Charm's v2 module paths, `charm.land/bubbles/v2/table`, `charm.land/bubbles/v2/progress`, and `charm.land/lipgloss/v2`, for pretty tabular output, reusable wait/progress rendering, and optional TTY styling. Color is enabled only when stdout is a terminal and `NO_COLOR` is not set. Non-TTY output remains ANSI-free plain text.

Implementation note: the new progress helper is available for command wait loops, but existing `--wait` command behavior still needs command-level lifecycle integration before live progress can appear during long-running server, image, volume, or delete operations.

Sources consulted:

* [Charm Bubbles](https://github.com/charmbracelet/bubbles), whose README describes table and progress components for Bubble Tea applications.
* [Charm Bubbles tags](https://github.com/charmbracelet/bubbles/tags), which show `v2.1.0` as the current v2 tag used by this implementation.
* [Charm Lip Gloss](https://github.com/charmbracelet/lipgloss), used for terminal styling.

## 2026-05-04: Default List and Show Output Parity Check

Work done: compared default `table` output from the local Python OSC oracle and the Go CLI against `cloud6` for representative `list` and `show` commands. Fixed shared renderer parity gaps for Python-style `None`, empty slice and map display, numeric list-column right alignment, typed empty maps from Gophercloud resources, and server network summaries. Also fixed field order and field inclusion for `role show` and `region show`, sorted `image list` by name to match the oracle on `cloud6`, and changed `router list` state display from boolean values to `UP` and `DOWN`.

Live observations on `cloud6`: the final default-output spot check matched byte-for-byte for `project list`, `project show admin`, `user list`, `user show admin`, `service list`, `service show nova`, `domain list`, `domain show default`, `region list`, `region show RegionOne`, `role list`, `role show admin`, `flavor list`, `image list`, `network list`, `router list`, `server list`, `security group list`, `floating ip list`, and `volume type list`.

Known remaining default-output mismatches from the same read-only pass are command-specific rather than shared table rendering: `flavor show m1.tiny`, `image show cirros`, `network show os6-lan`, `subnet show os6-subnet`, `router show testRouter`, `server show rocky`, `security group show 92e5d908-af34-4360-9dbc-a91c538fc44e`, `floating ip show 06376810-3b16-4160-b1d7-6135571d2efd`, `volume list`, and `volume show b95c61ef-5d8e-4530-91a1-2175aa378c54`. These need per-command field and formatter work, especially OpenStackSDK-style show field surfaces and command-specific list formatters.

Sources consulted:

* Local Python OSC oracle `/Users/ken/.local/bin/openstack`, which reports `openstack 9.0.0`.
* Local Go command `./bin/openstack`, built from this repository during the check.
* Local implementation files `internal/cli/output.go`, `internal/cli/table.go`, `internal/cli/identity_read.go`, and `internal/cli/core_read.go`.

## 2026-05-04: OS_PRETTY Environment Default

Work done: added `OS_PRETTY=1` as a Go-only environment default for enhanced pretty output. This mirrors the convenience of `OS_CLOUD`-style environment configuration without changing the Python-compatible default when the variable is absent.

Decision: `OS_PRETTY=1` is a default, not a hard override. An explicit `--format` value such as `-f json` wins over the environment default, while an explicit `--pretty` still forces pretty output even if a format flag is also present.

Sources consulted:

* Local implementation file `internal/cli/root.go`.
* Local parser and pretty-output tests in `internal/cli/root_test.go`.

## 2026-05-04: Pretty Multiline Wrapping

Work done: changed the `pretty` renderer to preserve explicit multiline cell values and wrap long cells into aligned continuation rows before passing them to the Bubbles table component. This prevents long values, such as `hypervisor show` `cpu_info`, from being truncated to a single ellipsis-ended row.

Live observations on `cloud6`: `OS_CLOUD=cloud6 ./bin/openstack --pretty hypervisor show 3e999761-e6fa-4ad7-9d89-ddd2592e7554` now renders `cpu_info` and `uptime` over multiple aligned rows instead of truncating them.

Sources consulted:

* Local implementation file `internal/cli/output.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.

## 2026-05-04: Pretty Structured Values

Work done: changed the `pretty` renderer to format structured values as readable multiline blocks instead of dense JSON. The formatter now detects maps, slices, arrays, structs, and JSON object or array strings, and renders them using indented `key: value` and list-item lines before table wrapping.

Live observations on `cloud6`: `OS_CLOUD=cloud6 ./bin/openstack --pretty hypervisor show 3e999761-e6fa-4ad7-9d89-ddd2592e7554` now renders `cpu_info` as `vendor`, `arch`, `model`, `features`, and `topology` sections instead of dense JSON.

Sources consulted:

* Local implementation file `internal/cli/output.go`.
* Local Gophercloud hypervisor type definition in `/Users/ken/Dev/openstack-go/.cache/gomod/github.com/gophercloud/gophercloud/v2@v2.12.0/openstack/compute/v2/hypervisors/results.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.

## 2026-05-04: Pretty IP Address Display

Work done: changed pretty-only address rendering for server, port, address group, and router output. Server list and show now group IP addresses by network name. Port list now shows a vertical list of fixed IP addresses without subnet UUID clutter, while port show keeps subnet detail under each IP address. Router show now formats external gateway fixed IPs and router interface IPs with readable labels instead of raw `ip_address` and `subnet_id` keys.

Implementation note: default output still uses the existing Python-compatible renderer and command values. The new formatters are passed only when `--pretty`, `--format=pretty`, or the existing `OS_PRETTY=1` default selects the pretty renderer.

Live observations on `cloud6`: `server list --pretty` showed `testNet` addresses as a vertical list, `server show rocky --pretty` showed `os6-lan` with its fixed IP on the following line, `port list --pretty` showed one fixed IP per row without subnet wrapping, `port show 23555be5-4030-4a3c-a80c-5c0e7ed7791f --pretty` kept `subnet` detail below `172.16.86.56`, and `router show testRouter --pretty` rendered `external fixed IPs` and `interfaces_info` as nested address blocks.

Sources consulted:

* Local implementation files `internal/cli/core_read.go` and `internal/cli/output.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.
* Gophercloud local type definitions for ports in `/Users/ken/Dev/openstack-go/.cache/gomod/github.com/gophercloud/gophercloud/v2@v2.12.0/openstack/networking/v2/ports/results.go`.
* Gophercloud local type definitions for router gateway fixed IPs in `/Users/ken/Dev/openstack-go/.cache/gomod/github.com/gophercloud/gophercloud/v2@v2.12.0/openstack/networking/v2/extensions/layer3/routers/results.go`.

## 2026-05-04: Pretty Semantic Color

Work done: added semantic colorization to the pretty renderer for TTY output. UUIDs, IP addresses, names, flavor names, flavor component numbers, booleans, and common status values now receive distinct colors. Non-TTY output and `NO_COLOR=1` remain ANSI-free.

Implementation note: pretty output still computes widths and wraps cells before applying semantic color. This keeps ANSI escape sequences out of the local wrapping math while still allowing Charm Bubbles and Lip Gloss to render styled cells.

Sources consulted:

* Local implementation files `internal/cli/output.go` and `internal/cli/command_list.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.
* Local Charm Bubbles table implementation in `/Users/ken/Dev/openstack-go/.cache/gomod/charm.land/bubbles/v2@v2.1.0/table/table.go`, used to confirm table rendering truncates ANSI-aware cell strings.
* Local Lip Gloss v2 implementation in `/Users/ken/Dev/openstack-go/.cache/gomod/charm.land/lipgloss/v2@v2.0.3`, used for semantic terminal styles.

## 2026-05-04: Pretty Color Refinements

Work done: refined UUID and image semantic color. UUID hex groups remain colored, but UUID hyphen separators are now left plain. Image values now color the `N/A` token, including values such as `N/A (booted from volume)`, without coloring the explanatory suffix.

Sources consulted:

* Local implementation file `internal/cli/output.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.

## 2026-05-04: Pretty Server Network Labels

Work done: changed pretty server address rendering so the network name repeats on every IP address line. This fixes rows with multiple IP addresses where only the first address visually carried the network name.

Live observations on `cloud6`: `server list --pretty` now renders multi-address rows as `testNet: 172.16.86.110` and `testNet: 172.17.36.42` on separate lines instead of showing `testNet:` once followed by unlabeled IP lines.

Sources consulted:

* Local implementation file `internal/cli/core_read.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.

## 2026-05-04: Pretty Entry Spacing

Work done: added a blank spacer line between logical pretty output entries. Wrapped lines for a single entry stay grouped together, and the spacer is inserted before the next entry.

Live observations on `cloud6`: `server list --pretty` now shows a blank line between server entries while keeping each server's wrapped ID, image, and network-address lines grouped together.

Sources consulted:

* Local implementation file `internal/cli/output.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.

## 2026-05-04: Pretty Header Spacing

Work done: added one blank line between the Bubbles table heading row and the first content row in pretty output. This keeps the existing blank spacer lines between entries, while making the heading area easier to scan.

Observation: the pinned Bubbles table API does not have a named "filled heading" option. It exposes a `table.Styles.Header` Lip Gloss style through `table.WithStyles`, and Bubbles applies that style to each rendered header cell. Because Lip Gloss styles support backgrounds, a filled header can be built by setting a background on the header style, but that would be a styling choice implemented in this CLI rather than a separate Bubbles table feature.

Sources consulted:

* Local implementation file `internal/cli/output.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.
* Local Charm Bubbles table implementation in `/Users/ken/Dev/openstack-go/.cache/gomod/charm.land/bubbles/v2@v2.1.0/table/table.go`.
* Local Lip Gloss v2 implementation in `/Users/ken/Dev/openstack-go/.cache/gomod/charm.land/lipgloss/v2@v2.0.3`.

## 2026-05-04: Pretty Label Prefix Color

Work done: changed TTY pretty output so lines that begin with a `label: value` shape render the `label:` prefix in bright white. The remaining value keeps the existing semantic coloring for UUIDs, IP addresses, status values, `N/A`, and other recognized tokens.

Implementation note: the colorizer handles this after wrapping and only when pretty color is enabled. Non-TTY pretty output and default OSC-compatible output remain plain and unchanged.

Sources consulted:

* Local implementation file `internal/cli/output.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.

## 2026-05-04: Pretty Label Value Contrast

Work done: changed labeled pretty output so the `label:` prefix and the value render with intentionally different styles. Labels now use bright white, while inline values and wrapped continuation values use a non-bright value style. This fixes structured columns such as `volume list` `Attached to`, where `attachment_id:` and similar labels can wrap onto a separate line from their value.

Implementation note: the wrapped-cell color pass now remembers when a line is a label with no inline value and applies the non-bright value style to following continuation lines until the next label, list marker, or blank line.

Sources consulted:

* Local implementation file `internal/cli/output.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.

## 2026-05-04: Pretty Structured Attachment Display

Work done: restored semantic coloring for values in `label: value` pretty output. Labels stay bright white, but values now pass back through the normal value colorizer so IP addresses, UUIDs, and other recognized values keep their colors. Wrapped continuation values under ID-like labels, such as `attachment_id:`, `server_id:`, and `volume_id:`, color UUID fragments after wrapping.

Work done: removed standalone `-` list markers for structured object lists. This removes the dangling dashes in `volume list` `Attached to` output while keeping scalar list markers, such as `- sse`, for simple lists.

Work done: hostnames now use the same green style as IP addresses in pretty output.

Sources consulted:

* Local implementation file `internal/cli/output.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.

## 2026-05-04: Pretty Color Palette Decision

Decision: pretty output uses a role-based color palette so that resource types and operational states remain visually consistent across commands. The palette is implemented with named constants in `internal/cli/output.go`; tests assert representative domain values so future changes do not silently drift.

Current palette:

| Role | xterm color | Use |
| --- | --- | --- |
| Success status | `82` | Healthy, active, available, in-use, running, up. |
| Warning status | `220` | Build, create, delete, attach, detach, migrate, resize, restore, upload, and similar transitional states. |
| Error status | `203` | Error, failed, down, disabled, deleted, shutoff, suspended, and related error states. |
| Volume | `93` | Volume names and non-ID volume references. |
| Device | `81` | Device paths such as `/dev/vda`. |
| Image | `130` fallback, plus OS brand colors | Image names and non-ID image references. Recognized operating-system images use OS-specific brand colors. `N/A` remains the explicit no-image color. |
| Flavor | `223` | Flavor names and non-ID flavor references. |
| Timestamp | `117` | ISO-like created, updated, attached, heartbeat, and guaranteed-until timestamps. |
| Address | `114` | IP addresses and hostnames. |
| UUID | `75` | ID, UUID, and GUID fields across resources, including volume, server, subnet, image, flavor, and attachment IDs. UUID hyphens remain uncolored. |
| Label | `15` | `label:` prefixes inside structured pretty values. This is the only value-level style that keeps bold/bright emphasis. |

Implementation notes: ID-like fields take precedence over resource colors so IDs use the same UUID/GUID formatting across commands. Resource colors still apply to non-ID names and references. Status matching normalizes hyphens and spaces to underscores, so Cinder statuses such as `in-use` are colored through the same status buckets as compute statuses. Pretty value styles keep their colors but avoid bold SGR; inline `label:` prefixes remain bright white and bold so nested structured output stays scannable.

Follow-up: device paths now use the device color, and nested `id:` fields inside volume attachment output use the generic UUID/GUID style. The top-level `volume list` `ID` column uses the generic UUID/GUID style, and the top-level `volume list` `Name` column uses the same generic name color as `server list`. Full UUIDs and UUID fragments continue to color only the UUID/GUID segments, leaving hyphen separators uncolored.

Follow-up: image values now use OS-specific colors when their name or image-related text contains a supported operating-system marker. The supported palette is AlmaLinux `#0069DA`, Alpine Linux `#0D597F`, Arch Linux `#1793D1`, CentOS `#262577`, CentOS Stream `#A14F8C`, CirrOS `#ED1844`, Debian `#CE0056`, deepin `#007CFF`, elementary OS `#64BAFF`, EndeavourOS `#7F7FFF`, Fedora `#3C6EB4`, FreeBSD `#E31E26`, Gentoo `#54487A`, Kali Linux `#557C94`, Linux Mint `#86BE43`, Manjaro `#35BFA4`, NetBSD `#F26711`, NixOS `#5277C3`, OpenBSD `#F2CA30`, openSUSE `#73BA25`, Oracle Linux `#E32124`, Pop!_OS `#48B9C7`, Qubes OS `#3874D8`, Red Hat/RHEL `#EE0000`, Rocky Linux `#10B981`, Solus `#5294E2`, SUSE `#30BA78`, Tails `#56347C`, Ubuntu `#E95420`, Void Linux `#478061`, VyOS `#FFBF12`, Windows `#0078D7`, and Zorin OS `#15A6F0`. Unknown image names still use the generic image color `130`, and `N/A` image values keep the no-image color.

Decision update: the OS image matcher keeps more specific names before broader family names. This makes `CentOS Stream` render with the newer CentOS Stream color instead of the classic CentOS color, `Oracle Linux` render with the Oracle color instead of falling into an enterprise Linux family, and `CirrOS` render with OpenStack logo red because CirrOS is primarily used as an OpenStack test image. Oracle Linux intentionally uses the brighter Oracle logo red `#E32124` for visual separation from Red Hat `#EE0000`; current Oracle Redwood brand materials also publish Oracle Red `#C74634`, so this is a reviewed display choice rather than a claim that every Oracle asset uses `#E32124`.

Work done: added `make os-test`, backed by `tools/os-test`, to render the supported OS image color palette with the same Fancy table path used by CLI output. The command is meant as a quick visual consistency check for the supported image color rules.

Sources consulted:

* Local implementation file `internal/cli/output.go`.
* Local volume command implementation in `internal/cli/core_read.go`.
* Local pretty-output tests in `internal/cli/root_test.go`.
* Ubuntu brand color palette, which lists Ubuntu orange as `#E95420`: https://design.ubuntu.com/brand/colour-palette.
* Debian logo page, which lists current red equivalents `#CE0056` and `#CE0058`; this implementation uses `#CE0056`: https://www.debian.org/logos/.
* Rocky Linux official brand assets repository, plus the Simple Icons entry that records Rocky Linux `#10B981`: https://github.com/rocky-linux/brand-kit and https://simple-icons.github.io/simple-icons-website/.
* Red Hat brand standards, which list Red Hat red as `#ee0000`: https://www.redhat.com/en/about/brand/standards/color.
* Fedora Project logo usage guidelines, which list Fedora blue as `#3C6EB4`: https://fedoraproject.org/wiki/Logo/UsageGuidelines.
* CentOS archived brand logo page, which lists logo colors including dark blue `#262577`: https://wiki.centos.org/ArtWork%282f%29Brand%282f%29Logo.html.
* openSUSE artwork brand page, which lists openSUSE green as `#73ba25`: https://en.opensuse.org/openSUSE%3AArtwork_brand.
* SUSE/Rancher brand page, which lists SUSE brand colors including Jungle Green `#30BA78`: https://ranchercomprd.eks-prod.suse.com/brand-guidelines.
* Simple Icons color data, used for Alpine Linux `#0D597F` and Arch Linux `#1793D1` after checking project/logo sources: https://pub.dev/documentation/simple_icons/latest/simple_icons/SimpleIconColors-class.html.
* Microsoft Windows unattend `WindowColor` documentation, which lists the default Windows accent as `0xff0078d7`: https://learn.microsoft.com/en-us/windows-hardware/customize/desktop/unattend/microsoft-windows-shell-setup-themes-windowcolor.
* AlmaLinux current site icon and theme metadata, used for AlmaLinux blue `#0069DA`: https://almalinux.org/images/icon.svg and https://almalinux.org/.
* CentOS 2022 logo SVG, used for CentOS Stream purple `#A14F8C`; classic CentOS keeps the archived brand/logo dark blue `#262577`: https://commons.wikimedia.org/wiki/File:Centos-logo-2022.svg.
* Oracle logo SVG, used for Oracle Linux bright red `#E32124`, and Oracle Redwood brand guide, which records the newer Oracle Red `#C74634` considered during review: https://commons.wikimedia.org/wiki/File:Oracle_Logo.svg and https://www.oracle.com/a/ocom/docs/oracle-brand-guidelines.pdf.
* OpenStack 2016 logo SVG, used for CirrOS/OpenStack red `#ED1844`: https://commons.wikimedia.org/wiki/File:OpenStack%C2%AE_Logo_2016.svg.
* VyOS official logo SVG, whose yellow/orange gradient starts at `#FFBF12`; the palette uses that yellow endpoint for VyOS: https://vyos.io/wp-content/themes/vyos_theme/images/main/vyos-logo.svg.
* Linux Mint official brand logo repository, used for Linux Mint green `#86BE43`: https://github.com/linuxmint/brand-logo.
* System76 Pop!_OS brand repository, used for Pop!_OS teal `#48B9C7`: https://github.com/system76/brand.
* FreeBSD Foundation brand page, used for FreeBSD red `#E31E26`: https://freebsdfoundation.org/brand/.
* NetBSD official logo page and NetBSD text logo SVG, used for NetBSD orange `#F26711`: https://www.netbsd.org/gallery/logos.html and https://commons.wikimedia.org/wiki/File:NetBSD_textlogo.svg.
* OpenBSD textual logo SVG, used for OpenBSD yellow `#F2CA30`: https://commons.wikimedia.org/wiki/File:OpenBSD_textual_logo.svg.
* Simple Icons source-backed color data, used where official project pages or logo sources did not expose a simple brand palette for popular distro entries such as deepin, elementary OS, EndeavourOS, Gentoo, Kali Linux, Manjaro, NixOS, Qubes OS, Solus, Tails, Void Linux, and Zorin OS: https://cdn.jsdelivr.net/gh/simple-icons/simple-icons/data/simple-icons.json.

## 2026-05-04: Top-Level Makefile

Decision: the repository now has a top-level `Makefile` as the standard entry point for local build, test, smoke, and compatibility-generation commands. The targets keep the underlying Go commands visible and use workspace-local `GOCACHE` and `GOMODCACHE` defaults so constrained environments do not write to user-level Go cache directories unless the caller explicitly overrides them.

Work done: added `help`, `build`, `test`, `fmt`, `check`, `smoke`, `catalog`, `matrix`, `compat`, and `clean` targets. The `help` target is self-documenting and lists every public target with its description.

Sources consulted:

* Local README build instructions in `README.md`.
* Local project commands in `cmd/openstack` and `tools/`.

## 2026-05-04: Matrix Generator Rename

Decision: rename the compatibility matrix generator from `compat-matrix` to `matrix`. The shorter name matches the artifact name, keeps the Make target simple, and avoids implying that the test cloud configuration is secondary to command compatibility data.

Work done: moved the generator to `tools/matrix`, renamed the Make target to `matrix`, updated documentation references, and updated generated file headers to name `tools/matrix`. The generator writes `compat/matrix.yaml`, `compat/test-matrix.yaml`, and `compat/test-clouds.yaml` by default; callers can override those destinations with `--matrix`, `--test-matrix`, and `--test-clouds`.

Sources consulted:

* Local generator implementation in `tools/matrix/main.go`.
* Local Make targets in `Makefile`.
* Local compatibility artifacts under `compat/`.

## 2026-05-04: Matrix Terminal Summary

Decision: `tools/matrix` should print a compact summary to stdout after writing artifacts. The terminal output lists the total command count, status counts, and the generated file paths. It intentionally does not dump the full YAML matrix because the command catalog has hundreds of rows and the files remain the durable review artifacts.

Work done: added generation summary output to `tools/matrix`, covered the status counts and summary renderer with unit tests, and documented the behavior in the README. The generator accepts `--report-format terminal` and `--report-format readme`; `terminal` is the default, and `readme` emits a Markdown table and generated-file list suitable for README updates. The earlier `--summary-format` flag remains as an alias.

Sources consulted:

* Local generator implementation in `tools/matrix/main.go`.
* Local matrix tests in `tools/matrix/main_test.go`.

## 2026-05-04: Command Status Markdown Report

Decision: `tools/matrix` should be able to emit a Markdown command compatibility table for README or planning updates. The report compares every command from the pinned Python OSC 9.0.0 catalog to the current Go CLI matrix row and labels each command source as `built-in` or `plugin`.

Status mapping is intentionally conservative. Matrix rows marked `golden-matched` render as `compatible`; rows marked `cloud-verified` render as `partially compatible`; rows marked `implemented` render as `implemented`; and unfinished, stubbed, blocked, SDK-covered, or shim-needed rows render as `partially implemented` with a note that behavior is not complete. This avoids overstating Python compatibility before oracle parity is recorded.

Work done: added `--report command-status`, `--report-format terminal|readme`, and `--report-output <path>` to `tools/matrix`; added `make report`; and covered status mapping, Markdown table output, and table-cell escaping with unit tests.

Sources consulted:

* Local matrix generator in `tools/matrix/main.go`.
* Local matrix tests in `tools/matrix/main_test.go`.
* Local Make targets in `Makefile`.

## 2026-05-04: Make Target Short Names

Decision: use shorter Make target names for generated project artifacts and reports. `catalog` replaces `compat-catalog`, and `report` replaces `matrix-command-status`.

Work done: updated the Makefile targets, the `compat` aggregate target dependency, README examples, and the earlier diary entries that listed target names.

Sources consulted:

* Local Make targets in `Makefile`.
* Local README build instructions in `README.md`.

## 2026-05-04: Bubble Table Fancy Output Experiment

Decision: start an experiment branch, `experiment/bubble-table-fancy`, to try `github.com/evertras/bubble-table` for color-enabled Fancy table rendering. The upstream project describes itself as a customizable, interactive Bubble Tea table component, with support for headers, rows, borders, column widths, flexible widths, horizontal scrolling, and styles at table, column, row, and cell scope.

Work done: added `github.com/evertras/bubble-table v0.17.2` and routed only color-enabled `--pretty` table output through `bubble-table`. Non-color pretty output keeps the existing Bubbles table path so tests and piped output do not change while this visual experiment is evaluated. The adapter preserves existing width calculation, wrapping, row spacers, semantic colorization, and label coloring before handing rows to `bubble-table`.

Observation: `bubble-table` provides native rounded table borders, header separators, and internal column dividers. It uses the pre-v2 `github.com/charmbracelet/lipgloss` module, while the rest of the CLI uses `charm.land/lipgloss/v2`; the implementation keeps those style types isolated to avoid crossing incompatible style APIs.

Work done: refined the bubble-table experiment so color-enabled Fancy tables use symmetric one-space padding around headers and cells and use rounded-border horizontal rules between logical items. UUID-like values now prefer hyphen split points during Fancy wrapping, and wrapped inline `label: value` output carries the label context forward so UUID fragments and timestamps keep their semantic color on continuation lines.

Implementation note: the local `bubble-table` v0.17.2 source exposes row styling but not a per-row separator API. The CLI therefore marks separator rows before rendering, lets `bubble-table` render the table, and replaces only those marked body rows with generated rounded-border separators. The non-color pretty path and the OSC-compatible default table renderer are left unchanged.

Work done: corrected Fancy width fitting so the renderer counts bubble-table padding, left and right borders, and internal column dividers before assigning column widths. Fancy output now allows compact one-character columns when needed to stay inside narrow terminals, while preserving a 12-character minimum for ID-like columns when the terminal has enough room so UUIDs can wrap at hyphen boundaries without splitting the final UUID group.

Work done: adjusted wrapped image `N/A (...)` values so the `N/A` token keeps the explicit no-image color, but the explanatory continuation stays neutral gray instead of falling back to image brown after wrapping.

Work done: changed Fancy `show` output so the left `Field` column uses the same bright white label style as inline `label:` prefixes. Field names now visually match nested labels while values keep their semantic colors.

Work done: removed the color-enabled Fancy top spacer row so the first body row starts directly under the header separator. Inter-item horizontal rules remain in place between logical rows.

Sources consulted:

* Upstream `bubble-table` README at https://github.com/Evertras/bubble-table/tree/main.
* Upstream `bubble-table` module source in the local Go module cache.
* Local implementation file `internal/cli/output.go`.

## 2026-05-04: Static Oracle Compatibility Harness

Decision: add a dedicated `tools/compat-check` harness for no-cloud Python-vs-Go compatibility checks. This is separate from `tools/matrix` because it executes commands and compares stdout, stderr, and exit status, while the matrix generator only reports planned or recorded status.

Work done: added `tools/compat-check`, `make compat-static`, and `make compat-static-all`. The default target compares required static cases for completion, `command list`, and representative leaf help output against the pinned Python OSC oracle. Known gaps, including root help, invalid command errors, invalid flag errors, and `module list`, are reported without failing the target so the harness can be used immediately while still making those gaps visible.

Implementation note: the harness reads the oracle path from `compat/osc/9.0.0/metadata.json`, disables color and Go-only pretty defaults for both commands, and normalizes line endings before comparison. `--all-help` adds every cataloged leaf command's `--help` output to the comparison set.

Sources consulted:

* Local Python OSC oracle metadata in `compat/osc/9.0.0/metadata.json`.
* Local generated command catalog in `compat/osc/9.0.0/commands.json`.
* Local CLI entry point in `cmd/openstack/main.go`.

## 2026-05-04: Typed Lookup And Error Helpers

Decision: add typed compatibility errors before broadening write and delete behavior. Name-or-ID lookup, partial failures, and HTTP response failures need machine-checkable error types so commands and tests do not have to parse user-facing strings.

Work done: added typed lookup errors for not-found and ambiguous lookup results, a partial-failure error helper, and a first normalized OpenStack HTTP error formatter. The existing generic `singleMatch` helper now returns typed lookup errors while preserving its current user-facing text.

Sources consulted:

* Local lookup helpers in `internal/cli/identity_read.go`.
* Local Gophercloud v2.12.0 error types in `.cache/gomod/github.com/gophercloud/gophercloud/v2@v2.12.0/errors.go`.

## 2026-05-04: Live Cloud Discovery Tool

Decision: add live cloud discovery as a standalone tool that writes non-secret JSON artifacts under `compat/live-clouds/`. This keeps discovery separate from compatibility matrix generation because discovery reads live cloud state and must be refreshed before lifecycle tests.

Work done: added `tools/cloud-discovery` and `make discover-cloud CLOUD=name[,name]`. The first discovery pass records service catalog endpoints and safe fixture candidates for images, flavors, networks, and volume types. It marks one sorted candidate per fixture type as a default candidate for diagnostics only; lifecycle tests must still re-query immediately before use.

Implementation note: role discovery, API version detail, extension detail, and structured test eligibility remain open. The generated report includes a note that artifacts must not contain tokens, passwords, application credential secrets, `clouds.yaml` contents, or debug logs.

Sources consulted:

* Gophercloud cloud config parser in `.cache/gomod/github.com/gophercloud/gophercloud/v2@v2.12.0/openstack/config/clouds`.
* Gophercloud service catalog, images, flavors, networks, and volume type packages in the local module cache.
* Local cloud safety decisions in `docs/openstack-cli-compatibility-plan.md`.

## 2026-05-04: Lifecycle Test Scaffolding

Decision: add lifecycle safety helpers before adding more write tests. Every lifecycle test needs unique resource names, fixture recording, LIFO cleanup, and retained diagnostics before it is safe to broaden write coverage on shared clouds.

Work done: added an internal lifecycle helper that generates `golang-osc-test-*` IDs, builds resource names from that prefix, records fixture values, registers cleanup callbacks, executes cleanup in reverse creation order, captures cleanup errors, and writes JSON diagnostics.

Sources consulted:

* Local cloud safety decisions in `docs/openstack-cli-compatibility-plan.md`.
* Existing partial-failure helper in `internal/cli/compat_errors.go`.

## 2026-05-04: First Golden-Matched Rows

Decision: use the compatibility harness to record `golden-matched` status only after Python and Go output match for the same command form. Static local commands can use no-cloud comparisons; service commands need a live cloud comparison against the same cloud state.

Work done: extended `tools/compat-check` with `--live-cloud` and `--live-command` so selected commands run with the same `OS_CLOUD` for both Python OSC and the Go binary. `command list` is now `golden-matched` for static table and JSON checks. `flavor list`, `image list`, and `network list` are now `golden-matched` for default output against `cloud6`.

Sources consulted:

* Local compatibility harness in `tools/compat-check`.
* Pinned Python OSC oracle at `/Users/ken/.local/bin/openstack`.
* Live `cloud6` compatibility run using default table output.

## 2026-05-05: Parser Error Parity And Fixture-Aware Checks

Decision: move invalid command and invalid flag behavior from known gaps into required static compatibility checks once the Go CLI can match Python OSC stdout, stderr, and exit code. Keep root help as a known gap for live static comparison because repeated Python OSC `--help` runs showed auth plugin option groups can appear in different orders between invocations, while the Go CLI must emit a deterministic embedded snapshot.

Work done: root `--help` now uses the embedded OSC help snapshot, invalid flags render the captured argparse-style usage block plus `unrecognized arguments`, and invalid commands use Cliff-style fuzzy suggestions with exit code `2`. `tools/compat-check` now treats invalid command and invalid flag cases as required checks.

Work done: `tools/compat-check` now supports live fixture placeholders such as `{server}`, `{volume}`, `{network}`, `{project}`, and `{security_group}`. Placeholders are resolved through the Python oracle on the selected `--live-cloud`, and unavailable fixtures are reported as `SKIP` with a reason instead of being confused with compatibility failures.

Live observations on `cloud6`: `server show {server}`, `volume show {volume}`, and `network show {network}` resolved real fixture IDs. The comparison then exposed default-output mismatches in those show commands, so they remain unpromoted.

Sources consulted:

* Local Cliff `App.get_fuzzy_matches` and `CommandManager.find_command` source from `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/cliff`.
* Local Python OSC oracle at `/Users/ken/.local/bin/openstack`.
* Local compatibility harness in `tools/compat-check`.

## 2026-05-05: Discovery Details And Keypair Lifecycle Smoke

Decision: extend live cloud discovery before broadening lifecycle coverage. The report should remain non-secret and read-only, but should include enough detail to explain why a suite is runnable or skipped on each cloud.

Work done: `tools/cloud-discovery` now records Compute, Network, and Block Storage API version details, extension probes, role fixture visibility where Keystone permits it, and structured test eligibility with skip reasons. Discovery reports were refreshed for `cloud6`, `flex-sjc`, `flex-dfw`, and `flex-iad`.

Decision: use keypairs for the first wired lifecycle smoke. Keypairs are project-scoped, low-risk, cheap to create, easy to clean up by unique name, and do not require image, flavor, network, or volume fixtures.

Work done: added `tools/lifecycle-smoke` and `make lifecycle-smoke CLOUD=name`. The smoke test preflights keypair listing, creates a unique `golang-osc-test-*` keypair using a generated public-key fixture, shows it, deletes it, and retains JSON diagnostics under `compat/lifecycle-diagnostics/` only on failure unless `--keep-success` is used. `make lifecycle-smoke CLOUD=cloud6` passed.

Sources consulted:

* Gophercloud v2.12.0 local module packages for Compute, Network, Block Storage API versions, common extensions, and Identity roles.
* Local lifecycle safety decisions in `docs/openstack-cli-compatibility-plan.md`.

## 2026-05-05: Server Command Expansion

Decision: implement the Python OSC `server` namespace through Gophercloud typed helpers wherever available, and use narrow Nova or Neutron REST shims only for operations that the local Gophercloud v2.12.0 module does not expose as typed commands or where OSC needs cross-service behavior. Keep `server ssh` as a stub for now because Python OSC delegates to local SSH behavior and this project has a standing requirement that production CLI behavior be self-contained and not shell out to the operating system.

Work done: added handlers and command-local flags for `server create/delete/set/unset`, lifecycle actions including start, stop, pause, unpause, suspend, resume, shelve, unshelve, reboot, rebuild, rescue, unrescue, resize, migrate, restore, lock, unlock, evacuate, and dump create, plus server image and backup creation, server network/port/fixed-IP/floating-IP/security-group/volume attach and detach commands, migration show/abort/force-complete, resize and migration confirm/revert aliases, and server volume set/update. `--wait` paths now use the existing Fancy progress helper only when `--pretty` is active.

Compatibility note: output and lifecycle behavior are implemented but not yet golden-matched against live Python OSC for the write commands. The first implementation uses existing list/show renderers, name-or-ID lookup helpers, and service clients. It still needs disposable server lifecycle tests on each cloud after discovery picks current-safe image, flavor, network, keypair, and quota defaults immediately before the test run.

Sources consulted:

* Gophercloud package docs for [Compute servers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers), [Compute attach interfaces](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces), [Compute volume attachments](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach), and [Compute tags](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/tags).
* Gophercloud package docs for [Networking ports](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports) and [Networking floating IPs](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips).
* Local OSC oracle help snapshots under `compat/osc/9.0.0/help/server/`.
* Local keypair implementation in `internal/cli/core_read.go`.

## 2026-05-05: Pretty Wait Progress Animation

Decision: keep default `server create --wait` output compatible with Python OSC, but make Pretty wait output feel alive. Pretty server-status waits now keep a local progress floor that advances by 5% on each poll when Nova does not report useful progress, poll once per second, use action labels such as `Creating` and `Deleting` instead of a generic waiting label, cap visible progress below 100% until the target status is reached, update the existing terminal progress bar instead of printing a new one per poll, and render the final completed state at 100%. Progress labels are padded so action labels and `complete` do not shift the progress bar column.

Decision: use Harmonica directly for terminal Pretty progress animation rather than switching the CLI to a Bubble Tea application loop. Bubbles’ progress model documents that `ViewAs` is static and that animation uses model updates; this CLI path is a synchronous command renderer, so a small terminal-only Harmonica spring keeps the behavior localized. Non-TTY Pretty output remains a single deterministic progress line.

Sources consulted:

* Charmbracelet Harmonica docs: [github.com/charmbracelet/harmonica](https://github.com/charmbracelet/harmonica) and [pkg.go.dev/github.com/charmbracelet/harmonica](https://pkg.go.dev/github.com/charmbracelet/harmonica).
* Bubbles progress source and docs in the pinned local module, especially `SetPercent`, `Update`, and `ViewAs`: [pkg.go.dev/charm.land/bubbles/v2/progress](https://pkg.go.dev/charm.land/bubbles/v2/progress).

## 2026-05-05: Volume Command Implementation Coverage

Decision: implement the pinned `openstack.volume.v3` command surface by wiring every Volume v3 catalog command to either a typed Gophercloud package or a narrow Cinder REST shim through the authenticated Block Storage service client. This keeps production behavior self-contained in Go and preserves the existing rule that Python OSC is only an oracle, not a runtime dependency.

Work done: implemented the remaining Volume v3 command families: block storage cleanup, cluster set, log-level set, manageable volume and snapshot lists, consistency groups, consistency group snapshots, volume groups, volume group snapshots, volume group types, host freeze/thaw, message delete, volume migrate, volume revert, backend capability display, remote-source volume and snapshot management, attachment mutations, backup mutations, QoS mutations, transfer mutations, service set, volume type mutations, and volume/snapshot set/unset/delete/create paths. The command registry now reports the Python Volume v3 command list without `(Not Implemented Yet)` markers, and the matrix generator marks the implemented commands and Cinder shim coverage.

Implementation validation: `go test ./internal/cli`, `go test ./...`, `make build`, `make matrix`, `./bin/openstack command list -f json --group openstack.volume.v3`, `/Users/ken/.local/bin/openstack command list -f json --group openstack.volume.v3`, and `OS_CLOUD=cloud6 ./bin/openstack volume list -f json` passed. The command-list outputs for the Go CLI and Python oracle contained the same Volume v3 commands. This is not full completion: the Volume v3 command family still needs output parity testing, parity fixes, mocked Cinder endpoint tests for raw shims, and safe live lifecycle validation before it can be called finished or `done`.

Sources consulted:

* Gophercloud Block Storage package docs for [attachments](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments), [backups](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups), [QoS](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos), [snapshots](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots), [transfers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers), [volumes](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes), and [volume types](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes).
* Local Python OSC source under `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/volume/`.
* Local cinderclient source under `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/cinderclient/v3/`.
* Local OpenStackSDK source under `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstack/block_storage/v3/`.
* Pinned local help snapshots under `compat/osc/9.0.0/help/`.

## 2026-05-05: Live Output Parity Expansion

Decision: promote a command to `golden-matched` only when the live compatibility harness has compared the Go command against the pinned Python OSC oracle for the same cloud fixture, and the observed default table and JSON output match exactly. This is still narrower than full command completion because flags, write paths, alternate formats, and other clouds may remain untested.

Work done: fixed table/JSON split values for nested and multiline output where Python OSC renders compact table strings but structured JSON. The fixes covered server networks, enriched `server show` flavor/image/address/attachment fields, volume list attachments, image properties ordering, security group rule ordering, router state and gateway ordering, port fixed-IP ordering, empty port hints, and flavor `rxtx_factor` JSON formatting.

Validation: `go test ./internal/cli`, `make build`, and `go test ./...` passed. The static compatibility gate passed with the two existing known gaps: nondeterministic Python OSC root help auth-option ordering and intentionally different Go `module list` reporting. The live `cloud6` compatibility run passed for default table output and JSON output for `flavor list/show`, `hypervisor list`, `image list/show`, `keypair list/show`, `network list/show`, `port list/show`, `project list/show`, `router list/show`, `security group list/show`, `server list/show`, `subnet list/show`, and `volume list/show`.

Remaining work: many implemented commands are still only `implemented` or `cloud-verified`; they need the same oracle-backed output parity treatment before they can be called compatible or finished. This includes write commands, admin-only commands, less common read commands, command flags, alternate output formats beyond table and JSON, and remote-cloud breadth checks.

Sources consulted:

* Local compatibility harness in `tools/compat-check`.
* Pinned Python OSC oracle at `/Users/ken/.local/bin/openstack`.
* Local Go CLI binary at `./bin/openstack`.
* Live `cloud6` fixture data resolved by the Python oracle through `tools/compat-check`.

## 2026-05-05: Expanded Read Parity Fixtures

Decision: treat live fields that change between the Python oracle process and the Go process as comparison-harness volatility, not CLI output differences. The CLI output remains Python-shaped; `tools/compat-check` now normalizes known volatile live fields such as service update timestamps, service-list row order, `hypervisor show` uptime-derived fields, and `network agent show` heartbeat timestamps before comparing default and JSON output.

Work done: added live fixture placeholders for additional Compute, Network, and Volume read commands, including aggregates, hypervisors, IP availability, network agents, security group rules, server groups, subnet pools, volume attachments, backups, groups, group types, messages, QoS specs, snapshots, and volume types. `volume_message` fixture discovery now carries `OS_VOLUME_API_VERSION=3.3`, matching Python OSC's explicit microversion requirement for message commands.

Parity fixes: default empty tables now emit Python OSC's single newline, numeric `json.Number` list columns right-align in table output, network agent `Alive` and `State` values render as `:-)` and `UP`, IP availability subnet rows use Python's compact row-string format, `hypervisor show` follows Python OSC's field set, `hypervisor stats show` emits Python's deprecation warning, `volume type show` includes `access_project_ids` and Python-style properties, `volume attachment show` renders blank detached timestamps and Python-style connection properties, and `volume message show` renders links as Python-style dictionaries while preserving structured JSON.

Validation: the expanded live `cloud6` default-output suite passed with 36 live passes, 0 required failures, 5 fixture skips, and the two existing known static gaps. The matching JSON suite also passed with 36 live passes, 0 required failures, 5 fixture skips, and the same known static gaps. Skipped commands were `aggregate show`, `server group show`, `volume snapshot show`, `subnet pool show`, and `volume qos show` because the Python oracle found no fixture rows on `cloud6` during the run. `make matrix` regenerated `compat/matrix.yaml`, `compat/test-matrix.yaml`, and `compat/test-clouds.yaml`, raising `golden-matched` rows to 50.

Sources consulted:

* Local Python OSC oracle at `/Users/ken/.local/bin/openstack`.
* Local compatibility harness in `tools/compat-check`.
* Local matrix generator in `tools/matrix`.
* Live `cloud6` data from the configured `clouds.yaml`.

## 2026-05-06: Volume Lifecycle and Read Parity Validation

Decision: treat the Volume command family as implemented but split validation status by safety and available cloud fixtures. Commands that mutate only test-created Cinder resources can be exercised by the routine lifecycle suite. Commands that change real service state, backend state, logging levels, migrations, failover state, existing messages, or backend-local unmanaged storage remain implemented but intentionally unpromoted until there is a scoped fixture or explicit approval for that run.

Work done: added a `--suite` selector to `tools/lifecycle-smoke`, added `make volume-lifecycle`, and implemented a Volume v3 lifecycle suite. The suite preflights Cinder service, backend pool, resource filter, manageable resource, backend capability, and volume type state. It creates unique `golang-osc-test-*` resources, records selected fixtures, waits for asynchronous status changes, cleans up in reverse order, and records structured diagnostics under `compat/lifecycle-diagnostics/`. That diagnostics directory is now ignored so cloud-specific run details are not committed accidentally.

Lifecycle coverage now includes normal `volume create/delete/set/unset`, `volume snapshot create/delete/set/unset`, `volume revert`, `volume transfer request create/delete/accept`, `volume qos create/delete/set/unset`, `volume group type create/delete/set`, `volume group create/delete/set`, `consistency group create/delete/set`, and the associated list/show paths. The suite also records deliberate skips for attachment mutations until a disposable server fixture exists, backup mutations while `cinder-backup` is down on `cloud6`, volume type mutations while both Python OSC and the Go CLI hit a Cinder HTTP 500 deleting a test-created type because `__DEFAULT__` is missing, group snapshot creation while `cloud6` rejects the empty source group, manage-existing remote-source calls without backend-local unmanaged storage fixtures, and real service/backend/admin mutations that are not scoped to test-created resources.

Parity fixes: Cinder group updates now accept empty Cinder responses instead of failing with EOF. Resource filter list/show keeps API order for JSON but sorts default-table output to match Python OSC. Manageable volume and snapshot list output now uses Python-style repr strings in table output while preserving JSON objects. Backend capability show now follows Python-compatible property ordering for common capability keys. Volume summary renders empty metadata as blank in the default table while preserving `{}` in JSON. The live compatibility harness also now supplies the Cinder microversions Python OSC requires for cluster, manageable resource, cleanup, log-level, resource filter, message, and group commands.

Validation: `go test ./tools/lifecycle-smoke`, `go test ./internal/cli ./tools/compat-check ./tools/lifecycle-smoke`, `make build`, and `make check` passed. `env OS_CLOUD=cloud6 make volume-lifecycle CLOUD=cloud6` passed, and a retained successful run wrote `compat/lifecycle-diagnostics/golang-osc-test-01f14f2262fd8ad6.json`. The expanded live Python-vs-Go read parity suite passed on `cloud6` for default table and JSON output for `block storage resource filter list`, `block storage resource filter show volume`, `block storage volume manageable list dell6.crandall.haus@lvm-1`, `block storage snapshot manageable list dell6.crandall.haus@lvm-1`, `volume backend capability show dell6.crandall.haus@lvm-1`, `block storage log level list`, and `volume summary`; the run reported 24 passes, 0 required failures, and the two existing static known gaps for root help ordering and Go module reporting.

Sources consulted:

* Local Python OSC oracle at `/Users/ken/.local/bin/openstack`.
* Local Go CLI binary at `./bin/openstack`.
* Local lifecycle suite in `tools/lifecycle-smoke`.
* Local compatibility harness in `tools/compat-check`.
* Live `cloud6` data from the configured `clouds.yaml`.

## 2026-05-06: Server, Volume Attachment, Quota Lifecycle Closure

Decision: keep lifecycle suites bounded by the test harness when a CLI wait path can otherwise spend a long time waiting for a cloud state that may never arrive on the current cloud. The production CLI wait behavior remains compatible, but the lifecycle suite should turn cloud-specific no-progress cases into structured skips with cleanup instead of tying up a run for the full internal wait window.

Work done: added `server` and `quota` suites to `tools/lifecycle-smoke`, added `make server-lifecycle` and `make quota-lifecycle`, and extended the existing Volume suite with a disposable server fixture for Cinder attachment mutation tests. The server suite dynamically discovers image, flavor, network, alternate network, alternate flavor, and volume type fixtures; creates only `golang-osc-test-*` resources; validates server create/delete, server group create/delete/show, event list/show, set/unset, lock/unlock, reboot, stop/start, rebuild, pause/unpause, suspend/resume, rescue/unrescue, volume attach/detach/list/set/update, security-group add/remove, port add/remove, and network add/remove where fixtures are available; and records skips for unsafe or cloud-blocked actions.

Quota work: the quota suite creates a dedicated Keystone project, mutates only that project's Compute, Volume, and Network quotas, verifies aggregate `quota show -f json` against the Python oracle before and after reset, resets each service, deletes the test project, and records `quota set --class` and `--default` as intentionally skipped because those paths change default quota-class state rather than only a test-owned project. The aggregate quota renderer now mirrors Python OSC's merged service row order, including Neutron replacing Nova's placeholder `networks` row and Cinder volume-type quotas following live `volume type list` order.

Pretty output update: status colorization now includes Compute lifecycle states such as `SHELVING_OFFLOADING`, `UNSHELVING`, `SHELVED`, and `SHELVED_OFFLOADED`. Transitional states use the warning color; inactive or intervention states such as `SHELVED_OFFLOADED`, `SHELVED`, `PAUSED`, and `RESCUE` use the error color.

Validation: `env OS_CLOUD=cloud6 make server-lifecycle CLOUD=cloud6`, `env OS_CLOUD=cloud6 make volume-lifecycle CLOUD=cloud6`, and `env OS_CLOUD=cloud6 make quota-lifecycle CLOUD=cloud6` passed on the current tree. `go test ./internal/cli`, `go test ./tools/lifecycle-smoke`, `go test ./tools/matrix`, `make build`, and `make matrix` passed. The matrix now reports 594 commands with 57 `golden-matched`, 239 `cloud-verified`, 105 `implemented`, 192 `unknown`, and 1 `blocked` row.

Sources consulted:

* Local Python OSC oracle at `/Users/ken/.local/bin/openstack`.
* Local Go CLI binary at `./bin/openstack`.
* Local lifecycle suite in `tools/lifecycle-smoke`.
* Local matrix generator in `tools/matrix`.
* Local Python OSC quota source at `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/common/quota.py`.
* Live `cloud6` data from the configured `clouds.yaml`.

## 2026-05-06: Pretty Server Network Labels

Decision: keep default `server list` and `server show` address output tied to Nova/Python OSC compatibility, but let Pretty enrich network labels through Neutron subnet data when it is available. This handles clouds where Nova's server address map groups multiple fixed IPs under one label even though the IPs belong to different Neutron networks.

Work done: Pretty server network output now builds a subnet CIDR to network-name map from Neutron and relabels each address by the subnet containing that IP. If Neutron is unavailable, the subnet cannot be parsed, or no subnet matches, Pretty falls back to the original Nova label. Default table and JSON output remain unchanged.

Validation: `go test ./internal/cli`, `go test ./...`, `make build`, `OS_CLOUD=cloud6 ./bin/openstack --pretty server list`, and `OS_CLOUD=cloud6 ./bin/openstack server list -f json` passed. The live Pretty output relabeled `172.16.86.110` and `172.16.86.177` as `os6-lan` while leaving `172.17.36.42` and `172.17.36.118` as `testNet`; JSON still matched the Nova/Python OSC label shape.

Sources consulted:

* Local Python OSC oracle at `/Users/ken/.local/bin/openstack`.
* Local Go CLI binary at `./bin/openstack`.
* Live `cloud6` Neutron `network list`, `subnet list`, and `port list --fixed-ip` data from the configured `clouds.yaml`.

## 2026-05-06: Write Output Parity Lifecycle Pass

Decision: mutating command output parity needs paired disposable resources when the Python oracle and Go CLI cannot safely run the same mutation against the same resource. The lifecycle runner now compares the Go command against the Python command on parallel test-created resources, normalizing only values proven to be cloud-generated or resource-specific, such as UUIDs, timestamps, Nova instance names, reservation IDs, generated admin passwords, IP addresses, Cinder attachment IDs, target IQNs, and Cinder connector credentials.

Work done: added paired Python-vs-Go write-output comparisons to the server lifecycle for safe server create/delete, set/unset, lock/unlock, reboot, stop/start, rebuild, pause/unpause, suspend/resume, rescue/unrescue, server volume add/remove/set/update, security-group add/remove, port add/remove, and network add/remove paths. Added paired Volume attachment output comparisons for `volume attachment create`, `volume attachment set`, `volume attachment complete`, and `volume attachment delete`. The quota lifecycle now compares `quota set` and `quota delete` output for the dedicated test project while keeping the existing aggregate `quota show -f json` oracle checks before and after reset.

Parity fixes found by the new checks: `server create --wait -f json` and `server rebuild --wait -f json` now render from the raw server detail path used by `server show`, with `server create` preserving `adminPass`. Create/rebuild output applies Python OSC's create-style `null` handling for empty `kernel_id`, `ramdisk_id`, access IPs, `config_drive`, `launch_index`, `locked`, `progress`, and empty `server_groups`, without changing `server show` output. Default table `server show` now renders empty `server_groups` as `[]`, matching Python OSC. `server reboot --wait` now prints `Complete`; `server delete --wait` remains silent, matching the oracle behavior observed on `cloud6`.

Validation: `make server-lifecycle CLOUD=cloud6`, `make volume-lifecycle CLOUD=cloud6`, and `make quota-lifecycle CLOUD=cloud6` passed after the parity fixes. `go test ./internal/cli`, `go test ./tools/lifecycle-smoke`, `go test ./tools/matrix`, and `make matrix` passed. `make matrix` now reports 89 `golden-matched`, 207 `cloud-verified`, 105 `implemented`, 192 `unknown`, and 1 `blocked` row.

Sources consulted:

* Local Python OSC oracle at `/Users/ken/.local/bin/openstack`.
* Local Go CLI binary at `./bin/openstack`.
* Local lifecycle suite in `tools/lifecycle-smoke`.
* Local matrix generator in `tools/matrix`.
* Live `cloud6` data from the configured `clouds.yaml`.

## 2026-05-06: Image, Network, And Object Write Parity

Decision: extend the same paired-resource parity model to Image, Network, and Object Store. The runner compares the Go command against the Python oracle on parallel test-created resources and normalizes only values observed to be generated by the cloud or resource pairing: UUIDs, hex IDs, timestamps, IP addresses, MAC addresses, Swift transaction IDs, Neutron revision numbers, provider segmentation IDs, and standard attribute IDs. JSON canonicalization is limited to paired comparisons where those replacements are active.

Work done: added `image`, `network`, and `object` suites to `tools/lifecycle-smoke`, with `make image-lifecycle`, `make network-lifecycle`, and `make object-lifecycle` targets. The image suite covers direct image create/show/list/save/set/unset/delete, staged image import where supported, shared image membership add/remove through a temporary project, and Glance metadef namespace/object/property/resource-type association mutation paths. The network suite covers disposable networks, subnets, ports, routers, router interface add/remove, router extra-route add/remove, security groups, security group rules, address groups, and address scopes, and records structured skips for extensions that `cloud6` does not expose. The object suite covers disposable Swift containers and objects, including `object save --file -`.

Parity fixes found by the new checks: Image properties now preserve Python OSC's observed JSON property order for hidden/hash fields, `image metadef object update` sends the current object name when no rename was requested, and shared image membership tests create shared images. Router show output now preserves Python-observed Neutron extension fields such as `gw_port_id`, `ha_vr_id`, `enable_default_route_bfd`, and `enable_default_route_ecmp` when present. The network lifecycle harness now confirms resources are already gone when Neutron deletes router interface ports but the Go delete command only reports a generic aggregate delete failure. Object Store account discovery accepts Swift account `HEAD` responses with either HTTP 200 or HTTP 204, matching the response observed on `flex-dfw`.

Validation: `make image-lifecycle CLOUD=cloud6`, `make network-lifecycle CLOUD=cloud6`, and `make object-lifecycle CLOUD=flex-dfw` passed. `go test ./tools/lifecycle-smoke`, `go test ./tools/matrix`, `go test ./internal/cli`, and `make matrix` passed during the implementation loop. `make matrix` now reports 594 commands with 152 `golden-matched`, 146 `cloud-verified`, 103 `implemented`, 192 `unknown`, and 1 `blocked` row.

Sources consulted:

* Local Python OSC oracle at `/Users/ken/.local/bin/openstack`.
* Local Go CLI binary at `./bin/openstack`.
* Local lifecycle suite in `tools/lifecycle-smoke`.
* Local matrix generator in `tools/matrix`.
* Live `cloud6` and `flex-dfw` data from the configured `clouds.yaml`.

## 2026-05-06: Pretty Command List Grouping

Decision: keep default `openstack command list` output compatible with Python OSC, but make Pretty mode group related command families into a more scannable table. Pretty now renders `Command Group`, `Command`, and `Subcommands`; command paths are grouped by their first word, so families such as `server`, `network`, `volume`, `image`, and `router` occupy one logical row with multiline subcommands.

Work done: changed only the Pretty renderer path for `command list`. The default table, JSON, and value formats still use the OSC-compatible command list rows. Added unit coverage for grouping behavior and updated the Pretty renderer smoke test.

Validation: `go test ./internal/cli`, `go test ./...`, `make build`, `git diff --check`, `./bin/openstack --pretty command list --group openstack.compute.v2`, `./bin/openstack command list --group openstack.compute.v2`, and `./bin/openstack command list --group openstack.compute.v2 -f json` passed.

Sources consulted:

* Local Go CLI binary at `./bin/openstack`.
* Local command list renderer in `internal/cli/command_list.go`.

## 2026-05-06: Pretty Compact Mode

Decision: add `--compact`, `--no-compact`, and `OS_COMPACT=1` as Go-only Pretty controls. The compact options are parsed globally but intentionally ignored by non-Pretty output formats so the Python-compatible default table, JSON, value, CSV, YAML, and shell output surfaces do not change. `--no-compact` is an explicit user override for environments where `OS_COMPACT=1` is set but the user wants expanded Pretty output for one command.

Work done: added `Options.Compact`, `Options.NoCompact`, the global `--compact` and `--no-compact` flags, and `OS_COMPACT` default handling. Pretty table rendering now uses an explicit row-separation mode; normal Pretty keeps row separators, while compact Pretty removes the extra separator rows entirely. Non-TTY Pretty also drops spacer rows in compact mode. During command pre-run, an explicit `--no-compact` clears compact mode after environment defaults are loaded.

Validation: `go test ./internal/cli` passed after adding tests for `OS_COMPACT`, `--no-compact` overriding `OS_COMPACT=1`, default-output no-op behavior, and TTY row-separator removal.

Sources consulted:

* Local Pretty renderer in `internal/cli/output.go`.
* Local parser setup in `internal/cli/root.go`.

## 2026-05-06: Pretty Server Network IP Indentation

Decision: Pretty server network addresses should keep the network label on the first IP for a network and indent additional IPs by four spaces. Default server address output remains Python-compatible and unchanged. This decision was superseded later on 2026-05-06 by dot-marked server network IP output.

Work done: updated the shared Pretty server network formatter used by `server list` and `server show`. A multi-IP network now renders as `network: first-ip` followed by `    next-ip` lines. Single-IP networks still render on one labeled line.

Validation: `go test ./internal/cli` passed with focused assertions for the four-space indentation and for unchanged single-IP network labels.

Sources consulted:

* Local Pretty server address formatter in `internal/cli/core_read.go`.

## 2026-05-06: Pretty Server Network IP Dot Marker

Decision: Pretty server network addresses should mark the first rendered IP line with a dot marker and align later IP labels under that marker. The marker is shown even when there is only one IP address for consistency. Pretty resource reference columns should use UUID coloring when the value is UUID-like, including undashed OpenStack project and user IDs and reference columns such as `Port`, `Router`, `Network`, `Floating Network`, `Server`, `Image`, `Flavor`, and `Volume`. Default server address output remains Python-compatible and unchanged.

Work done: replaced the four-space continuation-only layout with `dot network: ip` for the first rendered IP and `  network: ip` for subsequent IPs. The shared Pretty formatter is used by both `server list` and `server show`, and the Pretty label parser now treats the dot as a prefix so labels and IP values keep their existing color rules. The Pretty semantic colorizer now treats UUID-like resource reference values as IDs without coloring non-ID names as UUIDs, including short fragments produced when a narrow table wraps a resource ID.

Validation: focused unit tests cover the dot marker, aligned later IP labels, relabeled subnet output, unchanged default address output, and UUID-style coloring for project, user, port, router, and floating-network reference columns.

Sources consulted:

* Local Pretty server address formatter in `internal/cli/core_read.go`.
* Local Pretty label parser in `internal/cli/output.go`.

## 2026-05-07: Lifecycle Make Target Consolidation

Decision: use one public Make target for lifecycle validation: `make lifecycle CLOUD=name SUITE=suite`. The supported suites are `keypair`, `server`, `volume`, `quota`, `image`, `network`, `object`, and `all`. Remove the individual service-specific `*-lifecycle` Make targets so users do not have to discover multiple target names for the same runner.

Work done: replaced `lifecycle-smoke` and the individual `server-lifecycle`, `volume-lifecycle`, `quota-lifecycle`, `image-lifecycle`, `network-lifecycle`, and `object-lifecycle` Make targets with a single `lifecycle` target. `make help` now prints the lifecycle suite list and short suite descriptions. The lifecycle runner now accepts `--suite all` and runs every suite in sequence, continuing through all suites and returning a failure exit code if any suite fails.

Validation: focused unit tests cover the lifecycle suite help text and unknown-suite rejection.

Sources consulted:

* Local Make targets in `Makefile`.
* Local lifecycle runner in `tools/lifecycle-smoke/main.go`.

## 2026-05-07: Discovery Make Target And Terminal Summary

Decision: rename the cloud discovery Make target from `discover-cloud` to `discover` and have the discovery command print a terminal summary table while still writing the non-secret JSON reports under `compat/live-clouds/`.

Work done: updated the Makefile and README to use `make discover CLOUD=name[,name]`. `tools/cloud-discovery` now accumulates report results and prints one row per cloud with status, region, service count, API probe status counts, fixture status summary, lifecycle eligibility counts, and the JSON report path.

Validation: focused unit tests cover the terminal table summary and status-count ordering.

Sources consulted:

* Local Make targets in `Makefile`.
* Local discovery runner in `tools/cloud-discovery/main.go`.

## 2026-05-07: Pretty OS Color Contrast

Decision: keep OS image colors in the Fancy/Pretty renderer brand-recognizable, but treat readability as the stronger constraint for terminal text. `make os-test` now reports the measured contrast ratio for each supported OS color against the dark Fancy terminal baseline `#282C34`, and unit tests require every suggested color to meet at least `4.5:1`. CentOS, CentOS Stream, and CentOS Core share the readable CentOS green `#9CCD2A`. Gentoo uses the documented Gentoo grey `#DDDAEC` instead of the primary purple, Tails uses a lighter accessible purple tint because the primary purple was too dark on this background, and Talos Linux uses the source-backed Simple Icons orange `#FF7300`.

Work done: updated the OS image palette to use readable colors, added Flatcar Container Linux, Talos Linux, CentOS Core, and CoreOS/Fedora CoreOS match entries, made `make os-test` include a `Contrast` column, and added tests for match specificity and dark-background contrast.

Validation: focused unit tests cover the contrast threshold, `os-test` contrast reporting, and the new OS match ordering. Full validation results are recorded with the commit.

Sources consulted:

* W3C WCAG 2.2 SC 1.4.3 contrast guidance, which specifies `4.5:1` for normal text: https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html.
* Archived CentOS logo color page, which lists CentOS green `#9CCD2A`, orange `#EFA724`, purple `#a14F8C`, and dark blue `#262577`: https://wiki.centos.org/ArtWork%282f%29Brand%282f%29Logo.html.
* Gentoo artwork color page, which lists Gentoo purple variants and light grey `#DDDAEC`: https://wiki.gentoo.org/wiki/Project:Artwork/Colors.
* Flatcar Container Linux project page, used to confirm the distro name and supported image label: https://www.flatcar.org/.
* Talos Linux project page, used to confirm the distro name and supported image label: https://www.talos.dev/.
* Wikimedia CoreOS logo records, used as the available public reference for legacy CoreOS and Fedora CoreOS logo lineage: https://commons.wikimedia.org/wiki/File:CoreOS.svg and https://commons.wikimedia.org/wiki/File:Fedora_CoreOS_logo.svg.
* Simple Icons source-backed color data remains the fallback source for distro logo colors when an official page does not expose a terminal-usable palette. It currently records Talos as `#FF7300` from the Sidero Labs Talos logo SVG, Tails as `#56347C` from the Tails logo page, and Gentoo as `#54487A` from Gentoo artwork: https://cdn.jsdelivr.net/npm/simple-icons@latest/_data/simple-icons.json.

## 2026-05-07: Pretty OS Brand-First Palette Adjustment

Decision update: the previous `4.5:1` contrast threshold was too strict for this terminal-only Fancy palette because it pushed several readable, recognizable distro colors away from their brand. The OS image palette is now brand-first when the source color remains legible in the Fancy table. `make os-test` continues to report contrast ratios, and unit tests keep a low dark-background legibility guard to catch truly unreadable entries without forcing WCAG AA-style substitutions.

Work done: restored AlmaLinux blue `#0069DA`, Debian red `#CE0056`, and Red Hat/RHEL red `#EE0000`; kept SUSE green `#30BA78` and openSUSE green `#73BA25`; changed Oracle Linux to the red used in the Oracle Linux SVG and current Oracle logo SVG, `#C74634`; changed CentOS, CentOS Stream, and CentOS Core to CentOS-site purple `#A14F8C`; changed Talos Linux to Talos-site red-orange `#E8312C`; and changed elementary OS to the blue from the site's purchase button gradient, `#4CA7E4`.

Validation: passed focused OS-color renderer tests, `make os-test`, `go test ./...`, `make build`, and `git diff --check`.

Sources consulted:

* AlmaLinux current site icon and theme metadata, used for AlmaLinux blue `#0069DA`: https://almalinux.org/images/icon.svg and https://almalinux.org/.
* Debian logo page, which lists current red equivalents `#CE0056` and `#CE0058`; this implementation uses `#CE0056`: https://www.debian.org/logos/.
* Red Hat brand standards, which list Red Hat red as `#ee0000`: https://www.redhat.com/en/about/brand/standards/color.
* SUSE/Rancher brand page, which lists SUSE brand colors including Jungle Green `#30BA78`: https://ranchercomprd.eks-prod.suse.com/brand-guidelines.
* openSUSE artwork brand page, which lists openSUSE green as `#73ba25`: https://en.opensuse.org/openSUSE%3AArtwork_brand.
* Oracle Linux SVG and current Oracle logo SVG, which use `#c74634`/`#C74634`: https://commons.wikimedia.org/wiki/File:Oracle_linux_logo.svg and https://commons.wikimedia.org/wiki/File:Oracle_logo.svg.
* CentOS Stream project page and stylesheet, whose current page styling uses purple `#A14F8C`: https://www.centos.org/centos-stream/ and https://www.centos.org/assets/css/base/stylesheet.min.css.
* Talos Linux project page, whose site styling includes red-orange `#E8312C`: https://www.talos.dev/.
* elementary OS site and stylesheet, whose purchase button gradient includes blue `#4CA7E4`: https://elementary.io/ and https://elementary.io/styles/main-0d002cb065.css.

## 2026-05-07: Pretty OS Palette Data File

Decision: OS image color rules should be data-driven at build time. The editable source is `internal/cli/pretty_os_colors.json`, and the compiled binary embeds that file through Go's `embed` package. JSON was selected because Go parses it with the standard library, so this does not add another dependency. Runtime palette overrides remain deferred because the request was specifically for user edits before compilation.

Work done: moved OS image display names, hex colors, sample image labels, match keywords, source URLs, contrast background, and minimum legibility guard into `internal/cli/pretty_os_colors.json`. The Go loader validates the schema version, strict JSON fields, hex color syntax, duplicate names, required samples, and required match keywords before exposing the palette to the Pretty renderer. CirrOS now uses the OpenStack logo red `#ED1844` again.

Validation: passed focused Pretty OS color tests, `make os-test`, `go test ./...`, `make build`, and `git diff --check`.

Sources consulted:

* Go `embed` package documentation, used for compile-time embedding: https://pkg.go.dev/embed.
* Go `encoding/json` package documentation, used for the standard-library JSON parser: https://pkg.go.dev/encoding/json.
* OpenStack 2016 logo SVG, used for CirrOS/OpenStack red `#ED1844`: https://commons.wikimedia.org/wiki/File:OpenStack%C2%AE_Logo_2016.svg.
