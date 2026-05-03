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

## 2026-05-02: Common Read Command Expansion

Work done: added Common command implementations for `availability zone list`, `extension list`, `extension show`, and `limits show`. Availability-zone output now combines Nova, Cinder, and Neutron rows to match the default Python OSC behavior observed on `cloud6`. Limits output requires `--absolute` or `--rate`, matching Python OSC, and normalizes absolute-limit names to the OSC JSON names observed from `openstack limits show --absolute -f json`.

Compatibility note: Nova and Cinder availability zones, extensions, and limits use Gophercloud packages directly. Neutron availability zones are implemented as a narrow raw REST read through the authenticated Gophercloud Network v2 service client because no dedicated Neutron availability-zone package was found in the local Gophercloud v2.12.0 module inventory. That shim is intentionally recorded in the matrix and should move behind the service-scoped extras boundary when that plugin layer is ready. `quota list/show` and `versions show` remain unimplemented because they need broader service aggregation, project/domain policy handling, and oracle tests before they should replace their stubs.

Sources consulted: Gophercloud package docs and local module sources for [Compute availability zones](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/availabilityzones), [Block Storage availability zones](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/availabilityzones), [common extensions](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/common/extensions), [Compute limits](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/limits), and [Block Storage limits](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits).

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, `go run ./tools/compat-matrix`, and live `cloud6` JSON smoke checks passed for `availability zone list`, `availability zone list --long`, `extension list --network`, `extension show router`, `limits show --absolute`, and `limits show --rate`.

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

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and `go run ./tools/compat-matrix` passed. Live `cloud6` checks matched Python OSC for `volume attachment list -f json`, `volume attachment show <existing-attachment> -f json`, `volume qos list --print-empty`, and `volume summary -f json`. A temporary QoS spec named `golang-osc-test-64fb7c64-bee7-47ed-8dba-65b585469eb5` was created with Python OSC to provide a show fixture; Python and Go matched for `volume qos show -f json` and `volume qos list -f json`, and the temporary QoS spec was deleted successfully.

## 2026-05-03: Compute Events, Attachments, And Usage Read Expansion

Work done: added Compute v2 read implementations for `server event list`, `server event show`, `server volume list`, `usage list`, and `usage show`. Server events use Gophercloud's instance actions package, server volume attachment reads use Gophercloud's Nova volume attachments package with an extended extraction struct for the `attachment_id` and `bdm_uuid` fields observed from Python OSC, and usage reads use Gophercloud's tenant usage package.

Compatibility note: Python OSC/openstacksdk requested Compute microversion 2.89 for `server volume list` on `cloud6` so that `Attachment ID` and `BlockDeviceMapping UUID` appear instead of the older `ID` column. The Go command now discovers supported Compute microversions and uses the service maximum when no explicit `OS_COMPUTE_API_VERSION` is set for commands that need a higher minimum. If the user sets `OS_COMPUTE_API_VERSION`, the command honors it and returns a compatibility error when it is too low for the requested behavior. `usage show` also preserves Python's JSON detail shape for usage server rows, including the duplicate `name` key emitted by the local OSC oracle.

Sources consulted:

* [Gophercloud Compute instance actions](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/instanceactions), for server event list and show.
* [Gophercloud Compute volume attachments](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach), for Nova server volume attachment reads.
* [Gophercloud Compute usage](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/usage), for tenant usage list and show.
* [Gophercloud OpenStack utils](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/utils), for supported Compute microversion discovery.
* Local Python OSC source files `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/server_event.py`, `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/server_volume.py`, and `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/usage.py`, used only as the pinned local oracle implementation source.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and `go run ./tools/compat-matrix` passed with workspace-local Go caches. Live `cloud6` JSON checks matched Python OSC for `server event list`, `server event list --long`, `server event show`, `server volume list`, `usage list --start 2026-05-01 --end 2026-05-03`, and `usage show --start 2026-05-01 --end 2026-05-02`; `usage show` has dynamic `uptime` values, so only stable structure and non-time-varying fields should be used for golden tests.

## 2026-05-03: Compute Console Read Expansion

Work done: added `console log show` and `console url show`. Console log reads use Gophercloud's server console-output action and write raw log text, matching Python OSC's command behavior instead of routing through the structured formatter. Console URL reads use Gophercloud's remote console package and preserve the Python-observed `protocol`, `type`, and `url` field order for show output.

Compatibility note: `console url show` uses Compute microversion discovery with a minimum of 2.6 when no explicit `OS_COMPUTE_API_VERSION` is set. The `console connection show` command remains a stub because it needs a separate Nova console-token lookup path that is not exposed by the local Gophercloud `remoteconsoles` package.

Sources consulted:

* [Gophercloud Compute servers](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers), for console-output actions.
* [Gophercloud Compute remote consoles](https://pkg.go.dev/github.com/gophercloud/gophercloud/v2/openstack/compute/v2/remoteconsoles), for remote console URL creation.
* Local Python OSC source file `/Users/ken/.local/share/uv/tools/python-openstackclient/lib/python3.12/site-packages/openstackclient/compute/v2/console.py`, used only as the pinned local oracle implementation source.

Verification: `go test ./...`, `go build -o bin/openstack ./cmd/openstack`, and `go run ./tools/compat-matrix` passed with workspace-local Go caches. Live `cloud6` checks matched Python OSC for `console log show --lines 5 rocky` and the stable JSON fields of `console url show rocky -f json`; the URL token itself is expected to differ between calls.

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
