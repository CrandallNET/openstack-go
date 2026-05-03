package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	bsattachments "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups"
	bsqos "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/qos"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/schedulerstats"
	bsservices "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/services"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/transfers"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/instanceactions"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/remoteconsoles"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	computeservices "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/services"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/usage"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/members"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/tasks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/agents"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/addressscopes"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networkipavailabilities"
	qospolicies "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies"
	qosruletypes "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/ruletypes"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/addressgroups"
	secgroups "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	secgrouprules "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/segments"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	osutils "github.com/gophercloud/gophercloud/v2/openstack/utils"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"
)

func runCoreRead(path string, stdout io.Writer, opts *Options) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		clients, err := newOpenStackClients(cmd.Context(), opts)
		if err != nil {
			return err
		}

		switch path {
		case "block storage cluster list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageClusterList(cmd.Context(), stdout, opts, client)
		case "block storage cluster show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageClusterShow(cmd.Context(), stdout, opts, client, args)
		case "block storage log level list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageLogLevelList(cmd.Context(), stdout, opts, client)
		case "block storage resource filter list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageResourceFilterList(cmd.Context(), stdout, opts, client)
		case "block storage resource filter show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageResourceFilterShow(cmd.Context(), stdout, opts, client, args)
		case "address group list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressGroupList(cmd.Context(), stdout, opts, client)
		case "address group show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressGroupShow(cmd.Context(), stdout, opts, client, args)
		case "address scope list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressScopeList(cmd.Context(), stdout, opts, client)
		case "address scope show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressScopeShow(cmd.Context(), stdout, opts, client, args)
		case "aggregate list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return aggregateList(cmd.Context(), stdout, opts, client)
		case "aggregate show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return aggregateShow(cmd.Context(), stdout, opts, client, args)
		case "availability zone list":
			return availabilityZoneList(cmd.Context(), stdout, opts, clients)
		case "cached image list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return cachedImageList(cmd.Context(), stdout, opts, client)
		case "compute agent list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return computeAgentList(cmd.Context(), stdout, opts, client)
		case "compute service list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return computeServiceList(cmd.Context(), stdout, opts, client)
		case "console connection show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return consoleConnectionShow(cmd.Context(), stdout, opts, client, args)
		case "console log show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return consoleLogShow(cmd.Context(), stdout, opts, client, args)
		case "console url show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return consoleURLShow(cmd.Context(), stdout, opts, client, args)
		case "extension list":
			return extensionList(cmd.Context(), stdout, opts, clients)
		case "extension show":
			return extensionShow(cmd.Context(), stdout, opts, clients, args)
		case "flavor list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return computeFlavorList(cmd.Context(), stdout, opts, client)
		case "flavor show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return computeFlavorShow(cmd.Context(), stdout, opts, client, args)
		case "allocation candidate list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return allocationCandidateList(cmd.Context(), stdout, opts, client)
		case "container list":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return containerList(cmd.Context(), stdout, opts, client)
		case "container show":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return containerShow(cmd.Context(), stdout, opts, client, args)
		case "floating ip list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPList(cmd.Context(), stdout, opts, client)
		case "floating ip pool list":
			return floatingIPPoolList()
		case "floating ip show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPShow(cmd.Context(), stdout, opts, client, args)
		case "host list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "API has been deprecated; consider using 'hypervisor list' instead.")
			return hostList(cmd.Context(), stdout, opts, client)
		case "host show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "API has been deprecated; consider using 'hypervisor show' instead.")
			return hostShow(cmd.Context(), stdout, opts, client, args)
		case "ip availability list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return ipAvailabilityList(cmd.Context(), stdout, opts, clients, client)
		case "ip availability show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return ipAvailabilityShow(cmd.Context(), stdout, opts, client, args)
		case "image list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageList(cmd.Context(), stdout, opts, client)
		case "image member get":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMemberGet(cmd.Context(), stdout, opts, client, args)
		case "image member list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMemberList(cmd.Context(), stdout, opts, client, args)
		case "image metadef namespace list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefNamespaceList(cmd.Context(), stdout, opts, client)
		case "image metadef namespace show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefNamespaceShow(cmd.Context(), stdout, opts, client, args)
		case "image show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageShow(cmd.Context(), stdout, opts, client, args)
		case "image stores list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageStoresList(cmd.Context(), stdout, opts, client)
		case "image task list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageTaskList(cmd.Context(), stdout, opts, client)
		case "image task show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageTaskShow(cmd.Context(), stdout, opts, client, args)
		case "network list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkList(cmd.Context(), stdout, opts, client)
		case "network agent list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkAgentList(cmd.Context(), stdout, opts, client)
		case "network agent show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkAgentShow(cmd.Context(), stdout, opts, client, args)
		case "network service provider list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkServiceProviderList(cmd.Context(), stdout, opts, client)
		case "network qos policy list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSPolicyList(cmd.Context(), stdout, opts, client)
		case "network qos policy show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSPolicyShow(cmd.Context(), stdout, opts, client, args)
		case "network qos rule type list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSRuleTypeList(cmd.Context(), stdout, opts, client)
		case "network qos rule type show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSRuleTypeShow(cmd.Context(), stdout, opts, client, args)
		case "network rbac list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkRBACList(cmd.Context(), stdout, opts, client)
		case "network rbac show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkRBACShow(cmd.Context(), stdout, opts, client, args)
		case "network segment list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkSegmentList(cmd.Context(), stdout, opts, client)
		case "network segment show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkSegmentShow(cmd.Context(), stdout, opts, client, args)
		case "network show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkShow(cmd.Context(), stdout, opts, client, args)
		case "network trunk list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkTrunkList(cmd.Context(), stdout, opts, client)
		case "network trunk show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkTrunkShow(cmd.Context(), stdout, opts, client, args)
		case "object list":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectList(cmd.Context(), stdout, opts, client, args)
		case "object show":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectShow(cmd.Context(), stdout, opts, client, args)
		case "object store account show":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectStoreAccountShow(cmd.Context(), stdout, opts, client)
		case "quota list":
			return quotaList(cmd.Context(), stdout, opts, clients)
		case "quota show":
			return quotaShow(cmd.Context(), stdout, opts, clients, args)
		case "keypair list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return keypairList(cmd.Context(), stdout, opts, client)
		case "keypair show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return keypairShow(cmd.Context(), stdout, opts, client, args)
		case "hypervisor list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return hypervisorList(cmd.Context(), stdout, opts, client)
		case "hypervisor show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return hypervisorShow(cmd.Context(), stdout, opts, client, args)
		case "hypervisor stats show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return hypervisorStatsShow(cmd.Context(), stdout, opts, client)
		case "limits show":
			return limitsShow(cmd.Context(), stdout, opts, clients)
		case "port list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			computeClient, _ := clients.computeV2()
			return portList(cmd.Context(), stdout, opts, client, computeClient)
		case "port show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return portShow(cmd.Context(), stdout, opts, client, args)
		case "resource class list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceClassList(cmd.Context(), stdout, opts, client)
		case "resource class show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceClassShow(cmd.Context(), stdout, opts, client, args)
		case "resource provider aggregate list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderAggregateList(cmd.Context(), stdout, opts, client, args)
		case "resource provider allocation show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderAllocationShow(cmd.Context(), stdout, opts, client, args)
		case "resource provider inventory list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderInventoryList(cmd.Context(), stdout, opts, client, args)
		case "resource provider inventory show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderInventoryShow(cmd.Context(), stdout, opts, client, args)
		case "resource provider list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderList(cmd.Context(), stdout, opts, client)
		case "resource provider show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderShow(cmd.Context(), stdout, opts, client, args)
		case "resource provider trait list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderTraitList(cmd.Context(), stdout, opts, client, args)
		case "resource provider usage show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return resourceProviderUsageShow(cmd.Context(), stdout, opts, client, args)
		case "router list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerList(cmd.Context(), stdout, opts, client)
		case "router show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerShow(cmd.Context(), stdout, opts, client, args)
		case "security group list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupList(cmd.Context(), stdout, opts, client)
		case "security group show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupShow(cmd.Context(), stdout, opts, client, args)
		case "security group rule list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupRuleList(cmd.Context(), stdout, opts, client, args)
		case "security group rule show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupRuleShow(cmd.Context(), stdout, opts, client, args)
		case "server list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			imageClient, _ := clients.imageV2()
			return computeServerList(cmd.Context(), stdout, opts, client, imageClient)
		case "server show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return computeServerShow(cmd.Context(), stdout, opts, client, args)
		case "server event list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverEventList(cmd.Context(), stdout, opts, client, args)
		case "server event show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverEventShow(cmd.Context(), stdout, opts, client, args)
		case "server group list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverGroupList(cmd.Context(), stdout, opts, client)
		case "server group show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverGroupShow(cmd.Context(), stdout, opts, client, args)
		case "server migration list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMigrationList(cmd.Context(), stdout, opts, clients, client)
		case "server volume list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverVolumeList(cmd.Context(), stdout, opts, client, args)
		case "subnet list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetList(cmd.Context(), stdout, opts, client)
		case "subnet pool list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetPoolList(cmd.Context(), stdout, opts, client)
		case "subnet pool show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetPoolShow(cmd.Context(), stdout, opts, client, args)
		case "subnet show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetShow(cmd.Context(), stdout, opts, client, args)
		case "trait list":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return traitList(cmd.Context(), stdout, opts, client)
		case "trait show":
			client, err := clients.placementV1()
			if err != nil {
				return err
			}
			return traitShow(cmd.Context(), stdout, opts, client, args)
		case "volume list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeList(cmd.Context(), stdout, opts, client)
		case "volume group list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupList(cmd.Context(), stdout, opts, client)
		case "volume group show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupShow(cmd.Context(), stdout, opts, client, args)
		case "volume group snapshot list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupSnapshotList(cmd.Context(), stdout, opts, client)
		case "volume group snapshot show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupSnapshotShow(cmd.Context(), stdout, opts, client, args)
		case "volume group type list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupTypeList(cmd.Context(), stdout, opts, client)
		case "volume group type show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupTypeShow(cmd.Context(), stdout, opts, client, args)
		case "volume message list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeMessageList(cmd.Context(), stdout, opts, clients, client)
		case "volume message show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeMessageShow(cmd.Context(), stdout, opts, client, args)
		case "volume show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeShow(cmd.Context(), stdout, opts, client, args)
		case "volume backend pool list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackendPoolList(cmd.Context(), stdout, opts, client)
		case "volume attachment list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeAttachmentList(cmd.Context(), stdout, opts, client)
		case "volume attachment show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeAttachmentShow(cmd.Context(), stdout, opts, client, args)
		case "volume backup list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupList(cmd.Context(), stdout, opts, client)
		case "volume backup show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupShow(cmd.Context(), stdout, opts, client, args)
		case "volume service list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeServiceList(cmd.Context(), stdout, opts, client)
		case "volume snapshot list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotList(cmd.Context(), stdout, opts, client)
		case "volume snapshot show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotShow(cmd.Context(), stdout, opts, client, args)
		case "volume qos list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSList(cmd.Context(), stdout, opts, client)
		case "volume qos show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSShow(cmd.Context(), stdout, opts, client, args)
		case "volume summary":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSummary(cmd.Context(), stdout, opts, client)
		case "volume type list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTypeList(cmd.Context(), stdout, opts, client)
		case "volume type show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTypeShow(cmd.Context(), stdout, opts, client, args)
		case "volume transfer request list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTransferList(cmd.Context(), stdout, opts, client)
		case "volume transfer request show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTransferShow(cmd.Context(), stdout, opts, client, args)
		case "versions show":
			return versionsShow(cmd.Context(), stdout, opts, clients)
		case "usage list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return usageList(cmd.Context(), stdout, opts, clients, client)
		case "usage show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return usageShow(cmd.Context(), stdout, opts, clients, client)
		default:
			return fmt.Errorf("core read command %q is not wired", path)
		}
	}
}

func computeServerList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, imageClient *gophercloud.ServiceClient) error {
	page, err := servers.List(client, servers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := extractServers(page)
	if err != nil {
		return err
	}
	flavorNames := flavorNameMap(ctx, client)
	imageNames := imageNameMap(ctx, imageClient)
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":       item.ID,
			"Name":     item.Name,
			"Status":   item.Status,
			"Networks": serverNetworks(item.Addresses),
			"Image":    serverImage(item.Image, imageNames),
			"Flavor":   serverFlavor(item.Flavor, flavorNames),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Status", "Networks", "Image", "Flavor"}, rows)
}

func computeServerShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server show requires <server>")
	}
	item, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"status", item.Status},
		{"project_id", item.TenantID},
		{"user_id", item.UserID},
		{"created", item.Created},
		{"updated", item.Updated},
		{"image", serverImage(item.Image, nil)},
		{"flavor", serverFlavor(item.Flavor, nil)},
		{"addresses", serverNetworks(item.Addresses)},
		{"metadata", item.Metadata},
		{"key_name", nilIfEmpty(item.KeyName)},
	})
}

func aggregateList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := aggregates.List(client).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := aggregates.ExtractAggregates(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":                item.ID,
			"Name":              item.Name,
			"Availability Zone": item.AvailabilityZone,
		}
		if boolFlag(opts, "long") {
			row["Hosts"] = item.Hosts
			row["Properties"] = item.Metadata
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Availability Zone"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Hosts", "Properties")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func aggregateShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("aggregate show requires <aggregate>")
	}
	item, err := findAggregate(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"availability_zone", item.AvailabilityZone},
		{"hosts", item.Hosts},
		{"metadata", item.Metadata},
		{"uuid", item.UUID},
		{"created_at", oscTime(item.CreatedAt)},
		{"updated_at", oscTime(item.UpdatedAt)},
		{"deleted", item.Deleted},
	})
}

