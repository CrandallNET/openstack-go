package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
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
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/imageimport"
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
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
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
		case "address group create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return addressGroupCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "address group delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressGroupDelete(cmd.Context(), client, args)
		case "address group list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressGroupList(cmd.Context(), stdout, opts, client)
		case "address group set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressGroupSet(cmd.Context(), opts, client, args)
		case "address group show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressGroupShow(cmd.Context(), stdout, opts, client, args)
		case "address group unset":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressGroupUnset(cmd.Context(), opts, client, args)
		case "address scope create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return addressScopeCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "address scope delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressScopeDelete(cmd.Context(), client, args)
		case "address scope list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressScopeList(cmd.Context(), stdout, opts, client)
		case "address scope set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return addressScopeSet(cmd.Context(), opts, client, args)
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
		case "cached image clear":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return cachedImageClear(cmd.Context(), opts, client)
		case "cached image delete":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return cachedImageDelete(cmd.Context(), cmd.ErrOrStderr(), opts, client, args)
		case "cached image list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return cachedImageList(cmd.Context(), stdout, opts, client)
		case "cached image queue":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return cachedImageQueue(cmd.Context(), cmd.ErrOrStderr(), opts, client, args)
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
		case "container create":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return containerCreate(cmd.Context(), stdout, opts, client, args)
		case "container delete":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return containerDelete(cmd.Context(), opts, client, args)
		case "container set":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return containerSet(cmd.Context(), opts, client, args)
		case "container show":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return containerShow(cmd.Context(), stdout, opts, client, args)
		case "container unset":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return containerUnset(cmd.Context(), opts, client, args)
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
		case "image create":
			imageClient, err := clients.imageV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			volumeClient, err := clients.blockStorageV3()
			if err != nil {
				volumeClient = nil
			}
			return imageCreate(cmd.Context(), stdout, opts, imageClient, identityClient, volumeClient, args)
		case "image delete":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageDelete(cmd.Context(), cmd.ErrOrStderr(), opts, client, args)
		case "image import":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageImport(cmd.Context(), stdout, opts, client, args)
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
		case "image metadef namespace create":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefNamespaceCreate(cmd.Context(), stdout, opts, client, args)
		case "image metadef namespace delete":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefNamespaceDelete(cmd.Context(), opts, client, args)
		case "image metadef namespace list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefNamespaceList(cmd.Context(), stdout, opts, client)
		case "image metadef namespace set":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefNamespaceSet(cmd.Context(), opts, client, args)
		case "image metadef namespace show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefNamespaceShow(cmd.Context(), stdout, opts, client, args)
		case "image metadef object create":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefObjectCreate(cmd.Context(), stdout, opts, client, args)
		case "image metadef object delete":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefObjectDelete(cmd.Context(), client, args)
		case "image metadef object list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefObjectList(cmd.Context(), stdout, opts, client, args)
		case "image metadef object property show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefObjectPropertyShow(cmd.Context(), stdout, opts, client, args)
		case "image metadef object show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefObjectShow(cmd.Context(), stdout, opts, client, args)
		case "image metadef object update":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefObjectUpdate(cmd.Context(), opts, client, args)
		case "image metadef property create":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefPropertyCreate(cmd.Context(), stdout, opts, client, args)
		case "image metadef property delete":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefPropertyDelete(cmd.Context(), client, args)
		case "image metadef property list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefPropertyList(cmd.Context(), stdout, opts, client, args)
		case "image metadef property set":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefPropertySet(cmd.Context(), opts, client, args)
		case "image metadef property show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefPropertyShow(cmd.Context(), stdout, opts, client, args)
		case "image metadef resource type association create":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefResourceTypeAssociationCreate(cmd.Context(), stdout, opts, client, args)
		case "image metadef resource type association delete":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefResourceTypeAssociationDelete(cmd.Context(), opts, client, args)
		case "image metadef resource type association list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefResourceTypeAssociationList(cmd.Context(), stdout, opts, client, args)
		case "image metadef resource type list":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageMetadefResourceTypeList(cmd.Context(), stdout, opts, client)
		case "image import info":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageImportInfo(cmd.Context(), stdout, opts, client)
		case "image add project":
			imageClient, err := clients.imageV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return imageAddProject(cmd.Context(), stdout, opts, imageClient, identityClient, args)
		case "image remove project":
			imageClient, err := clients.imageV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return imageRemoveProject(cmd.Context(), opts, imageClient, identityClient, args)
		case "image save":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageSave(cmd.Context(), stdout, opts, client, args)
		case "image set":
			imageClient, err := clients.imageV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return imageSet(cmd.Context(), opts, imageClient, identityClient, args)
		case "image show":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageShow(cmd.Context(), stdout, opts, client, args)
		case "image stage":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageStage(cmd.Context(), opts, client, args)
		case "image unset":
			client, err := clients.imageV2()
			if err != nil {
				return err
			}
			return imageUnset(cmd.Context(), cmd.ErrOrStderr(), opts, client, args)
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
		case "network create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return networkCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "network delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkDelete(cmd.Context(), client, args)
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
		case "network set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkSet(cmd.Context(), opts, client, args)
		case "network show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkShow(cmd.Context(), stdout, opts, client, args)
		case "network unset":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkUnset(cmd.Context(), opts, client, args)
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
		case "object create":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectCreate(cmd.Context(), stdout, opts, client, args)
		case "object delete":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectDelete(cmd.Context(), client, args)
		case "object save":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectSave(cmd.Context(), stdout, opts, client, args)
		case "object set":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectSet(cmd.Context(), opts, client, args)
		case "object show":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectShow(cmd.Context(), stdout, opts, client, args)
		case "object unset":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectUnset(cmd.Context(), opts, client, args)
		case "object store account show":
			client, err := clients.objectStorageV1()
			if err != nil {
				return err
			}
			return objectStoreAccountShow(cmd.Context(), stdout, opts, client)
		case "quota list":
			return quotaList(cmd.Context(), stdout, opts, clients)
		case "quota delete":
			return quotaDelete(cmd.Context(), opts, clients, args)
		case "quota set":
			return quotaSet(cmd.Context(), opts, clients, args)
		case "quota show":
			return quotaShow(cmd.Context(), stdout, opts, clients, args)
		case "keypair list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return keypairList(cmd.Context(), stdout, opts, client)
		case "keypair create":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return keypairCreate(cmd.Context(), stdout, opts, clients, client, args)
		case "keypair delete":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return keypairDelete(cmd.Context(), opts, clients, client, args)
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
		case "security group create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return securityGroupCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "security group delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupDelete(cmd.Context(), client, args)
		case "security group set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupSet(cmd.Context(), opts, client, args)
		case "security group show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupShow(cmd.Context(), stdout, opts, client, args)
		case "security group unset":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupUnset(cmd.Context(), opts, client, args)
		case "security group rule create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return securityGroupRuleCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "security group rule delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return securityGroupRuleDelete(cmd.Context(), client, args)
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
			client, err = computeClientWithMaximumMicroversion(cmd.Context(), client, "2.64")
			if err != nil {
				return err
			}
			return serverGroupList(cmd.Context(), stdout, opts, client)
		case "server group create":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			client, err = computeClientWithMaximumMicroversion(cmd.Context(), client, "2.64")
			if err != nil {
				return err
			}
			return serverGroupCreate(cmd.Context(), stdout, opts, client, args)
		case "server group delete":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			client, err = computeClientWithMaximumMicroversion(cmd.Context(), client, "2.64")
			if err != nil {
				return err
			}
			return serverGroupDelete(cmd.Context(), client, args)
		case "server group show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			client, err = computeClientWithMaximumMicroversion(cmd.Context(), client, "2.64")
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
		case "subnet pool create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return subnetPoolCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "subnet pool delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetPoolDelete(cmd.Context(), client, args)
		case "subnet pool list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetPoolList(cmd.Context(), stdout, opts, client)
		case "subnet pool set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetPoolSet(cmd.Context(), opts, client, args)
		case "subnet pool show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetPoolShow(cmd.Context(), stdout, opts, client, args)
		case "subnet pool unset":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetPoolUnset(cmd.Context(), opts, client, args)
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

func imageImportInfo(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	info, err := imageimport.Get(ctx, client).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{{Name: "import-methods", Value: info.ImportMethods.Value}})
}

type imageCreateOpts struct {
	images.CreateOpts
	Owner string
}

func (opts imageCreateOpts) ToImageCreateMap() (map[string]any, error) {
	body, err := opts.CreateOpts.ToImageCreateMap()
	if err != nil {
		return nil, err
	}
	if opts.Owner != "" {
		body["owner"] = opts.Owner
	}
	return body, nil
}

func imageCreate(ctx context.Context, stdout io.Writer, opts *Options, imageClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image create requires <image-name>")
	}
	if flagChanged(opts, "file") && flagChanged(opts, "volume") {
		return fmt.Errorf("argument --volume: not allowed with argument --file")
	}
	if volume := flagValue(opts, "volume"); volume != "" {
		return imageCreateFromVolume(ctx, stdout, opts, volumeClient, args[0], volume)
	}
	if flagValue(opts, "sign-key-path") != "" || flagValue(opts, "sign-cert-id") != "" {
		if flagValue(opts, "file") == "" {
			return fmt.Errorf("signing an image requires the --file option, passing files via stdin when signing is not supported.")
		}
		if flagValue(opts, "sign-key-path") == "" || flagValue(opts, "sign-cert-id") == "" {
			return fmt.Errorf("'sign-key-path' and 'sign-cert-id' must both be specified when attempting to sign an image.")
		}
		return fmt.Errorf("image signing is not yet implemented")
	}

	data, closeData, err := imageCreateDataReader(opts)
	if err != nil {
		return err
	}
	if closeData != nil {
		defer closeData.Close()
	}

	createOpts, err := buildImageCreateOpts(ctx, opts, identityClient, args[0])
	if err != nil {
		return err
	}
	item, err := images.Create(ctx, imageClient, createOpts).Extract()
	if err != nil {
		return err
	}

	if data != nil {
		if boolFlag(opts, "import") {
			if err := imagedata.Stage(ctx, imageClient, item.ID, data).ExtractErr(); err != nil {
				return err
			}
			if err := imageimport.Create(ctx, imageClient, item.ID, imageimport.CreateOpts{Name: imageimport.GlanceDirectMethod}).ExtractErr(); err != nil {
				return err
			}
		} else {
			if err := imagedata.Upload(ctx, imageClient, item.ID, data).ExtractErr(); err != nil {
				return err
			}
		}
	}

	refreshed, err := images.Get(ctx, imageClient, item.ID).Extract()
	if err == nil {
		item = refreshed
	}
	return renderShowOutput(stdout, opts, imageCreateFields(item))
}

func imageCreateFromVolume(ctx context.Context, stdout io.Writer, opts *Options, volumeClient *gophercloud.ServiceClient, imageName string, volumeNameOrID string) error {
	if volumeClient == nil {
		return fmt.Errorf("volume service is required for image create --volume")
	}
	data, closeData, err := imageCreateDataReader(opts)
	if err != nil {
		return err
	}
	if closeData != nil {
		defer closeData.Close()
	}
	if data != nil {
		return fmt.Errorf("Uploading data and using container are not allowed at the same time")
	}
	volumeClient, err = blockStorageClientForImageUpload(ctx, volumeClient, opts)
	if err != nil {
		return err
	}
	volume, err := findVolume(ctx, volumeClient, volumeNameOrID)
	if err != nil {
		return err
	}
	uploadOpts := volumes.UploadImageOpts{
		ImageName:       imageName,
		ContainerFormat: imageCreateContainerFormat(opts),
		DiskFormat:      imageCreateDiskFormat(opts),
		Force:           boolFlag(opts, "force"),
	}
	if visibility, err := imageCreateVisibility(opts); err != nil {
		return err
	} else if visibility != nil {
		uploadOpts.Visibility = string(*visibility)
	} else if volumeClient.Microversion != "" && microversionAtLeast(volumeClient.Microversion, "3.1") {
		uploadOpts.Visibility = "private"
	}
	if protected, err := imageCreateProtected(opts); err != nil {
		return err
	} else if protected != nil {
		uploadOpts.Protected = *protected
	}
	item, err := volumes.UploadImage(ctx, volumeClient, volume.ID, uploadOpts).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, imageCreateVolumeFields(item))
}

func buildImageCreateOpts(ctx context.Context, opts *Options, identityClient *gophercloud.ServiceClient, name string) (imageCreateOpts, error) {
	visibility, err := imageCreateVisibility(opts)
	if err != nil {
		return imageCreateOpts{}, err
	}
	protected, err := imageCreateProtected(opts)
	if err != nil {
		return imageCreateOpts{}, err
	}
	properties, err := imageCreateProperties(opts)
	if err != nil {
		return imageCreateOpts{}, err
	}
	if properties == nil {
		properties = map[string]string{}
	}
	properties["owner_specified.openstack.md5"] = ""
	properties["owner_specified.openstack.object"] = "images/" + name
	properties["owner_specified.openstack.sha256"] = ""
	createOpts := imageCreateOpts{
		CreateOpts: images.CreateOpts{
			Name:            name,
			ID:              flagValue(opts, "id"),
			ContainerFormat: imageCreateContainerFormat(opts),
			DiskFormat:      imageCreateDiskFormat(opts),
			Visibility:      visibility,
			Protected:       protected,
			Tags:            flagValues(opts, "tag"),
			Properties:      properties,
		},
	}
	if value := flagValue(opts, "min-disk"); value != "" {
		createOpts.MinDisk, _ = strconv.Atoi(value)
	}
	if value := flagValue(opts, "min-ram"); value != "" {
		createOpts.MinRAM, _ = strconv.Atoi(value)
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return imageCreateOpts{}, err
		}
		createOpts.Owner = project.ID
	}
	return createOpts, nil
}

func imageDelete(ctx context.Context, stderr io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image delete requires <image> [<image> ...]")
	}
	failures := 0
	store := flagValue(opts, "store")
	for _, imageArg := range args {
		image, err := findImage(ctx, client, imageArg)
		if err == nil {
			if store != "" {
				resp, deleteErr := client.Delete(ctx, client.ServiceURL("stores", url.PathEscape(store), url.PathEscape(image.ID)), nil)
				_, _, err = gophercloud.ParseResponse(resp, deleteErr)
				if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
					return fmt.Errorf("Multi Backend support not enabled.")
				}
			} else {
				err = images.Delete(ctx, client, image.ID).ExtractErr()
			}
		} else if strings.Contains(err.Error(), "no resource found") {
			return fmt.Errorf("Multi Backend support not enabled.")
		}
		if err != nil {
			failures++
			fmt.Fprintf(stderr, "Failed to delete image with name or ID '%s': %s\n", imageArg, err)
		}
	}
	if failures > 0 {
		return fmt.Errorf("Failed to delete %d of %d images.", failures, len(args))
	}
	return nil
}