func computeServiceList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := computeservices.List(client, computeservices.ListOpts{
		Binary: flagValue(opts, "service"),
		Host:   flagValue(opts, "host"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := computeservices.ExtractServices(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":         item.ID,
			"Binary":     item.Binary,
			"Host":       item.Host,
			"Zone":       item.Zone,
			"Status":     item.Status,
			"State":      item.State,
			"Updated At": oscTime(item.UpdatedAt),
		}
		if boolFlag(opts, "long") {
			row["Disabled Reason"] = nilIfEmpty(item.DisabledReason)
			row["Forced Down"] = item.ForcedDown
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Binary", "Host", "Zone", "Status", "State", "Updated At"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Disabled Reason", "Forced Down")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func computeAgentList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client.Microversion = "2.1"
	requestURL := client.ServiceURL("os-agents")
	if hypervisor := flagValue(opts, "hypervisor"); hypervisor != "" {
		query := url.Values{}
		query.Set("hypervisor", hypervisor)
		requestURL += "?" + query.Encode()
	}
	var body struct {
		Agents []map[string]any `json:"agents"`
	}
	resp, err := client.Get(ctx, requestURL, &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	rows := make([]outputRow, 0, len(body.Agents))
	for _, item := range body.Agents {
		rows = append(rows, outputRow{
			"Agent ID":     mapValueOrEmpty(item, "agent_id"),
			"Hypervisor":   mapValueOrEmpty(item, "hypervisor"),
			"OS":           mapValueOrEmpty(item, "os"),
			"Architecture": mapValueOrEmpty(item, "architecture"),
			"Version":      mapValueOrEmpty(item, "version"),
			"Md5Hash":      mapValueOrEmpty(item, "md5hash"),
			"URL":          mapValueOrEmpty(item, "url"),
		})
	}
	return renderListOutput(stdout, opts, []string{"Agent ID", "Hypervisor", "OS", "Architecture", "Version", "Md5Hash", "URL"}, rows)
}

func consoleConnectionShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("console connection show requires <token>")
	}
	client, err := computeClientWithMaximumMicroversion(ctx, client, "2.99")
	if err != nil {
		return err
	}
	var body struct {
		Console map[string]any `json:"console"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("os-console-auth-tokens", args[0]), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscResourceNotFoundError(err, "ConsoleAuthToken", args[0])
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"host", mapValueOrEmpty(body.Console, "host")},
		{"instance_uuid", mapValueOrEmpty(body.Console, "instance_uuid")},
		{"internal_access_path", mapValueOrEmpty(body.Console, "internal_access_path")},
		{"port", mapValueOrEmpty(body.Console, "port")},
		{"tls_port", mapValueOrEmpty(body.Console, "tls_port")},
	})
}

func hostList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client.Microversion = "2.1"
	var body struct {
		Hosts []map[string]any `json:"hosts"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("os-hosts"), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	zone := flagValue(opts, "zone")
	rows := make([]outputRow, 0, len(body.Hosts))
	for _, item := range body.Hosts {
		if zone != "" && mapValueString(item, "zone") != zone {
			continue
		}
		rows = append(rows, outputRow{
			"Host Name": mapValueOrEmpty(item, "host_name"),
			"Service":   mapValueOrEmpty(item, "service"),
			"Zone":      mapValueOrEmpty(item, "zone"),
		})
	}
	return renderListOutput(stdout, opts, []string{"Host Name", "Service", "Zone"}, rows)
}

func hostShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("host show requires <host>")
	}
	client.Microversion = "2.1"
	var body struct {
		Host []struct {
			Resource map[string]any `json:"resource"`
		} `json:"host"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("os-hosts", args[0]), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	rows := make([]outputRow, 0, len(body.Host))
	for _, item := range body.Host {
		resource := item.Resource
		rows = append(rows, outputRow{
			"Host":      mapValueOrEmpty(resource, "host"),
			"Project":   mapValueOrEmpty(resource, "project"),
			"CPU":       mapValueOrEmpty(resource, "cpu"),
			"Memory MB": mapValueOrEmpty(resource, "memory_mb"),
			"Disk GB":   mapValueOrEmpty(resource, "disk_gb"),
		})
	}
	return renderListOutput(stdout, opts, []string{"Host", "Project", "CPU", "Memory MB", "Disk GB"}, rows)
}

func consoleLogShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("console log show requires <server>")
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	output, err := servers.ShowConsoleOutput(ctx, client, server.ID, servers.ShowConsoleOutputOpts{
		Length: intFlag(opts, "lines"),
	}).Extract()
	if err != nil {
		return err
	}
	if output == "" {
		return nil
	}
	if _, err := fmt.Fprint(stdout, output); err != nil {
		return err
	}
	if !strings.HasSuffix(output, "\n") {
		_, err = fmt.Fprintln(stdout)
		return err
	}
	return nil
}

func consoleURLShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("console url show requires <server>")
	}
	client, err := computeClientWithMinimumMicroversion(ctx, client, "2.6")
	if err != nil {
		return err
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	console, err := remoteconsoles.Create(ctx, client, server.ID, consoleURLOpts(opts)).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"protocol", console.Protocol},
		{"type", console.Type},
		{"url", console.URL},
	})
}

func serverEventList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server event list requires <server>")
	}
	minimum := "2.21"
	if flagValue(opts, "changes-since") != "" || flagValue(opts, "limit") != "" || flagValue(opts, "marker") != "" {
		minimum = "2.58"
	}
	if flagValue(opts, "changes-before") != "" {
		minimum = "2.66"
	}
	client, err := computeClientWithMinimumMicroversion(ctx, client, minimum)
	if err != nil {
		return err
	}
	serverID, err := serverIDForEventLookup(ctx, client, args[0])
	if err != nil {
		return err
	}
	listOpts := instanceactions.ListOpts{
		Limit:  intFlag(opts, "limit"),
		Marker: flagValue(opts, "marker"),
	}
	if value := flagValue(opts, "changes-since"); value != "" {
		parsed, err := parseRFC3339Flag("changes-since", value)
		if err != nil {
			return err
		}
		listOpts.ChangesSince = &parsed
	}
	if value := flagValue(opts, "changes-before"); value != "" {
		parsed, err := parseRFC3339Flag("changes-before", value)
		if err != nil {
			return err
		}
		listOpts.ChangesBefore = &parsed
	}
	page, err := instanceactions.List(client, serverID, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := instanceactions.ExtractInstanceActions(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"Request ID": item.RequestID,
			"Server ID":  item.InstanceUUID,
			"Action":     item.Action,
			"Start Time": oscTime(item.StartTime),
		}
		if boolFlag(opts, "long") {
			row["Message"] = nilIfEmpty(item.Message)
			row["Project ID"] = item.ProjectID
			row["User ID"] = item.UserID
		}
		rows = append(rows, row)
	}
	columns := []string{"Request ID", "Server ID", "Action", "Start Time"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Message", "Project ID", "User ID")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func serverEventShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server event show requires <server> <request-id>")
	}
	client, err := computeClientWithMinimumMicroversion(ctx, client, "2.50")
	if err != nil {
		return err
	}
	serverID, err := serverIDForEventLookup(ctx, client, args[0])
	if err != nil {
		return err
	}
	item, err := instanceactions.Get(ctx, client, serverID, args[1]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"action", item.Action},
		{"events", serverEventDetails(item.Events)},
		{"id", item.RequestID},
		{"message", nilIfEmpty(item.Message)},
		{"project_id", item.ProjectID},
		{"request_id", item.RequestID},
		{"start_time", oscTime(item.StartTime)},
		{"user_id", item.UserID},
	})
}

func serverMigrationList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) error {
	client, err := computeClientWithMaximumMicroversion(ctx, client, "2.80")
	if err != nil {
		return err
	}
	query := url.Values{}
	if value := flagValue(opts, "host"); value != "" {
		query.Set("host", value)
	}
	if value := flagValue(opts, "status"); value != "" {
		query.Set("status", value)
	}
	if value := flagValue(opts, "server"); value != "" {
		server, err := findServer(ctx, client, value)
		if err != nil {
			return err
		}
		query.Set("instance_uuid", server.ID)
	}
	if value := flagValue(opts, "type"); value != "" {
		if value == "cold-migration" {
			value = "migration"
		}
		query.Set("migration_type", value)
	}
	if value := flagValue(opts, "marker"); value != "" {
		if !microversionAtLeast(client.Microversion, "2.59") {
			return fmt.Errorf("--os-compute-api-version 2.59 or greater is required to support the --marker option")
		}
		query.Set("marker", value)
	}
	if value := intFlag(opts, "limit"); value > 0 {
		if !microversionAtLeast(client.Microversion, "2.59") {
			return fmt.Errorf("--os-compute-api-version 2.59 or greater is required to support the --limit option")
		}
		query.Set("limit", strconv.Itoa(value))
	}
	if value := flagValue(opts, "changes-since"); value != "" {
		if !microversionAtLeast(client.Microversion, "2.59") {
			return fmt.Errorf("--os-compute-api-version 2.59 or greater is required to support the --changes-since option")
		}
		query.Set("changes-since", value)
	}
	if value := flagValue(opts, "changes-before"); value != "" {
		if !microversionAtLeast(client.Microversion, "2.66") {
			return fmt.Errorf("--os-compute-api-version 2.66 or greater is required to support the --changes-before option")
		}
		query.Set("changes-before", value)
	}
	if value := flagValue(opts, "project"); value != "" {
		if !microversionAtLeast(client.Microversion, "2.80") {
			return fmt.Errorf("--os-compute-api-version 2.80 or greater is required to support the --project option")
		}
		identityClient, err := clients.identityV3()
		if err != nil {
			return err
		}
		project, err := findProjectWithDomain(ctx, identityClient, value, flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		query.Set("project_id", project.ID)
	}
	if value := flagValue(opts, "user"); value != "" {
		if !microversionAtLeast(client.Microversion, "2.80") {
			return fmt.Errorf("--os-compute-api-version 2.80 or greater is required to support the --user option")
		}
		identityClient, err := clients.identityV3()
		if err != nil {
			return err
		}
		user, err := findUserWithDomain(ctx, identityClient, value, flagValue(opts, "user-domain"))
		if err != nil {
			return err
		}
		query.Set("user_id", user.ID)
	}
	requestURL := client.ServiceURL("os-migrations")
	if encoded := query.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	var body struct {
		Migrations []map[string]any `json:"migrations"`
	}
	resp, err := client.Get(ctx, requestURL, &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	columns, keys := serverMigrationListColumns(client.Microversion, opts)
	rows := make([]outputRow, 0, len(body.Migrations))
	for _, item := range body.Migrations {
		row := outputRow{}
		for i, column := range columns {
			row[column] = mapValueOrEmpty(item, keys[i])
		}
		rows = append(rows, row)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func serverVolumeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server volume list requires <server>")
	}
	client, err := computeClientWithMinimumMicroversion(ctx, client, "2.89")
	if err != nil {
		return err
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	page, err := volumeattach.List(client, server.ID).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := extractComputeServerVolumeAttachments(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Device":                  item.Device,
			"Server ID":               item.ServerID,
			"Volume ID":               item.VolumeID,
			"Tag":                     stringPtrValue(item.Tag),
			"Delete On Termination?":  boolPtrValue(item.DeleteOnTermination),
			"Attachment ID":           nilIfEmpty(item.AttachmentID),
			"BlockDeviceMapping UUID": nilIfEmpty(item.BlockDeviceMappingUUID),
		})
	}
	return renderListOutput(stdout, opts, []string{"Device", "Server ID", "Volume ID", "Tag", "Delete On Termination?", "Attachment ID", "BlockDeviceMapping UUID"}, rows)
}

func usageList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) error {
	start, end, err := usageDateRange(opts)
	if err != nil {
		return err
	}
	page, err := usage.AllTenants(client, usage.AllTenantsOpts{
		Detailed: true,
		Start:    &start,
		End:      &end,
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := usage.ExtractAllTenants(page)
	if err != nil {
		return err
	}
	projectNames := map[string]string{}
	if outputWantsHumanProjectNames(opts) {
		projectNames = projectNameMap(ctx, clients)
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Project":       usageProjectValue(item.TenantID, projectNames),
			"Servers":       nil,
			"RAM MB-Hours":  usageFloat(item.TotalMemoryMBUsage),
			"CPU Hours":     usageFloat(item.TotalVCPUsUsage),
			"Disk GB-Hours": usageFloat(item.TotalLocalGBUsage),
		})
	}
	if opts.Format == "" || opts.Format == "table" {
		if len(rows) > 0 {
			if _, err := fmt.Fprintf(stdout, "Usage from %s to %s: \n", start.Format("2006-01-02"), end.Format("2006-01-02")); err != nil {
				return err
			}
		}
	}
	return renderListOutput(stdout, opts, []string{"Project", "Servers", "RAM MB-Hours", "CPU Hours", "Disk GB-Hours"}, rows)
}

func usageShow(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) error {
	start, end, err := usageDateRange(opts)
	if err != nil {
		return err
	}
	projectID, err := usageProjectID(ctx, opts, clients)
	if err != nil {
		return err
	}
	var item *usage.TenantUsage
	err = usage.SingleTenant(client, projectID, usage.SingleTenantOpts{
		Start: &start,
		End:   &end,
	}).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		extracted, err := usage.ExtractSingleTenant(page)
		if err != nil {
			return false, err
		}
		item = extracted
		return false, nil
	})
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("usage not found for project %s", projectID)
	}
	if opts.Format == "" || opts.Format == "table" {
		if _, err := fmt.Fprintf(stdout, "Usage from %s to %s on project %s: \n", start.Format("2006-01-02"), end.Format("2006-01-02"), projectID); err != nil {
			return err
		}
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"Project", item.TenantID},
		{"Servers", usageServerOutputs(item.ServerUsages)},
		{"RAM MB-Hours", usageFloat(item.TotalMemoryMBUsage)},
		{"CPU Hours", usageFloat(item.TotalVCPUsUsage)},
		{"Disk GB-Hours", usageFloat(item.TotalLocalGBUsage)},
	})
}

func hypervisorList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := hypervisors.ListOpts{}
	if matching := flagValue(opts, "matching"); matching != "" {
		listOpts.HypervisorHostnamePattern = &matching
	}
	if boolFlag(opts, "with-servers") {
		withServers := true
		listOpts.WithServers = &withServers
	}
	page, err := hypervisors.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := hypervisors.ExtractHypervisors(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":                  item.ID,
			"Hypervisor Hostname": item.HypervisorHostname,
			"Hypervisor Type":     item.HypervisorType,
			"Host IP":             item.HostIP,
			"State":               item.State,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Hypervisor Hostname", "Hypervisor Type", "Host IP", "State"}, rows)
}

func hypervisorShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("hypervisor show requires <hypervisor>")
	}
	item, err := findHypervisor(ctx, client, args[0])
	if err != nil {
		return err
	}
	uptime, _ := hypervisors.GetUptime(ctx, client, item.ID).Extract()
	fields := []outputField{
		{"id", item.ID},
		{"hypervisor_hostname", item.HypervisorHostname},
		{"hypervisor_type", item.HypervisorType},
		{"hypervisor_version", item.HypervisorVersion},
		{"host_ip", item.HostIP},
		{"state", item.State},
		{"status", item.Status},
		{"service_host", item.Service.Host},
		{"service_id", item.Service.ID},
		{"vcpus", item.VCPUs},
		{"vcpus_used", item.VCPUsUsed},
		{"memory_mb", item.MemoryMB},
		{"memory_mb_used", item.MemoryMBUsed},
		{"local_gb", item.LocalGB},
		{"local_gb_used", item.LocalGBUsed},
		{"free_disk_gb", item.FreeDiskGB},
		{"free_ram_mb", item.FreeRamMB},
		{"running_vms", item.RunningVMs},
		{"current_workload", item.CurrentWorkload},
		{"cpu_info", item.CPUInfo},
	}
	if uptime != nil {
		fields = append(fields, outputField{"uptime", uptime.Uptime})
	}
	return renderShowOutput(stdout, opts, fields)
}

func hypervisorStatsShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	stats, err := hypervisors.GetStatistics(ctx, client).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"count", stats.Count},
		{"current_workload", stats.CurrentWorkload},
		{"disk_available_least", stats.DiskAvailableLeast},
		{"free_disk_gb", stats.FreeDiskGB},
		{"free_ram_mb", stats.FreeRamMB},
		{"local_gb", stats.LocalGB},
		{"local_gb_used", stats.LocalGBUsed},
		{"memory_mb", stats.MemoryMB},
		{"memory_mb_used", stats.MemoryMBUsed},
		{"running_vms", stats.RunningVMs},
		{"vcpus", stats.VCPUs},
		{"vcpus_used", stats.VCPUsUsed},
	})
}

func computeFlavorList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := flavors.ListDetail(client, nil).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := flavors.ExtractFlavors(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":        item.ID,
			"Name":      item.Name,
			"RAM":       item.RAM,
			"Disk":      item.Disk,
			"Ephemeral": item.Ephemeral,
			"VCPUs":     item.VCPUs,
			"Is Public": item.IsPublic,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "RAM", "Disk", "Ephemeral", "VCPUs", "Is Public"}, rows)
}

func computeFlavorShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("flavor show requires <flavor>")
	}
	item, err := findFlavor(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"ram", item.RAM},
		{"disk", item.Disk},
		{"ephemeral", item.Ephemeral},
		{"vcpus", item.VCPUs},
		{"is_public", item.IsPublic},
		{"swap", item.Swap},
		{"rxtx_factor", item.RxTxFactor},
		{"properties", item.ExtraSpecs},
	})
}

func cachedImageList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client.Microversion = imageMicroversionAtMost(client.Microversion, "2.14")
	var body struct {
		CachedImages []map[string]any `json:"cached_images"`
		QueuedImages []string         `json:"queued_images"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("cache"), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscResourceNotFoundError(err, "Cache", "None")
	}
	rows := make([]outputRow, 0, len(body.CachedImages)+len(body.QueuedImages))
	for _, item := range body.CachedImages {
		rows = append(rows, outputRow{
			"ID":                  mapValueOrEmpty(item, "image_id"),
			"State":               "cached",
			"Last Accessed (UTC)": unixSecondsISO(mapValueOrEmpty(item, "last_accessed")),
			"Last Modified (UTC)": unixSecondsISO(mapValueOrEmpty(item, "last_modified")),
			"Size":                mapValueOrEmpty(item, "size"),
			"Hits":                mapValueOrEmpty(item, "hits"),
		})
	}
	for _, imageID := range body.QueuedImages {
		rows = append(rows, outputRow{
			"ID":                  imageID,
			"State":               "queued",
			"Last Accessed (UTC)": "N/A",
			"Last Modified (UTC)": "N/A",
			"Size":                "N/A",
			"Hits":                "N/A",
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "State", "Last Accessed (UTC)", "Last Modified (UTC)", "Size", "Hits"}, rows)
}

func imageList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := images.List(client, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := images.ExtractImages(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"ID": item.ID, "Name": item.Name, "Status": string(item.Status)})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Status"}, rows)
}

func imageShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image show requires <image>")
	}
	item, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"status", string(item.Status)},
		{"visibility", string(item.Visibility)},
		{"protected", item.Protected},
		{"container_format", item.ContainerFormat},
		{"disk_format", item.DiskFormat},
		{"min_disk", item.MinDiskGigabytes},
		{"min_ram", item.MinRAMMegabytes},
		{"owner", item.Owner},
		{"size", item.SizeBytes},
		{"checksum", item.Checksum},
		{"tags", item.Tags},
		{"created_at", oscTime(item.CreatedAt)},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func imageMetadefNamespaceList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	requestURL := client.ServiceURL("metadefs", "namespaces")
	query := url.Values{}
	if value := flagValue(opts, "resource-types"); value != "" {
		query.Set("resource_types", value)
	}
	if value := flagValue(opts, "visibility"); value != "" {
		query.Set("visibility", value)
	}
	if encoded := query.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	var rows []outputRow
	for requestURL != "" {
		var body struct {
			Namespaces []map[string]any `json:"namespaces"`
			Next       string           `json:"next"`
		}
		resp, err := client.Get(ctx, requestURL, &body, nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
		for _, item := range body.Namespaces {
			rows = append(rows, outputRow{"namespace": mapValueOrEmpty(item, "namespace")})
		}
		requestURL = resolveServiceNextURL(client, body.Next)
	}
	return renderListOutput(stdout, opts, []string{"namespace"}, rows)
}

func imageMetadefNamespaceShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef namespace show requires <namespace>")
	}
	var body map[string]any
	resp, err := client.Get(ctx, client.ServiceURL("metadefs", "namespaces", args[0]), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscResourceNotFoundError(err, "MetadefNamespace", args[0])
	}
	fields := make([]outputField, 0, 11)
	for _, name := range []string{"created_at", "description", "display_name", "namespace", "owner"} {
		fields = appendMapField(fields, body, name, name)
	}
	fields = appendMapField(fields, body, "protected", "protected")
	if associations, ok := body["resource_type_associations"]; ok && associations != nil {
		fields = append(fields, outputField{Name: "resource_type_associations", Value: metadefResourceTypeNames(associations)})
	}
	for _, name := range []string{"updated_at", "visibility"} {
		fields = appendMapField(fields, body, name, name)
	}
	return renderShowOutput(stdout, opts, fields)
}

func imageMemberList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image member list requires <image>")
	}
	image, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	page, err := members.List(client, image.ID).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := members.ExtractMembers(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Image ID":  item.ImageID,
			"Member ID": item.MemberID,
			"Status":    item.Status,
		})
	}
	return renderListOutput(stdout, opts, []string{"Image ID", "Member ID", "Status"}, rows)
}

func imageMemberGet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image member get requires <image> <project>")
	}
	image, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	item, err := members.Get(ctx, client, image.ID, args[1]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"image_id", item.ImageID},
		{"member_id", item.MemberID},
		{"schema", item.Schema},
		{"status", item.Status},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

type imageTaskListOpts struct {
	Limit   int    `q:"limit"`
	Marker  string `q:"marker"`
	SortDir string `q:"sort_dir"`
	SortKey string `q:"sort_key"`
	Type    string `q:"type"`
	Status  string `q:"status"`
}

func (opts imageTaskListOpts) ToTaskListQuery() (string, error) {
	query, err := gophercloud.BuildQueryString(opts)
	return query.String(), err
}

func imageTaskList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := tasks.List(client, imageTaskListOpts{
		Limit:   intFlag(opts, "limit"),
		Marker:  flagValue(opts, "marker"),
		SortDir: flagValue(opts, "sort-dir"),
		SortKey: flagValue(opts, "sort-key"),
		Type:    flagValue(opts, "type"),
		Status:  flagValue(opts, "status"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := tasks.ExtractTasks(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":     item.ID,
			"Type":   item.Type,
			"Status": item.Status,
			"Owner":  item.Owner,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Type", "Status", "Owner"}, rows)
}

func imageTaskShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image task show requires <Task ID>")
	}
	item, err := tasks.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"expires_at", oscTime(item.ExpiresAt)},
		{"id", item.ID},
		{"input", item.Input},
		{"message", nilIfEmpty(item.Message)},
		{"owner_id", item.Owner},
		{"properties", map[string]any{}},
		{"result", item.Result},
		{"status", item.Status},
		{"type", item.Type},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

type imageStore struct {
	ID          string         `json:"id"`
	Description any            `json:"description"`
	Default     any            `json:"default"`
	Properties  map[string]any `json:"properties"`
}

func imageStoresList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	url := client.ServiceURL("info", "stores")
	if boolFlag(opts, "detail") {
		url = client.ServiceURL("info", "stores", "detail")
	}
	var response struct {
		Stores []imageStore `json:"stores"`
	}
	_, err := client.Get(ctx, url, &response, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return fmt.Errorf("Multi Backend support not enabled")
		}
		return err
	}
	rows := make([]outputRow, 0, len(response.Stores))
	for _, item := range response.Stores {
		row := outputRow{
			"ID":          item.ID,
			"Description": item.Description,
			"Default":     storeDefault(item.Default),
		}
		if boolFlag(opts, "detail") {
			row["Properties"] = item.Properties
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Description", "Default"}
	if boolFlag(opts, "detail") {
		columns = append(columns, "Properties")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func storeDefault(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case bool:
		return typed
	case string:
		if typed == "" {
			return nil
		}
		return strings.EqualFold(typed, "true")
	default:
		return typed
	}
}

func networkList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := networks.List(client, networks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := networks.ExtractNetworks(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"ID": item.ID, "Name": item.Name, "Subnets": item.Subnets})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Subnets"}, rows)
}

func networkShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network show requires <network>")
	}
	item, err := findNetwork(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"status", item.Status},
		{"project_id", item.ProjectID},
		{"admin_state_up", item.AdminStateUp},
		{"shared", item.Shared},
		{"subnets", item.Subnets},
		{"description", item.Description},
		{"tags", item.Tags},
	})
}

func addressGroupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := addressgroups.List(client, addressgroups.ListOpts{
		Name:      flagValue(opts, "name"),
		ProjectID: flagValue(opts, "project"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := addressgroups.ExtractGroups(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Description": item.Description,
			"Project":     item.ProjectID,
			"Addresses":   item.Addresses,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Description", "Project", "Addresses"}, rows)
}

func addressGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("address group show requires <address-group>")
	}
	item, err := findAddressGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"addresses", item.Addresses},
		{"description", item.Description},
		{"id", item.ID},
		{"name", item.Name},
		{"project_id", item.ProjectID},
	})
}

func addressScopeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := addressscopes.ListOpts{
		Name:      flagValue(opts, "name"),
		ProjectID: flagValue(opts, "project"),
		IPVersion: intFlag(opts, "ip-version"),
	}
	if boolFlag(opts, "share") {
		shared := true
		listOpts.Shared = &shared
	}
	if boolFlag(opts, "no-share") {
		shared := false
		listOpts.Shared = &shared
	}
	page, err := addressscopes.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := addressscopes.ExtractAddressScopes(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":         item.ID,
			"Name":       item.Name,
			"IP Version": item.IPVersion,
			"Shared":     item.Shared,
			"Project":    firstNonEmpty(item.ProjectID, item.TenantID),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "IP Version", "Shared", "Project"}, rows)
}

func addressScopeShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("address scope show requires <address-scope>")
	}
	item, err := findAddressScope(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"ip_version", item.IPVersion},
		{"name", item.Name},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"shared", item.Shared},
	})
}

func networkAgentList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := agents.ListOpts{
		AgentType: neutronAgentType(flagValue(opts, "agent-type")),
		Host:      flagValue(opts, "host"),
	}
	page, err := agents.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := agents.ExtractAgents(page)
	if err != nil {
		return err
	}
	if network := flagValue(opts, "network"); network != "" {
		items = filterAgentsByNetwork(ctx, client, items, resolveNetworkID(ctx, client, network))
	}
	if router := flagValue(opts, "router"); router != "" {
		items = filterAgentsByRouter(ctx, client, items, resolveRouterID(ctx, client, router))
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":                item.ID,
			"Agent Type":        item.AgentType,
			"Host":              item.Host,
			"Availability Zone": nilIfEmpty(item.AvailabilityZone),
			"Alive":             item.Alive,
			"State":             item.AdminStateUp,
			"Binary":            item.Binary,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Agent Type", "Host", "Availability Zone", "Alive", "State", "Binary"}, rows)
}

func networkAgentShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network agent show requires <agent-id>")
	}
	item, err := networkAgentRaw(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"admin_state_up", item["admin_state_up"]},
		{"agent_type", item["agent_type"]},
		{"alive", item["alive"]},
		{"availability_zone", item["availability_zone"]},
		{"binary", item["binary"]},
		{"configuration", item["configurations"]},
		{"created_at", item["created_at"]},
		{"description", item["description"]},
		{"ha_state", item["ha_state"]},
		{"host", item["host"]},
		{"id", item["id"]},
		{"last_heartbeat_at", item["heartbeat_timestamp"]},
		{"resources_synced", item["resources_synced"]},
		{"started_at", item["started_at"]},
		{"topic", item["topic"]},
	})
}

func networkAgentRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var response struct {
		Agent map[string]any `json:"agent"`
	}
	_, err := client.Get(ctx, client.ServiceURL("agents", id), &response, nil)
	return response.Agent, err
}

func networkRBACList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := rbacpolicies.List(client, rbacpolicies.ListOpts{
		ObjectType:   flagValue(opts, "type"),
		Action:       rbacpolicies.PolicyAction(flagValue(opts, "action")),
		TargetTenant: flagValue(opts, "target-project"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := rbacpolicies.ExtractRBACPolicies(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":          item.ID,
			"Object Type": item.ObjectType,
			"Object ID":   item.ObjectID,
		}
		if boolFlag(opts, "long") {
			row["Action"] = item.Action
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Object Type", "Object ID"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Action")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func networkRBACShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network rbac show requires <rbac-policy>")
	}
	item, err := rbacpolicies.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"action", item.Action},
		{"id", item.ID},
		{"object_id", item.ObjectID},
		{"object_type", item.ObjectType},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"target_project_id", item.TargetTenant},
	})
}

func networkSegmentList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := segments.List(client, segments.ListOpts{
		NetworkID: resolveNetworkID(ctx, client, flagValue(opts, "network")),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := segments.ExtractSegments(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":           item.ID,
			"Network":      item.NetworkID,
			"Network Type": item.NetworkType,
			"Segment":      item.SegmentationID,
		}
		if boolFlag(opts, "long") {
			row["Name"] = item.Name
			row["Physical Network"] = nilIfEmpty(item.PhysicalNetwork)
			row["Description"] = item.Description
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Network", "Network Type", "Segment"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Name", "Physical Network", "Description")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func networkSegmentShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network segment show requires <network-segment>")
	}
	item, err := findNetworkSegment(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"id", item.ID},
		{"name", item.Name},
		{"network_id", item.NetworkID},
		{"network_type", item.NetworkType},
		{"physical_network", nilIfEmpty(item.PhysicalNetwork)},
		{"revision_number", item.RevisionNumber},
		{"segmentation_id", item.SegmentationID},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func networkTrunkList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := trunks.List(client, trunks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := trunks.ExtractTrunks(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Parent Port": item.PortID,
			"Description": item.Description,
			"Project":     firstNonEmpty(item.ProjectID, item.TenantID),
			"State":       item.Status,
		}
		if boolFlag(opts, "long") {
			row["Subports"] = trunkSubports(item.Subports)
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Parent Port", "Description", "Project", "State"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Subports", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func networkTrunkShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network trunk show requires <trunk>")
	}
	item, err := findNetworkTrunk(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"admin_state_up", item.AdminStateUp},
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"id", item.ID},
		{"name", item.Name},
		{"port_id", item.PortID},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"revision_number", item.RevisionNumber},
		{"status", item.Status},
		{"sub_ports", trunkSubports(item.Subports)},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func networkQoSPolicyList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := qospolicies.ListOpts{
		ProjectID: flagValue(opts, "project"),
	}
	if boolFlag(opts, "share") {
		shared := true
		listOpts.Shared = &shared
	}
	if boolFlag(opts, "no-share") {
		shared := false
		listOpts.Shared = &shared
	}
	page, err := qospolicies.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := qospolicies.ExtractPolicies(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":      item.ID,
			"Name":    item.Name,
			"Shared":  item.Shared,
			"Default": item.IsDefault,
			"Project": firstNonEmpty(item.ProjectID, item.TenantID),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Shared", "Default", "Project"}, rows)
}

func networkQoSPolicyShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network qos policy show requires <qos-policy>")
	}
	item, err := findNetworkQoSPolicy(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"id", item.ID},
		{"is_default", item.IsDefault},
		{"name", item.Name},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"revision_number", item.RevisionNumber},
		{"rules", item.Rules},
		{"shared", item.Shared},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func networkQoSRuleTypeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := qosruletypes.ListRuleTypes(client).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := qosruletypes.ExtractRuleTypes(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{"Type": item.Type}
		if boolFlag(opts, "all-supported") || boolFlag(opts, "all-rules") {
			row["Drivers"] = qosRuleTypeDrivers(item.Drivers)
		}
		rows = append(rows, row)
	}
	columns := []string{"Type"}
	if boolFlag(opts, "all-supported") || boolFlag(opts, "all-rules") {
		columns = append(columns, "Drivers")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func networkQoSRuleTypeShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network qos rule type show requires <qos-rule-type-name>")
	}
	item, err := qosruletypes.GetRuleType(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"drivers", qosRuleTypeDrivers(item.Drivers)},
		{"type", item.Type},
	})
}

func routerList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := routers.ListOpts{
		Name:       flagValue(opts, "name"),
		ProjectID:  flagValue(opts, "project"),
		Tags:       firstFlag(opts, "tags"),
		TagsAny:    firstFlag(opts, "any-tags", "tags-any"),
		NotTags:    firstFlag(opts, "not-tags"),
		NotTagsAny: firstFlag(opts, "not-any-tags", "not-tags-any"),
	}
	if boolFlag(opts, "enable") {
		enabled := true
		listOpts.AdminStateUp = &enabled
	}
	if boolFlag(opts, "disable") {
		disabled := false
		listOpts.AdminStateUp = &disabled
	}
	page, err := routers.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := routers.ExtractRouters(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Status":      item.Status,
			"State":       item.AdminStateUp,
			"Project":     item.ProjectID,
			"Distributed": item.Distributed,
			"HA":          false,
		}
		if boolFlag(opts, "long") {
			row["Routes"] = item.Routes
			row["External gateway info"] = item.GatewayInfo
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Status", "State", "Project", "Distributed", "HA"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Routes", "External gateway info", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func routerShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("router show requires <router>")
	}
	item, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"admin_state_up", item.AdminStateUp},
		{"availability_zone_hints", item.AvailabilityZoneHints},
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"distributed", item.Distributed},
		{"external_gateway_info", item.GatewayInfo},
		{"ha", false},
		{"id", item.ID},
		{"interfaces_info", routerInterfacesInfo(ctx, client, item.ID)},
		{"name", item.Name},
		{"project_id", item.ProjectID},
		{"revision_number", item.RevisionNumber},
		{"routes", item.Routes},
		{"status", item.Status},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func routerInterfacesInfo(ctx context.Context, client *gophercloud.ServiceClient, routerID string) []map[string]string {
	page, err := ports.List(client, ports.ListOpts{DeviceID: routerID}).AllPages(ctx)
	if err != nil {
		return nil
	}
	items, err := ports.ExtractPorts(page)
	if err != nil {
		return nil
	}
	var rows []map[string]string
	for _, port := range items {
		if port.DeviceOwner == "network:router_gateway" {
			continue
		}
		for _, fixedIP := range port.FixedIPs {
			rows = append(rows, map[string]string{
				"port_id":    port.ID,
				"ip_address": fixedIP.IPAddress,
				"subnet_id":  fixedIP.SubnetID,
			})
		}
	}
	return rows
}

func securityGroupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := secgroups.List(client, secgroups.ListOpts{
		ProjectID:  flagValue(opts, "project"),
		Tags:       firstFlag(opts, "tags"),
		TagsAny:    firstFlag(opts, "any-tags", "tags-any"),
		NotTags:    firstFlag(opts, "not-tags"),
		NotTagsAny: firstFlag(opts, "not-any-tags", "not-tags-any"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := secgroups.ExtractGroups(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Description": item.Description,
			"Project":     item.ProjectID,
			"Tags":        item.Tags,
			"Shared":      false,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Description", "Project", "Tags", "Shared"}, rows)
}

func securityGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group show requires <group>")
	}
	item, err := findSecurityGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"id", item.ID},
		{"name", item.Name},
		{"project_id", item.ProjectID},
		{"revision_number", item.RevisionNumber},
		{"rules", securityGroupRuleDetails(item.Rules)},
		{"stateful", item.Stateful},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func securityGroupRuleDetails(items []secgrouprules.SecGroupRule) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"id":                      item.ID,
			"description":             item.Description,
			"direction":               item.Direction,
			"ethertype":               item.EtherType,
			"protocol":                nilIfEmpty(item.Protocol),
			"port_range_min":          zeroNil(item.PortRangeMin),
			"port_range_max":          zeroNil(item.PortRangeMax),
			"remote_ip_prefix":        remoteIPPrefix(item),
			"remote_group_id":         nilIfEmpty(item.RemoteGroupID),
			"remote_address_group_id": nilIfEmpty(item.RemoteAddressGroupID),
			"security_group_id":       item.SecGroupID,
			"project_id":              item.ProjectID,
		})
	}
	return rows
}

func securityGroupRuleList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	listOpts := secgrouprules.ListOpts{
		Protocol:  protocolFilter(opts),
		EtherType: ethertypeFilter(opts),
		ProjectID: flagValue(opts, "project"),
	}
	if boolFlag(opts, "ingress") {
		listOpts.Direction = "ingress"
	}
	if boolFlag(opts, "egress") {
		listOpts.Direction = "egress"
	}
	includeSecurityGroup := true
	if len(args) > 0 {
		group, err := findSecurityGroup(ctx, client, args[0])
		if err != nil {
			return err
		}
		listOpts.SecGroupID = group.ID
		includeSecurityGroup = false
	}
	page, err := secgrouprules.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := secgrouprules.ExtractRules(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":                    item.ID,
			"IP Protocol":           nilIfEmpty(item.Protocol),
			"Ethertype":             item.EtherType,
			"IP Range":              remoteIPPrefix(item),
			"Port Range":            portRange(item),
			"Direction":             item.Direction,
			"Remote Security Group": nilIfEmpty(item.RemoteGroupID),
			"Remote Address Group":  nilIfEmpty(item.RemoteAddressGroupID),
		}
		if includeSecurityGroup {
			row["Security Group"] = item.SecGroupID
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "IP Protocol", "Ethertype", "IP Range", "Port Range", "Direction", "Remote Security Group", "Remote Address Group"}
	if includeSecurityGroup {
		columns = append(columns, "Security Group")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func securityGroupRuleShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group rule show requires <rule>")
	}
	item, err := secgrouprules.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"direction", item.Direction},
		{"ether_type", item.EtherType},
		{"id", item.ID},
		{"port_range_max", zeroNil(item.PortRangeMax)},
		{"port_range_min", zeroNil(item.PortRangeMin)},
		{"project_id", item.ProjectID},
		{"protocol", nilIfEmpty(item.Protocol)},
		{"remote_address_group_id", nilIfEmpty(item.RemoteAddressGroupID)},
		{"remote_group_id", nilIfEmpty(item.RemoteGroupID)},
		{"remote_ip_prefix", remoteIPPrefix(*item)},
		{"revision_number", item.RevisionNumber},
		{"security_group_id", item.SecGroupID},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func protocolFilter(opts *Options) string {
	value := strings.ToLower(flagValue(opts, "protocol"))
	if value == "any" {
		return ""
	}
	return value
}

func ethertypeFilter(opts *Options) string {
	value := flagValue(opts, "ethertype")
	switch strings.ToLower(value) {
	case "ipv4":
		return "IPv4"
	case "ipv6":
		return "IPv6"
	default:
		return value
	}
}

func remoteIPPrefix(rule secgrouprules.SecGroupRule) any {
	if rule.RemoteIPPrefix != "" {
		return rule.RemoteIPPrefix
	}
	if rule.EtherType == "IPv6" {
		return "::/0"
	}
	if rule.EtherType == "IPv4" {
		return "0.0.0.0/0"
	}
	return nil
}

func portRange(rule secgrouprules.SecGroupRule) string {
	if rule.PortRangeMin == 0 && rule.PortRangeMax == 0 {
		return ""
	}
	if rule.PortRangeMin == rule.PortRangeMax {
		return strconv.Itoa(rule.PortRangeMin)
	}
	return fmt.Sprintf("%d:%d", rule.PortRangeMin, rule.PortRangeMax)
}

func zeroNil(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func volumeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := volumes.List(client, volumes.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := volumes.ExtractVolumes(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Status":      item.Status,
			"Size":        item.Size,
			"Attached to": volumeAttachments(item.Attachments),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Status", "Size", "Attached to"}, rows)
}

func volumeShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume show requires <volume>")
	}
	item, err := findVolume(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"status", item.Status},
		{"size", item.Size},
		{"availability_zone", item.AvailabilityZone},
		{"bootable", item.Bootable},
		{"encrypted", item.Encrypted},
		{"volume_type", item.VolumeType},
		{"attachments", volumeAttachments(item.Attachments)},
		{"metadata", item.Metadata},
		{"description", item.Description},
		{"created_at", item.CreatedAt},
	})
}

func volumeBackupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := backups.List(client, backups.ListOpts{
		AllTenants: boolFlag(opts, "all-projects") || flagValue(opts, "project") != "",
		Name:       flagValue(opts, "name"),
		Status:     flagValue(opts, "status"),
		TenantID:   flagValue(opts, "project"),
		VolumeID:   flagValue(opts, "volume"),
		Limit:      intFlag(opts, "limit"),
		Marker:     flagValue(opts, "marker"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := backups.ExtractBackups(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Description": item.Description,
			"Status":      item.Status,
			"Size":        item.Size,
			"Incremental": item.IsIncremental,
			"Created At":  oscTime(item.CreatedAt),
		}
		if boolFlag(opts, "long") {
			row["Availability Zone"] = stringPtrValue(item.AvailabilityZone)
			row["Volume"] = item.VolumeID
			row["Container"] = item.Container
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Description", "Status", "Size", "Incremental", "Created At"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Availability Zone", "Volume", "Container")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeBackupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume backup show requires <backup>")
	}
	item, err := findVolumeBackup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"availability_zone", stringPtrValue(item.AvailabilityZone)},
		{"container", item.Container},
		{"created_at", oscTime(item.CreatedAt)},
		{"data_timestamp", oscTime(item.DataTimestamp)},
		{"description", item.Description},
		{"encryption_key_id", nil},
		{"fail_reason", nilIfEmpty(item.FailReason)},
		{"has_dependent_backups", item.HasDependentBackups},
		{"id", item.ID},
		{"is_incremental", item.IsIncremental},
		{"metadata", mapPtrValue(item.Metadata)},
		{"name", item.Name},
		{"object_count", item.ObjectCount},
		{"project_id", nilIfEmpty(item.ProjectID)},
		{"size", item.Size},
		{"snapshot_id", nilIfEmpty(item.SnapshotID)},
		{"status", item.Status},
		{"updated_at", oscTime(item.UpdatedAt)},
		{"user_id", nil},
		{"volume_id", item.VolumeID},
	})
}

func volumeServiceList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := bsservices.List(client, bsservices.ListOpts{
		Binary: flagValue(opts, "service"),
		Host:   flagValue(opts, "host"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := bsservices.ExtractServices(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"Binary":        item.Binary,
			"Host":          item.Host,
			"Zone":          item.Zone,
			"Status":        item.Status,
			"State":         item.State,
			"Updated At":    oscTime(item.UpdatedAt),
			"Cluster":       nilIfEmpty(item.Cluster),
			"Backend State": nil,
		}
		if boolFlag(opts, "long") {
			row["Disabled Reason"] = nilIfEmpty(item.DisabledReason)
		}
		rows = append(rows, row)
	}
	columns := []string{"Binary", "Host", "Zone", "Status", "State", "Updated At", "Cluster", "Backend State"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Disabled Reason")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeBackendPoolList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := schedulerstats.List(client, schedulerstats.ListOpts{Detail: boolFlag(opts, "long")}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := schedulerstats.ExtractStoragePools(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{"Name": item.Name}
		if boolFlag(opts, "long") {
			row["Capabilities"] = storagePoolCapabilities(item.Capabilities)
		}
		rows = append(rows, row)
	}
	columns := []string{"Name"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Capabilities")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeAttachmentList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.27")
	if err != nil {
		return err
	}
	page, err := bsattachments.List(client, bsattachments.ListOpts{
		AllTenants: boolFlag(opts, "all-projects") || flagValue(opts, "project") != "",
		ProjectID:  flagValue(opts, "project"),
		VolumeID:   flagValue(opts, "volume-id"),
		Status:     flagValue(opts, "status"),
		Limit:      intFlag(opts, "limit"),
		Marker:     flagValue(opts, "marker"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := bsattachments.ExtractAttachments(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":        item.ID,
			"Volume ID": item.VolumeID,
			"Server ID": item.Instance,
			"Status":    item.Status,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Volume ID", "Server ID", "Status"}, rows)
}

func volumeAttachmentShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume attachment show requires <attachment>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.27")
	if err != nil {
		return err
	}
	item, err := bsattachments.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"ID", item.ID},
		{"Volume ID", item.VolumeID},
		{"Instance ID", item.Instance},
		{"Status", item.Status},
		{"Attach Mode", item.AttachMode},
		{"Attached At", valueString(oscTime(item.AttachedAt))},
		{"Detached At", valueString(oscTime(item.DetachedAt))},
		{"Properties", item.ConnectionInfo},
	})
}

func volumeQoSList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := bsqos.List(client, bsqos.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := bsqos.ExtractQoS(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		associations, err := volumeQoSAssociations(ctx, client, item.ID)
		if err != nil {
			return err
		}
		rows = append(rows, outputRow{
			"ID":           item.ID,
			"Name":         item.Name,
			"Consumer":     item.Consumer,
			"Associations": associations,
			"Properties":   item.Specs,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Consumer", "Associations", "Properties"}, rows)
}

func volumeQoSShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume qos show requires <qos-spec>")
	}
	item, err := findVolumeQoS(ctx, client, args[0])
	if err != nil {
		return err
	}
	associations, err := volumeQoSAssociations(ctx, client, item.ID)
	if err != nil {
		return err
	}
	fields := []outputField{
		{"consumer", item.Consumer},
		{"id", item.ID},
		{"name", item.Name},
		{"properties", item.Specs},
	}
	if len(associations) > 0 {
		fields = append([]outputField{{"associations", associations}}, fields...)
	}
	return renderShowOutput(stdout, opts, fields)
}

func volumeSummary(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.12")
	if err != nil {
		return err
	}
	url := client.ServiceURL("volumes", "summary")
	if boolFlag(opts, "all-projects") {
		url += "?all_tenants=True"
	} else {
		url += "?all_tenants=False"
	}

	var response struct {
		Summary struct {
			TotalCount int            `json:"total_count"`
			TotalSize  int            `json:"total_size"`
			Metadata   map[string]any `json:"metadata"`
		} `json:"volume-summary"`
	}
	_, err = client.Get(ctx, url, &response, nil)
	if err != nil {
		return err
	}
	fields := []outputField{
		{"Total Count", response.Summary.TotalCount},
		{"Total Size", response.Summary.TotalSize},
	}
	if microversionAtLeast(client.Microversion, "3.36") {
		metadata := response.Summary.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}
		fields = append(fields, outputField{"Metadata", metadata})
	}
	return renderShowOutput(stdout, opts, fields)
}

type blockStorageClusterRecord struct {
	Name              string `json:"name"`
	Binary            string `json:"binary"`
	State             string `json:"state"`
	Status            string `json:"status"`
	DisabledReason    string `json:"disabled_reason"`
	NumHosts          any    `json:"num_hosts"`
	NumDownHosts      any    `json:"num_down_hosts"`
	LastHeartbeat     string `json:"last_heartbeat"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	ReplicationStatus string `json:"replication_status"`
	Frozen            any    `json:"frozen"`
	ActiveBackendID   string `json:"active_backend_id"`
}

func blockStorageClusterList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.7", "block storage cluster list")
	if err != nil {
		return err
	}
	requestURL := client.ServiceURL("clusters")
	if boolFlag(opts, "long") {
		requestURL = client.ServiceURL("clusters", "detail")
	}
	query := url.Values{}
	if value := flagValue(opts, "cluster"); value != "" {
		query.Set("name", value)
	}
	if value := flagValue(opts, "binary"); value != "" {
		query.Set("binary", value)
	}
	if boolFlag(opts, "up") {
		query.Set("is_up", "True")
	}
	if boolFlag(opts, "disabled") {
		query.Set("disabled", "True")
	}
	if value := flagValue(opts, "num-hosts"); value != "" {
		query.Set("num_hosts", value)
	}
	if value := flagValue(opts, "num-down-hosts"); value != "" {
		query.Set("num_down_hosts", value)
	}
	if encoded := query.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	var response struct {
		Clusters []blockStorageClusterRecord `json:"clusters"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	columns := []string{"Name", "Binary", "State", "Status"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Num Hosts", "Num Down Hosts", "Last Heartbeat", "Disabled Reason", "Created At", "Updated At")
	}
	rows := make([]outputRow, 0, len(response.Clusters))
	for _, item := range response.Clusters {
		row := outputRow{
			"Name":   item.Name,
			"Binary": item.Binary,
			"State":  item.State,
			"Status": item.Status,
		}
		if boolFlag(opts, "long") {
			row["Num Hosts"] = item.NumHosts
			row["Num Down Hosts"] = item.NumDownHosts
			row["Last Heartbeat"] = nilIfEmpty(item.LastHeartbeat)
			row["Disabled Reason"] = nilIfEmpty(item.DisabledReason)
			row["Created At"] = nilIfEmpty(item.CreatedAt)
			row["Updated At"] = nilIfEmpty(item.UpdatedAt)
		}
		rows = append(rows, row)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func blockStorageClusterShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("block storage cluster show requires <cluster>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.7", "block storage cluster show")
	if err != nil {
		return err
	}
	requestURL := client.ServiceURL("clusters", args[0])
	if value := flagValue(opts, "binary"); value != "" {
		query := url.Values{}
		query.Set("binary", value)
		requestURL += "?" + query.Encode()
	}
	var response struct {
		Cluster blockStorageClusterRecord `json:"cluster"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	item := response.Cluster
	return renderShowOutput(stdout, opts, []outputField{
		{"Name", item.Name},
		{"Binary", item.Binary},
		{"State", item.State},
		{"Status", item.Status},
		{"Disabled Reason", nilIfEmpty(item.DisabledReason)},
		{"Hosts", item.NumHosts},
		{"Down Hosts", item.NumDownHosts},
		{"Last Heartbeat", nilIfEmpty(item.LastHeartbeat)},
		{"Created At", nilIfEmpty(item.CreatedAt)},
		{"Updated At", nilIfEmpty(item.UpdatedAt)},
		{"Replication Status", nilIfEmpty(item.ReplicationStatus)},
		{"Frozen", item.Frozen},
		{"Active Backend ID", nilIfEmpty(item.ActiveBackendID)},
	})
}

type blockStorageResourceFilterRecord struct {
	Resource string   `json:"resource"`
	Filters  []string `json:"filters"`
}

func blockStorageResourceFilterList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	items, err := listBlockStorageResourceFilters(ctx, client, "")
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Resource": item.Resource,
			"Filters":  item.Filters,
		})
	}
	return renderListOutput(stdout, opts, []string{"Resource", "Filters"}, rows)
}

func blockStorageResourceFilterShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("block storage resource filter show requires <resource>")
	}
	items, err := listBlockStorageResourceFilters(ctx, client, args[0])
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("No resource filter with a name of {parsed_args.resource}' exists.")
	}
	item := items[0]
	return renderShowOutput(stdout, opts, []outputField{
		{"Resource", item.Resource},
		{"Filters", item.Filters},
	})
}

func listBlockStorageResourceFilters(ctx context.Context, client *gophercloud.ServiceClient, resource string) ([]blockStorageResourceFilterRecord, error) {
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.33")
	if err != nil {
		return nil, err
	}
	requestURL := client.ServiceURL("resource_filters")
	if resource != "" {
		query := url.Values{}
		query.Set("resource", resource)
		requestURL += "?" + query.Encode()
	}
	var response struct {
		ResourceFilters []blockStorageResourceFilterRecord `json:"resource_filters"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return response.ResourceFilters, nil
}

type blockStorageLogLevelRecord struct {
	Binary string            `json:"binary"`
	Host   string            `json:"host"`
	Levels map[string]string `json:"levels"`
}

func blockStorageLogLevelList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.32")
	if err != nil {
		return err
	}
	request := map[string]any{
		"binary": flagValue(opts, "service"),
		"server": nilIfEmpty(flagValue(opts, "host")),
		"prefix": nilIfEmpty(flagValue(opts, "log-prefix")),
	}
	var response struct {
		LogLevels []blockStorageLogLevelRecord `json:"log_levels"`
	}
	resp, err := client.Put(ctx, client.ServiceURL("os-services", "get-log"), request, &response, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusOK},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	rows := []outputRow{}
	for _, item := range response.LogLevels {
		prefixes := make([]string, 0, len(item.Levels))
		for prefix := range item.Levels {
			prefixes = append(prefixes, prefix)
		}
		sort.Strings(prefixes)
		for _, prefix := range prefixes {
			rows = append(rows, outputRow{
				"Binary": item.Binary,
				"Host":   item.Host,
				"Prefix": prefix,
				"Level":  item.Levels[prefix],
			})
		}
	}
	return renderListOutput(stdout, opts, []string{"Binary", "Host", "Prefix", "Level"}, rows)
}

type volumeMessageRecord struct {
	ID              string              `json:"id"`
	EventID         string              `json:"event_id"`
	ResourceType    string              `json:"resource_type"`
	ResourceUUID    string              `json:"resource_uuid"`
	MessageLevel    string              `json:"message_level"`
	UserMessage     string              `json:"user_message"`
	RequestID       string              `json:"request_id"`
	CreatedAt       string              `json:"created_at"`
	GuaranteedUntil string              `json:"guaranteed_until"`
	Links           []volumeMessageLink `json:"links"`
}

type volumeMessageLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func volumeMessageList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) error {
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.3", "volume message list")
	if err != nil {
		return err
	}
	query := url.Values{}
	if project := flagValue(opts, "project"); project != "" {
		identityClient, err := clients.identityV3()
		if err != nil {
			return err
		}
		item, err := findProjectWithDomain(ctx, identityClient, project, flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		query.Set("project_id", item.ID)
	}
	if value := flagValue(opts, "marker"); value != "" {
		query.Set("marker", value)
	}
	if value := intFlag(opts, "limit"); value > 0 {
		query.Set("limit", strconv.Itoa(value))
	}
	requestURL := client.ServiceURL("messages")
	if encoded := query.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	var response struct {
		Messages []volumeMessageRecord `json:"messages"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(response.Messages))
	for _, item := range response.Messages {
		rows = append(rows, outputRow{
			"ID":               item.ID,
			"Event ID":         item.EventID,
			"Resource Type":    item.ResourceType,
			"Resource UUID":    item.ResourceUUID,
			"Message Level":    item.MessageLevel,
			"User Message":     item.UserMessage,
			"Request ID":       item.RequestID,
			"Created At":       item.CreatedAt,
			"Guaranteed Until": item.GuaranteedUntil,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Event ID", "Resource Type", "Resource UUID", "Message Level", "User Message", "Request ID", "Created At", "Guaranteed Until"}, rows)
}

func volumeMessageShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume message show requires <message-id>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.3", "volume message show")
	if err != nil {
		return err
	}
	var response struct {
		Message volumeMessageRecord `json:"message"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("messages", args[0]), &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	item := response.Message
	return renderShowOutput(stdout, opts, []outputField{
		{"created_at", item.CreatedAt},
		{"event_id", item.EventID},
		{"guaranteed_until", item.GuaranteedUntil},
		{"id", item.ID},
		{"links", item.Links},
		{"message_level", item.MessageLevel},
		{"request_id", item.RequestID},
		{"resource_type", item.ResourceType},
		{"resource_uuid", item.ResourceUUID},
		{"user_message", item.UserMessage},
	})
}

type volumeGroupRecord struct {
	ID               string   `json:"id"`
	Status           string   `json:"status"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	GroupType        string   `json:"group_type"`
	VolumeTypes      []string `json:"volume_types"`
	AvailabilityZone string   `json:"availability_zone"`
	CreatedAt        string   `json:"created_at"`
	Volumes          []any    `json:"volumes"`
	GroupSnapshotID  string   `json:"group_snapshot_id"`
	SourceGroupID    string   `json:"source_group_id"`
}

func volumeGroupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.13", "volume group list")
	if err != nil {
		return err
	}
	items, err := listVolumeGroups(ctx, client, boolFlag(opts, "all-projects"), false)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":     item.ID,
			"Status": item.Status,
			"Name":   item.Name,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Status", "Name"}, rows)
}

func volumeGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group show requires <group>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.13", "volume group show")
	if err != nil {
		return err
	}
	if flagValue(opts, "volumes") != "" || flagValue(opts, "no-volumes") != "" {
		if !microversionAtLeast(client.Microversion, "3.25") {
			return fmt.Errorf("--os-volume-api-version 3.25 or greater is required to support the '--(no-)volumes' option")
		}
	}
	if flagValue(opts, "replication-targets") != "" || flagValue(opts, "no-replication-targets") != "" {
		if !microversionAtLeast(client.Microversion, "3.38") {
			return fmt.Errorf("--os-volume-api-version 3.38 or greater is required to support the '--(no-)replication-targets' option")
		}
	}
	item, err := findVolumeGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	requestURL := client.ServiceURL("groups", item.ID)
	if boolFlag(opts, "volumes") {
		query := url.Values{}
		query.Set("list_volume", "True")
		requestURL += "?" + query.Encode()
	}
	var response struct {
		Group volumeGroupRecord `json:"group"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	item = response.Group
	return renderShowOutput(stdout, opts, []outputField{
		{"ID", item.ID},
		{"Status", item.Status},
		{"Name", item.Name},
		{"Description", nilIfEmpty(item.Description)},
		{"Group Type", nilIfEmpty(item.GroupType)},
		{"Volume Types", item.VolumeTypes},
		{"Availability Zone", nilIfEmpty(item.AvailabilityZone)},
		{"Created At", nilIfEmpty(item.CreatedAt)},
		{"Volumes", item.Volumes},
		{"Group Snapshot ID", nilIfEmpty(item.GroupSnapshotID)},
		{"Source Group ID", nilIfEmpty(item.SourceGroupID)},
	})
}

func listVolumeGroups(ctx context.Context, client *gophercloud.ServiceClient, allProjects bool, listVolumes bool) ([]volumeGroupRecord, error) {
	requestURL := client.ServiceURL("groups", "detail")
	query := url.Values{}
	if allProjects {
		query.Set("all_tenants", "True")
	}
	if listVolumes {
		query.Set("list_volume", "True")
	}
	if encoded := query.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	var response struct {
		Groups []volumeGroupRecord `json:"groups"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return response.Groups, nil
}

func findVolumeGroup(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string) (volumeGroupRecord, error) {
	items, err := listVolumeGroups(ctx, client, false, false)
	if err != nil {
		return volumeGroupRecord{}, err
	}
	for _, item := range items {
		if item.ID == nameOrID || item.Name == nameOrID {
			return item, nil
		}
	}
	var response struct {
		Group volumeGroupRecord `json:"group"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("groups", nameOrID), &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return volumeGroupRecord{}, err
	}
	return response.Group, nil
}

type volumeGroupSnapshotRecord struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Name        string `json:"name"`
	Description string `json:"description"`
	GroupID     string `json:"group_id"`
	GroupTypeID string `json:"group_type_id"`
}

func volumeGroupSnapshotList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.14")
	if err != nil {
		return err
	}
	items, err := listVolumeGroupSnapshots(ctx, client, boolFlag(opts, "all-projects"))
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":     item.ID,
			"Status": item.Status,
			"Name":   item.Name,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Status", "Name"}, rows)
}

func volumeGroupSnapshotShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group snapshot show requires <snapshot>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.14")
	if err != nil {
		return err
	}
	item, err := findVolumeGroupSnapshot(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"ID", item.ID},
		{"Status", item.Status},
		{"Name", item.Name},
		{"Description", nilIfEmpty(item.Description)},
		{"Group", nilIfEmpty(item.GroupID)},
		{"Group Type", nilIfEmpty(item.GroupTypeID)},
	})
}

func listVolumeGroupSnapshots(ctx context.Context, client *gophercloud.ServiceClient, allProjects bool) ([]volumeGroupSnapshotRecord, error) {
	requestURL := client.ServiceURL("group_snapshots", "detail")
	if allProjects {
		query := url.Values{}
		query.Set("all_tenants", "True")
		requestURL += "?" + query.Encode()
	}
	var response struct {
		GroupSnapshots []volumeGroupSnapshotRecord `json:"group_snapshots"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return response.GroupSnapshots, nil
}

func findVolumeGroupSnapshot(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string) (volumeGroupSnapshotRecord, error) {
	items, err := listVolumeGroupSnapshots(ctx, client, false)
	if err != nil {
		return volumeGroupSnapshotRecord{}, err
	}
	for _, item := range items {
		if item.ID == nameOrID || item.Name == nameOrID {
			nameOrID = item.ID
			break
		}
	}
	var response struct {
		GroupSnapshot volumeGroupSnapshotRecord `json:"group_snapshot"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("group_snapshots", nameOrID), &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return volumeGroupSnapshotRecord{}, err
	}
	return response.GroupSnapshot, nil
}

type volumeGroupTypeRecord struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	IsPublic    bool           `json:"is_public"`
	GroupSpecs  map[string]any `json:"group_specs"`
}

func volumeGroupTypeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.11", "volume group type list")
	if err != nil {
		return err
	}
	var items []volumeGroupTypeRecord
	if boolFlag(opts, "default") {
		item, err := getVolumeGroupType(ctx, client, "default")
		if err != nil {
			return err
		}
		items = []volumeGroupTypeRecord{item}
	} else {
		items, err = listVolumeGroupTypes(ctx, client)
		if err != nil {
			return err
		}
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":         item.ID,
			"Name":       item.Name,
			"Is Public":  item.IsPublic,
			"Properties": mapAnyIfNil(item.GroupSpecs),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Is Public", "Properties"}, rows)
}

func volumeGroupTypeShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group type show requires <group_type>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.11", "volume group type show")
	if err != nil {
		return err
	}
	item, err := findVolumeGroupType(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"ID", item.ID},
		{"Name", item.Name},
		{"Description", nilIfEmpty(item.Description)},
		{"Is Public", item.IsPublic},
		{"Properties", mapAnyIfNil(item.GroupSpecs)},
	})
}

func listVolumeGroupTypes(ctx context.Context, client *gophercloud.ServiceClient) ([]volumeGroupTypeRecord, error) {
	var response struct {
		GroupTypes []volumeGroupTypeRecord `json:"group_types"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("group_types"), &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return response.GroupTypes, nil
}

func findVolumeGroupType(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string) (volumeGroupTypeRecord, error) {
	items, err := listVolumeGroupTypes(ctx, client)
	if err != nil {
		return volumeGroupTypeRecord{}, err
	}
	for _, item := range items {
		if item.ID == nameOrID || item.Name == nameOrID {
			return getVolumeGroupType(ctx, client, item.ID)
		}
	}
	return getVolumeGroupType(ctx, client, nameOrID)
}

func getVolumeGroupType(ctx context.Context, client *gophercloud.ServiceClient, id string) (volumeGroupTypeRecord, error) {
	var response struct {
		GroupType volumeGroupTypeRecord `json:"group_type"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("group_types", id), &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return volumeGroupTypeRecord{}, err
	}
	return response.GroupType, nil
}

func mapAnyIfNil(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func volumeQoSAssociations(ctx context.Context, client *gophercloud.ServiceClient, qosID string) ([]string, error) {
	page, err := bsqos.ListAssociations(client, qosID).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := bsqos.ExtractAssociations(page)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names, nil
}

func volumeTransferList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := transfers.List(client, transfers.ListOpts{AllTenants: boolFlag(opts, "all-projects")}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := transfers.ExtractTransfers(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":     item.ID,
			"Name":   item.Name,
			"Volume": item.VolumeID,
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Volume"}, rows)
}

func volumeTransferShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume transfer request show requires <transfer-request>")
	}
	item, err := findVolumeTransfer(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"auth_key", nilIfEmpty(item.AuthKey)},
		{"created_at", oscTime(item.CreatedAt)},
		{"id", item.ID},
		{"name", item.Name},
		{"volume_id", item.VolumeID},
	})
}

func stringPtrValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func mapPtrValue(value *map[string]string) any {
	if value == nil {
		return map[string]string{}
	}
	return *value
}

func storagePoolCapabilities(value schedulerstats.Capabilities) map[string]any {
	return map[string]any{
		"allocated_capacity_gb":       value.AllocatedCapacityGB,
		"driver_version":              value.DriverVersion,
		"filter_function":             value.FilterFunction,
		"free_capacity_gb":            value.FreeCapacityGB,
		"goodness_function":           value.GoodnessFunction,
		"location_info":               value.LocationInfo,
		"max_over_subscription_ratio": value.MaxOverSubscriptionRatio,
		"multiattach":                 value.Multiattach,
		"provisioned_capacity_gb":     value.ProvisionedCapacityGB,
		"qos_support":                 value.QoSSupport,
		"reserved_percentage":         value.ReservedPercentage,
		"sparse_copy_volume":          value.SparseCopyVolume,
		"storage_protocol":            value.StorageProtocol,
		"thick_provisioning_support":  value.ThickProvisioningSupport,
		"thin_provisioning_support":   value.ThinProvisioningSupport,
		"total_capacity_gb":           value.TotalCapacityGB,
		"total_volumes":               value.TotalVolumes,
		"vendor_name":                 value.VendorName,
		"volume_backend_name":         value.VolumeBackendName,
	}
}

func subnetList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	enableDHCP, hasDHCPFilter := subnetDHCPFilter(opts)
	listOpts := subnets.ListOpts{
		Name:         flagValue(opts, "name"),
		GatewayIP:    flagValue(opts, "gateway"),
		CIDR:         flagValue(opts, "subnet-range"),
		SubnetPoolID: flagValue(opts, "subnet-pool"),
		ProjectID:    flagValue(opts, "project"),
		Tags:         firstFlag(opts, "tags"),
		TagsAny:      firstFlag(opts, "any-tags", "tags-any"),
		NotTags:      firstFlag(opts, "not-tags"),
		NotTagsAny:   firstFlag(opts, "not-any-tags", "not-tags-any"),
	}
	if value := intFlag(opts, "ip-version"); value != 0 {
		listOpts.IPVersion = value
	}
	if hasDHCPFilter {
		listOpts.EnableDHCP = &enableDHCP
	}
	if network := flagValue(opts, "network"); network != "" {
		listOpts.NetworkID = resolveNetworkID(ctx, client, network)
	}

	page, err := subnets.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := subnets.ExtractSubnets(page)
	if err != nil {
		return err
	}
	serviceType := flagValue(opts, "service-type")
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		if serviceType != "" && !stringSliceContains(item.ServiceTypes, serviceType) {
			continue
		}
		row := outputRow{
			"ID":      item.ID,
			"Name":    item.Name,
			"Network": item.NetworkID,
			"Subnet":  item.CIDR,
		}
		if boolFlag(opts, "long") {
			row["Project"] = item.ProjectID
			row["DHCP"] = item.EnableDHCP
			row["Gateway"] = item.GatewayIP
			row["IP Version"] = item.IPVersion
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Network", "Subnet"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Project", "DHCP", "Gateway", "IP Version", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func subnetShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet show requires <subnet>")
	}
	item, err := findSubnet(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"network_id", item.NetworkID},
		{"project_id", item.ProjectID},
		{"cidr", item.CIDR},
		{"ip_version", item.IPVersion},
		{"gateway_ip", item.GatewayIP},
		{"enable_dhcp", item.EnableDHCP},
		{"dns_nameservers", item.DNSNameservers},
		{"allocation_pools", item.AllocationPools},
		{"host_routes", item.HostRoutes},
		{"service_types", item.ServiceTypes},
		{"subnetpool_id", item.SubnetPoolID},
		{"description", item.Description},
		{"tags", item.Tags},
		{"created_at", item.CreatedAt},
		{"updated_at", item.UpdatedAt},
	})
}