func imageSave(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image save requires <image>")
	}
	image, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	body, err := imagedata.Download(ctx, client, image.ID).Extract()
	if err != nil {
		return err
	}
	defer body.Close()

	output := stdout
	var file *os.File
	if filename := flagValue(opts, "file"); filename != "" {
		file, err = os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		output = file
	}
	chunkSize := 1024
	if value := flagValue(opts, "chunk-size"); value != "" {
		if parsed, parseErr := strconv.Atoi(value); parseErr == nil && parsed > 0 {
			chunkSize = parsed
		}
	}
	_, err = io.CopyBuffer(output, body, make([]byte, chunkSize))
	return err
}

func imageStage(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image stage requires <image>")
	}
	image, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	if image.Status != images.ImageStatusQueued {
		return fmt.Errorf("Image stage is only possible for images in the queued state. Current state is %s", image.Status)
	}
	data, closeData, err := imageCreateDataReader(opts)
	if err != nil {
		return err
	}
	if closeData != nil {
		defer closeData.Close()
	}
	if data == nil {
		data = bytes.NewReader(nil)
	}
	return imagedata.Stage(ctx, client, image.ID, data).ExtractErr()
}

func imageImport(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image import requires <image>")
	}
	info, err := imageimport.Get(ctx, client).Extract()
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return fmt.Errorf("The Image Import feature is not supported by this deployment")
		}
		return err
	}
	method := flagValue(opts, "method")
	if method == "" {
		method = "glance-direct"
	}
	if !stringInSlice(method, info.ImportMethods.Value) {
		return fmt.Errorf("The '%s' import method is not supported by this deployment. Supported: %s", method, strings.Join(info.ImportMethods.Value, ", "))
	}
	if method == "web-download" {
		uri := flagValue(opts, "uri")
		if uri == "" {
			return fmt.Errorf("The '--uri' option is required when using '--method=web-download'")
		}
		parsed, err := url.Parse(uri)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("'%s' is not a valid url", uri)
		}
	} else if flagValue(opts, "uri") != "" {
		return fmt.Errorf("The '--uri' option is only supported when using '--method=web-download'")
	}
	if method == "glance-download" {
		if flagValue(opts, "remote-region") == "" || flagValue(opts, "remote-image") == "" {
			return fmt.Errorf("The '--remote-region' and '--remote-image' options are required when using '--method=web-download'")
		}
	} else {
		if flagValue(opts, "remote-region") != "" {
			return fmt.Errorf("The '--remote-region' option is only supported when using '--method=glance-download'")
		}
		if flagValue(opts, "remote-image") != "" {
			return fmt.Errorf("The '--remote-image' option is only supported when using '--method=glance-download'")
		}
		if flagValue(opts, "remote-service-interface") != "" {
			return fmt.Errorf("The '--remote-service-interface' option is only supported when using '--method=glance-download'")
		}
	}
	stores := flagValues(opts, "store")
	if method == "copy-image" && len(stores) == 0 && !boolFlag(opts, "all-stores") {
		return fmt.Errorf("The '--stores' or '--all-stores' options are required when using '--method=copy-image'")
	}
	image, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	if image.ContainerFormat == "" && image.DiskFormat == "" {
		return fmt.Errorf("The 'container_format' and 'disk_format' properties must be set on an image before it can be imported")
	}
	switch method {
	case "glance-direct":
		if image.Status != images.ImageStatusUploading {
			return fmt.Errorf("The 'glance-direct' import method can only be used with an image in status 'uploading'")
		}
	case "web-download":
		if image.Status != images.ImageStatusQueued {
			return fmt.Errorf("The 'web-download' import method can only be used with an image in status 'queued'")
		}
	case "copy-image":
		if image.Status != images.ImageStatusActive {
			return fmt.Errorf("The 'copy-image' import method can only be used with an image in status 'active'")
		}
	}
	request := map[string]any{
		"method":                  map[string]any{"name": method},
		"all_stores_must_succeed": !imageImportAllowFailure(opts),
	}
	methodBody := request["method"].(map[string]any)
	if uri := flagValue(opts, "uri"); uri != "" {
		methodBody["uri"] = uri
	}
	if remoteRegion := flagValue(opts, "remote-region"); remoteRegion != "" {
		methodBody["glance_region"] = remoteRegion
	}
	if remoteImage := flagValue(opts, "remote-image"); remoteImage != "" {
		methodBody["glance_image_id"] = remoteImage
	}
	if remoteInterface := flagValue(opts, "remote-service-interface"); remoteInterface != "" {
		methodBody["glance_service_interface"] = remoteInterface
	}
	if boolFlag(opts, "all-stores") {
		request["all_stores"] = true
	}
	if len(stores) > 0 {
		request["stores"] = stores
	}
	resp, err := client.Post(ctx, client.ServiceURL("images", url.PathEscape(image.ID), "import"), request, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusAccepted}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, imageCreateFields(image))
}

func imageImportAllowFailure(opts *Options) bool {
	if boolFlag(opts, "disallow-failure") {
		return false
	}
	return true
}

func stringInSlice(value string, values []string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func imageSet(ctx context.Context, opts *Options, imageClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image set requires <image>")
	}
	image, err := findImage(ctx, imageClient, args[0])
	if err != nil {
		return err
	}
	if boolFlag(opts, "deactivate") && boolFlag(opts, "activate") {
		return fmt.Errorf("only one activation option may be specified")
	}
	if boolFlag(opts, "deactivate") {
		if err := imageAction(ctx, imageClient, image.ID, "deactivate"); err != nil {
			return err
		}
	}
	if boolFlag(opts, "activate") {
		if err := imageAction(ctx, imageClient, image.ID, "reactivate"); err != nil {
			return err
		}
	}

	projectID := ""
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		projectID = project.ID
	}
	membership, err := imageMembershipStatus(opts)
	if err != nil {
		return err
	}
	if membership != "" {
		if projectID == "" {
			return fmt.Errorf("image set membership requires --project")
		}
		if _, err := members.Update(ctx, imageClient, image.ID, projectID, members.UpdateOpts{Status: membership}).Extract(); err != nil {
			return err
		}
	}

	patches, err := imageSetPatches(opts, image, projectID, membership != "")
	if err != nil {
		return err
	}
	if len(patches) == 0 {
		return nil
	}
	_, err = images.Update(ctx, imageClient, image.ID, patches).Extract()
	return err
}

func imageSetPatches(opts *Options, image *images.Image, projectID string, membershipOnlyProject bool) (images.UpdateOpts, error) {
	patches := images.UpdateOpts{}
	if value := flagValue(opts, "name"); value != "" {
		patches = append(patches, images.ReplaceImageName{NewName: value})
	}
	if value := flagValue(opts, "min-disk"); value != "" {
		parsed, _ := strconv.Atoi(value)
		patches = append(patches, images.ReplaceImageMinDisk{NewMinDisk: parsed})
	}
	if value := flagValue(opts, "min-ram"); value != "" {
		parsed, _ := strconv.Atoi(value)
		patches = append(patches, images.ReplaceImageMinRam{NewMinRam: parsed})
	}
	if value := flagValue(opts, "container-format"); value != "" {
		patches = append(patches, images.UpdateImageProperty{Op: images.AddOp, Name: "container_format", Value: value})
	}
	if value := flagValue(opts, "disk-format"); value != "" {
		patches = append(patches, images.UpdateImageProperty{Op: images.AddOp, Name: "disk_format", Value: value})
	}
	if protected, err := imageCreateProtected(opts); err != nil {
		return nil, err
	} else if protected != nil {
		patches = append(patches, images.ReplaceImageProtected{NewProtected: *protected})
	}
	if visibility, err := imageCreateVisibility(opts); err != nil {
		return nil, err
	} else if visibility != nil {
		patches = append(patches, images.UpdateVisibility{Visibility: *visibility})
	}
	if hidden, ok, err := imageHiddenFlag(opts); err != nil {
		return nil, err
	} else if ok {
		patches = append(patches, images.ReplaceImageHidden{NewHidden: hidden})
	}
	if projectID != "" && !membershipOnlyProject {
		patches = append(patches, images.UpdateImageProperty{Op: images.AddOp, Name: "owner", Value: projectID})
	}
	tags := flagValues(opts, "tag")
	if len(tags) > 0 {
		patches = append(patches, images.ReplaceImageTags{NewTags: mergeImageTags(image.Tags, tags)})
	}
	properties, err := imageCreateProperties(opts)
	if err != nil {
		return nil, err
	}
	for key, value := range properties {
		patches = append(patches, images.UpdateImageProperty{Op: images.AddOp, Name: key, Value: value})
	}
	for _, field := range []string{"architecture", "instance-id", "instance-uuid", "kernel-id", "os-distro", "os-version", "ramdisk-id"} {
		if value := flagValue(opts, field); value != "" {
			patches = append(patches, images.UpdateImageProperty{Op: images.AddOp, Name: imagePropertyFieldName(field), Value: value})
		}
	}
	return patches, nil
}

func imageUnset(ctx context.Context, stderr io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image unset requires <image>")
	}
	image, err := findImage(ctx, client, args[0])
	if err != nil {
		return err
	}
	tagFailures := 0
	tags := flagValues(opts, "tag")
	for _, tag := range tags {
		resp, deleteErr := client.Delete(ctx, client.ServiceURL("images", url.PathEscape(image.ID), "tags", url.PathEscape(tag)), nil)
		_, _, err = gophercloud.ParseResponse(resp, deleteErr)
		if err != nil {
			tagFailures++
			fmt.Fprintf(stderr, "tag unset failed, '%s' is a nonexistent tag \n", tag)
		}
	}

	propertyFailures := 0
	properties := flagValues(opts, "property")
	patches := images.UpdateOpts{}
	for _, property := range properties {
		path, ok := imageUnsetPropertyPath(image, property)
		if !ok {
			propertyFailures++
			fmt.Fprintf(stderr, "property unset failed, '%s' is a nonexistent property \n", property)
			continue
		}
		patches = append(patches, images.UpdateImageProperty{Op: images.RemoveOp, Name: path})
	}
	if len(patches) > 0 {
		if _, err := images.Update(ctx, client, image.ID, patches).Extract(); err != nil {
			return err
		}
	}
	if tagFailures > 0 && propertyFailures > 0 {
		return fmt.Errorf("Failed to unset %d of %d tags,Failed to unset %d of %d properties.", tagFailures, len(tags), propertyFailures, len(properties))
	}
	if tagFailures > 0 {
		return fmt.Errorf("Failed to unset %d of %d tags.", tagFailures, len(tags))
	}
	if propertyFailures > 0 {
		return fmt.Errorf("Failed to unset %d of %d properties.", propertyFailures, len(properties))
	}
	return nil
}