func subnetPoolList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := subnetpools.ListOpts{
		Name:       flagValue(opts, "name"),
		ProjectID:  flagValue(opts, "project"),
		Tags:       firstFlag(opts, "tags"),
		TagsAny:    firstFlag(opts, "any-tags", "tags-any"),
		NotTags:    firstFlag(opts, "not-tags"),
		NotTagsAny: firstFlag(opts, "not-any-tags", "not-tags-any"),
	}
	if boolFlag(opts, "share") {
		shared := true
		listOpts.Shared = &shared
	}
	if boolFlag(opts, "no-share") {
		shared := false
		listOpts.Shared = &shared
	}
	if boolFlag(opts, "default") {
		isDefault := true
		listOpts.IsDefault = &isDefault
	}
	if boolFlag(opts, "no-default") {
		isDefault := false
		listOpts.IsDefault = &isDefault
	}
	if addressScope := flagValue(opts, "address-scope"); addressScope != "" {
		listOpts.AddressScopeID = resolveAddressScopeID(ctx, client, addressScope)
	}
	page, err := subnetpools.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := subnetpools.ExtractSubnetPools(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":       item.ID,
			"Name":     item.Name,
			"Prefixes": item.Prefixes,
		}
		if boolFlag(opts, "long") {
			row["Default Prefix Length"] = item.DefaultPrefixLen
			row["Address Scope"] = nilIfEmpty(item.AddressScopeID)
			row["Default"] = item.IsDefault
			row["Shared"] = item.Shared
			row["Project"] = firstNonEmpty(item.ProjectID, item.TenantID)
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Prefixes"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Default Prefix Length", "Address Scope", "Default", "Shared", "Project", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func subnetPoolShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet pool show requires <subnet-pool>")
	}
	item, err := findSubnetPool(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"address_scope_id", nilIfEmpty(item.AddressScopeID)},
		{"created_at", oscTime(item.CreatedAt)},
		{"default_prefixlen", item.DefaultPrefixLen},
		{"default_quota", item.DefaultQuota},
		{"description", item.Description},
		{"id", item.ID},
		{"ip_version", item.IPversion},
		{"is_default", item.IsDefault},
		{"max_prefixlen", item.MaxPrefixLen},
		{"min_prefixlen", item.MinPrefixLen},
		{"name", item.Name},
		{"prefixes", item.Prefixes},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"revision_number", item.RevisionNumber},
		{"shared", item.Shared},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	})
}

func portList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, computeClient *gophercloud.ServiceClient) error {
	listOpts := ports.ListOpts{
		Name:           flagValue(opts, "name"),
		Status:         flagValue(opts, "status"),
		DeviceOwner:    flagValue(opts, "device-owner"),
		MACAddress:     flagValue(opts, "mac-address"),
		ProjectID:      flagValue(opts, "project"),
		SecurityGroups: compactStrings([]string{flagValue(opts, "security-group")}),
		Tags:           firstFlag(opts, "tags"),
		TagsAny:        firstFlag(opts, "any-tags", "tags-any"),
		NotTags:        firstFlag(opts, "not-tags"),
		NotTagsAny:     firstFlag(opts, "not-any-tags", "not-tags-any"),
	}
	if network := flagValue(opts, "network"); network != "" {
		listOpts.NetworkID = resolveNetworkID(ctx, client, network)
	}
	if deviceID := firstFlag(opts, "device-id", "router"); deviceID != "" {
		listOpts.DeviceID = deviceID
	}
	if server := flagValue(opts, "server"); server != "" {
		listOpts.DeviceID = resolveServerID(ctx, computeClient, server)
	}
	if fixedIP := flagValue(opts, "fixed-ip"); fixedIP != "" {
		listOpts.FixedIPs = append(listOpts.FixedIPs, fixedIPFilter(ctx, client, fixedIP))
	}

	page, err := ports.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := ports.ExtractPorts(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":                 item.ID,
			"Name":               item.Name,
			"MAC Address":        item.MACAddress,
			"Fixed IP Addresses": portFixedIPs(item.FixedIPs),
			"Status":             item.Status,
		}
		if boolFlag(opts, "long") {
			row["Project"] = item.ProjectID
			row["Network"] = item.NetworkID
			row["Device Owner"] = item.DeviceOwner
			row["Device ID"] = item.DeviceID
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "MAC Address", "Fixed IP Addresses", "Status"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Project", "Network", "Device Owner", "Device ID", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func portShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("port show requires <port>")
	}
	item, err := findPort(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"network_id", item.NetworkID},
		{"project_id", item.ProjectID},
		{"status", item.Status},
		{"admin_state_up", item.AdminStateUp},
		{"mac_address", item.MACAddress},
		{"fixed_ips", item.FixedIPs},
		{"device_id", item.DeviceID},
		{"device_owner", item.DeviceOwner},
		{"security_group_ids", item.SecurityGroups},
		{"allowed_address_pairs", item.AllowedAddressPairs},
		{"description", item.Description},
		{"tags", item.Tags},
		{"created_at", item.CreatedAt},
		{"updated_at", item.UpdatedAt},
	})
}

func floatingIPList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := floatingips.ListOpts{
		FixedIP:    flagValue(opts, "fixed-ip-address"),
		FloatingIP: flagValue(opts, "floating-ip-address"),
		Status:     flagValue(opts, "status"),
		ProjectID:  flagValue(opts, "project"),
		RouterID:   flagValue(opts, "router"),
		Tags:       firstFlag(opts, "tags"),
		TagsAny:    firstFlag(opts, "any-tags", "tags-any"),
		NotTags:    firstFlag(opts, "not-tags"),
		NotTagsAny: firstFlag(opts, "not-any-tags", "not-tags-any"),
	}
	if network := flagValue(opts, "network"); network != "" {
		listOpts.FloatingNetworkID = resolveNetworkID(ctx, client, network)
	}
	if port := flagValue(opts, "port"); port != "" {
		listOpts.PortID = resolvePortID(ctx, client, port)
	}
	page, err := floatingips.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := floatingips.ExtractFloatingIPs(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":                  item.ID,
			"Floating IP Address": item.FloatingIP,
			"Fixed IP Address":    item.FixedIP,
			"Port":                item.PortID,
			"Floating Network":    item.FloatingNetworkID,
		}
		if boolFlag(opts, "long") {
			row["Project"] = item.ProjectID
			row["Router"] = item.RouterID
			row["Status"] = item.Status
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Floating IP Address", "Fixed IP Address", "Port", "Floating Network"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Project", "Router", "Status", "Tags")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func floatingIPShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("floating ip show requires <floating-ip>")
	}
	item, err := findFloatingIP(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"floating_ip_address", item.FloatingIP},
		{"fixed_ip_address", item.FixedIP},
		{"floating_network_id", item.FloatingNetworkID},
		{"port_id", item.PortID},
		{"router_id", item.RouterID},
		{"project_id", item.ProjectID},
		{"status", item.Status},
		{"description", item.Description},
		{"tags", item.Tags},
		{"created_at", item.CreatedAt},
		{"updated_at", item.UpdatedAt},
	})
}

func floatingIPPoolList() error {
	return fmt.Errorf("Floating ip pool operations are only available for Compute v2 network.")
}

func ipAvailabilityList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) error {
	listOpts := networkipavailabilities.ListOpts{}
	if version := flagValue(opts, "ip-version"); version != "" {
		listOpts.IPVersion = version
	}
	if project := flagValue(opts, "project"); project != "" {
		identityClient, err := clients.identityV3()
		if err != nil {
			return err
		}
		item, err := findProjectWithDomain(ctx, identityClient, project, flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		listOpts.ProjectID = item.ID
	}
	page, err := networkipavailabilities.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := networkipavailabilities.ExtractNetworkIPAvailabilities(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Network ID":   item.NetworkID,
			"Network Name": item.NetworkName,
			"Total IPs":    jsonNumberString(item.TotalIPs),
			"Used IPs":     jsonNumberString(item.UsedIPs),
		})
	}
	return renderListOutput(stdout, opts, []string{"Network ID", "Network Name", "Total IPs", "Used IPs"}, rows)
}

func ipAvailabilityShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ip availability show requires <network>")
	}
	network, err := findNetwork(ctx, client, args[0])
	if err != nil {
		return err
	}
	item, err := networkipavailabilities.Get(ctx, client, network.ID).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"network_id", item.NetworkID},
		{"network_name", item.NetworkName},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"subnet_ip_availability", ipAvailabilitySubnets(item.SubnetIPAvailabilities)},
		{"total_ips", jsonNumberString(item.TotalIPs)},
		{"used_ips", jsonNumberString(item.UsedIPs)},
	})
}

type ipAvailabilitySubnetOutput struct {
	SubnetID   string `json:"subnet_id"`
	IPVersion  int    `json:"ip_version"`
	CIDR       string `json:"cidr"`
	SubnetName string `json:"subnet_name"`
	UsedIPs    any    `json:"used_ips"`
	TotalIPs   any    `json:"total_ips"`
}

func ipAvailabilitySubnets(items []networkipavailabilities.SubnetIPAvailability) []ipAvailabilitySubnetOutput {
	values := make([]ipAvailabilitySubnetOutput, 0, len(items))
	for _, item := range items {
		values = append(values, ipAvailabilitySubnetOutput{
			SubnetID:   item.SubnetID,
			IPVersion:  item.IPVersion,
			CIDR:       item.CIDR,
			SubnetName: item.SubnetName,
			UsedIPs:    jsonNumberString(item.UsedIPs),
			TotalIPs:   jsonNumberString(item.TotalIPs),
		})
	}
	return values
}

func jsonNumberString(value string) any {
	if value == "" {
		return nil
	}
	return json.Number(value)
}

type networkServiceProviderRecord struct {
	ServiceType string `json:"service_type"`
	Name        string `json:"name"`
	Default     bool   `json:"default"`
}