func imageAction(ctx context.Context, client *gophercloud.ServiceClient, imageID string, action string) error {
	resp, err := client.Post(ctx, client.ServiceURL("images", url.PathEscape(imageID), "actions", action), nil, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func imageMembershipStatus(opts *Options) (string, error) {
	selected := []string{}
	if boolFlag(opts, "accept") {
		selected = append(selected, "accepted")
	}
	if boolFlag(opts, "reject") {
		selected = append(selected, "rejected")
	}
	if boolFlag(opts, "pending") {
		selected = append(selected, "pending")
	}
	if len(selected) > 1 {
		return "", fmt.Errorf("only one membership option may be specified")
	}
	if len(selected) == 0 {
		return "", nil
	}
	return selected[0], nil
}

func imageHiddenFlag(opts *Options) (bool, bool, error) {
	hidden := boolFlag(opts, "hidden")
	unhidden := boolFlag(opts, "unhidden")
	if hidden && unhidden {
		return false, false, fmt.Errorf("only one hidden option may be specified")
	}
	if hidden {
		return true, true, nil
	}
	if unhidden {
		return false, true, nil
	}
	return false, false, nil
}

func mergeImageTags(existing []string, additions []string) []string {
	seen := map[string]bool{}
	var merged []string
	for _, tag := range existing {
		if !seen[tag] {
			seen[tag] = true
			merged = append(merged, tag)
		}
	}
	for _, tag := range additions {
		if !seen[tag] {
			seen[tag] = true
			merged = append(merged, tag)
		}
	}
	sort.Strings(merged)
	return merged
}

func imagePropertyFieldName(flag string) string {
	switch flag {
	case "instance-id", "instance-uuid":
		return "instance_id"
	case "kernel-id":
		return "kernel_id"
	case "os-distro":
		return "os_distro"
	case "os-version":
		return "os_version"
	case "ramdisk-id":
		return "ramdisk_id"
	default:
		return flag
	}
}

func imageUnsetPropertyPath(image *images.Image, property string) (string, bool) {
	if _, ok := image.Properties[property]; ok {
		return property, true
	}
	attributeNames := map[string]bool{
		"architecture":     true,
		"container_format": true,
		"disk_format":      true,
		"instance_id":      true,
		"kernel_id":        true,
		"min_disk":         true,
		"min_ram":          true,
		"name":             true,
		"os_distro":        true,
		"os_hidden":        true,
		"os_version":       true,
		"owner":            true,
		"protected":        true,
		"ramdisk_id":       true,
		"visibility":       true,
	}
	if attributeNames[property] {
		return property, true
	}
	customNames := map[string]string{
		"is_hidden":                    "os_hidden",
		"is_protected":                 "protected",
		"hash_algo":                    "os_hash_algo",
		"hash_value":                   "os_hash_value",
		"needs_config_drive":           "img_config_drive",
		"needs_secure_boot":            "os_secure_boot",
		"is_hw_vif_multiqueue_enabled": "hw_vif_multiqueue_enabled",
		"is_hw_boot_menu_enabled":      "hw_boot_menu",
		"has_auto_disk_config":         "auto_disk_config",
	}
	if mapped, ok := customNames[property]; ok {
		return mapped, true
	}
	return "", false
}

func imageCreateDataReader(opts *Options) (io.Reader, io.Closer, error) {
	return imageCreateDataReaderForFile(flagValue(opts, "file"))
}

func imageCreateDataReaderForFile(filename string) (io.Reader, io.Closer, error) {
	if filename != "" {
		file, err := os.Open(filename)
		if err != nil {
			return nil, nil, fmt.Errorf("'%s' is not a valid file", filename)
		}
		return file, file, nil
	}
	_, err := os.Stdin.Stat()
	if err != nil {
		return nil, nil, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return os.Stdin, nil, nil
	}
	return nil, nil, nil
}

func imageCreateContainerFormat(opts *Options) string {
	if value := flagValue(opts, "container-format"); value != "" {
		return value
	}
	return "bare"
}

func imageCreateDiskFormat(opts *Options) string {
	if value := flagValue(opts, "disk-format"); value != "" {
		return value
	}
	return "raw"
}

func imageCreateVisibility(opts *Options) (*images.ImageVisibility, error) {
	choices := []string{"public", "private", "community", "shared"}
	var selected []string
	for _, choice := range choices {
		if boolFlag(opts, choice) {
			selected = append(selected, choice)
		}
	}
	if len(selected) > 1 {
		return nil, fmt.Errorf("only one image visibility option may be specified")
	}
	if len(selected) == 0 {
		return nil, nil
	}
	visibility := images.ImageVisibility(selected[0])
	return &visibility, nil
}

func imageCreateProtected(opts *Options) (*bool, error) {
	protected := boolFlag(opts, "protected")
	unprotected := boolFlag(opts, "unprotected")
	if protected && unprotected {
		return nil, fmt.Errorf("only one protected option may be specified")
	}
	if protected {
		value := true
		return &value, nil
	}
	if unprotected {
		value := false
		return &value, nil
	}
	return nil, nil
}

func imageCreateProperties(opts *Options) (map[string]string, error) {
	properties := map[string]string{}
	for _, property := range flagValues(opts, "property") {
		key, value, ok := strings.Cut(property, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid property %q, expected <key=value>", property)
		}
		properties[key] = value
	}
	if len(properties) == 0 {
		return nil, nil
	}
	return properties, nil
}

func imageCreateFields(item *images.Image) []outputField {
	properties := map[string]any{}
	for key, value := range item.Properties {
		if value == nil {
			continue
		}
		properties[key] = value
	}
	if _, ok := properties["os_hidden"]; !ok {
		properties["os_hidden"] = item.Hidden
	}
	fields := []outputField{}
	add := func(name string, value any, include bool) {
		if include {
			fields = append(fields, outputField{Name: name, Value: value})
		}
	}
	add("checksum", item.Checksum, item.Checksum != "")
	add("container_format", item.ContainerFormat, item.ContainerFormat != "")
	add("created_at", imageTime(item.CreatedAt), !item.CreatedAt.IsZero())
	add("disk_format", item.DiskFormat, item.DiskFormat != "")
	add("file", item.File, item.File != "")
	add("id", item.ID, item.ID != "")
	add("min_disk", item.MinDiskGigabytes, true)
	add("min_ram", item.MinRAMMegabytes, true)
	add("name", item.Name, item.Name != "")
	add("owner", item.Owner, item.Owner != "")
	if len(properties) > 0 {
		add("properties", properties, true)
	}
	add("protected", item.Protected, true)
	add("schema", item.Schema, item.Schema != "")
	add("size", item.SizeBytes, item.SizeBytes > 0 || item.Status == images.ImageStatusActive)
	add("status", string(item.Status), item.Status != "")
	tags := item.Tags
	if tags == nil {
		tags = []string{}
	}
	add("tags", tags, true)
	add("updated_at", imageTime(item.UpdatedAt), !item.UpdatedAt.IsZero())
	add("virtual_size", item.VirtualSize, item.VirtualSize > 0)
	add("visibility", string(item.Visibility), item.Visibility != "")
	return fields
}

func imageCreateVolumeFields(item volumes.VolumeImage) []outputField {
	volumeType := any(nil)
	if item.VolumeType.Name != "" {
		volumeType = item.VolumeType.Name
	}
	fields := []outputField{
		{"container_format", item.ContainerFormat},
		{"disk_format", item.DiskFormat},
		{"display_description", nilIfEmpty(item.Description)},
		{"id", item.VolumeID},
		{"image_id", item.ImageID},
		{"image_name", item.ImageName},
		{"protected", item.Protected},
		{"size", item.Size},
		{"status", item.Status},
		{"updated_at", imageTime(item.UpdatedAt)},
		{"visibility", nilIfEmpty(item.Visibility)},
		{"volume_type", volumeType},
	}
	sort.Slice(fields, func(i int, j int) bool { return fields[i].Name < fields[j].Name })
	return fields
}

func imageTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func blockStorageClientForImageUpload(ctx context.Context, client *gophercloud.ServiceClient, opts *Options) (*gophercloud.ServiceClient, error) {
	needsMicroversion := boolFlag(opts, "public") || boolFlag(opts, "private") || boolFlag(opts, "community") || boolFlag(opts, "shared") || boolFlag(opts, "protected") || boolFlag(opts, "unprotected")
	withMinimum, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.1")
	if err != nil {
		if needsMicroversion {
			return nil, fmt.Errorf("--os-volume-api-version 3.1 or greater is required to support the --public, --private, --community, --shared or --protected option.")
		}
		return client, nil
	}
	return withMinimum, nil
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

func cachedImageClear(ctx context.Context, opts *Options, client *gophercloud.ServiceClient) error {
	client.Microversion = imageMicroversionAtMost(client.Microversion, "2.14")
	headers := map[string]string{}
	if boolFlag(opts, "cache") {
		headers["x-image-cache-clear-target"] = "cache"
	}
	if boolFlag(opts, "queue") {
		headers["x-image-cache-clear-target"] = "queue"
	}
	resp, err := client.Delete(ctx, client.ServiceURL("cache"), &gophercloud.RequestOpts{MoreHeaders: headers})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return fmt.Errorf("Failed to clear image cache")
	}
	return nil
}

func cachedImageDelete(ctx context.Context, stderr io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("cached image delete requires <image> [<image> ...]")
	}
	client.Microversion = imageMicroversionAtMost(client.Microversion, "2.14")
	failures := 0
	for _, imageArg := range args {
		image, err := findImage(ctx, client, imageArg)
		if err == nil {
			resp, deleteErr := client.Delete(ctx, client.ServiceURL("cache", url.PathEscape(image.ID)), nil)
			_, _, err = gophercloud.ParseResponse(resp, deleteErr)
			if cachedImageDeleteIgnoreMissing(err) {
				err = nil
			}
		}
		if err != nil {
			failures++
			fmt.Fprintf(stderr, "Failed to delete image with name or ID '%s': %s\n", imageArg, oscCacheCommandError(err))
		}
	}
	if failures > 0 {
		return fmt.Errorf("Failed to delete %d of %d images.", failures, len(args))
	}
	return nil
}

func cachedImageQueue(ctx context.Context, stderr io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("cached image queue requires <image> [<image> ...]")
	}
	client.Microversion = imageMicroversionAtMost(client.Microversion, "2.14")
	failures := 0
	for _, imageArg := range args {
		image, err := findImage(ctx, client, imageArg)
		if err == nil {
			resp, queueErr := client.Put(ctx, client.ServiceURL("cache", url.PathEscape(image.ID)), nil, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusAccepted, http.StatusNoContent}})
			_, _, err = gophercloud.ParseResponse(resp, queueErr)
		}
		if err != nil {
			failures++
			fmt.Fprintf(stderr, "Failed to queue image with name or ID '%s': %s\n", imageArg, oscCacheCommandError(err))
		}
	}
	if failures > 0 {
		return fmt.Errorf("Failed to queue %d of %d images", failures, len(args))
	}
	return nil
}

func cachedImageDeleteIgnoreMissing(err error) bool {
	codeErr, ok := unexpectedResponseCode(err)
	return ok && codeErr.Actual == http.StatusNotFound
}

func oscCacheCommandError(err error) error {
	if codeErr, ok := unexpectedResponseCode(err); ok && codeErr.Actual == http.StatusNotFound {
		return fmt.Errorf("NotFoundException: 404: Client Error for url: %s, %s", codeErr.URL, openStackFaultMessage(codeErr.Body))
	}
	return err
}

func imageMetadefNamespaceCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef namespace create requires <namespace>")
	}
	request := imageMetadefNamespaceRequest(opts, args[0], true)
	var body map[string]any
	resp, err := client.Post(ctx, client.ServiceURL("metadefs", "namespaces"), request, &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	return renderShowOutput(stdout, opts, imageMetadefNamespaceFields(body))
}

func imageMetadefNamespaceDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef namespace delete requires <namespace> [<namespace> ...]")
	}
	failures := 0
	for _, namespace := range args {
		resp, err := client.Delete(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace)), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d namespace failed to delete.", failures, len(args))
	}
	return nil
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
	return renderShowOutput(stdout, opts, imageMetadefNamespaceFields(body))
}

func imageMetadefNamespaceSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef namespace set requires <namespace>")
	}
	request := imageMetadefNamespaceRequest(opts, args[0], false)
	resp, err := client.Put(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(args[0])), request, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	return nil
}

func imageMetadefNamespaceRequest(opts *Options, namespace string, includeCreateDefaults bool) map[string]any {
	request := map[string]any{"namespace": namespace}
	if value := flagValue(opts, "display-name"); value != "" {
		request["display_name"] = value
	}
	if value := flagValue(opts, "description"); value != "" {
		request["description"] = value
	}
	if boolFlag(opts, "public") {
		request["visibility"] = "public"
	}
	if boolFlag(opts, "private") {
		request["visibility"] = "private"
	}
	if boolFlag(opts, "protected") {
		request["protected"] = true
	}
	if boolFlag(opts, "unprotected") {
		request["protected"] = false
	}
	if includeCreateDefaults {
		return request
	}
	return request
}

func imageMetadefNamespaceFields(body map[string]any) []outputField {
	values := map[string]any{}
	for _, name := range []string{"created_at", "description", "display_name", "namespace", "owner", "protected", "updated_at", "visibility"} {
		if value, ok := body[name]; ok && value != nil {
			values[name] = value
		}
	}
	if associations, ok := body["resource_type_associations"]; ok && associations != nil {
		values["resource_type_associations"] = metadefResourceTypeNames(associations)
	}
	return sortedFieldsFromMap(values, true)
}

func imageMetadefObjectCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef object create requires <metadef-object-name>")
	}
	namespace := flagValue(opts, "namespace")
	if namespace == "" {
		return fmt.Errorf("image metadef object create requires --namespace <namespace>")
	}
	request := map[string]any{"name": args[0]}
	var body []byte
	resp, err := client.Post(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace), "objects"), request, nil, &gophercloud.RequestOpts{KeepResponseBody: true})
	reader, _, err := gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	if reader != nil {
		defer reader.Close()
		body, err = io.ReadAll(reader)
		if err != nil {
			return err
		}
	}
	item, err := orderedJSONTopObject(body)
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, imageMetadefObjectFields(item, namespace, args[0], false))
}

func imageMetadefObjectDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef object delete requires <namespace> [<object> ...]")
	}
	namespace := args[0]
	if len(args) == 1 {
		resp, err := client.Delete(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace), "objects"), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		return oscHTTPException(err)
	}
	failures := 0
	for _, objectName := range args[1:] {
		resp, err := client.Delete(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace), "objects", url.PathEscape(objectName)), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d object failed to delete.", failures, len(namespace))
	}
	return nil
}

func imageMetadefObjectList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef object list requires <namespace>")
	}
	requestURL := client.ServiceURL("metadefs", "namespaces", url.PathEscape(args[0]), "objects")
	var rows []outputRow
	for requestURL != "" {
		var body struct {
			Objects []map[string]any `json:"objects"`
			Next    string           `json:"next"`
		}
		resp, err := client.Get(ctx, requestURL, &body, nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
		for _, item := range body.Objects {
			rows = append(rows, outputRow{
				"name":        mapValueOrEmpty(item, "name"),
				"description": mapValueOrEmpty(item, "description"),
			})
		}
		requestURL = resolveServiceNextURL(client, body.Next)
	}
	return renderListOutput(stdout, opts, []string{"name", "description"}, rows)
}

func imageMetadefObjectShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image metadef object show requires <namespace> <object>")
	}
	item, err := imageMetadefObject(ctx, client, args[0], args[1])
	if err != nil {
		return oscResourceNotFoundError(err, "MetadefObject", "None")
	}
	return renderShowOutput(stdout, opts, imageMetadefObjectFields(item, args[0], args[1], true))
}

func imageMetadefObjectUpdate(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image metadef object update requires <namespace> <object>")
	}
	request := map[string]any{}
	if value := flagValue(opts, "name"); value != "" {
		request["name"] = value
	}
	resp, err := client.Put(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(args[0]), "objects", url.PathEscape(args[1])), request, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	return nil
}

func imageMetadefObjectFields(item orderedJSONObject, namespace string, objectName string, defaultRequiredEmpty bool) []outputField {
	required := orderedMapValueOrNil(item, "required")
	if defaultRequiredEmpty && required == nil {
		required = []any{}
	}
	return []outputField{
		{Name: "created_at", Value: orderedMapValueOrNil(item, "created_at")},
		{Name: "description", Value: orderedMapValueOrNil(item, "description")},
		{Name: "name", Value: orderedMapValueOrDefault(item, "name", objectName)},
		{Name: "namespace_name", Value: orderedMapValueOrDefault(item, "namespace_name", namespace)},
		{Name: "properties", Value: orderedMapValueOrNil(item, "properties")},
		{Name: "required", Value: required},
		{Name: "updated_at", Value: orderedMapValueOrNil(item, "updated_at")},
	}
}

func imageMetadefObjectPropertyShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("image metadef object property show requires <namespace_name> <object_name> <property>")
	}
	item, err := imageMetadefObject(ctx, client, args[0], args[1])
	if err != nil {
		return oscResourceNotFoundError(err, "MetadefObject", "None")
	}
	properties, ok := orderedMapValueAsObject(item, "properties")
	if !ok {
		return fmt.Errorf("Property %s not found in object %s.", args[2], args[1])
	}
	property, ok := orderedJSONValueAsObject(properties.values[args[2]])
	if !ok {
		return fmt.Errorf("Property %s not found in object %s.", args[2], args[1])
	}
	values := make(map[string]any, len(property.values)+1)
	for key, value := range property.values {
		values[key] = value
	}
	values["name"] = args[2]
	return renderShowOutput(stdout, opts, sortedFieldsFromMap(values, false))
}

func imageMetadefPropertyCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef property create requires <namespace>")
	}
	request, err := imageMetadefPropertyRequest(opts, true)
	if err != nil {
		return err
	}
	var body map[string]any
	resp, err := client.Post(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(args[0]), "properties"), request, &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	item := orderedJSONObject{values: map[string]any{}}
	for key, value := range body {
		item.keys = append(item.keys, key)
		item.values[key] = value
	}
	return renderShowOutput(stdout, opts, metadefPropertyFields(item, flagValue(opts, "name")))
}

func imageMetadefPropertyDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef property delete requires <namespace> [<property> ...]")
	}
	namespace := args[0]
	if len(args) == 1 {
		resp, err := client.Delete(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace), "properties"), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		return oscHTTPException(err)
	}
	failures := 0
	for _, property := range args[1:] {
		resp, err := client.Delete(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace), "properties", url.PathEscape(property)), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d properties failed to delete.", failures, len(namespace))
	}
	return nil
}

func imageMetadefPropertyList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef property list requires <namespace>")
	}
	body, err := readServiceJSONRaw(ctx, client, client.ServiceURL("metadefs", "namespaces", url.PathEscape(args[0]), "properties"))
	if err != nil {
		return oscHTTPException(err)
	}
	item, err := orderedJSONTopObject(body)
	if err != nil {
		return err
	}
	properties, _ := orderedMapValueAsObject(item, "properties")
	rows := make([]outputRow, 0, len(properties.keys))
	for _, name := range properties.keys {
		property, _ := orderedJSONValueAsObject(properties.values[name])
		rows = append(rows, outputRow{
			"name":  name,
			"title": orderedMapValueOrNil(property, "title"),
			"type":  orderedMapValueOrNil(property, "type"),
		})
	}
	return renderListOutput(stdout, opts, []string{"name", "title", "type"}, rows)
}

func imageMetadefPropertySet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image metadef property set requires <namespace> <property>")
	}
	current, err := readServiceJSONRaw(ctx, client, client.ServiceURL("metadefs", "namespaces", url.PathEscape(args[0]), "properties", url.PathEscape(args[1])))
	if err != nil {
		return oscResourceNotFoundError(err, "MetadefProperty", args[1])
	}
	item, err := orderedJSONTopObject(current)
	if err != nil {
		return err
	}
	request := make(map[string]any, len(item.values))
	for key, value := range item.values {
		request[key] = value
	}
	updates, err := imageMetadefPropertyRequest(opts, false)
	if err != nil {
		return err
	}
	for key, value := range updates {
		request[key] = value
	}
	resp, err := client.Put(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(args[0]), "properties", url.PathEscape(args[1])), request, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	return nil
}

func imageMetadefPropertyShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image metadef property show requires <namespace> <property>")
	}
	body, err := readServiceJSONRaw(ctx, client, client.ServiceURL("metadefs", "namespaces", url.PathEscape(args[0]), "properties", url.PathEscape(args[1])))
	if err != nil {
		return oscResourceNotFoundError(err, "MetadefProperty", args[1])
	}
	item, err := orderedJSONTopObject(body)
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, metadefPropertyFields(item, args[1]))
}

func imageMetadefPropertyRequest(opts *Options, requireCore bool) (map[string]any, error) {
	request := map[string]any{}
	for _, key := range []string{"name", "title", "type"} {
		value := flagValue(opts, key)
		if value == "" {
			if requireCore {
				return nil, fmt.Errorf("image metadef property create requires --%s", key)
			}
			continue
		}
		request[key] = value
	}
	schema := flagValue(opts, "schema")
	if schema == "" {
		if requireCore {
			return nil, fmt.Errorf("image metadef property create requires --schema")
		}
		return request, nil
	}
	var schemaValues map[string]any
	if err := json.Unmarshal([]byte(schema), &schemaValues); err != nil {
		return nil, fmt.Errorf("Failed to load JSON schema: %v", err)
	}
	for key, value := range schemaValues {
		request[key] = value
	}
	return request, nil
}

func imageMetadefObject(ctx context.Context, client *gophercloud.ServiceClient, namespace string, object string) (orderedJSONObject, error) {
	body, err := readServiceJSONRaw(ctx, client, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace), "objects", url.PathEscape(object)))
	if err != nil {
		return orderedJSONObject{}, err
	}
	return orderedJSONTopObject(body)
}

func metadefPropertyFields(item orderedJSONObject, propertyName string) []outputField {
	values := map[string]any{}
	addPropertyValue := func(outputName string, inputNames ...string) {
		for _, inputName := range inputNames {
			if value, ok := item.values[inputName]; ok && value != nil {
				values[outputName] = value
				return
			}
		}
	}
	addPropertyValue("namespace_name", "namespace_name")
	if value, ok := item.values["name"]; ok && value != nil {
		values["name"] = value
	} else if propertyName != "" {
		values["name"] = propertyName
	}
	addPropertyValue("type", "type")
	addPropertyValue("title", "title")
	addPropertyValue("description", "description")
	addPropertyValue("operators", "operators")
	addPropertyValue("default", "default")
	addPropertyValue("is_readonly", "is_readonly", "readonly")
	addPropertyValue("minimum", "minimum")
	addPropertyValue("maximum", "maximum")
	addPropertyValue("enum", "enum")
	addPropertyValue("pattern", "pattern")
	addPropertyValue("min_length", "min_length", "minLength")
	addPropertyValue("max_length", "max_length", "maxLength")
	addPropertyValue("items", "items")
	addPropertyValue("require_unique_items", "require_unique_items", "uniqueItems")
	addPropertyValue("min_items", "min_items", "minItems")
	addPropertyValue("max_items", "max_items", "maxItems")
	addPropertyValue("allow_additional_items", "allow_additional_items", "additionalItems")
	return sortedFieldsFromMap(values, true)
}

func imageMetadefResourceTypeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	rows, err := imageMetadefNameRows(ctx, client, client.ServiceURL("metadefs", "resource_types"), "resource_types")
	if err != nil {
		return err
	}
	return renderListOutput(stdout, opts, []string{"Name"}, rows)
}

func imageMetadefResourceTypeAssociationCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image metadef resource type association create requires <namespace> <name>")
	}
	request := map[string]any{
		"name": args[1],
	}
	if value := flagValue(opts, "properties-target"); value != "" {
		request["properties_target"] = value
	}
	var body map[string]any
	resp, err := client.Post(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(args[0]), "resource_types"), request, &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	fields := []outputField{
		{Name: "created_at", Value: mapValueOrEmpty(body, "created_at")},
		{Name: "id", Value: mapValueOrEmpty(body, "id", "name")},
		{Name: "name", Value: mapValueOrEmpty(body, "name")},
		{Name: "prefix", Value: nilIfMissing(body, "prefix")},
		{Name: "properties_target", Value: nilIfMissing(body, "properties_target")},
		{Name: "updated_at", Value: mapValueOrEmpty(body, "updated_at")},
	}
	return renderShowOutput(stdout, opts, fields)
}

func imageMetadefResourceTypeAssociationDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image metadef resource type association delete requires <metadef_namespace> <name> [<name> ...]")
	}
	failures := 0
	for _, resourceType := range args[1:] {
		if err := imageMetadefResourceTypeAssociationDeleteOne(ctx, opts, client, args[0], resourceType); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d resource type failed to delete.", failures, len(args[0]))
	}
	return nil
}

func imageMetadefResourceTypeAssociationDeleteOne(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, namespace string, resourceType string) error {
	wasProtected := false
	if boolFlag(opts, "force") {
		var namespaceBody map[string]any
		resp, err := client.Get(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace)), &namespaceBody, nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
		wasProtected = boolValue(namespaceBody["protected"])
		if wasProtected {
			if err := imageMetadefNamespaceProtected(ctx, client, namespace, false); err != nil {
				return err
			}
		}
	}
	if wasProtected {
		defer imageMetadefNamespaceProtected(ctx, client, namespace, true)
	}
	resp, err := client.Delete(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace), "resource_types", url.PathEscape(resourceType)), nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	return nil
}

func imageMetadefNamespaceProtected(ctx context.Context, client *gophercloud.ServiceClient, namespace string, protected bool) error {
	request := map[string]any{
		"namespace": namespace,
		"protected": protected,
	}
	resp, err := client.Put(ctx, client.ServiceURL("metadefs", "namespaces", url.PathEscape(namespace)), request, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	return nil
}

func imageMetadefResourceTypeAssociationList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image metadef resource type association list requires <metadef_namespace>")
	}
	rows, err := imageMetadefNameRows(ctx, client, client.ServiceURL("metadefs", "namespaces", args[0], "resource_types"), "resource_type_associations")
	if err != nil {
		return err
	}
	return renderListOutput(stdout, opts, []string{"Name"}, rows)
}

func imageMetadefNameRows(ctx context.Context, client *gophercloud.ServiceClient, requestURL string, key string) ([]outputRow, error) {
	var rows []outputRow
	for requestURL != "" {
		var body map[string]any
		resp, err := client.Get(ctx, requestURL, &body, nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return nil, oscHTTPException(err)
		}
		for _, item := range anySlice(body[key]) {
			if itemMap, ok := item.(map[string]any); ok {
				rows = append(rows, outputRow{"Name": mapValueOrEmpty(itemMap, "name")})
			}
		}
		requestURL = resolveServiceNextURL(client, valueString(body["next"]))
	}
	return rows, nil
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

func imageAddProject(ctx context.Context, stdout io.Writer, opts *Options, imageClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image add project requires <image> <project>")
	}
	image, err := findImage(ctx, imageClient, args[0])
	if err != nil {
		return err
	}
	project, err := findProjectWithDomain(ctx, identityClient, args[1], flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	item, err := members.Create(ctx, imageClient, image.ID, project.ID).Extract()
	if err != nil {
		return err
	}
	return renderImageMemberShow(stdout, opts, item)
}

func imageRemoveProject(ctx context.Context, opts *Options, imageClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("image remove project requires <image> <project>")
	}
	image, err := findImage(ctx, imageClient, args[0])
	if err != nil {
		return err
	}
	project, err := findProjectWithDomain(ctx, identityClient, args[1], flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	return members.Delete(ctx, imageClient, image.ID, project.ID).ExtractErr()
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
	return renderImageMemberShow(stdout, opts, item)
}

func renderImageMemberShow(stdout io.Writer, opts *Options, item *members.Member) error {
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

type addressGroupCreateOpts struct {
	Name        string
	Description string
	ProjectID   string
	Addresses   []string
	Extra       map[string]any
}

func (opts addressGroupCreateOpts) ToAddressGroupCreateMap() (map[string]any, error) {
	group := map[string]any{
		"name":      opts.Name,
		"addresses": opts.Addresses,
	}
	if opts.Description != "" {
		group["description"] = opts.Description
	}
	if opts.ProjectID != "" {
		group["project_id"] = opts.ProjectID
	}
	for key, value := range opts.Extra {
		group[key] = value
	}
	return map[string]any{"address_group": group}, nil
}

type addressGroupUpdateOpts struct {
	addressgroups.UpdateOpts
	Extra map[string]any
}

func (opts addressGroupUpdateOpts) ToAddressGroupUpdateMap() (map[string]any, error) {
	body, err := opts.UpdateOpts.ToAddressGroupUpdateMap()
	if err != nil {
		return nil, err
	}
	group, ok := body["address_group"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid address group update request body")
	}
	for key, value := range opts.Extra {
		group[key] = value
	}
	return body, nil
}

func addressGroupCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("address group create requires <name>")
	}
	addresses, err := normalizeAddressGroupAddresses(flagValues(opts, "address"))
	if err != nil {
		return err
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	createOpts := addressGroupCreateOpts{
		Name:      args[0],
		Addresses: addresses,
		Extra:     extra,
	}
	if description := flagValue(opts, "description"); description != "" {
		createOpts.Description = description
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		createOpts.ProjectID = project.ID
	}
	result := addressgroups.Create(ctx, networkClient, createOpts)
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if raw, ok := addressGroupRawFromBody(result.Body); ok {
		return renderShowOutput(stdout, opts, addressGroupRawFields(raw))
	}
	return renderShowOutput(stdout, opts, addressGroupFields(item))
}

func addressGroupDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("address group delete requires <address-group> [<address-group> ...]")
	}
	failures := 0
	for _, groupArg := range args {
		item, err := findAddressGroup(ctx, client, groupArg)
		if err != nil {
			failures++
			continue
		}
		if err := addressgroups.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d address groups failed to delete.", failures, len(args))
	}
	return nil
}

func addressGroupSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("address group set requires <address-group>")
	}
	item, err := findAddressGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	updateOpts := addressGroupUpdateOpts{Extra: extra}
	hasUpdate := len(extra) > 0
	if flagChanged(opts, "name") {
		name := flagValue(opts, "name")
		updateOpts.Name = &name
		hasUpdate = true
	}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		updateOpts.Description = &description
		hasUpdate = true
	}
	if hasUpdate {
		if _, err := addressgroups.Update(ctx, client, item.ID, updateOpts).Extract(); err != nil {
			return err
		}
	}
	addresses, err := normalizeAddressGroupAddresses(flagValues(opts, "address"))
	if err != nil {
		return err
	}
	if len(addresses) > 0 {
		if _, err := addressgroups.AddAddresses(ctx, client, item.ID, addressgroups.UpdateAddressesOpts{Addresses: addresses}).Extract(); err != nil {
			return err
		}
	}
	return nil
}

func addressGroupUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("address group unset requires <address-group>")
	}
	item, err := findAddressGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	addresses, err := normalizeAddressGroupAddresses(flagValues(opts, "address"))
	if err != nil {
		return err
	}
	if len(addresses) > 0 {
		if _, err := addressgroups.RemoveAddresses(ctx, client, item.ID, addressgroups.UpdateAddressesOpts{Addresses: addresses}).Extract(); err != nil {
			return err
		}
	}
	return nil
}

func normalizeAddressGroupAddresses(values []string) ([]string, error) {
	addresses := make([]string, 0, len(values))
	for _, value := range values {
		address, err := normalizeAddressGroupAddress(value)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, address)
	}
	return addresses, nil
}

func normalizeAddressGroupAddress(value string) (string, error) {
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.String(), nil
	}
	addr, err := netip.ParseAddr(value)
	if err != nil {
		return "", fmt.Errorf("invalid address %q", value)
	}
	if addr.Is4() {
		return addr.String() + "/32", nil
	}
	return addr.String() + "/128", nil
}