func networkServiceProviderList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	var body struct {
		ServiceProviders []networkServiceProviderRecord `json:"service_providers"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("service-providers"), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(body.ServiceProviders))
	for _, item := range body.ServiceProviders {
		rows = append(rows, outputRow{
			"Service Type": item.ServiceType,
			"Name":         item.Name,
			"Default":      item.Default,
		})
	}
	return renderListOutput(stdout, opts, []string{"Service Type", "Name", "Default"}, rows)
}

func keypairList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := keypairs.List(client, keypairs.ListOpts{UserID: flagValue(opts, "user")}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := keypairs.ExtractKeyPairs(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Name":        item.Name,
			"Fingerprint": item.Fingerprint,
			"Type":        item.Type,
		})
	}
	return renderListOutput(stdout, opts, []string{"Name", "Fingerprint", "Type"}, rows)
}

func keypairShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("keypair show requires <key>")
	}
	item, err := keypairs.Get(ctx, client, args[0], keypairs.GetOpts{UserID: flagValue(opts, "user")}).Extract()
	if err != nil {
		return err
	}
	if boolFlag(opts, "public-key") {
		_, err := fmt.Fprintln(stdout, item.PublicKey)
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"name", item.Name},
		{"fingerprint", item.Fingerprint},
		{"public_key", item.PublicKey},
		{"user_id", item.UserID},
		{"type", item.Type},
	})
}

func serverGroupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := servergroups.ListOpts{
		AllProjects: boolFlag(opts, "all-projects"),
		Limit:       intFlag(opts, "limit"),
		Offset:      intFlag(opts, "offset"),
	}
	page, err := servergroups.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := servergroups.ExtractServerGroups(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{"ID": item.ID, "Name": item.Name, "Policies": item.Policies}
		if boolFlag(opts, "long") {
			row["Project"] = item.ProjectID
			row["User"] = item.UserID
			row["Members"] = item.Members
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Policies"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Project", "User", "Members")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func serverGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server group show requires <server-group>")
	}
	item, err := findServerGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"policies", item.Policies},
		{"members", item.Members},
		{"project_id", item.ProjectID},
		{"user_id", item.UserID},
		{"metadata", item.Metadata},
		{"policy", item.Policy},
		{"rules", item.Rules},
	})
}

func volumeTypeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := volumetypes.ListOpts{Name: flagValue(opts, "name")}
	if boolFlag(opts, "public") {
		listOpts.IsPublic = volumetypes.VisibilityPublic
	}
	if boolFlag(opts, "private") {
		listOpts.IsPublic = volumetypes.VisibilityPrivate
	}
	page, err := volumetypes.List(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := volumetypes.ExtractVolumeTypes(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		if !volumeTypeMatches(item, opts) {
			continue
		}
		row := outputRow{
			"ID":        item.ID,
			"Name":      item.Name,
			"Is Public": volumeTypeIsPublic(item),
		}
		if boolFlag(opts, "long") {
			row["Description"] = item.Description
			row["Properties"] = item.ExtraSpecs
			row["Qos Spec ID"] = item.QosSpecID
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Is Public"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Description", "Properties", "Qos Spec ID")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeTypeShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume type show requires <volume-type>")
	}
	item, err := findVolumeType(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"description", item.Description},
		{"is_public", volumeTypeIsPublic(*item)},
		{"properties", item.ExtraSpecs},
		{"qos_specs_id", item.QosSpecID},
	})
}

func volumeSnapshotList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := snapshots.ListOpts{
		AllTenants: boolFlag(opts, "all-projects"),
		Name:       flagValue(opts, "name"),
		Status:     flagValue(opts, "status"),
		TenantID:   flagValue(opts, "project"),
		Limit:      intFlag(opts, "limit"),
		Marker:     flagValue(opts, "marker"),
	}
	if volume := flagValue(opts, "volume"); volume != "" {
		listOpts.VolumeID = resolveVolumeID(ctx, client, volume)
	}
	page, err := snapshots.ListDetail(client, listOpts).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := snapshots.ExtractSnapshots(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Description": item.Description,
			"Status":      item.Status,
			"Size":        item.Size,
		}
		if boolFlag(opts, "long") {
			row["Volume"] = item.VolumeID
			row["Created At"] = item.CreatedAt
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Description", "Status", "Size"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Volume", "Created At")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func volumeSnapshotShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume snapshot show requires <snapshot>")
	}
	item, err := findVolumeSnapshot(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"description", item.Description},
		{"status", item.Status},
		{"size", item.Size},
		{"volume_id", item.VolumeID},
		{"metadata", item.Metadata},
		{"created_at", item.CreatedAt},
		{"updated_at", item.UpdatedAt},
	})
}

func findServer(ctx context.Context, client *gophercloud.ServiceClient, value string) (*servers.Server, error) {
	result := servers.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := servers.List(client, servers.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := extractServers(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item servers.Server) string { return item.Name })
}

func serverIDForEventLookup(ctx context.Context, client *gophercloud.ServiceClient, value string) (string, error) {
	server, err := findServer(ctx, client, value)
	if err == nil {
		return server.ID, nil
	}
	if isUUIDLike(value) {
		return value, nil
	}
	return "", err
}

func findAggregate(ctx context.Context, client *gophercloud.ServiceClient, value string) (*aggregates.Aggregate, error) {
	if id, err := strconv.Atoi(value); err == nil {
		result := aggregates.Get(ctx, client, id)
		if result.Err == nil {
			return result.Extract()
		}
	}
	page, err := aggregates.List(client).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	items, err := aggregates.ExtractAggregates(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item aggregates.Aggregate) string { return item.Name })
}

func findHypervisor(ctx context.Context, client *gophercloud.ServiceClient, value string) (*hypervisors.Hypervisor, error) {
	result := hypervisors.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := hypervisors.List(client, hypervisors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := hypervisors.ExtractHypervisors(page)
	if err != nil {
		return nil, err
	}
	var matches []hypervisors.Hypervisor
	for _, item := range items {
		if item.HypervisorHostname == value || item.ID == value {
			matches = append(matches, item)
		}
	}
	return singleMatch(value, matches)
}

func findFlavor(ctx context.Context, client *gophercloud.ServiceClient, value string) (*flavors.Flavor, error) {
	result := flavors.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := flavors.ListDetail(client, flavors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := flavors.ExtractFlavors(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item flavors.Flavor) string { return item.Name })
}

func findImage(ctx context.Context, client *gophercloud.ServiceClient, value string) (*images.Image, error) {
	result := images.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := images.List(client, images.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := images.ExtractImages(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item images.Image) string { return item.Name })
}

func findNetwork(ctx context.Context, client *gophercloud.ServiceClient, value string) (*networks.Network, error) {
	result := networks.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := networks.List(client, networks.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := networks.ExtractNetworks(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item networks.Network) string { return item.Name })
}

func findAddressGroup(ctx context.Context, client *gophercloud.ServiceClient, value string) (*addressgroups.AddressGroup, error) {
	result := addressgroups.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := addressgroups.List(client, addressgroups.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := addressgroups.ExtractGroups(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item addressgroups.AddressGroup) string { return item.Name })
}

func findAddressScope(ctx context.Context, client *gophercloud.ServiceClient, value string) (*addressscopes.AddressScope, error) {
	result := addressscopes.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := addressscopes.List(client, addressscopes.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := addressscopes.ExtractAddressScopes(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item addressscopes.AddressScope) string { return item.Name })
}

func findRouter(ctx context.Context, client *gophercloud.ServiceClient, value string) (*routers.Router, error) {
	result := routers.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := routers.List(client, routers.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := routers.ExtractRouters(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item routers.Router) string { return item.Name })
}

func findNetworkSegment(ctx context.Context, client *gophercloud.ServiceClient, value string) (*segments.Segment, error) {
	result := segments.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := segments.List(client, segments.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := segments.ExtractSegments(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item segments.Segment) string { return item.Name })
}

func findNetworkTrunk(ctx context.Context, client *gophercloud.ServiceClient, value string) (*trunks.Trunk, error) {
	result := trunks.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := trunks.List(client, trunks.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := trunks.ExtractTrunks(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item trunks.Trunk) string { return item.Name })
}

func findNetworkQoSPolicy(ctx context.Context, client *gophercloud.ServiceClient, value string) (*qospolicies.Policy, error) {
	result := qospolicies.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := qospolicies.List(client, qospolicies.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := qospolicies.ExtractPolicies(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item qospolicies.Policy) string { return item.Name })
}

func findSecurityGroup(ctx context.Context, client *gophercloud.ServiceClient, value string) (*secgroups.SecGroup, error) {
	result := secgroups.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := secgroups.List(client, secgroups.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := secgroups.ExtractGroups(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item secgroups.SecGroup) string { return item.Name })
}

func findVolume(ctx context.Context, client *gophercloud.ServiceClient, value string) (*volumes.Volume, error) {
	result := volumes.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := volumes.List(client, volumes.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := volumes.ExtractVolumes(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item volumes.Volume) string { return item.Name })
}

func findVolumeBackup(ctx context.Context, client *gophercloud.ServiceClient, value string) (*backups.Backup, error) {
	result := backups.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := backups.List(client, backups.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := backups.ExtractBackups(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item backups.Backup) string { return item.Name })
}

func findVolumeTransfer(ctx context.Context, client *gophercloud.ServiceClient, value string) (*transfers.Transfer, error) {
	result := transfers.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := transfers.List(client, transfers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := transfers.ExtractTransfers(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item transfers.Transfer) string { return item.Name })
}

func findVolumeQoS(ctx context.Context, client *gophercloud.ServiceClient, value string) (*bsqos.QoS, error) {
	result := bsqos.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := bsqos.List(client, bsqos.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := bsqos.ExtractQoS(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item bsqos.QoS) string { return item.Name })
}

func findSubnet(ctx context.Context, client *gophercloud.ServiceClient, value string) (*subnets.Subnet, error) {
	result := subnets.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := subnets.List(client, subnets.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := subnets.ExtractSubnets(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item subnets.Subnet) string { return item.Name })
}

func findSubnetPool(ctx context.Context, client *gophercloud.ServiceClient, value string) (*subnetpools.SubnetPool, error) {
	result := subnetpools.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := subnetpools.List(client, subnetpools.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := subnetpools.ExtractSubnetPools(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item subnetpools.SubnetPool) string { return item.Name })
}

func findPort(ctx context.Context, client *gophercloud.ServiceClient, value string) (*ports.Port, error) {
	result := ports.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := ports.List(client, ports.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := ports.ExtractPorts(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item ports.Port) string { return item.Name })
}

func findFloatingIP(ctx context.Context, client *gophercloud.ServiceClient, value string) (*floatingips.FloatingIP, error) {
	result := floatingips.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := floatingips.List(client, floatingips.ListOpts{FloatingIP: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := floatingips.ExtractFloatingIPs(page)
	if err != nil {
		return nil, err
	}
	return singleMatch(value, items)
}

func findServerGroup(ctx context.Context, client *gophercloud.ServiceClient, value string) (*servergroups.ServerGroup, error) {
	result := servergroups.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := servergroups.List(client, servergroups.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := servergroups.ExtractServerGroups(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item servergroups.ServerGroup) string { return item.Name })
}

func findVolumeType(ctx context.Context, client *gophercloud.ServiceClient, value string) (*volumetypes.VolumeType, error) {
	result := volumetypes.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := volumetypes.List(client, volumetypes.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := volumetypes.ExtractVolumeTypes(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item volumetypes.VolumeType) string { return item.Name })
}

func findVolumeSnapshot(ctx context.Context, client *gophercloud.ServiceClient, value string) (*snapshots.Snapshot, error) {
	result := snapshots.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := snapshots.ListDetail(client, snapshots.ListOpts{Name: value}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := snapshots.ExtractSnapshots(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item snapshots.Snapshot) string { return item.Name })
}

func extractServers(page pagination.Page) ([]servers.Server, error) {
	var items []servers.Server
	err := servers.ExtractServersInto(page, &items)
	return items, err
}

func serverImage(image map[string]any, imageNames map[string]string) any {
	if len(image) == 0 {
		return "N/A (booted from volume)"
	}
	if id, ok := image["id"].(string); ok && id != "" {
		if name := imageNames[id]; name != "" {
			return name
		}
	}
	if name, ok := image["name"].(string); ok && name != "" {
		return name
	}
	if id, ok := image["id"].(string); ok && id != "" {
		return id
	}
	return image
}

func serverFlavor(flavor map[string]any, flavorNames map[string]string) any {
	if len(flavor) == 0 {
		return ""
	}
	if id, ok := flavor["id"].(string); ok && id != "" {
		if name := flavorNames[id]; name != "" {
			return name
		}
	}
	if original, ok := flavor["original_name"].(string); ok && original != "" {
		return original
	}
	if name, ok := flavor["name"].(string); ok && name != "" {
		return name
	}
	if id, ok := flavor["id"].(string); ok && id != "" {
		return id
	}
	return flavor
}

func flavorNameMap(ctx context.Context, client *gophercloud.ServiceClient) map[string]string {
	if client == nil {
		return nil
	}
	page, err := flavors.ListDetail(client, nil).AllPages(ctx)
	if err != nil {
		return nil
	}
	items, err := flavors.ExtractFlavors(page)
	if err != nil {
		return nil
	}
	names := make(map[string]string, len(items))
	for _, item := range items {
		names[item.ID] = item.Name
	}
	return names
}

func imageNameMap(ctx context.Context, client *gophercloud.ServiceClient) map[string]string {
	if client == nil {
		return nil
	}
	page, err := images.List(client, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil
	}
	items, err := images.ExtractImages(page)
	if err != nil {
		return nil
	}
	names := make(map[string]string, len(items))
	for _, item := range items {
		names[item.ID] = item.Name
	}
	return names
}

func serverNetworks(addresses map[string]any) map[string][]string {
	networks := make(map[string][]string, len(addresses))
	for name, rawAddresses := range addresses {
		for _, rawAddress := range anySlice(rawAddresses) {
			if address, ok := rawAddress.(map[string]any); ok {
				if addr, ok := address["addr"].(string); ok && addr != "" {
					networks[name] = append(networks[name], addr)
				}
				continue
			}
			if addr, ok := rawAddress.(string); ok && addr != "" {
				networks[name] = append(networks[name], addr)
			}
		}
	}
	for _, values := range networks {
		sort.Strings(values)
	}
	return networks
}

func anySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		values := make([]any, len(typed))
		for i, item := range typed {
			values[i] = item
		}
		return values
	default:
		return nil
	}
}

func volumeAttachments(attachments []volumes.Attachment) []map[string]any {
	values := make([]map[string]any, 0, len(attachments))
	for _, item := range attachments {
		values = append(values, map[string]any{
			"id":            item.ID,
			"attachment_id": item.AttachmentID,
			"volume_id":     item.VolumeID,
			"server_id":     item.ServerID,
			"host_name":     item.HostName,
			"device":        item.Device,
			"attached_at":   item.AttachedAt.Format("2006-01-02T15:04:05.000000"),
		})
	}
	return values
}

func subnetDHCPFilter(opts *Options) (bool, bool) {
	if boolFlag(opts, "dhcp") {
		return true, true
	}
	if boolFlag(opts, "no-dhcp") {
		return false, true
	}
	return false, false
}

func firstFlag(opts *Options, names ...string) string {
	for _, name := range names {
		if value := flagValue(opts, name); value != "" {
			return value
		}
	}
	return ""
}

func mapValueOrEmpty(item map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			return value
		}
	}
	return ""
}

func mapValueString(item map[string]any, keys ...string) string {
	return valueString(mapValueOrEmpty(item, keys...))
}

func appendMapField(fields []outputField, item map[string]any, key string, name string) []outputField {
	value, ok := item[key]
	if !ok || value == nil {
		return fields
	}
	return append(fields, outputField{Name: name, Value: value})
}

func metadefResourceTypeNames(value any) []string {
	items := anySlice(value)
	names := make([]string, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case map[string]any:
			if name := valueString(typed["name"]); name != "" {
				names = append(names, name)
			}
		case map[string]string:
			if name := typed["name"]; name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

func resolveServiceNextURL(client *gophercloud.ServiceClient, next string) string {
	if strings.TrimSpace(next) == "" {
		return ""
	}
	parsedNext, err := url.Parse(next)
	if err != nil || parsedNext.IsAbs() {
		return next
	}
	parsedEndpoint, err := url.Parse(client.Endpoint)
	if err != nil {
		return next
	}
	return parsedEndpoint.ResolveReference(parsedNext).String()
}

func oscHTTPException(err error) error {
	if err == nil {
		return nil
	}
	if codeErr, ok := unexpectedResponseCode(err); ok {
		return formatOSCHTTPException(codeErr)
	}
	return err
}

func oscResourceNotFoundError(err error, resource string, id string) error {
	codeErr, ok := unexpectedResponseCode(err)
	if !ok || codeErr.Actual != http.StatusNotFound {
		return oscHTTPException(err)
	}
	message := openStackFaultMessage(codeErr.Body)
	if message == "" {
		message = strings.TrimSpace(string(codeErr.Body))
	}
	return fmt.Errorf("No %s found for %s: Client Error for url: %s, %s", resource, id, codeErr.URL, message)
}

func unexpectedResponseCode(err error) (gophercloud.ErrUnexpectedResponseCode, bool) {
	var valueErr gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &valueErr) {
		return valueErr, true
	}
	var ptrErr *gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &ptrErr) && ptrErr != nil {
		return *ptrErr, true
	}
	return gophercloud.ErrUnexpectedResponseCode{}, false
}

func formatOSCHTTPException(err gophercloud.ErrUnexpectedResponseCode) error {
	class := "Error"
	if err.Actual >= 500 {
		class = "Server Error"
	} else if err.Actual >= 400 {
		class = "Client Error"
	}
	message := openStackFaultMessage(err.Body)
	if message == "" {
		message = strings.TrimSpace(string(err.Body))
	}
	return fmt.Errorf("HttpException: %d: %s for url: %s, %s", err.Actual, class, err.URL, message)
}

func openStackFaultMessage(body []byte) string {
	var decoded map[string]map[string]any
	if err := json.Unmarshal(body, &decoded); err == nil {
		for _, value := range decoded {
			if message, ok := value["message"].(string); ok {
				return cleanOpenStackMessage(message)
			}
		}
	}
	var flat map[string]any
	if err := json.Unmarshal(body, &flat); err != nil {
		return ""
	}
	message, _ := flat["message"].(string)
	message = cleanOpenStackMessage(message)
	if message == "" {
		return ""
	}
	if code := valueString(flat["code"]); code != "" {
		return code + ": " + message
	}
	return message
}

func cleanOpenStackMessage(message string) string {
	for _, value := range []string{"<br />", "<br/>", "<br>"} {
		message = strings.ReplaceAll(message, value, "\n")
	}
	lines := strings.Split(message, "\n")
	compact := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			compact = append(compact, line)
		}
	}
	return strings.Join(compact, "\n")
}

func serverMigrationListColumns(microversion string, opts *Options) ([]string, []string) {
	columns := []string{
		"Source Node",
		"Dest Node",
		"Source Compute",
		"Dest Compute",
		"Dest Host",
		"Status",
		"Server UUID",
		"Old Flavor",
		"New Flavor",
		"Created At",
		"Updated At",
	}
	keys := []string{
		"source_node",
		"dest_node",
		"source_compute",
		"dest_compute",
		"dest_host",
		"status",
		"instance_uuid",
		"old_instance_type_id",
		"new_instance_type_id",
		"created_at",
		"updated_at",
	}
	if microversionAtLeast(microversion, "2.59") {
		columns = insertStringAt(columns, 0, "UUID")
		keys = insertStringAt(keys, 0, "uuid")
	}
	if microversionAtLeast(microversion, "2.23") {
		columns = insertStringAt(columns, 0, "Id")
		keys = insertStringAt(keys, 0, "id")
		index := len(columns) - 2
		columns = insertStringAt(columns, index, "Type")
		keys = insertStringAt(keys, index, "migration_type")
	}
	if microversionAtLeast(microversion, "2.80") {
		if flagValue(opts, "project") != "" {
			index := len(columns) - 2
			columns = insertStringAt(columns, index, "Project")
			keys = insertStringAt(keys, index, "project_id")
		}
		if flagValue(opts, "user") != "" {
			index := len(columns) - 2
			columns = insertStringAt(columns, index, "User")
			keys = insertStringAt(keys, index, "user_id")
		}
	}
	return columns, keys
}

func insertStringAt(values []string, index int, value string) []string {
	values = append(values, "")
	copy(values[index+1:], values[index:])
	values[index] = value
	return values
}

func imageMicroversionAtMost(current string, maximum string) string {
	if current == "" || microversionAtLeast(current, maximum) {
		return maximum
	}
	return current
}

func unixSecondsISO(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case float64:
		return time.Unix(int64(typed), 0).UTC().Format("2006-01-02T15:04:05")
	case int64:
		return time.Unix(typed, 0).UTC().Format("2006-01-02T15:04:05")
	case int:
		return time.Unix(int64(typed), 0).UTC().Format("2006-01-02T15:04:05")
	case json.Number:
		seconds, err := typed.Int64()
		if err == nil {
			return time.Unix(seconds, 0).UTC().Format("2006-01-02T15:04:05")
		}
	case string:
		seconds, err := strconv.ParseInt(typed, 10, 64)
		if err == nil {
			return time.Unix(seconds, 0).UTC().Format("2006-01-02T15:04:05")
		}
	}
	return value
}

func intFlag(opts *Options, name string) int {
	value := flagValue(opts, name)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func resolveNetworkID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findNetwork(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolveAddressScopeID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findAddressScope(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolveRouterID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findRouter(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolvePortID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findPort(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolveSubnetID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findSubnet(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolveServerID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findServer(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func resolveVolumeID(ctx context.Context, client *gophercloud.ServiceClient, value string) string {
	if client == nil || value == "" {
		return value
	}
	item, err := findVolume(ctx, client, value)
	if err != nil {
		return value
	}
	return item.ID
}

func fixedIPFilter(ctx context.Context, client *gophercloud.ServiceClient, value string) ports.FixedIPOpts {
	filter := ports.FixedIPOpts{}
	for _, part := range strings.Split(value, ",") {
		key, raw, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		raw = strings.TrimSpace(raw)
		switch key {
		case "subnet":
			filter.SubnetID = resolveSubnetID(ctx, client, raw)
		case "ip-address", "ip_address":
			filter.IPAddress = raw
		case "ip-substring", "ip_address_substr", "ip-substr":
			filter.IPAddressSubstr = raw
		}
	}
	return filter
}

func neutronAgentType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "bgp":
		return "BGP dynamic routing agent"
	case "dhcp":
		return "DHCP agent"
	case "open-vswitch", "openvswitch", "ovs":
		return "Open vSwitch agent"
	case "linux-bridge":
		return "Linux bridge agent"
	case "l3":
		return "L3 agent"
	case "metadata":
		return "Metadata agent"
	case "metering":
		return "Metering agent"
	default:
		return value
	}
}

func filterAgentsByNetwork(ctx context.Context, client *gophercloud.ServiceClient, items []agents.Agent, networkID string) []agents.Agent {
	if networkID == "" {
		return items
	}
	var filtered []agents.Agent
	for _, item := range items {
		networks, err := agents.ListDHCPNetworks(ctx, client, item.ID).Extract()
		if err != nil {
			continue
		}
		for _, network := range networks {
			if network.ID == networkID {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

func filterAgentsByRouter(ctx context.Context, client *gophercloud.ServiceClient, items []agents.Agent, routerID string) []agents.Agent {
	if routerID == "" {
		return items
	}
	var filtered []agents.Agent
	for _, item := range items {
		routers, err := agents.ListL3Routers(ctx, client, item.ID).Extract()
		if err != nil {
			continue
		}
		for _, router := range routers {
			if router.ID == routerID {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

func trunkSubports(items []trunks.Subport) []map[string]any {
	values := make([]map[string]any, 0, len(items))
	for _, item := range items {
		values = append(values, map[string]any{
			"port_id":           item.PortID,
			"segmentation_id":   item.SegmentationID,
			"segmentation_type": item.SegmentationType,
		})
	}
	return values
}

func qosRuleTypeDrivers(items []qosruletypes.Driver) []map[string]any {
	values := make([]map[string]any, 0, len(items))
	for _, item := range items {
		values = append(values, map[string]any{
			"name":                 item.Name,
			"supported_parameters": qosRuleTypeParameters(item.SupportedParameters),
		})
	}
	return values
}

func qosRuleTypeParameters(items []qosruletypes.SupportedParameter) []map[string]any {
	values := make([]map[string]any, 0, len(items))
	for _, item := range items {
		values = append(values, map[string]any{
			"parameter_name":   item.ParameterName,
			"parameter_type":   item.ParameterType,
			"parameter_values": item.ParameterValues,
		})
	}
	return values
}

func portFixedIPs(fixedIPs []ports.IP) []string {
	values := make([]string, 0, len(fixedIPs))
	for _, item := range fixedIPs {
		switch {
		case item.IPAddress != "" && item.SubnetID != "":
			values = append(values, fmt.Sprintf("%s, subnet_id=%s", item.IPAddress, item.SubnetID))
		case item.IPAddress != "":
			values = append(values, item.IPAddress)
		case item.SubnetID != "":
			values = append(values, fmt.Sprintf("subnet_id=%s", item.SubnetID))
		}
	}
	sort.Strings(values)
	return values
}

func volumeTypeMatches(item volumetypes.VolumeType, opts *Options) bool {
	property := flagValue(opts, "property")
	if property != "" {
		key, value, ok := strings.Cut(property, "=")
		if !ok || item.ExtraSpecs[strings.TrimSpace(key)] != strings.TrimSpace(value) {
			return false
		}
	}
	if boolFlag(opts, "multiattach") && !extraSpecTruthy(item.ExtraSpecs, "multiattach") {
		return false
	}
	if boolFlag(opts, "cacheable") && !extraSpecTruthy(item.ExtraSpecs, "cacheable") {
		return false
	}
	if boolFlag(opts, "replicated") && !extraSpecTruthy(item.ExtraSpecs, "replication_enabled") {
		return false
	}
	availabilityZone := flagValue(opts, "availability-zone")
	if availabilityZone != "" && !volumeTypeAvailabilityZoneMatches(item.ExtraSpecs, availabilityZone) {
		return false
	}
	return true
}

func volumeTypeIsPublic(item volumetypes.VolumeType) bool {
	return item.IsPublic || item.PublicAccess
}

func extraSpecTruthy(specs map[string]string, key string) bool {
	value := strings.TrimSpace(strings.ToLower(specs[key]))
	return value == "true" || value == "<is> true" || value == "yes" || value == "1"
}

func volumeTypeAvailabilityZoneMatches(specs map[string]string, availabilityZone string) bool {
	availabilityZone = strings.TrimSpace(availabilityZone)
	if availabilityZone == "" {
		return true
	}
	for key, value := range specs {
		if key == "RESKEY:availability_zones:"+availabilityZone || value == availabilityZone {
			return true
		}
	}
	return false
}

type computeServerVolumeAttachment struct {
	VolumeID                 string  `json:"volumeId"`
	ServerID                 string  `json:"serverId"`
	Device                   string  `json:"device"`
	Tag                      *string `json:"tag"`
	DeleteOnTermination      *bool   `json:"delete_on_termination"`
	AttachmentID             string  `json:"attachment_id"`
	BlockDeviceMappingUUID   string  `json:"bdm_uuid"`
	LegacyVolumeAttachmentID string  `json:"id"`
}

func extractComputeServerVolumeAttachments(page pagination.Page) ([]computeServerVolumeAttachment, error) {
	var body struct {
		VolumeAttachments []computeServerVolumeAttachment `json:"volumeAttachments"`
	}
	err := page.(volumeattach.VolumeAttachmentPage).ExtractInto(&body)
	return body.VolumeAttachments, err
}

func serverEventDetails(events *[]instanceactions.Event) []map[string]any {
	if events == nil {
		return nil
	}
	rows := make([]map[string]any, 0, len(*events))
	for _, event := range *events {
		rows = append(rows, map[string]any{
			"details":     nil,
			"event":       event.Event,
			"finish_time": oscTime(event.FinishTime),
			"host":        stringPtrValue(event.Host),
			"host_id":     stringPtrValue(event.HostID),
			"result":      event.Result,
			"start_time":  oscTime(event.StartTime),
			"traceback":   nilIfEmpty(event.Traceback),
		})
	}
	return rows
}

func usageDateRange(opts *Options) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -28)
	end := now.AddDate(0, 0, 1)
	var err error
	if value := flagValue(opts, "start"); value != "" {
		start, err = parseUsageDate("start", value)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if value := flagValue(opts, "end"); value != "" {
		end, err = parseUsageDate("end", value)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	return start, end, nil
}

func parseUsageDate(name string, value string) (time.Time, error) {
	parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s value: %s", name, value)
	}
	return parsed, nil
}

func parseRFC3339Flag(name string, value string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid %s value: %s", name, value)
}

func consoleURLOpts(opts *Options) remoteconsoles.CreateOpts {
	switch {
	case boolFlag(opts, "xvpvnc"):
		return remoteconsoles.CreateOpts{Protocol: remoteconsoles.ConsoleProtocolVNC, Type: remoteconsoles.ConsoleTypeXVPVNC}
	case boolFlag(opts, "spice"):
		return remoteconsoles.CreateOpts{Protocol: remoteconsoles.ConsoleProtocolSPICE, Type: remoteconsoles.ConsoleTypeSPICEHTML5}
	case boolFlag(opts, "spice-direct"):
		return remoteconsoles.CreateOpts{Protocol: remoteconsoles.ConsoleProtocolSPICE, Type: remoteconsoles.ConsoleType("spice-direct")}
	case boolFlag(opts, "rdp"):
		return remoteconsoles.CreateOpts{Protocol: remoteconsoles.ConsoleProtocolRDP, Type: remoteconsoles.ConsoleTypeRDPHTML5}
	case boolFlag(opts, "serial"):
		return remoteconsoles.CreateOpts{Protocol: remoteconsoles.ConsoleProtocolSerial, Type: remoteconsoles.ConsoleTypeSerial}
	case boolFlag(opts, "mks"):
		return remoteconsoles.CreateOpts{Protocol: remoteconsoles.ConsoleProtocolMKS, Type: remoteconsoles.ConsoleTypeWebMKS}
	default:
		return remoteconsoles.CreateOpts{Protocol: remoteconsoles.ConsoleProtocolVNC, Type: remoteconsoles.ConsoleTypeNoVNC}
	}
}

func usageProjectID(ctx context.Context, opts *Options, clients *openStackClients) (string, error) {
	if project := flagValue(opts, "project"); project != "" {
		identityClient, err := clients.identityV3()
		if err != nil {
			return "", err
		}
		item, err := findProject(ctx, identityClient, project)
		if err != nil {
			return "", err
		}
		return item.ID, nil
	}
	if projectID := currentProjectID(clients); projectID != "" {
		return projectID, nil
	}
	if projectID := os.Getenv("OS_PROJECT_ID"); projectID != "" {
		return projectID, nil
	}
	if projectID := os.Getenv("OS_TENANT_ID"); projectID != "" {
		return projectID, nil
	}
	return "", fmt.Errorf("usage show requires --project when the current token is not project-scoped")
}

func projectNameMap(ctx context.Context, clients *openStackClients) map[string]string {
	values := map[string]string{}
	identityClient, err := clients.identityV3()
	if err != nil {
		return values
	}
	page, err := projects.List(identityClient, projects.ListOpts{}).AllPages(ctx)
	if err != nil {
		return values
	}
	items, err := projects.ExtractProjects(page)
	if err != nil {
		return values
	}
	for _, item := range items {
		values[item.ID] = item.Name
	}
	return values
}

func outputWantsHumanProjectNames(opts *Options) bool {
	return opts == nil || opts.Format == "" || opts.Format == "table" || opts.Format == "pretty"
}

func usageProjectValue(projectID string, projectNames map[string]string) string {
	if name := projectNames[projectID]; name != "" {
		return name
	}
	return projectID
}

type usageServerOutput struct {
	Hours     usageFloat
	Flavor    string
	Instance  string
	Name      string
	ProjectID string
	MemoryMB  int
	LocalGB   int
	VCPUs     int
	StartedAt any
	EndedAt   any
	State     string
	Uptime    int
}

func usageServerOutputs(items []usage.ServerUsage) []usageServerOutput {
	rows := make([]usageServerOutput, 0, len(items))
	for _, item := range items {
		rows = append(rows, usageServerOutput{
			Hours:     usageFloat(item.Hours),
			Flavor:    item.Flavor,
			Instance:  item.InstanceID,
			Name:      item.Name,
			ProjectID: item.TenantID,
			MemoryMB:  item.MemoryMB,
			LocalGB:   item.LocalGB,
			VCPUs:     item.VCPUs,
			StartedAt: oscTime(item.StartedAt),
			EndedAt:   oscTime(item.EndedAt),
			State:     item.State,
			Uptime:    item.Uptime,
		})
	}
	return rows
}

type usageFloat float64

func (value usageFloat) MarshalJSON() ([]byte, error) {
	formatted := strconv.FormatFloat(float64(value), 'f', -1, 64)
	if !strings.ContainsAny(formatted, ".eE") {
		formatted += ".0"
	}
	return []byte(formatted), nil
}

func (item usageServerOutput) MarshalJSON() ([]byte, error) {
	fields := []struct {
		name  string
		value any
	}{
		{"hours", item.Hours},
		{"flavor", item.Flavor},
		{"instance_id", item.Instance},
		{"name", item.Name},
		{"project_id", item.ProjectID},
		{"memory_mb", item.MemoryMB},
		{"local_gb", item.LocalGB},
		{"vcpus", item.VCPUs},
		{"started_at", item.StartedAt},
		{"ended_at", item.EndedAt},
		{"state", item.State},
		{"uptime", item.Uptime},
		{"id", nil},
		{"name", item.Name},
		{"location", nil},
	}
	var builder strings.Builder
	builder.WriteByte('{')
	for i, field := range fields {
		if i > 0 {
			builder.WriteByte(',')
		}
		key, err := json.Marshal(field.name)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(field.value)
		if err != nil {
			return nil, err
		}
		builder.Write(key)
		builder.WriteByte(':')
		builder.Write(value)
	}
	builder.WriteByte('}')
	return []byte(builder.String()), nil
}

func boolPtrValue(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func isUUIDLike(value string) bool {
	cleaned := strings.ToLower(strings.Trim(strings.TrimPrefix(strings.TrimPrefix(value, "urn:"), "uuid:"), "{}"))
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	if len(cleaned) != 32 {
		return false
	}
	for _, char := range cleaned {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}

func computeClientWithMinimumMicroversion(ctx context.Context, client *gophercloud.ServiceClient, minimum string) (*gophercloud.ServiceClient, error) {
	if os.Getenv("OS_COMPUTE_API_VERSION") != "" {
		if !microversionAtLeast(client.Microversion, minimum) {
			return nil, fmt.Errorf("--os-compute-api-version %s or greater is required", minimum)
		}
		return client, nil
	}
	supported, err := osutils.GetSupportedMicroversions(ctx, client)
	if err != nil {
		client.Microversion = minimum
		return client, nil
	}
	max := fmt.Sprintf("%d.%d", supported.MaxMajor, supported.MaxMinor)
	if !microversionAtLeast(max, minimum) {
		return nil, fmt.Errorf("--os-compute-api-version %s or greater is required", minimum)
	}
	client.Microversion = max
	return client, nil
}

func computeClientWithMaximumMicroversion(ctx context.Context, client *gophercloud.ServiceClient, maximum string) (*gophercloud.ServiceClient, error) {
	if os.Getenv("OS_COMPUTE_API_VERSION") != "" {
		if microversionAtLeast(client.Microversion, maximum) {
			client.Microversion = maximum
		}
		return client, nil
	}
	supported, err := osutils.GetSupportedMicroversions(ctx, client)
	if err != nil {
		if microversionAtLeast(client.Microversion, maximum) {
			client.Microversion = maximum
		}
		return client, nil
	}
	maximumSupported := fmt.Sprintf("%d.%d", supported.MaxMajor, supported.MaxMinor)
	if microversionAtLeast(maximumSupported, maximum) {
		client.Microversion = maximum
	} else {
		client.Microversion = maximumSupported
	}
	return client, nil
}

func blockStorageClientWithMinimumMicroversion(ctx context.Context, client *gophercloud.ServiceClient, minimum string) (*gophercloud.ServiceClient, error) {
	if client.Microversion != "" {
		if !microversionAtLeast(client.Microversion, minimum) {
			return nil, fmt.Errorf("--os-volume-api-version %s or greater is required", minimum)
		}
		return client, nil
	}
	supported, err := discoverBlockStorageMicroversions(ctx, client)
	if err != nil {
		client.Microversion = minimum
		return client, nil
	}
	max := fmt.Sprintf("%d.%d", supported.MaxMajor, supported.MaxMinor)
	if !microversionAtLeast(max, minimum) {
		return nil, fmt.Errorf("--os-volume-api-version %s or greater is required", minimum)
	}
	client.Microversion = max
	return client, nil
}

func blockStorageClientWithExplicitMinimumMicroversion(client *gophercloud.ServiceClient, minimum string, command string) (*gophercloud.ServiceClient, error) {
	if client.Microversion == "" || !microversionAtLeast(client.Microversion, minimum) {
		return nil, fmt.Errorf("--os-volume-api-version %s or greater is required to support the '%s' command", minimum, command)
	}
	return client, nil
}

func discoverBlockStorageMicroversions(ctx context.Context, client *gophercloud.ServiceClient) (osutils.SupportedMicroversions, error) {
	endpoint := client.Endpoint
	if versioned, err := osutils.BaseVersionedEndpoint(client.Endpoint); err == nil && versioned != "" {
		endpoint = versioned
	}
	versions, err := osutils.GetServiceVersions(ctx, client.ProviderClient, endpoint, true)
	if err != nil {
		return osutils.SupportedMicroversions{}, err
	}
	for _, version := range versions {
		if version.Major == 3 {
			return version.SupportedMicroversions, nil
		}
	}
	return osutils.SupportedMicroversions{}, fmt.Errorf("block storage v3 microversions not found")
}

func microversionAtLeast(value string, minimum string) bool {
	valueMajor, valueMinor, err := osutils.ParseMicroversion(value)
	if err != nil {
		return false
	}
	minMajor, minMinor, err := osutils.ParseMicroversion(minimum)
	if err != nil {
		return false
	}
	if valueMajor != minMajor {
		return valueMajor > minMajor
	}
	return valueMinor >= minMinor
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func compactStrings(values []string) []string {
	var compacted []string
	for _, value := range values {
		if value != "" {
			compacted = append(compacted, value)
		}
	}
	return compacted
}