type networkCreateOpts struct {
	Values map[string]any
}

func (opts networkCreateOpts) ToNetworkCreateMap() (map[string]any, error) {
	return map[string]any{"network": opts.Values}, nil
}

type networkUpdateOpts struct {
	Values map[string]any
}

func (opts networkUpdateOpts) ToNetworkUpdateMap() (map[string]any, error) {
	return map[string]any{"network": opts.Values}, nil
}

func networkCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network create requires <name>")
	}
	if boolFlag(opts, "no-tag") && len(flagValues(opts, "tag")) > 0 {
		return fmt.Errorf("argument --no-tag: not allowed with argument --tag")
	}
	values, err := networkCreateValues(ctx, opts, networkClient, identityClient, args[0])
	if err != nil {
		return err
	}
	result := networks.Create(ctx, networkClient, networkCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	tags := flagValues(opts, "tag")
	if len(tags) > 0 {
		target, err := setNeutronResourceTags(ctx, networkClient, "networks", item.ID, item.Tags, tags, boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
		item.Tags = target
	}
	if raw, ok := networkRawFromBody(result.Body); ok {
		if len(tags) > 0 {
			raw["tags"] = item.Tags
		}
		return renderShowOutput(stdout, opts, networkRawFields(raw))
	}
	return renderShowOutput(stdout, opts, networkFields(item))
}

func networkCreateValues(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, name string) (map[string]any, error) {
	values := map[string]any{
		"name":           name,
		"admin_state_up": true,
	}
	if flagChanged(opts, "enable") && flagChanged(opts, "disable") {
		return nil, fmt.Errorf("argument --disable: not allowed with argument --enable")
	}
	if boolFlag(opts, "disable") {
		values["admin_state_up"] = false
	}
	if shared, err := networkBoolFlag(opts, "share", "no-share"); err != nil {
		return nil, err
	} else if shared != nil {
		values["shared"] = *shared
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return nil, err
		}
		values["project_id"] = project.ID
	}
	if err := networkApplyCommonValues(ctx, opts, networkClient, values, true); err != nil {
		return nil, err
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return nil, err
	}
	for key, value := range extra {
		values[key] = value
	}
	return values, nil
}

func networkDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network delete requires <network> [<network> ...]")
	}
	failures := 0
	for _, networkArg := range args {
		item, err := findNetwork(ctx, client, networkArg)
		if err != nil {
			failures++
			continue
		}
		if err := networks.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d networks failed to delete.", failures, len(args))
	}
	return nil
}

func networkSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network set requires <network>")
	}
	item, err := findNetwork(ctx, client, args[0])
	if err != nil {
		return err
	}
	values, err := networkSetValues(ctx, opts, client)
	if err != nil {
		return err
	}
	if len(values) > 0 {
		if _, err := networks.Update(ctx, client, item.ID, networkUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "no-tag") || len(flagValues(opts, "tag")) > 0 {
		_, err := setNeutronResourceTags(ctx, client, "networks", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
	}
	return nil
}

func networkSetValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient) (map[string]any, error) {
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if adminState, err := networkBoolFlag(opts, "enable", "disable"); err != nil {
		return nil, err
	} else if adminState != nil {
		values["admin_state_up"] = *adminState
	}
	if shared, err := networkBoolFlag(opts, "share", "no-share"); err != nil {
		return nil, err
	} else if shared != nil {
		values["shared"] = *shared
	}
	if err := networkApplyCommonValues(ctx, opts, client, values, false); err != nil {
		return nil, err
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return nil, err
	}
	for key, value := range extra {
		values[key] = value
	}
	return values, nil
}

func networkUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network unset requires <network>")
	}
	if len(flagValues(opts, "tag")) > 0 && boolFlag(opts, "all-tag") {
		return fmt.Errorf("argument --all-tag: not allowed with argument --tag")
	}
	item, err := findNetwork(ctx, client, args[0])
	if err != nil {
		return err
	}
	extra, err := parseUnsetExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	if len(extra) > 0 {
		if _, err := networks.Update(ctx, client, item.ID, networkUpdateOpts{Values: extra}).Extract(); err != nil {
			return err
		}
	}
	_, err = unsetNeutronResourceTags(ctx, client, "networks", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "all-tag"))
	return err
}

func networkApplyCommonValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, values map[string]any, create bool) error {
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if flagChanged(opts, "mtu") {
		mtu, err := strconv.Atoi(flagValue(opts, "mtu"))
		if err != nil {
			return fmt.Errorf("argument --mtu: invalid int value: %q", flagValue(opts, "mtu"))
		}
		values["mtu"] = mtu
	}
	if hints := flagValues(opts, "availability-zone-hint"); len(hints) > 0 {
		values["availability_zone_hints"] = hints
	}
	if portSecurity, err := networkBoolFlag(opts, "enable-port-security", "disable-port-security"); err != nil {
		return err
	} else if portSecurity != nil {
		values["port_security_enabled"] = *portSecurity
	}
	if external, err := networkBoolFlag(opts, "external", "internal"); err != nil {
		return err
	} else if external != nil {
		values["router:external"] = *external
	}
	if isDefault, err := networkBoolFlag(opts, "default", "no-default"); err != nil {
		return err
	} else if isDefault != nil {
		values["is_default"] = *isDefault
	}
	if qosPolicy := flagValue(opts, "qos-policy"); qosPolicy != "" {
		policy, err := findNetworkQoSPolicy(ctx, client, qosPolicy)
		if err != nil {
			return err
		}
		values["qos_policy_id"] = policy.ID
	}
	if boolFlag(opts, "no-qos-policy") {
		if flagChanged(opts, "qos-policy") {
			return fmt.Errorf("argument --no-qos-policy: not allowed with argument --qos-policy")
		}
		values["qos_policy_id"] = nil
	}
	if vlanTransparent, err := networkBoolFlag(opts, "transparent-vlan", "no-transparent-vlan"); err != nil {
		return err
	} else if vlanTransparent != nil {
		values["vlan_transparent"] = *vlanTransparent
	}
	if vlanQinQ, err := networkBoolFlag(opts, "qinq-vlan", "no-qinq-vlan"); err != nil {
		return err
	} else if vlanQinQ != nil {
		values["vlan_qinq"] = *vlanQinQ
	}
	if enabledBool(values, "vlan_transparent") && enabledBool(values, "vlan_qinq") {
		return fmt.Errorf("--transparent-vlan and --qinq-vlan can not be both enabled for the network.")
	}
	if providerType := flagValue(opts, "provider-network-type"); providerType != "" {
		values["provider:network_type"] = providerType
	}
	if physicalNetwork := flagValue(opts, "provider-physical-network"); physicalNetwork != "" {
		values["provider:physical_network"] = physicalNetwork
	}
	if segment := flagValue(opts, "provider-segment"); segment != "" {
		if create && flagValue(opts, "provider-network-type") == "" {
			return fmt.Errorf("--provider-segment requires --provider-network-type to be specified.")
		}
		values["provider:segmentation_id"] = segment
	}
	if flagChanged(opts, "dns-domain") {
		values["dns_domain"] = flagValue(opts, "dns-domain")
	}
	return nil
}

func networkBoolFlag(opts *Options, trueFlag string, falseFlag string) (*bool, error) {
	if flagChanged(opts, trueFlag) && flagChanged(opts, falseFlag) {
		return nil, fmt.Errorf("argument --%s: not allowed with argument --%s", falseFlag, trueFlag)
	}
	if boolFlag(opts, trueFlag) {
		value := true
		return &value, nil
	}
	if boolFlag(opts, falseFlag) {
		value := false
		return &value, nil
	}
	return nil, nil
}

func enabledBool(values map[string]any, key string) bool {
	value, ok := values[key].(bool)
	return ok && value
}

func networkList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	page, err := networks.List(client, networks.ListOpts{
		Name:      flagValue(opts, "name"),
		ProjectID: flagValue(opts, "project"),
		Status:    flagValue(opts, "status"),
	}).AllPages(ctx)
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
	if raw, err := neutronNetworkRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, networkRawFields(raw))
	}
	return renderShowOutput(stdout, opts, networkFields(item))
}

func neutronNetworkRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		Network map[string]any `json:"network"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("networks", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Network, nil
}

func networkRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		Network map[string]any `json:"network"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.Network == nil {
		return nil, false
	}
	return wrapper.Network, true
}

func networkRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"admin_state_up":            true,
		"availability_zone_hints":   true,
		"availability_zones":        true,
		"created_at":                true,
		"description":               true,
		"dns_domain":                true,
		"id":                        true,
		"ipv4_address_scope":        true,
		"ipv6_address_scope":        true,
		"is_default":                true,
		"location":                  true,
		"mtu":                       true,
		"name":                      true,
		"port_security_enabled":     true,
		"project_id":                true,
		"provider:network_type":     true,
		"provider:physical_network": true,
		"provider:segmentation_id":  true,
		"qos_policy_id":             true,
		"revision_number":           true,
		"router:external":           true,
		"segments":                  true,
		"shared":                    true,
		"status":                    true,
		"subnets":                   true,
		"tags":                      true,
		"tenant_id":                 true,
		"updated_at":                true,
		"vlan_qinq":                 true,
		"vlan_transparent":          true,
	}
	fields := []outputField{
		{"admin_state_up", raw["admin_state_up"]},
		{"availability_zone_hints", raw["availability_zone_hints"]},
		{"availability_zones", raw["availability_zones"]},
		{"created_at", raw["created_at"]},
		{"description", raw["description"]},
		{"dns_domain", raw["dns_domain"]},
		{"id", raw["id"]},
		{"ipv4_address_scope", raw["ipv4_address_scope"]},
		{"ipv6_address_scope", raw["ipv6_address_scope"]},
		{"is_default", raw["is_default"]},
		{"is_vlan_qinq", raw["vlan_qinq"]},
		{"is_vlan_transparent", raw["vlan_transparent"]},
		{"mtu", rawNumber(raw["mtu"])},
		{"name", raw["name"]},
		{"port_security_enabled", raw["port_security_enabled"]},
		{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		{"provider:network_type", raw["provider:network_type"]},
		{"provider:physical_network", raw["provider:physical_network"]},
		{"provider:segmentation_id", rawNumber(raw["provider:segmentation_id"])},
		{"qos_policy_id", raw["qos_policy_id"]},
		{"revision_number", rawNumber(raw["revision_number"])},
		{"router:external", raw["router:external"]},
		{"segments", raw["segments"]},
		{"shared", raw["shared"]},
		{"status", raw["status"]},
		{"subnets", raw["subnets"]},
		{"tags", raw["tags"]},
		{"updated_at", raw["updated_at"]},
	}
	var extras []string
	for key := range raw {
		if !known[key] {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		fields = append(fields, outputField{key, raw[key]})
	}
	return fields
}

func networkFields(item *networks.Network) []outputField {
	return []outputField{
		{"admin_state_up", item.AdminStateUp},
		{"availability_zone_hints", item.AvailabilityZoneHints},
		{"availability_zones", nil},
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"dns_domain", nil},
		{"id", item.ID},
		{"ipv4_address_scope", nil},
		{"ipv6_address_scope", nil},
		{"is_default", nil},
		{"is_vlan_qinq", nil},
		{"is_vlan_transparent", nil},
		{"mtu", nil},
		{"name", item.Name},
		{"port_security_enabled", nil},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"provider:network_type", nil},
		{"provider:physical_network", nil},
		{"provider:segmentation_id", nil},
		{"qos_policy_id", nil},
		{"revision_number", item.RevisionNumber},
		{"router:external", nil},
		{"segments", nil},
		{"shared", item.Shared},
		{"status", item.Status},
		{"subnets", item.Subnets},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	}
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
	if raw, err := neutronAddressGroupRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, addressGroupRawFields(raw))
	}
	return renderShowOutput(stdout, opts, addressGroupFields(item))
}

func neutronAddressGroupRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		AddressGroup map[string]any `json:"address_group"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("address-groups", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.AddressGroup, nil
}

func addressGroupRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		AddressGroup map[string]any `json:"address_group"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.AddressGroup == nil {
		return nil, false
	}
	return wrapper.AddressGroup, true
}

func addressGroupRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"addresses":       true,
		"created_at":      true,
		"description":     true,
		"id":              true,
		"location":        true,
		"name":            true,
		"project_id":      true,
		"revision_number": true,
		"tenant_id":       true,
		"updated_at":      true,
	}
	fields := []outputField{
		{"addresses", raw["addresses"]},
		{"created_at", raw["created_at"]},
		{"description", raw["description"]},
		{"id", raw["id"]},
		{"name", raw["name"]},
		{"project_id", raw["project_id"]},
		{"revision_number", raw["revision_number"]},
		{"updated_at", raw["updated_at"]},
	}
	var extras []string
	for key := range raw {
		if !known[key] {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		fields = append(fields, outputField{key, raw[key]})
	}
	return fields
}

func addressGroupFields(item *addressgroups.AddressGroup) []outputField {
	return []outputField{
		{"addresses", item.Addresses},
		{"description", item.Description},
		{"id", item.ID},
		{"name", item.Name},
		{"project_id", item.ProjectID},
	}
}

type addressScopeCreateOpts struct {
	Name      string
	ProjectID string
	IPVersion int
	Shared    *bool
	Extra     map[string]any
}

func (opts addressScopeCreateOpts) ToAddressScopeCreateMap() (map[string]any, error) {
	scope := map[string]any{
		"name":       opts.Name,
		"ip_version": opts.IPVersion,
	}
	if opts.ProjectID != "" {
		scope["project_id"] = opts.ProjectID
	}
	if opts.Shared != nil {
		scope["shared"] = *opts.Shared
	}
	for key, value := range opts.Extra {
		scope[key] = value
	}
	return map[string]any{"address_scope": scope}, nil
}

type addressScopeUpdateOpts struct {
	addressscopes.UpdateOpts
	Extra map[string]any
}

func (opts addressScopeUpdateOpts) ToAddressScopeUpdateMap() (map[string]any, error) {
	body, err := opts.UpdateOpts.ToAddressScopeUpdateMap()
	if err != nil {
		return nil, err
	}
	scope, ok := body["address_scope"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid address scope update request body")
	}
	for key, value := range opts.Extra {
		scope[key] = value
	}
	return body, nil
}

func addressScopeCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("address scope create requires <name>")
	}
	ipVersion := intFlag(opts, "ip-version")
	if ipVersion == 0 {
		ipVersion = 4
	}
	if ipVersion != 4 && ipVersion != 6 {
		return fmt.Errorf("argument --ip-version: invalid choice: %d (choose from 4, 6)", ipVersion)
	}
	shared, err := addressScopeSharedFlag(opts)
	if err != nil {
		return err
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	createOpts := addressScopeCreateOpts{
		Name:      args[0],
		IPVersion: ipVersion,
		Shared:    shared,
		Extra:     extra,
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		createOpts.ProjectID = project.ID
	}
	result := addressscopes.Create(ctx, networkClient, createOpts)
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if raw, ok := addressScopeRawFromBody(result.Body); ok {
		return renderShowOutput(stdout, opts, addressScopeRawFields(raw))
	}
	return renderShowOutput(stdout, opts, addressScopeFields(item))
}

func addressScopeDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("address scope delete requires <address-scope> [<address-scope> ...]")
	}
	failures := 0
	for _, scopeArg := range args {
		item, err := findAddressScope(ctx, client, scopeArg)
		if err != nil {
			failures++
			continue
		}
		if err := addressscopes.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d address scopes failed to delete.", failures, len(args))
	}
	return nil
}

func addressScopeSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("address scope set requires <address-scope>")
	}
	item, err := findAddressScope(ctx, client, args[0])
	if err != nil {
		return err
	}
	shared, err := addressScopeSharedFlag(opts)
	if err != nil {
		return err
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	updateOpts := addressScopeUpdateOpts{Extra: extra}
	hasUpdate := len(extra) > 0
	if flagChanged(opts, "name") {
		name := flagValue(opts, "name")
		updateOpts.Name = &name
		hasUpdate = true
	}
	if shared != nil {
		updateOpts.Shared = shared
		hasUpdate = true
	}
	if hasUpdate {
		if _, err := addressscopes.Update(ctx, client, item.ID, updateOpts).Extract(); err != nil {
			return err
		}
	}
	return nil
}

func addressScopeSharedFlag(opts *Options) (*bool, error) {
	if boolFlag(opts, "share") && boolFlag(opts, "no-share") {
		return nil, fmt.Errorf("argument --no-share: not allowed with argument --share")
	}
	if boolFlag(opts, "share") {
		shared := true
		return &shared, nil
	}
	if boolFlag(opts, "no-share") {
		shared := false
		return &shared, nil
	}
	return nil, nil
}

func addressScopeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	listOpts := addressscopes.ListOpts{
		Name:      flagValue(opts, "name"),
		ProjectID: flagValue(opts, "project"),
		IPVersion: intFlag(opts, "ip-version"),
	}
	shared, err := addressScopeSharedFlag(opts)
	if err != nil {
		return err
	}
	if shared != nil {
		listOpts.Shared = shared
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
	if raw, err := neutronAddressScopeRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, addressScopeRawFields(raw))
	}
	return renderShowOutput(stdout, opts, addressScopeFields(item))
}

func neutronAddressScopeRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		AddressScope map[string]any `json:"address_scope"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("address-scopes", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.AddressScope, nil
}

func addressScopeRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		AddressScope map[string]any `json:"address_scope"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.AddressScope == nil {
		return nil, false
	}
	return wrapper.AddressScope, true
}

func addressScopeRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"id":         true,
		"ip_version": true,
		"location":   true,
		"name":       true,
		"project_id": true,
		"shared":     true,
		"tenant_id":  true,
	}
	fields := []outputField{
		{"id", raw["id"]},
		{"ip_version", raw["ip_version"]},
		{"name", raw["name"]},
		{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		{"shared", raw["shared"]},
	}
	var extras []string
	for key := range raw {
		if !known[key] {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		fields = append(fields, outputField{key, raw[key]})
	}
	return fields
}

func addressScopeFields(item *addressscopes.AddressScope) []outputField {
	return []outputField{
		{"id", item.ID},
		{"ip_version", item.IPVersion},
		{"name", item.Name},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"shared", item.Shared},
	}
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

type securityGroupCreateOpts struct {
	secgroups.CreateOpts
	Extra map[string]any
}

func (opts securityGroupCreateOpts) ToSecGroupCreateMap() (map[string]any, error) {
	body, err := opts.CreateOpts.ToSecGroupCreateMap()
	if err != nil {
		return nil, err
	}
	if len(opts.Extra) == 0 {
		return body, nil
	}
	group, ok := body["security_group"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid security group create request body")
	}
	for key, value := range opts.Extra {
		group[key] = value
	}
	return body, nil
}

type securityGroupUpdateOpts struct {
	secgroups.UpdateOpts
	Extra map[string]any
}

func (opts securityGroupUpdateOpts) ToSecGroupUpdateMap() (map[string]any, error) {
	body, err := opts.UpdateOpts.ToSecGroupUpdateMap()
	if err != nil {
		return nil, err
	}
	group, ok := body["security_group"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid security group update request body")
	}
	for key, value := range opts.Extra {
		group[key] = value
	}
	return body, nil
}

func securityGroupCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group create requires <name>")
	}
	if flagChanged(opts, "stateful") && flagChanged(opts, "stateless") {
		return fmt.Errorf("argument --stateless: not allowed with argument --stateful")
	}
	if len(flagValues(opts, "tag")) > 0 && boolFlag(opts, "no-tag") {
		return fmt.Errorf("argument --no-tag: not allowed with argument --tag")
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	createOpts := securityGroupCreateOpts{
		CreateOpts: secgroups.CreateOpts{
			Name:        args[0],
			Description: args[0],
		},
		Extra: extra,
	}
	if flagChanged(opts, "description") {
		createOpts.Description = flagValue(opts, "description")
	}
	if flagChanged(opts, "stateful") {
		stateful := true
		createOpts.Stateful = &stateful
	}
	if flagChanged(opts, "stateless") {
		stateful := false
		createOpts.Stateful = &stateful
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		createOpts.ProjectID = project.ID
	}

	result := secgroups.Create(ctx, networkClient, createOpts)
	item, err := result.Extract()
	if err != nil {
		return err
	}
	raw, _ := securityGroupRawFromBody(result.Body)
	tags, err := setNeutronResourceTags(ctx, networkClient, "security-groups", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "no-tag"))
	if err != nil {
		return err
	}
	item.Tags = tags
	if raw != nil {
		raw["tags"] = tags
		return renderShowOutput(stdout, opts, securityGroupRawFields(raw))
	}
	return renderShowOutput(stdout, opts, securityGroupFields(item))
}

func securityGroupSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group set requires <group>")
	}
	if flagChanged(opts, "stateful") && flagChanged(opts, "stateless") {
		return fmt.Errorf("argument --stateless: not allowed with argument --stateful")
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	item, err := findSecurityGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	updateOpts := securityGroupUpdateOpts{Extra: extra}
	if flagChanged(opts, "name") {
		updateOpts.Name = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		description := flagValue(opts, "description")
		updateOpts.Description = &description
	}
	if flagChanged(opts, "stateful") {
		stateful := true
		updateOpts.Stateful = &stateful
	}
	if flagChanged(opts, "stateless") {
		stateful := false
		updateOpts.Stateful = &stateful
	}
	if _, err := secgroups.Update(ctx, client, item.ID, updateOpts).Extract(); err != nil {
		return err
	}
	_, err = setNeutronResourceTags(ctx, client, "security-groups", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "no-tag"))
	return err
}

func securityGroupUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group unset requires <group>")
	}
	if len(flagValues(opts, "tag")) > 0 && boolFlag(opts, "all-tag") {
		return fmt.Errorf("argument --all-tag: not allowed with argument --tag")
	}
	item, err := findSecurityGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	_, err = unsetNeutronResourceTags(ctx, client, "security-groups", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "all-tag"))
	return err
}

func securityGroupDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group delete requires <group> [<group> ...]")
	}
	failures := 0
	for _, groupArg := range args {
		item, err := findSecurityGroup(ctx, client, groupArg)
		if err != nil {
			failures++
			continue
		}
		if err := secgroups.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d security groups failed to delete.", failures, len(args))
	}
	return nil
}

func securityGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group show requires <group>")
	}
	item, err := findSecurityGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	if raw, err := neutronSecurityGroupRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, securityGroupRawFields(raw))
	}
	return renderShowOutput(stdout, opts, securityGroupFields(item))
}

func securityGroupFields(item *secgroups.SecGroup) []outputField {
	return []outputField{
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
	}
}

func neutronSecurityGroupRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		SecurityGroup map[string]any `json:"security_group"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("security-groups", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.SecurityGroup, nil
}

func securityGroupRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		SecurityGroup map[string]any `json:"security_group"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.SecurityGroup == nil {
		return nil, false
	}
	return wrapper.SecurityGroup, true
}

func securityGroupRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"created_at":           true,
		"description":          true,
		"id":                   true,
		"is_shared":            true,
		"location":             true,
		"name":                 true,
		"project_id":           true,
		"revision_number":      true,
		"rules":                true,
		"security_group_rules": true,
		"shared":               true,
		"stateful":             true,
		"tags":                 true,
		"tenant_id":            true,
		"updated_at":           true,
	}
	fields := []outputField{
		{"created_at", raw["created_at"]},
		{"description", raw["description"]},
		{"id", raw["id"]},
		{"is_shared", firstPresent(raw, "is_shared", "shared")},
		{"name", raw["name"]},
		{"project_id", raw["project_id"]},
		{"revision_number", raw["revision_number"]},
		{"rules", firstPresent(raw, "rules", "security_group_rules")},
		{"stateful", raw["stateful"]},
		{"tags", raw["tags"]},
		{"updated_at", raw["updated_at"]},
	}
	var extras []string
	for key := range raw {
		if !known[key] {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		fields = append(fields, outputField{key, raw[key]})
	}
	return fields
}

func firstPresent(values map[string]any, names ...string) any {
	for _, name := range names {
		if value, ok := values[name]; ok {
			return value
		}
	}
	return nil
}

func parseExtraProperties(values []string) (map[string]any, error) {
	if len(values) == 0 {
		return nil, nil
	}
	properties := make(map[string]any, len(values))
	for _, value := range values {
		parts := map[string]string{}
		for _, part := range strings.Split(value, ",") {
			key, raw, ok := strings.Cut(part, "=")
			if !ok || strings.TrimSpace(key) == "" {
				return nil, fmt.Errorf("invalid extra property %q, expected type=<type>,name=<name>,value=<value>", value)
			}
			parts[strings.TrimSpace(key)] = raw
		}
		name := parts["name"]
		if name == "" {
			return nil, fmt.Errorf("invalid extra property %q, missing name", value)
		}
		parsed, err := parseExtraPropertyValue(parts["type"], parts["value"])
		if err != nil {
			return nil, fmt.Errorf("invalid extra property %q: %w", value, err)
		}
		properties[name] = parsed
	}
	return properties, nil
}

func parseUnsetExtraProperties(values []string) (map[string]any, error) {
	if len(values) == 0 {
		return nil, nil
	}
	properties := make(map[string]any, len(values))
	for _, value := range values {
		parts := map[string]string{}
		for _, part := range strings.Split(value, ",") {
			key, raw, ok := strings.Cut(part, "=")
			if !ok || strings.TrimSpace(key) == "" {
				return nil, fmt.Errorf("invalid extra property %q, expected type=<type>,name=<name>,value=<value>", value)
			}
			parts[strings.TrimSpace(key)] = raw
		}
		name := parts["name"]
		if name == "" {
			return nil, fmt.Errorf("invalid extra property %q, missing name", value)
		}
		properties[name] = nil
	}
	return properties, nil
}

func parseExtraPropertyValue(valueType string, raw string) (any, error) {
	switch valueType {
	case "", "str", "string":
		return raw, nil
	case "bool":
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid bool value %q", raw)
		}
		return parsed, nil
	case "int":
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid int value %q", raw)
		}
		return parsed, nil
	case "list":
		if raw == "" {
			return []string{}, nil
		}
		return strings.Split(raw, ";"), nil
	case "dict":
		result := map[string]string{}
		if raw == "" {
			return result, nil
		}
		for _, pair := range strings.Split(raw, ";") {
			key, value, ok := strings.Cut(pair, ":")
			if !ok || key == "" {
				return nil, fmt.Errorf("invalid dict value %q", raw)
			}
			result[key] = value
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", valueType)
	}
}

func setNeutronResourceTags(ctx context.Context, client *gophercloud.ServiceClient, resourcePath string, id string, existing []string, additions []string, clear bool) ([]string, error) {
	target := neutronResourceTargetTags(existing, additions, clear)
	if len(additions) == 0 && !clear {
		return target, nil
	}
	if stringSlicesEqual(uniqueSortedStrings(existing), target) {
		return target, nil
	}
	resp, err := client.Put(ctx, client.ServiceURL(resourcePath, url.PathEscape(id), "tags"), map[string]any{"tags": target}, nil, &gophercloud.RequestOpts{OkCodes: []int{200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return target, err
}

func neutronResourceTargetTags(existing []string, additions []string, clear bool) []string {
	target := []string{}
	if !clear {
		target = append(target, existing...)
	}
	target = append(target, additions...)
	return uniqueSortedStrings(target)
}

func unsetNeutronResourceTags(ctx context.Context, client *gophercloud.ServiceClient, resourcePath string, id string, existing []string, removals []string, clear bool) ([]string, error) {
	target := neutronResourceTagsAfterRemoval(existing, removals, clear)
	if len(removals) == 0 && !clear {
		return target, nil
	}
	if stringSlicesEqual(uniqueSortedStrings(existing), target) {
		return target, nil
	}
	resp, err := client.Put(ctx, client.ServiceURL(resourcePath, url.PathEscape(id), "tags"), map[string]any{"tags": target}, nil, &gophercloud.RequestOpts{OkCodes: []int{200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return target, err
}

func neutronResourceTagsAfterRemoval(existing []string, removals []string, clear bool) []string {
	if clear {
		return []string{}
	}
	remove := map[string]bool{}
	for _, value := range removals {
		remove[value] = true
	}
	target := make([]string, 0, len(existing))
	for _, value := range existing {
		if !remove[value] {
			target = append(target, value)
		}
	}
	return uniqueSortedStrings(target)
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

type securityGroupRuleCreateOpts struct {
	secgrouprules.CreateOpts
	Extra map[string]any
}

func (opts securityGroupRuleCreateOpts) ToSecGroupRuleCreateMap() (map[string]any, error) {
	body, err := opts.CreateOpts.ToSecGroupRuleCreateMap()
	if err != nil {
		return nil, err
	}
	rule, ok := body["security_group_rule"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid security group rule create request body")
	}
	for key, value := range opts.Extra {
		rule[key] = value
	}
	return body, nil
}

func securityGroupRuleCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group rule create requires <group>")
	}
	if err := validateSecurityGroupRuleCreateFlags(opts); err != nil {
		return err
	}
	group, err := findSecurityGroup(ctx, networkClient, args[0])
	if err != nil {
		return err
	}
	protocol := securityGroupRuleCreateProtocol(opts)
	ethertype := securityGroupRuleCreateEthertype(opts, protocol)
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	createOpts := securityGroupRuleCreateOpts{
		CreateOpts: secgrouprules.CreateOpts{
			Direction:  secgrouprules.DirIngress,
			EtherType:  secgrouprules.RuleEtherType(ethertype),
			SecGroupID: group.ID,
			Protocol:   secgrouprules.RuleProtocol(protocol),
		},
		Extra: extra,
	}
	if boolFlag(opts, "egress") {
		createOpts.Direction = secgrouprules.DirEgress
	}
	if flagChanged(opts, "description") {
		createOpts.Description = flagValue(opts, "description")
	}
	if portMin, portMax, ok, err := securityGroupRulePortRange(opts); err != nil {
		return err
	} else if ok && !isICMPProtocol(protocol) {
		createOpts.PortRangeMin = portMin
		createOpts.PortRangeMax = portMax
	}
	if flagChanged(opts, "icmp-type") {
		createOpts.PortRangeMin = intFlag(opts, "icmp-type")
	}
	if flagChanged(opts, "icmp-code") {
		createOpts.PortRangeMax = intFlag(opts, "icmp-code")
	}
	if remoteGroup := flagValue(opts, "remote-group"); remoteGroup != "" {
		remote, err := findSecurityGroup(ctx, networkClient, remoteGroup)
		if err != nil {
			return err
		}
		createOpts.RemoteGroupID = remote.ID
	} else if remoteAddressGroup := flagValue(opts, "remote-address-group"); remoteAddressGroup != "" {
		remote, err := findAddressGroup(ctx, networkClient, remoteAddressGroup)
		if err != nil {
			return err
		}
		createOpts.RemoteAddressGroupID = remote.ID
	} else if remoteIP := flagValue(opts, "remote-ip"); remoteIP != "" {
		createOpts.RemoteIPPrefix = remoteIP
	} else if ethertype == "IPv6" {
		createOpts.RemoteIPPrefix = "::/0"
	} else {
		createOpts.RemoteIPPrefix = "0.0.0.0/0"
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return err
		}
		createOpts.ProjectID = project.ID
	}
	result := secgrouprules.Create(ctx, networkClient, createOpts)
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if raw, ok := securityGroupRuleRawFromBody(result.Body); ok {
		return renderShowOutput(stdout, opts, securityGroupRuleRawFields(raw))
	}
	return renderShowOutput(stdout, opts, securityGroupRuleFields(item))
}

func securityGroupRuleDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("security group rule delete requires <rule> [<rule> ...]")
	}
	failures := 0
	for _, ruleID := range args {
		if err := secgrouprules.Delete(ctx, client, ruleID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d security group rules failed to delete.", failures, len(args))
	}
	return nil
}

func validateSecurityGroupRuleCreateFlags(opts *Options) error {
	remoteSelectors := 0
	for _, name := range []string{"remote-ip", "remote-group", "remote-address-group"} {
		if flagChanged(opts, name) {
			remoteSelectors++
		}
	}
	if remoteSelectors > 1 {
		return fmt.Errorf("argument --remote-address-group: not allowed with argument --remote-ip or --remote-group")
	}
	if flagChanged(opts, "ingress") && flagChanged(opts, "egress") {
		return fmt.Errorf("argument --egress: not allowed with argument --ingress")
	}
	if flagChanged(opts, "dst-port") && (flagChanged(opts, "icmp-type") || flagChanged(opts, "icmp-code")) {
		return fmt.Errorf("Argument --dst-port not allowed with arguments --icmp-type and --icmp-code")
	}
	if !flagChanged(opts, "icmp-type") && flagChanged(opts, "icmp-code") {
		return fmt.Errorf("Argument --icmp-type required with argument --icmp-code")
	}
	protocol := securityGroupRuleCreateProtocol(opts)
	if !isICMPProtocol(protocol) && (flagChanged(opts, "icmp-type") || flagChanged(opts, "icmp-code")) {
		return fmt.Errorf("ICMP IP protocol required with arguments --icmp-type and --icmp-code")
	}
	return nil
}

func securityGroupRuleCreateProtocol(opts *Options) string {
	value := strings.ToLower(firstFlag(opts, "protocol", "proto"))
	if value == "" || value == "any" {
		return ""
	}
	return value
}

func securityGroupRuleCreateEthertype(opts *Options, protocol string) string {
	if value := flagValue(opts, "ethertype"); value != "" {
		if strings.EqualFold(value, "ipv6") {
			return "IPv6"
		}
		return "IPv4"
	}
	if isIPv6Protocol(protocol) {
		return "IPv6"
	}
	return "IPv4"
}

func securityGroupRulePortRange(opts *Options) (int, int, bool, error) {
	value := flagValue(opts, "dst-port")
	if value == "" {
		return 0, 0, false, nil
	}
	parts := strings.Split(value, ":")
	if len(parts) > 2 {
		return 0, 0, false, fmt.Errorf("invalid range %q, too many values", value)
	}
	min, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid range %q", value)
	}
	max := min
	if len(parts) == 2 {
		max, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, false, fmt.Errorf("invalid range %q", value)
		}
		if min > max {
			return 0, 0, false, fmt.Errorf("invalid range %q, minimum is greater than maximum", value)
		}
	}
	return min, max, true, nil
}

func isICMPProtocol(protocol string) bool {
	switch protocol {
	case "icmp", "icmpv6", "ipv6-icmp", "1", "58":
		return true
	default:
		return false
	}
}

func isIPv6Protocol(protocol string) bool {
	if strings.HasPrefix(protocol, "ipv6-") {
		return true
	}
	switch protocol {
	case "icmpv6", "41", "43", "44", "58", "59", "60":
		return true
	default:
		return false
	}
}

func neutronSecurityGroupRuleRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		SecurityGroupRule map[string]any `json:"security_group_rule"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("security-group-rules", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.SecurityGroupRule, nil
}

func securityGroupRuleRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		SecurityGroupRule map[string]any `json:"security_group_rule"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.SecurityGroupRule == nil {
		return nil, false
	}
	return wrapper.SecurityGroupRule, true
}

func securityGroupRuleRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"belongs_to_default_sg":   true,
		"created_at":              true,
		"description":             true,
		"direction":               true,
		"ether_type":              true,
		"ethertype":               true,
		"id":                      true,
		"location":                true,
		"name":                    true,
		"normalized_cidr":         true,
		"port_range_max":          true,
		"port_range_min":          true,
		"project_id":              true,
		"protocol":                true,
		"remote_address_group_id": true,
		"remote_group_id":         true,
		"remote_ip_prefix":        true,
		"revision_number":         true,
		"security_group_id":       true,
		"tags":                    true,
		"tenant_id":               true,
		"updated_at":              true,
	}
	fields := []outputField{
		{"belongs_to_default_sg", raw["belongs_to_default_sg"]},
		{"created_at", raw["created_at"]},
		{"description", raw["description"]},
		{"direction", raw["direction"]},
		{"ether_type", firstPresent(raw, "ether_type", "ethertype")},
		{"id", raw["id"]},
		{"normalized_cidr", raw["normalized_cidr"]},
		{"port_range_max", raw["port_range_max"]},
		{"port_range_min", raw["port_range_min"]},
		{"project_id", raw["project_id"]},
		{"protocol", raw["protocol"]},
		{"remote_address_group_id", raw["remote_address_group_id"]},
		{"remote_group_id", raw["remote_group_id"]},
		{"remote_ip_prefix", raw["remote_ip_prefix"]},
		{"revision_number", raw["revision_number"]},
		{"security_group_id", raw["security_group_id"]},
		{"updated_at", raw["updated_at"]},
	}
	var extras []string
	for key := range raw {
		if !known[key] {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		fields = append(fields, outputField{key, raw[key]})
	}
	return fields
}

func securityGroupRuleFields(item *secgrouprules.SecGroupRule) []outputField {
	return []outputField{
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
	}
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
	if raw, err := neutronSecurityGroupRuleRaw(ctx, client, args[0]); err == nil {
		return renderShowOutput(stdout, opts, securityGroupRuleRawFields(raw))
	}
	item, err := secgrouprules.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, securityGroupRuleFields(item))
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
	shared, err := subnetPoolBoolFlag(opts, "share", "no-share")
	if err != nil {
		return err
	}
	if shared != nil {
		listOpts.Shared = shared
	}
	isDefault, err := subnetPoolBoolFlag(opts, "default", "no-default")
	if err != nil {
		return err
	}
	if isDefault != nil {
		listOpts.IsDefault = isDefault
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

type subnetPoolCreateOpts struct {
	Values map[string]any
}

func (opts subnetPoolCreateOpts) ToSubnetPoolCreateMap() (map[string]any, error) {
	return map[string]any{"subnetpool": opts.Values}, nil
}

type subnetPoolUpdateOpts struct {
	Values map[string]any
}

func (opts subnetPoolUpdateOpts) ToSubnetPoolUpdateMap() (map[string]any, error) {
	return map[string]any{"subnetpool": opts.Values}, nil
}

func subnetPoolCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet pool create requires <name>")
	}
	if boolFlag(opts, "no-tag") && len(flagValues(opts, "tag")) > 0 {
		return fmt.Errorf("argument --no-tag: not allowed with argument --tag")
	}
	values, err := subnetPoolCreateValues(ctx, opts, networkClient, identityClient, args[0])
	if err != nil {
		return err
	}
	result := subnetpools.Create(ctx, networkClient, subnetPoolCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	tags := flagValues(opts, "tag")
	if len(tags) > 0 {
		target, err := setNeutronResourceTags(ctx, networkClient, "subnetpools", item.ID, item.Tags, tags, boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
		item.Tags = target
	}
	if raw, ok := subnetPoolRawFromBody(result.Body); ok {
		if len(tags) > 0 {
			raw["tags"] = item.Tags
		}
		return renderShowOutput(stdout, opts, subnetPoolRawFields(raw))
	}
	return renderShowOutput(stdout, opts, subnetPoolFields(item))
}

func subnetPoolCreateValues(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, name string) (map[string]any, error) {
	prefixes := flagValues(opts, "pool-prefix")
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("argument --pool-prefix is required")
	}
	values := map[string]any{
		"name":     name,
		"prefixes": prefixes,
	}
	if err := subnetPoolAddPrefixLengthValues(opts, values); err != nil {
		return nil, err
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return nil, err
		}
		values["project_id"] = project.ID
	}
	if addressScope := flagValue(opts, "address-scope"); addressScope != "" {
		values["address_scope_id"] = resolveAddressScopeID(ctx, networkClient, addressScope)
	}
	isDefault, err := subnetPoolBoolFlag(opts, "default", "no-default")
	if err != nil {
		return nil, err
	}
	if isDefault != nil {
		values["is_default"] = *isDefault
	}
	shared, err := subnetPoolBoolFlag(opts, "share", "no-share")
	if err != nil {
		return nil, err
	}
	if shared != nil {
		values["shared"] = *shared
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if flagChanged(opts, "default-quota") {
		values["default_quota"] = intFlag(opts, "default-quota")
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return nil, err
	}
	for key, value := range extra {
		values[key] = value
	}
	return values, nil
}

func subnetPoolDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet pool delete requires <subnet-pool> [<subnet-pool> ...]")
	}
	failures := 0
	for _, poolArg := range args {
		item, err := findSubnetPool(ctx, client, poolArg)
		if err != nil {
			failures++
			continue
		}
		if err := subnetpools.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d subnet pools failed to delete.", failures, len(args))
	}
	return nil
}

func subnetPoolSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet pool set requires <subnet-pool>")
	}
	item, err := findSubnetPool(ctx, client, args[0])
	if err != nil {
		return err
	}
	values, err := subnetPoolSetValues(ctx, opts, client, item)
	if err != nil {
		return err
	}
	if len(values) > 0 {
		if _, err := subnetpools.Update(ctx, client, item.ID, subnetPoolUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "no-tag") || len(flagValues(opts, "tag")) > 0 {
		_, err := setNeutronResourceTags(ctx, client, "subnetpools", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
	}
	return nil
}

func subnetPoolSetValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, item *subnetpools.SubnetPool) (map[string]any, error) {
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if prefixes := flagValues(opts, "pool-prefix"); len(prefixes) > 0 {
		merged := append([]string{}, prefixes...)
		merged = append(merged, item.Prefixes...)
		values["prefixes"] = merged
	}
	if err := subnetPoolAddPrefixLengthValues(opts, values); err != nil {
		return nil, err
	}
	if flagChanged(opts, "address-scope") && boolFlag(opts, "no-address-scope") {
		return nil, fmt.Errorf("argument --no-address-scope: not allowed with argument --address-scope")
	}
	if addressScope := flagValue(opts, "address-scope"); addressScope != "" {
		values["address_scope_id"] = resolveAddressScopeID(ctx, client, addressScope)
	}
	if boolFlag(opts, "no-address-scope") {
		values["address_scope_id"] = nil
	}
	isDefault, err := subnetPoolBoolFlag(opts, "default", "no-default")
	if err != nil {
		return nil, err
	}
	if isDefault != nil {
		values["is_default"] = *isDefault
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if flagChanged(opts, "default-quota") {
		values["default_quota"] = intFlag(opts, "default-quota")
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return nil, err
	}
	for key, value := range extra {
		values[key] = value
	}
	return values, nil
}

func subnetPoolUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet pool unset requires <subnet-pool>")
	}
	if len(flagValues(opts, "tag")) > 0 && boolFlag(opts, "all-tag") {
		return fmt.Errorf("argument --all-tag: not allowed with argument --tag")
	}
	item, err := findSubnetPool(ctx, client, args[0])
	if err != nil {
		return err
	}
	_, err = unsetNeutronResourceTags(ctx, client, "subnetpools", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "all-tag"))
	return err
}

func subnetPoolAddPrefixLengthValues(opts *Options, values map[string]any) error {
	for flagName, fieldName := range map[string]string{
		"default-prefix-length": "default_prefixlen",
		"min-prefix-length":     "min_prefixlen",
		"max-prefix-length":     "max_prefixlen",
	} {
		if !flagChanged(opts, flagName) {
			continue
		}
		value := intFlag(opts, flagName)
		if value < 0 {
			return fmt.Errorf("argument --%s: invalid non-negative int value: %d", flagName, value)
		}
		values[fieldName] = value
	}
	return nil
}

func subnetPoolBoolFlag(opts *Options, trueFlag string, falseFlag string) (*bool, error) {
	if boolFlag(opts, trueFlag) && boolFlag(opts, falseFlag) {
		return nil, fmt.Errorf("argument --%s: not allowed with argument --%s", falseFlag, trueFlag)
	}
	if boolFlag(opts, trueFlag) {
		value := true
		return &value, nil
	}
	if boolFlag(opts, falseFlag) {
		value := false
		return &value, nil
	}
	return nil, nil
}

func subnetPoolShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet pool show requires <subnet-pool>")
	}
	item, err := findSubnetPool(ctx, client, args[0])
	if err != nil {
		return err
	}
	if raw, err := neutronSubnetPoolRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, subnetPoolRawFields(raw))
	}
	return renderShowOutput(stdout, opts, subnetPoolFields(item))
}

func neutronSubnetPoolRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		SubnetPool map[string]any `json:"subnetpool"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("subnetpools", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.SubnetPool, nil
}

func subnetPoolRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		SubnetPool map[string]any `json:"subnetpool"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.SubnetPool == nil {
		return nil, false
	}
	return wrapper.SubnetPool, true
}

func subnetPoolRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"address_scope_id":  true,
		"created_at":        true,
		"default_prefixlen": true,
		"default_quota":     true,
		"description":       true,
		"id":                true,
		"ip_version":        true,
		"is_default":        true,
		"location":          true,
		"max_prefixlen":     true,
		"min_prefixlen":     true,
		"name":              true,
		"prefixes":          true,
		"project_id":        true,
		"revision_number":   true,
		"shared":            true,
		"tags":              true,
		"tenant_id":         true,
		"updated_at":        true,
	}
	fields := []outputField{
		{"address_scope_id", raw["address_scope_id"]},
		{"created_at", raw["created_at"]},
		{"default_prefixlen", rawNumber(raw["default_prefixlen"])},
		{"default_quota", rawNumber(raw["default_quota"])},
		{"description", raw["description"]},
		{"id", raw["id"]},
		{"ip_version", rawNumber(raw["ip_version"])},
		{"is_default", raw["is_default"]},
		{"max_prefixlen", rawNumber(raw["max_prefixlen"])},
		{"min_prefixlen", rawNumber(raw["min_prefixlen"])},
		{"name", raw["name"]},
		{"prefixes", raw["prefixes"]},
		{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		{"revision_number", rawNumber(raw["revision_number"])},
		{"shared", raw["shared"]},
		{"tags", raw["tags"]},
		{"updated_at", raw["updated_at"]},
	}
	var extras []string
	for key := range raw {
		if !known[key] {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		fields = append(fields, outputField{key, raw[key]})
	}
	return fields
}

func rawNumber(value any) any {
	text, ok := value.(string)
	if !ok {
		return value
	}
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return value
	}
	return parsed
}

func subnetPoolFields(item *subnetpools.SubnetPool) []outputField {
	return []outputField{
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
	}
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

func keypairCreate(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("keypair create requires <name>")
	}
	if flagChanged(opts, "public-key") && flagChanged(opts, "private-key") {
		return fmt.Errorf("argument --private-key: not allowed with argument --public-key")
	}
	keyType := flagValue(opts, "type")
	if keyType != "" && keyType != "ssh" && keyType != "x509" {
		return fmt.Errorf("argument --type: invalid choice: %q (choose from 'ssh', 'x509')", keyType)
	}
	createOpts := keypairs.CreateOpts{
		Name: args[0],
		Type: keyType,
	}
	if userID, err := keypairUserID(ctx, opts, clients); err != nil {
		return err
	} else {
		createOpts.UserID = userID
	}

	generatedPrivateKey := ""
	if publicKeyPath := flagValue(opts, "public-key"); publicKeyPath != "" {
		data, err := os.ReadFile(expandUserPath(publicKeyPath))
		if err != nil {
			return fmt.Errorf("Key file %s not found: %v", publicKeyPath, err)
		}
		createOpts.PublicKey = string(data)
	} else {
		keypair, err := generateEd25519Keypair()
		if err != nil {
			return err
		}
		generatedPrivateKey = keypair.PrivateKey
		createOpts.PublicKey = keypair.PublicKey
		if privateKeyPath := flagValue(opts, "private-key"); privateKeyPath != "" {
			if err := os.WriteFile(expandUserPath(privateKeyPath), []byte(generatedPrivateKey), 0600); err != nil {
				return fmt.Errorf("Key file %s can not be saved: %v", privateKeyPath, err)
			}
		}
	}

	result := keypairs.Create(ctx, client, createOpts)
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if flagValue(opts, "public-key") == "" && flagValue(opts, "private-key") == "" {
		_, err := fmt.Fprint(stdout, generatedPrivateKey)
		return err
	}
	return renderKeypairShow(stdout, opts, item, keypairBodyMap(result.Body), true)
}

func keypairDelete(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("keypair delete requires <key>")
	}
	userID, err := keypairUserID(ctx, opts, clients)
	if err != nil {
		return err
	}
	failures := 0
	for _, name := range args {
		err := keypairs.Delete(ctx, client, name, keypairs.DeleteOpts{UserID: userID}).ExtractErr()
		if err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d keys failed to delete.", failures, len(args))
	}
	return nil
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
	result := keypairs.Get(ctx, client, args[0], keypairs.GetOpts{UserID: flagValue(opts, "user")})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if boolFlag(opts, "public-key") {
		_, err := fmt.Fprintln(stdout, item.PublicKey)
		return err
	}
	return renderKeypairShow(stdout, opts, item, keypairBodyMap(result.Body), false)
}

func renderKeypairShow(stdout io.Writer, opts *Options, item *keypairs.KeyPair, raw map[string]any, createOutput bool) error {
	fields := []outputField{
		{"created_at", keypairRawValue(raw, "created_at", nil)},
		{"fingerprint", item.Fingerprint},
		{"id", item.Name},
		{"is_deleted", keypairDeletedValue(raw, createOutput)},
		{"name", item.Name},
	}
	if !createOutput {
		fields = append(fields, outputField{"private_key", keypairRawValue(raw, "private_key", nil)})
	}
	fields = append(fields,
		outputField{"type", item.Type},
		outputField{"user_id", item.UserID},
	)
	return renderShowOutput(stdout, opts, fields)
}

func keypairBodyMap(body any) map[string]any {
	data, err := json.Marshal(body)
	if err != nil {
		return nil
	}
	var envelope map[string]map[string]any
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil
	}
	return envelope["keypair"]
}

func keypairRawValue(raw map[string]any, key string, fallback any) any {
	if raw == nil {
		return fallback
	}
	if value, ok := raw[key]; ok {
		return value
	}
	return fallback
}

func keypairDeletedValue(raw map[string]any, createOutput bool) any {
	if raw == nil {
		return nil
	}
	if value, ok := raw["is_deleted"]; ok {
		return value
	}
	if value, ok := raw["deleted"]; ok {
		if createOutput && value == false {
			return nil
		}
		return value
	}
	return nil
}

type generatedKeypair struct {
	PrivateKey string
	PublicKey  string
}

func generateEd25519Keypair() (generatedKeypair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return generatedKeypair{}, err
	}
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return generatedKeypair{}, err
	}
	privateBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return generatedKeypair{}, err
	}
	return generatedKeypair{
		PrivateKey: string(pem.EncodeToMemory(privateBlock)),
		PublicKey:  string(ssh.MarshalAuthorizedKey(sshPublicKey)),
	}, nil
}

func keypairUserID(ctx context.Context, opts *Options, clients *openStackClients) (string, error) {
	value := flagValue(opts, "user")
	if value == "" {
		return "", nil
	}
	identityClient, err := clients.identityV3()
	if err != nil {
		return "", err
	}
	user, err := findUserWithDomain(ctx, identityClient, value, flagValue(opts, "user-domain"))
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func expandUserPath(path string) string {
	if path == "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return home + path[1:]
	}
	return path
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
	policyColumn := "Policies"
	if microversionAtLeast(client.Microversion, "2.64") {
		policyColumn = "Policy"
	}
	for _, item := range items {
		row := outputRow{"ID": item.ID, "Name": item.Name}
		if policyColumn == "Policy" {
			row[policyColumn] = stringPtrValue(item.Policy)
		} else {
			row[policyColumn] = item.Policies
		}
		if boolFlag(opts, "long") {
			row["Members"] = item.Members
			row["Project Id"] = item.ProjectID
			row["User Id"] = item.UserID
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", policyColumn}
	if boolFlag(opts, "long") {
		columns = append(columns, "Members", "Project Id", "User Id")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func serverGroupCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server group create requires <name>")
	}
	policy := flagValue(opts, "policy")
	if policy == "" {
		policy = "affinity"
	}
	if !serverGroupPolicyAllowed(policy) {
		return fmt.Errorf("argument --policy: invalid choice: %q (choose from 'affinity', 'anti-affinity', 'soft-affinity', 'soft-anti-affinity')", policy)
	}
	if (policy == "soft-affinity" || policy == "soft-anti-affinity") && !microversionAtLeast(client.Microversion, "2.15") {
		return fmt.Errorf("--os-compute-api-version 2.15 or greater is required to support the %s policy", policy)
	}
	rules, err := serverGroupRules(opts, client)
	if err != nil {
		return err
	}
	createOpts := servergroups.CreateOpts{Name: args[0]}
	if microversionAtLeast(client.Microversion, "2.64") {
		createOpts.Policy = policy
		createOpts.Rules = rules
	} else {
		createOpts.Policies = []string{policy}
	}
	item, err := servergroups.Create(ctx, client, createOpts).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, serverGroupFields(item, client))
}

func serverGroupDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server group delete requires <server-group>")
	}
	failures := 0
	for _, value := range args {
		item, err := findServerGroup(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if err := servergroups.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d server groups failed to delete.", failures, len(args))
	}
	return nil
}

func serverGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server group show requires <server-group>")
	}
	item, err := findServerGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, serverGroupFields(item, client))
}

func serverGroupFields(item *servergroups.ServerGroup, client *gophercloud.ServiceClient) []outputField {
	if microversionAtLeast(client.Microversion, "2.64") {
		return []outputField{
			{"id", item.ID},
			{"members", item.Members},
			{"name", item.Name},
			outputField{"policy", stringPtrValue(item.Policy)},
			{"project_id", item.ProjectID},
			{"rules", serverGroupRulesOutput(item.Rules)},
			{"user_id", item.UserID},
		}
	}
	return []outputField{
		{"id", item.ID},
		{"members", item.Members},
		{"name", item.Name},
		{"policies", item.Policies},
		{"project_id", item.ProjectID},
		{"user_id", item.UserID},
	}
}

func serverGroupPolicyAllowed(policy string) bool {
	switch policy {
	case "affinity", "anti-affinity", "soft-affinity", "soft-anti-affinity":
		return true
	default:
		return false
	}
}

func serverGroupRules(opts *Options, client *gophercloud.ServiceClient) (*servergroups.Rules, error) {
	values := flagValues(opts, "rule")
	if len(values) == 0 {
		return nil, nil
	}
	if !microversionAtLeast(client.Microversion, "2.64") {
		return nil, fmt.Errorf("--os-compute-api-version 2.64 or greater is required to support the --rule option")
	}
	rules := &servergroups.Rules{}
	for _, value := range values {
		key, raw, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid rule %q, expected <key=value>", value)
		}
		switch key {
		case "max_server_per_host":
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid max_server_per_host rule value %q", raw)
			}
			rules.MaxServerPerHost = parsed
		default:
			return nil, fmt.Errorf("unsupported server group rule %q", key)
		}
	}
	return rules, nil
}

func serverGroupRulesOutput(rules *servergroups.Rules) any {
	if rules == nil || rules.MaxServerPerHost == 0 {
		return map[string]any{}
	}
	return map[string]any{"max_server_per_host": rules.MaxServerPerHost}
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

func nilIfMissing(item map[string]any, key string) any {
	value, ok := item[key]
	if !ok {
		return nil
	}
	return value
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	default:
		return false
	}
}

func appendMapField(fields []outputField, item map[string]any, key string, name string) []outputField {
	value, ok := item[key]
	if !ok || value == nil {
		return fields
	}
	return append(fields, outputField{Name: name, Value: value})
}

func readServiceJSONRaw(ctx context.Context, client *gophercloud.ServiceClient, requestURL string) ([]byte, error) {
	resp, err := client.Get(ctx, requestURL, nil, &gophercloud.RequestOpts{KeepResponseBody: true})
	body, _, err := gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}
	defer body.Close()
	return io.ReadAll(body)
}

func orderedJSONTopObject(body []byte) (orderedJSONObject, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	value, err := decodeOrderedJSONValue(decoder)
	if err != nil {
		return orderedJSONObject{}, err
	}
	item, ok := orderedJSONValueAsObject(value)
	if !ok {
		return orderedJSONObject{}, fmt.Errorf("expected JSON object")
	}
	return item, nil
}

func decodeOrderedJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delim {
	case '{':
		item := orderedJSONObject{values: map[string]any{}}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, fmt.Errorf("expected JSON object key")
			}
			value, err := decodeOrderedJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			item.keys = append(item.keys, key)
			item.values[key] = value
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return item, nil
	case '[':
		items := []any{}
		for decoder.More() {
			value, err := decodeOrderedJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}

func orderedJSONValueAsObject(value any) (orderedJSONObject, bool) {
	item, ok := value.(orderedJSONObject)
	return item, ok
}

func orderedMapValueAsObject(item orderedJSONObject, key string) (orderedJSONObject, bool) {
	return orderedJSONValueAsObject(item.values[key])
}

func orderedMapValueOrNil(item orderedJSONObject, key string) any {
	if value, ok := item.values[key]; ok {
		return value
	}
	return nil
}

func orderedMapValueOrDefault(item orderedJSONObject, key string, defaultValue any) any {
	if value, ok := item.values[key]; ok && value != nil {
		return value
	}
	return defaultValue
}

func sortedFieldsFromMap(values map[string]any, skipNil bool) []outputField {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if skipNil && value == nil {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]outputField, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, outputField{Name: key, Value: values[key]})
	}
	return fields
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
