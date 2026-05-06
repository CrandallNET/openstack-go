package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"reflect"
	"regexp"
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
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/instanceactions"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/remoteconsoles"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	computeservices "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/services"
	computetags "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/tags"
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
	portforwarding "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/portforwarding"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networkipavailabilities"
	qospolicies "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies"
	qosrules "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/rules"
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
		case "block storage cleanup":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageCleanup(cmd.Context(), stdout, opts, client)
		case "block storage cluster list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageClusterList(cmd.Context(), stdout, opts, client)
		case "block storage cluster set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageClusterSet(cmd.Context(), stdout, opts, client, args)
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
		case "block storage log level set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageLogLevelSet(cmd.Context(), opts, client, args)
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
		case "block storage snapshot manageable list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageManageableList(cmd.Context(), stdout, opts, client, args, "snapshots")
		case "block storage volume manageable list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return blockStorageManageableList(cmd.Context(), stdout, opts, client, args, "volumes")
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
		case "consistency group add volume":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupAddRemoveVolume(cmd.Context(), opts, client, args, true)
		case "consistency group create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupCreate(cmd.Context(), stdout, opts, client, args)
		case "consistency group delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupDelete(cmd.Context(), opts, client, args)
		case "consistency group list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupList(cmd.Context(), stdout, opts, client)
		case "consistency group remove volume":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupAddRemoveVolume(cmd.Context(), opts, client, args, false)
		case "consistency group set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupSet(cmd.Context(), opts, client, args)
		case "consistency group show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupShow(cmd.Context(), stdout, opts, client, args)
		case "consistency group snapshot create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupSnapshotCreate(cmd.Context(), stdout, opts, client, args)
		case "consistency group snapshot delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupSnapshotDelete(cmd.Context(), opts, client, args)
		case "consistency group snapshot list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupSnapshotList(cmd.Context(), stdout, opts, client)
		case "consistency group snapshot show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return consistencyGroupSnapshotShow(cmd.Context(), stdout, opts, client, args)
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
		case "floating ip create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return floatingIPCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "floating ip delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPDelete(cmd.Context(), client, args)
		case "floating ip list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPList(cmd.Context(), stdout, opts, client)
		case "floating ip pool list":
			return floatingIPPoolList()
		case "floating ip port forwarding create":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPPortForwardingCreate(cmd.Context(), stdout, opts, client, args)
		case "floating ip port forwarding delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPPortForwardingDelete(cmd.Context(), client, args)
		case "floating ip port forwarding list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPPortForwardingList(cmd.Context(), stdout, opts, client, args)
		case "floating ip port forwarding set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPPortForwardingSet(cmd.Context(), opts, client, args)
		case "floating ip port forwarding show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPPortForwardingShow(cmd.Context(), stdout, opts, client, args)
		case "floating ip set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPSet(cmd.Context(), opts, client, args)
		case "floating ip show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPShow(cmd.Context(), stdout, opts, client, args)
		case "floating ip unset":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return floatingIPUnset(cmd.Context(), opts, client, args)
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
		case "network qos policy create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return networkQoSPolicyCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "network qos policy delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSPolicyDelete(cmd.Context(), client, args)
		case "network qos policy set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSPolicySet(cmd.Context(), opts, client, args)
		case "network qos policy show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSPolicyShow(cmd.Context(), stdout, opts, client, args)
		case "network qos rule create":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSRuleCreate(cmd.Context(), stdout, opts, client, args)
		case "network qos rule delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSRuleDelete(cmd.Context(), client, args)
		case "network qos rule list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSRuleList(cmd.Context(), stdout, opts, client, args)
		case "network qos rule set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSRuleSet(cmd.Context(), opts, client, args)
		case "network qos rule show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkQoSRuleShow(cmd.Context(), stdout, opts, client, args)
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
		case "network rbac create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return networkRBACCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "network rbac delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkRBACDelete(cmd.Context(), client, args)
		case "network rbac set":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return networkRBACSet(cmd.Context(), opts, networkClient, identityClient, args)
		case "network rbac show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkRBACShow(cmd.Context(), stdout, opts, client, args)
		case "network segment create":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkSegmentCreate(cmd.Context(), stdout, opts, client, args)
		case "network segment delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkSegmentDelete(cmd.Context(), client, args)
		case "network segment list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkSegmentList(cmd.Context(), stdout, opts, client)
		case "network segment set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkSegmentSet(cmd.Context(), opts, client, args)
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
		case "network subport list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkSubportList(cmd.Context(), stdout, opts, client)
		case "network trunk create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return networkTrunkCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "network trunk delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkTrunkDelete(cmd.Context(), client, args)
		case "network trunk list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkTrunkList(cmd.Context(), stdout, opts, client)
		case "network trunk set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkTrunkSet(cmd.Context(), opts, client, args)
		case "network trunk show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkTrunkShow(cmd.Context(), stdout, opts, client, args)
		case "network trunk unset":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return networkTrunkUnset(cmd.Context(), opts, client, args)
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
			fmt.Fprintln(cmd.ErrOrStderr(), "This command is deprecated.")
			return hypervisorStatsShow(cmd.Context(), stdout, opts, client)
		case "limits show":
			return limitsShow(cmd.Context(), stdout, opts, clients)
		case "port create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return portCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "port delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return portDelete(cmd.Context(), client, args)
		case "port list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			computeClient, _ := clients.computeV2()
			return portList(cmd.Context(), stdout, opts, client, computeClient)
		case "port set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return portSet(cmd.Context(), opts, client, args)
		case "port show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return portShow(cmd.Context(), stdout, opts, client, args)
		case "port unset":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return portUnset(cmd.Context(), opts, client, args)
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
		case "router add port":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerAddPort(cmd.Context(), client, args)
		case "router add gateway":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerAddGateway(cmd.Context(), stdout, opts, client, args)
		case "router add route":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerAddRoute(cmd.Context(), stdout, opts, client, args)
		case "router add subnet":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerAddSubnet(cmd.Context(), client, args)
		case "router create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return routerCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "router delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerDelete(cmd.Context(), client, args)
		case "router list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerList(cmd.Context(), stdout, opts, client)
		case "router remove port":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerRemovePort(cmd.Context(), client, args)
		case "router remove gateway":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerRemoveGateway(cmd.Context(), stdout, opts, client, args)
		case "router remove route":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerRemoveRoute(cmd.Context(), stdout, opts, client, args)
		case "router remove subnet":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerRemoveSubnet(cmd.Context(), client, args)
		case "router set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerSet(cmd.Context(), opts, client, args)
		case "router show":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerShow(cmd.Context(), stdout, opts, client, args)
		case "router unset":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return routerUnset(cmd.Context(), opts, client, args)
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
		case "server add fixed ip":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			return serverAddFixedIP(cmd.Context(), stdout, opts, computeClient, networkClient, args)
		case "server add floating ip":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			return serverAddFloatingIP(cmd.Context(), opts, computeClient, networkClient, args)
		case "server add network":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			return serverAddNetwork(cmd.Context(), opts, computeClient, networkClient, args)
		case "server add port":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			return serverAddPort(cmd.Context(), opts, computeClient, networkClient, args)
		case "server add security group":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			return serverSecurityGroupAction(cmd.Context(), opts, computeClient, networkClient, args, "addSecurityGroup")
		case "server add volume":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			volumeClient, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return serverAddVolume(cmd.Context(), stdout, opts, computeClient, volumeClient, args)
		case "server backup create":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			imageClient, _ := clients.imageV2()
			return serverBackupCreate(cmd.Context(), stdout, opts, computeClient, imageClient, args)
		case "server create":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			imageClient, _ := clients.imageV2()
			networkClient, _ := clients.networkV2()
			volumeClient, _ := clients.blockStorageV3()
			return serverCreate(cmd.Context(), stdout, opts, computeClient, imageClient, networkClient, volumeClient, args)
		case "server delete":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverDelete(cmd.Context(), stdout, opts, client, args)
		case "server dump create":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverSimpleRawAction(cmd.Context(), client, args, "trigger_crash_dump")
		case "server evacuate":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverEvacuate(cmd.Context(), stdout, opts, client, args)
		case "server image create":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			imageClient, _ := clients.imageV2()
			return serverImageCreate(cmd.Context(), stdout, opts, computeClient, imageClient, args)
		case "server list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			imageClient, _ := clients.imageV2()
			networkClient, _ := clients.networkV2()
			return computeServerList(cmd.Context(), stdout, opts, client, imageClient, networkClient)
		case "server lock":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverLock(cmd.Context(), opts, client, args)
		case "server migrate":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMigrate(cmd.Context(), stdout, opts, client, args)
		case "server migrate confirm", "server migration confirm", "server resize confirm":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverSingleAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.ConfirmResize(ctx, client, id).ExtractErr()
			})
		case "server migrate revert", "server migration revert", "server resize revert":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverSingleAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.RevertResize(ctx, client, id).ExtractErr()
			})
		case "server migration abort":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMigrationDeleteAction(cmd.Context(), client, args)
		case "server migration force complete":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMigrationForceComplete(cmd.Context(), client, args)
		case "server show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, _ := clients.networkV2()
			return computeServerShow(cmd.Context(), stdout, opts, client, networkClient, args)
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
		case "server migration show":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMigrationShow(cmd.Context(), stdout, opts, client, args)
		case "server pause":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMultiAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.Pause(ctx, client, id).ExtractErr()
			})
		case "server reboot":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverReboot(cmd.Context(), stdout, opts, client, args)
		case "server rebuild":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			imageClient, _ := clients.imageV2()
			return serverRebuild(cmd.Context(), stdout, opts, computeClient, imageClient, args)
		case "server remove fixed ip":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverRemoveFixedIP(cmd.Context(), client, args)
		case "server remove floating ip":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			return serverRemoveFloatingIP(cmd.Context(), computeClient, networkClient, args)
		case "server remove network":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			return serverRemoveNetwork(cmd.Context(), computeClient, networkClient, args)
		case "server remove port":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			return serverRemovePort(cmd.Context(), computeClient, networkClient, args)
		case "server remove security group":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			return serverSecurityGroupAction(cmd.Context(), opts, computeClient, networkClient, args, "removeSecurityGroup")
		case "server remove volume":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			volumeClient, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return serverRemoveVolume(cmd.Context(), computeClient, volumeClient, args)
		case "server rescue":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			imageClient, _ := clients.imageV2()
			return serverRescue(cmd.Context(), stdout, opts, computeClient, imageClient, args)
		case "server resize":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverResize(cmd.Context(), stdout, opts, client, args)
		case "server restore":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverSimpleRawAction(cmd.Context(), client, args, "restore")
		case "server resume":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMultiAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.Resume(ctx, client, id).ExtractErr()
			})
		case "server set":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverSet(cmd.Context(), opts, client, args)
		case "server shelve":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverShelve(cmd.Context(), stdout, opts, client, args)
		case "server start":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMultiAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.Start(ctx, client, id).ExtractErr()
			})
		case "server stop":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMultiAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.Stop(ctx, client, id).ExtractErr()
			})
		case "server suspend":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMultiAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.Suspend(ctx, client, id).ExtractErr()
			})
		case "server unlock":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMultiAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.Unlock(ctx, client, id).ExtractErr()
			})
		case "server unpause":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMultiAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.Unpause(ctx, client, id).ExtractErr()
			})
		case "server unrescue":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverMultiAction(cmd.Context(), client, args, func(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
				return servers.Unrescue(ctx, client, id).ExtractErr()
			})
		case "server unset":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverUnset(cmd.Context(), opts, client, args)
		case "server unshelve":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverUnshelve(cmd.Context(), stdout, opts, client, args)
		case "server volume list":
			client, err := clients.computeV2()
			if err != nil {
				return err
			}
			return serverVolumeList(cmd.Context(), stdout, opts, client, args)
		case "server volume set", "server volume update":
			computeClient, err := clients.computeV2()
			if err != nil {
				return err
			}
			volumeClient, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return serverVolumeSet(cmd.Context(), opts, computeClient, volumeClient, args)
		case "subnet create":
			networkClient, err := clients.networkV2()
			if err != nil {
				return err
			}
			identityClient, err := clients.identityV3()
			if err != nil {
				return err
			}
			return subnetCreate(cmd.Context(), stdout, opts, networkClient, identityClient, args)
		case "subnet delete":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetDelete(cmd.Context(), client, args)
		case "subnet list":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetList(cmd.Context(), stdout, opts, client)
		case "subnet set":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetSet(cmd.Context(), opts, client, args)
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
		case "subnet unset":
			client, err := clients.networkV2()
			if err != nil {
				return err
			}
			return subnetUnset(cmd.Context(), opts, client, args)
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
			computeClient, _ := clients.computeV2()
			return volumeList(cmd.Context(), stdout, opts, client, computeClient)
		case "volume create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			imageClient, _ := clients.imageV2()
			return volumeCreate(cmd.Context(), stdout, opts, client, imageClient, args)
		case "volume delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeDelete(cmd.Context(), opts, client, args)
		case "volume set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSet(cmd.Context(), opts, client, args)
		case "volume unset":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeUnset(cmd.Context(), opts, client, args)
		case "volume group list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupList(cmd.Context(), stdout, opts, client)
		case "volume group create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupCreate(cmd.Context(), stdout, opts, client, args)
		case "volume group delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupDelete(cmd.Context(), opts, client, args)
		case "volume group failover":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupFailover(cmd.Context(), opts, client, args)
		case "volume group set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupSet(cmd.Context(), stdout, opts, client, args)
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
		case "volume group snapshot create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupSnapshotCreate(cmd.Context(), stdout, opts, client, args)
		case "volume group snapshot delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupSnapshotDelete(cmd.Context(), opts, client, args)
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
		case "volume group type create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupTypeCreate(cmd.Context(), stdout, opts, client, args)
		case "volume group type delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupTypeDelete(cmd.Context(), opts, client, args)
		case "volume group type set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupTypeSet(cmd.Context(), stdout, opts, client, args)
		case "volume group type show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeGroupTypeShow(cmd.Context(), stdout, opts, client, args)
		case "volume host set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeHostSet(cmd.Context(), opts, client, args)
		case "volume message list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeMessageList(cmd.Context(), stdout, opts, clients, client)
		case "volume message delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeMessageDelete(cmd.Context(), client, args)
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
		case "volume migrate":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeMigrate(cmd.Context(), opts, client, args)
		case "volume revert":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeRevert(cmd.Context(), opts, client, args)
		case "volume backend capability show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackendCapabilityShow(cmd.Context(), stdout, opts, client, args)
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
		case "volume attachment create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			computeClient, _ := clients.computeV2()
			return volumeAttachmentCreate(cmd.Context(), stdout, opts, client, computeClient, args)
		case "volume attachment delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeAttachmentDelete(cmd.Context(), opts, client, args)
		case "volume attachment set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeAttachmentSet(cmd.Context(), stdout, opts, client, args)
		case "volume attachment complete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeAttachmentComplete(cmd.Context(), opts, client, args)
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
		case "volume backup create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupCreate(cmd.Context(), stdout, opts, client, args)
		case "volume backup delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupDelete(cmd.Context(), opts, client, args)
		case "volume backup restore":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupRestore(cmd.Context(), stdout, opts, client, args)
		case "volume backup set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupSet(cmd.Context(), opts, client, args)
		case "volume backup unset":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupUnset(cmd.Context(), opts, client, args)
		case "volume backup record export":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupRecordExport(cmd.Context(), stdout, opts, client, args)
		case "volume backup record import":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeBackupRecordImport(cmd.Context(), stdout, opts, client, args)
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
		case "volume service set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeServiceSet(cmd.Context(), opts, client, args)
		case "volume snapshot list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotList(cmd.Context(), stdout, opts, client)
		case "volume snapshot create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotCreate(cmd.Context(), stdout, opts, client, args)
		case "volume snapshot delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotDelete(cmd.Context(), opts, client, args)
		case "volume snapshot set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotSet(cmd.Context(), opts, client, args)
		case "volume snapshot show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotShow(cmd.Context(), stdout, opts, client, args)
		case "volume snapshot unset":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeSnapshotUnset(cmd.Context(), opts, client, args)
		case "volume qos list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSList(cmd.Context(), stdout, opts, client)
		case "volume qos create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSCreate(cmd.Context(), stdout, opts, client, args)
		case "volume qos delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSDelete(cmd.Context(), opts, client, args)
		case "volume qos set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSSet(cmd.Context(), opts, client, args)
		case "volume qos show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSShow(cmd.Context(), stdout, opts, client, args)
		case "volume qos unset":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSUnset(cmd.Context(), opts, client, args)
		case "volume qos associate":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSAssociate(cmd.Context(), opts, client, args)
		case "volume qos disassociate":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeQoSDisassociate(cmd.Context(), opts, client, args)
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
		case "volume type create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTypeCreate(cmd.Context(), stdout, opts, clients, client, args)
		case "volume type delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTypeDelete(cmd.Context(), opts, client, args)
		case "volume type set":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTypeSet(cmd.Context(), opts, clients, client, args)
		case "volume type show":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTypeShow(cmd.Context(), stdout, opts, client, args)
		case "volume type unset":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTypeUnset(cmd.Context(), opts, clients, client, args)
		case "volume transfer request list":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTransferList(cmd.Context(), stdout, opts, client)
		case "volume transfer request create":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTransferCreate(cmd.Context(), stdout, opts, client, args)
		case "volume transfer request delete":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTransferDelete(cmd.Context(), opts, client, args)
		case "volume transfer request accept":
			client, err := clients.blockStorageV3()
			if err != nil {
				return err
			}
			return volumeTransferAccept(cmd.Context(), stdout, opts, client, args)
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

func computeServerList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, imageClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient) error {
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
	networkLabels := serverNetworkLabelsForPretty(ctx, opts, networkClient)
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		networks := tableValue{
			Value:  serverNetworksInAPISliceOrder(item.Addresses),
			Table:  serverNetworksSummary(item.Addresses),
			Pretty: serverPrettyNetworkAddresses(item.Addresses, networkLabels),
		}
		rows = append(rows, outputRow{
			"ID":       item.ID,
			"Name":     item.Name,
			"Status":   item.Status,
			"Networks": networks,
			"Image":    serverImage(item.Image, imageNames),
			"Flavor":   serverFlavor(item.Flavor, flavorNames),
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Status", "Networks", "Image", "Flavor"}, rows)
}

func computeServerShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server show requires <server>")
	}
	if os.Getenv("OS_COMPUTE_API_VERSION") == "" {
		if withMinimum, err := computeClientWithMinimumMicroversion(ctx, client, "2.96"); err == nil {
			client = withMinimum
		}
	}
	item, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	if raw, err := computeServerRaw(ctx, client, item.ID); err == nil {
		enrichServerRaw(ctx, client, raw)
		return renderShowOutput(stdout, opts, serverRawFields(raw, serverNetworkLabelsForPretty(ctx, opts, networkClient)))
	}
	addresses := any(serverNetworksSummary(item.Addresses))
	if prettyOutput(opts) {
		addresses = serverPrettyNetworkAddresses(item.Addresses, serverNetworkLabelsForPretty(ctx, opts, networkClient))
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
		{"addresses", addresses},
		{"metadata", item.Metadata},
		{"key_name", nilIfEmpty(item.KeyName)},
	})
}

func computeServerRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		Server map[string]any `json:"server"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("servers", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Server, nil
}

func serverRawFields(raw map[string]any, networkLabels []serverNetworkLabel) []outputField {
	return []outputField{
		{"OS-DCF:diskConfig", raw["OS-DCF:diskConfig"]},
		{"OS-EXT-AZ:availability_zone", raw["OS-EXT-AZ:availability_zone"]},
		{"OS-EXT-SRV-ATTR:host", raw["OS-EXT-SRV-ATTR:host"]},
		{"OS-EXT-SRV-ATTR:hostname", raw["OS-EXT-SRV-ATTR:hostname"]},
		{"OS-EXT-SRV-ATTR:hypervisor_hostname", raw["OS-EXT-SRV-ATTR:hypervisor_hostname"]},
		{"OS-EXT-SRV-ATTR:instance_name", raw["OS-EXT-SRV-ATTR:instance_name"]},
		{"OS-EXT-SRV-ATTR:kernel_id", raw["OS-EXT-SRV-ATTR:kernel_id"]},
		{"OS-EXT-SRV-ATTR:launch_index", rawNumber(raw["OS-EXT-SRV-ATTR:launch_index"])},
		{"OS-EXT-SRV-ATTR:ramdisk_id", raw["OS-EXT-SRV-ATTR:ramdisk_id"]},
		{"OS-EXT-SRV-ATTR:reservation_id", raw["OS-EXT-SRV-ATTR:reservation_id"]},
		{"OS-EXT-SRV-ATTR:root_device_name", raw["OS-EXT-SRV-ATTR:root_device_name"]},
		{"OS-EXT-SRV-ATTR:user_data", raw["OS-EXT-SRV-ATTR:user_data"]},
		{"OS-EXT-STS:power_state", serverPowerStateValue(raw["OS-EXT-STS:power_state"])},
		{"OS-EXT-STS:task_state", raw["OS-EXT-STS:task_state"]},
		{"OS-EXT-STS:vm_state", raw["OS-EXT-STS:vm_state"]},
		{"OS-SRV-USG:launched_at", raw["OS-SRV-USG:launched_at"]},
		{"OS-SRV-USG:terminated_at", raw["OS-SRV-USG:terminated_at"]},
		{"accessIPv4", raw["accessIPv4"]},
		{"accessIPv6", raw["accessIPv6"]},
		{"addresses", serverAddressesValue(raw["addresses"], networkLabels)},
		{"config_drive", raw["config_drive"]},
		{"created", raw["created"]},
		{"description", raw["description"]},
		{"flavor", serverFlavorShowValue(raw["flavor"])},
		{"hostId", raw["hostId"]},
		{"host_status", raw["host_status"]},
		{"id", raw["id"]},
		{"image", serverImageShowValue(raw["image"])},
		{"key_name", raw["key_name"]},
		{"locked", raw["locked"]},
		{"locked_reason", raw["locked_reason"]},
		{"name", raw["name"]},
		{"pinned_availability_zone", raw["pinned_availability_zone"]},
		{"progress", rawNumber(raw["progress"])},
		{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		{"properties", serverPropertyTableValue(raw["metadata"])},
		{"scheduler_hints", serverSchedulerHintsTableValue(raw["scheduler_hints"])},
		{"security_groups", serverKeyValueList(raw["security_groups"], []string{"name"})},
		{"server_groups", serverPythonListTableValue(raw["server_groups"])},
		{"status", raw["status"]},
		{"tags", blankEmptyListValue(raw["tags"])},
		{"trusted_image_certificates", raw["trusted_image_certificates"]},
		{"updated", raw["updated"]},
		{"user_id", raw["user_id"]},
		{"volumes_attached", serverKeyValueListWithJSONKeys(raw["os-extended-volumes:volumes_attached"], []string{"delete_on_termination", "id"}, []string{"id", "delete_on_termination"})},
	}
}

func serverCreateRawFields(raw map[string]any, networkLabels []serverNetworkLabel) []outputField {
	fields := serverRawFields(raw, networkLabels)
	result := make([]outputField, 0, len(fields)+1)
	for _, field := range fields {
		switch field.Name {
		case "OS-EXT-SRV-ATTR:kernel_id", "OS-EXT-SRV-ATTR:ramdisk_id", "accessIPv4", "accessIPv6", "config_drive":
			field.Value = nilIfEmptyAny(field.Value)
		case "OS-EXT-SRV-ATTR:launch_index", "progress":
			field.Value = nilIfZeroOrEmpty(field.Value)
		case "locked":
			field.Value = nilIfFalse(field.Value)
		case "server_groups":
			field.Value = serverGroupsShowValue(raw["server_groups"])
		}
		result = append(result, field)
		if field.Name == "addresses" && !emptyServerShowValue(raw["adminPass"]) {
			result = append(result, outputField{"adminPass", raw["adminPass"]})
		}
	}
	return result
}

func enrichServerRaw(ctx context.Context, client *gophercloud.ServiceClient, raw map[string]any) {
	flavor, ok := raw["flavor"].(map[string]any)
	if !ok {
		return
	}
	ref := firstNonEmpty(
		stringValue(firstPresent(flavor, "id")),
		stringValue(firstPresent(flavor, "name")),
		stringValue(firstPresent(flavor, "original_name")),
	)
	if ref == "" {
		return
	}
	item, err := findFlavor(ctx, client, ref)
	if err != nil {
		return
	}
	setIfMissing(flavor, "id", firstNonEmpty(stringValue(firstPresent(flavor, "original_name")), item.Name, item.ID))
	setIfMissing(flavor, "name", item.Name)
	setIfMissing(flavor, "original_name", item.Name)
	setIfMissing(flavor, "disk", item.Disk)
	setIfMissing(flavor, "ephemeral", item.Ephemeral)
	setIfMissing(flavor, "is_public", item.IsPublic)
	setIfMissing(flavor, "ram", item.RAM)
	setIfMissing(flavor, "swap", item.Swap)
	setIfMissing(flavor, "vcpus", item.VCPUs)
	if _, ok := flavor["extra_specs"]; !ok {
		flavor["extra_specs"] = map[string]any{}
	}
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func setIfMissing(values map[string]any, key string, value any) {
	if !emptyServerShowValue(values[key]) {
		return
	}
	if emptyServerShowValue(value) {
		return
	}
	values[key] = value
}

func serverPowerStateValue(value any) any {
	number, ok := intFromAny(value)
	if !ok {
		return value
	}
	states := map[int]string{
		0: "NOSTATE",
		1: "Running",
		3: "Paused",
		4: "Shutdown",
		6: "Crashed",
		7: "Suspended",
	}
	label, ok := states[number]
	if !ok {
		return value
	}
	return tableValue{Value: value, Table: label, Pretty: label}
}

func serverAddressesValue(value any, networkLabels []serverNetworkLabel) any {
	addresses, ok := value.(map[string]any)
	if !ok {
		return value
	}
	return tableValue{
		Value:  serverNetworksInAPISliceOrder(addresses),
		Table:  serverNetworksSummary(addresses),
		Pretty: serverPrettyNetworkAddresses(addresses, networkLabels),
	}
}

func serverFlavorShowValue(value any) any {
	flavor, ok := value.(map[string]any)
	if !ok {
		return value
	}
	tableKeys := []string{"description", "disk", "ephemeral", "extra_specs", "id", "is_disabled", "is_public", "location", "name", "original_name", "ram", "rxtx_factor", "swap", "vcpus"}
	jsonKeys := []string{"name", "original_name", "description", "disk", "is_public", "ram", "vcpus", "swap", "ephemeral", "is_disabled", "rxtx_factor", "extra_specs", "id", "location"}
	values := map[string]any{
		"description":   firstPresent(flavor, "description"),
		"disk":          firstPresent(flavor, "disk"),
		"ephemeral":     firstPresent(flavor, "ephemeral", "OS-FLV-EXT-DATA:ephemeral"),
		"extra_specs":   firstPresent(flavor, "extra_specs"),
		"id":            firstPresent(flavor, "id"),
		"is_disabled":   firstPresent(flavor, "is_disabled", "OS-FLV-DISABLED:disabled"),
		"is_public":     firstPresent(flavor, "is_public", "os-flavor-access:is_public"),
		"location":      firstPresent(flavor, "location"),
		"name":          firstPresent(flavor, "name"),
		"original_name": firstPresent(flavor, "original_name"),
		"ram":           firstPresent(flavor, "ram"),
		"rxtx_factor":   firstPresent(flavor, "rxtx_factor"),
		"swap":          serverFlavorSwapShowValue(firstPresent(flavor, "swap")),
		"vcpus":         firstPresent(flavor, "vcpus"),
	}
	parts := make([]string, 0, len(tableKeys))
	for _, key := range tableKeys {
		if key == "extra_specs" {
			parts = append(parts, "")
			continue
		}
		parts = append(parts, serverShowKeyValueToken(key, values[key]))
	}
	return tableValue{
		Value:  orderedMapFromKeys(values, jsonKeys),
		Table:  strings.Join(parts, ", "),
		Pretty: value,
	}
}

func serverFlavorSwapShowValue(value any) any {
	text := strings.TrimSpace(valueString(value))
	if text == "" || text == "None" {
		return 0
	}
	return rawNumber(value)
}

func serverImageShowValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return value
	case string:
		if strings.TrimSpace(typed) == "" {
			return "N/A (booted from volume)"
		}
		return typed
	case map[string]any:
		if len(typed) == 0 {
			return "N/A (booted from volume)"
		}
		display := serverImageShowString(typed, nil)
		return tableValue{Value: display, Table: display, Pretty: display}
	default:
		return value
	}
}

func serverKeyValueList(value any, keys []string) tableValue {
	return serverKeyValueListWithJSONKeys(value, keys, keys)
}

func serverKeyValueListWithJSONKeys(value any, tableKeys []string, jsonKeys []string) tableValue {
	items := anySlice(value)
	jsonValues := make([]any, 0, len(items))
	lines := make([]string, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		values := make(map[string]any, len(jsonKeys))
		presentKeys := make([]string, 0, len(jsonKeys))
		parts := make([]string, 0, len(tableKeys))
		for _, key := range jsonKeys {
			itemValue, ok := itemMap[key]
			if ok {
				values[key] = itemValue
				presentKeys = append(presentKeys, key)
			}
		}
		for _, key := range tableKeys {
			itemValue, ok := itemMap[key]
			if !ok || emptyServerShowValue(itemValue) {
				continue
			}
			parts = append(parts, serverShowQuotedKeyValueToken(key, itemValue))
		}
		jsonValues = append(jsonValues, orderedJSONObject{keys: presentKeys, values: values})
		if len(parts) > 0 {
			lines = append(lines, strings.Join(parts, ", "))
		}
	}
	return tableValue{Value: jsonValues, Table: strings.Join(lines, "\n"), Pretty: value}
}

func serverPropertyTableValue(value any) tableValue {
	values := mapValueOrEmptyAny(value)
	parts := make([]string, 0, len(values))
	for _, key := range sortedMapKeys(values) {
		if emptyServerShowValue(values[key]) {
			continue
		}
		parts = append(parts, serverShowQuotedKeyValueToken(key, values[key]))
	}
	return tableValue{Value: values, Table: strings.Join(parts, ", "), Pretty: values}
}

func serverSchedulerHintsTableValue(value any) tableValue {
	values := mapValueOrEmptyAny(value)
	parts := make([]string, 0, len(values))
	for _, key := range sortedMapKeys(values) {
		if emptyServerShowValue(values[key]) {
			continue
		}
		parts = append(parts, key+"="+serverSchedulerHintTableString(values[key]))
	}
	return tableValue{Value: values, Table: strings.Join(parts, ", "), Pretty: values}
}

func serverSchedulerHintTableString(value any) string {
	values := anySlice(value)
	if len(values) == 0 {
		return valueString(value)
	}
	parts := make([]string, 0, len(values))
	for _, item := range values {
		parts = append(parts, valueString(item))
	}
	return strings.Join(parts, ", ")
}

func serverPythonListTableValue(value any) tableValue {
	values := anySlice(value)
	table := ""
	if len(values) > 0 {
		table = pythonRepr(values)
	} else if value != nil {
		table = "[]"
	}
	return tableValue{Value: values, Table: table, Pretty: values}
}

func serverGroupsShowValue(value any) any {
	values := anySlice(value)
	if len(values) == 0 {
		return nil
	}
	return serverPythonListTableValue(value)
}

func serverShowKeyValueToken(key string, value any) string {
	if emptyServerShowValue(value) {
		return key + "="
	}
	return serverShowQuotedKeyValueToken(key, value)
}

func serverShowQuotedKeyValueToken(key string, value any) string {
	return fmt.Sprintf("%s='%s'", key, strings.ReplaceAll(valueString(value), "'", "\\'"))
}

func emptyServerShowValue(value any) bool {
	if value == nil {
		return true
	}
	text := strings.TrimSpace(valueString(value))
	return text == "" || text == "None"
}

func nilIfEmptyAny(value any) any {
	if emptyServerShowValue(value) {
		return nil
	}
	return value
}

func nilIfZeroOrEmpty(value any) any {
	if emptyServerShowValue(value) {
		return nil
	}
	number := rawNumber(value)
	switch typed := number.(type) {
	case int:
		if typed == 0 {
			return nil
		}
	case int8:
		if typed == 0 {
			return nil
		}
	case int16:
		if typed == 0 {
			return nil
		}
	case int32:
		if typed == 0 {
			return nil
		}
	case int64:
		if typed == 0 {
			return nil
		}
	case float32:
		if typed == 0 {
			return nil
		}
	case float64:
		if typed == 0 {
			return nil
		}
	case string:
		if strings.TrimSpace(typed) == "0" {
			return nil
		}
	}
	return number
}

func nilIfFalse(value any) any {
	typed, ok := value.(bool)
	if ok && !typed {
		return nil
	}
	return value
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

type serverActionFunc func(context.Context, *gophercloud.ServiceClient, string) error

func serverMultiAction(ctx context.Context, client *gophercloud.ServiceClient, args []string, action serverActionFunc) error {
	if len(args) < 1 {
		return fmt.Errorf("server command requires <server>")
	}
	failures := 0
	for _, value := range args {
		server, err := findServer(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if err := action(ctx, client, server.ID); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d servers failed.", failures, len(args))
	}
	return nil
}

func serverSingleAction(ctx context.Context, client *gophercloud.ServiceClient, args []string, action serverActionFunc) error {
	if len(args) < 1 {
		return fmt.Errorf("server command requires <server>")
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	return action(ctx, client, server.ID)
}

func serverDelete(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server delete requires <server>")
	}
	deleted := make([]string, 0, len(args))
	failures := 0
	for _, value := range args {
		server, err := findServerForServerCommand(ctx, client, value, boolFlag(opts, "all-projects"))
		if err != nil {
			failures++
			continue
		}
		if boolFlag(opts, "force") {
			err = servers.ForceDelete(ctx, client, server.ID).ExtractErr()
		} else {
			err = servers.Delete(ctx, client, server.ID).ExtractErr()
		}
		if err != nil {
			failures++
			continue
		}
		deleted = append(deleted, server.ID)
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d servers failed to delete.", failures, len(args))
	}
	if boolFlag(opts, "wait") {
		for _, id := range deleted {
			if err := waitForServerGone(ctx, stdout, opts, client, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func serverReboot(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server reboot requires <server>")
	}
	method := servers.SoftReboot
	if boolFlag(opts, "hard") {
		method = servers.HardReboot
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	if err := servers.Reboot(ctx, client, server.ID, servers.RebootOpts{Type: method}).ExtractErr(); err != nil {
		return err
	}
	if boolFlag(opts, "wait") {
		if err := waitForServerStatus(ctx, stdout, opts, client, server.ID, "Rebooting", []string{"ACTIVE"}, []string{"ERROR"}); err != nil {
			return err
		}
		renderWaitComplete(stdout, opts)
	}
	return nil
}

func serverResize(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server resize requires <server>")
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	switch {
	case boolFlag(opts, "confirm"):
		return servers.ConfirmResize(ctx, client, server.ID).ExtractErr()
	case boolFlag(opts, "revert"):
		return servers.RevertResize(ctx, client, server.ID).ExtractErr()
	}
	flavorName := flagValue(opts, "flavor")
	if flavorName == "" {
		return fmt.Errorf("server resize requires --flavor, --confirm, or --revert")
	}
	flavor, err := findFlavor(ctx, client, flavorName)
	if err != nil {
		return err
	}
	if err := servers.Resize(ctx, client, server.ID, servers.ResizeOpts{FlavorRef: flavor.ID}).ExtractErr(); err != nil {
		return err
	}
	if boolFlag(opts, "wait") {
		return waitForServerStatus(ctx, stdout, opts, client, server.ID, "Resizing", []string{"VERIFY_RESIZE"}, []string{"ERROR"})
	}
	return nil
}

func serverMigrate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server migrate requires <server>")
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	if boolFlag(opts, "live-migration") {
		var host *string
		if value := flagValue(opts, "host"); value != "" {
			host = &value
		}
		var block *bool
		if boolFlag(opts, "block-migration") {
			value := true
			block = &value
		} else if boolFlag(opts, "shared-migration") {
			value := false
			block = &value
		}
		var overcommit *bool
		if boolFlag(opts, "disk-overcommit") {
			value := true
			overcommit = &value
		} else if boolFlag(opts, "no-disk-overcommit") {
			value := false
			overcommit = &value
		}
		if err := servers.LiveMigrate(ctx, client, server.ID, servers.LiveMigrateOpts{Host: host, BlockMigration: block, DiskOverCommit: overcommit}).ExtractErr(); err != nil {
			return err
		}
		if boolFlag(opts, "wait") {
			return waitForServerStatus(ctx, stdout, opts, client, server.ID, "Migrating", []string{"ACTIVE"}, []string{"ERROR"})
		}
		return nil
	}
	body := any(nil)
	if host := flagValue(opts, "host"); host != "" {
		body = map[string]any{"host": host}
	}
	if err := serverRawAction(ctx, client, server.ID, "migrate", body, nil, http.StatusAccepted, http.StatusNoContent); err != nil {
		return err
	}
	if boolFlag(opts, "wait") {
		return waitForServerStatus(ctx, stdout, opts, client, server.ID, "Migrating", []string{"VERIFY_RESIZE"}, []string{"ERROR"})
	}
	return nil
}

func serverRebuild(ctx context.Context, stdout io.Writer, opts *Options, computeClient *gophercloud.ServiceClient, imageClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server rebuild requires <server>")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	imageID := ""
	if imageValue := flagValue(opts, "image"); imageValue != "" {
		imageID, err = resolveServerImageID(ctx, imageClient, imageValue)
		if err != nil {
			return err
		}
	} else if id, ok := server.Image["id"].(string); ok {
		imageID = id
	}
	if imageID == "" && !boolFlag(opts, "reimage-boot-volume") {
		return fmt.Errorf("server rebuild requires --image for servers without an image reference")
	}
	metadata, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	rebuild := serverRebuildOpts{
		RebuildOpts: servers.RebuildOpts{
			AdminPass: flagValue(opts, "password"),
			ImageRef:  imageID,
			Name:      flagValue(opts, "name"),
			Metadata:  metadata,
		},
		Description:              flagValue(opts, "description"),
		KeyName:                  flagValue(opts, "key-name"),
		UnsetKeyName:             boolFlag(opts, "no-key-name"),
		TrustedImageCertificates: flagValues(opts, "trusted-image-cert"),
		UnsetTrustedImageCerts:   boolFlag(opts, "no-trusted-image-certs"),
		Hostname:                 flagValue(opts, "hostname"),
		ReimageBootVolume:        boolFlag(opts, "reimage-boot-volume"),
		NoReimageBootVolume:      boolFlag(opts, "no-reimage-boot-volume"),
	}
	if flagChanged(opts, "user-data") {
		userData, err := os.ReadFile(expandUserPath(flagValue(opts, "user-data")))
		if err != nil {
			return err
		}
		rebuild.UserData = userData
	}
	if boolFlag(opts, "no-user-data") {
		rebuild.UnsetUserData = true
	}
	rebuilt, err := servers.Rebuild(ctx, computeClient, server.ID, rebuild).Extract()
	if err != nil {
		return err
	}
	adminPass := rebuilt.AdminPass
	if boolFlag(opts, "wait") {
		if err := waitForServerStatus(ctx, stdout, opts, computeClient, server.ID, "Rebuilding", []string{"ACTIVE"}, []string{"ERROR"}); err != nil {
			return err
		}
		rebuilt, err = servers.Get(ctx, computeClient, server.ID).Extract()
		if err != nil {
			return err
		}
	}
	showClient := computeClient
	if os.Getenv("OS_COMPUTE_API_VERSION") == "" {
		if withMinimum, err := computeClientWithMinimumMicroversion(ctx, computeClient, "2.96"); err == nil {
			showClient = withMinimum
		}
	}
	if raw, err := computeServerRaw(ctx, showClient, server.ID); err == nil {
		enrichServerRaw(ctx, showClient, raw)
		if adminPass != "" {
			raw["adminPass"] = adminPass
		}
		return renderShowOutput(stdout, opts, serverCreateRawFields(raw, nil))
	}
	return renderServerShow(stdout, opts, rebuilt, nil)
}

func serverRescue(ctx context.Context, stdout io.Writer, opts *Options, computeClient *gophercloud.ServiceClient, imageClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server rescue requires <server>")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	imageID := ""
	if imageValue := flagValue(opts, "image"); imageValue != "" {
		imageID, err = resolveServerImageID(ctx, imageClient, imageValue)
		if err != nil {
			return err
		}
	}
	result := servers.Rescue(ctx, computeClient, server.ID, servers.RescueOpts{AdminPass: flagValue(opts, "password"), RescueImageRef: imageID})
	if result.Err != nil {
		return result.Err
	}
	body := mapAnyFromRaw(result.Body)
	if len(body) == 0 {
		_ = result.ExtractInto(&body)
	}
	fields := []outputField{}
	if value, ok := body["adminPass"]; ok {
		fields = []outputField{{"adminPass", value}}
	}
	if len(fields) == 0 {
		return nil
	}
	return renderShowOutput(stdout, opts, fields)
}

func serverEvacuate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server evacuate requires <server>")
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	result := servers.Evacuate(ctx, client, server.ID, servers.EvacuateOpts{
		Host:            flagValue(opts, "host"),
		OnSharedStorage: boolFlag(opts, "shared-storage"),
		AdminPass:       flagValue(opts, "password"),
	})
	adminPass, err := result.ExtractAdminPass()
	if err != nil {
		return err
	}
	if boolFlag(opts, "wait") {
		if err := waitForServerStatus(ctx, stdout, opts, client, server.ID, "Evacuating", []string{"ACTIVE"}, []string{"ERROR"}); err != nil {
			return err
		}
	}
	if adminPass == "" {
		return nil
	}
	return renderShowOutput(stdout, opts, []outputField{{"adminPass", adminPass}})
}

func serverShelve(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server shelve requires <server>")
	}
	for _, value := range args {
		server, err := findServer(ctx, client, value)
		if err != nil {
			return err
		}
		if err := servers.Shelve(ctx, client, server.ID).ExtractErr(); err != nil {
			return err
		}
		if boolFlag(opts, "offload") {
			if err := servers.ShelveOffload(ctx, client, server.ID).ExtractErr(); err != nil {
				return err
			}
		}
		if boolFlag(opts, "wait") {
			targets := []string{"SHELVED", "SHELVED_OFFLOADED"}
			if boolFlag(opts, "offload") {
				targets = []string{"SHELVED_OFFLOADED"}
			}
			if err := waitForServerStatus(ctx, stdout, opts, client, server.ID, "Shelving", targets, []string{"ERROR"}); err != nil {
				return err
			}
		}
	}
	return nil
}

func serverUnshelve(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server unshelve requires <server>")
	}
	if boolFlag(opts, "no-availability-zone") || flagValue(opts, "host") != "" {
		client, _ = computeClientWithMinimumMicroversion(ctx, client, "2.91")
	} else if flagValue(opts, "availability-zone") != "" {
		client, _ = computeClientWithMinimumMicroversion(ctx, client, "2.77")
	}
	for _, value := range args {
		server, err := findServer(ctx, client, value)
		if err != nil {
			return err
		}
		body := any(nil)
		if boolFlag(opts, "no-availability-zone") || flagValue(opts, "host") != "" {
			payload := map[string]any{}
			if !boolFlag(opts, "no-availability-zone") && flagValue(opts, "availability-zone") != "" {
				payload["availability_zone"] = flagValue(opts, "availability-zone")
			}
			if host := flagValue(opts, "host"); host != "" {
				payload["host"] = host
			}
			body = payload
		} else {
			body = servers.UnshelveOpts{AvailabilityZone: flagValue(opts, "availability-zone")}
		}
		err = nil
		if typed, ok := body.(servers.UnshelveOpts); ok {
			err = servers.Unshelve(ctx, client, server.ID, typed).ExtractErr()
		} else {
			err = serverRawAction(ctx, client, server.ID, "unshelve", body, nil, http.StatusAccepted, http.StatusNoContent)
		}
		if err != nil {
			return err
		}
		if boolFlag(opts, "wait") {
			if err := waitForServerStatus(ctx, stdout, opts, client, server.ID, "Unshelving", []string{"ACTIVE"}, []string{"ERROR"}); err != nil {
				return err
			}
		}
	}
	return nil
}

func serverLock(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server lock requires <server>")
	}
	for _, value := range args {
		server, err := findServer(ctx, client, value)
		if err != nil {
			return err
		}
		if reason := flagValue(opts, "reason"); reason != "" {
			if err := serverRawAction(ctx, client, server.ID, "lock", map[string]any{"locked_reason": reason}, nil, http.StatusAccepted, http.StatusNoContent); err != nil {
				return err
			}
			continue
		}
		if err := servers.Lock(ctx, client, server.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func serverSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server set requires <server>")
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := serverUpdateOpts{}
	if flagChanged(opts, "name") {
		update.Name = flagValue(opts, "name")
	}
	if flagChanged(opts, "hostname") {
		hostname := flagValue(opts, "hostname")
		update.Hostname = &hostname
	}
	if flagChanged(opts, "description") {
		update.Description = valueStringPtr(flagValue(opts, "description"))
	}
	if update.hasValues() {
		if _, err := servers.Update(ctx, client, server.ID, update).Extract(); err != nil {
			return err
		}
	}
	if flagChanged(opts, "password") {
		if err := servers.ChangeAdminPassword(ctx, client, server.ID, flagValue(opts, "password")).ExtractErr(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "no-password") {
		if err := serverDeleteAdminPassword(ctx, client, server.ID); err != nil {
			return err
		}
	}
	if values := flagValues(opts, "property"); len(values) > 0 {
		metadata, err := parseStringMap(values, "property")
		if err != nil {
			return err
		}
		if _, err := servers.UpdateMetadata(ctx, client, server.ID, servers.MetadataOpts(metadata)).Extract(); err != nil {
			return err
		}
	}
	if state := strings.ToLower(strings.TrimSpace(flagValue(opts, "state"))); state != "" {
		if state != "active" && state != "error" {
			return fmt.Errorf("argument --state: invalid choice: %q (choose from 'active', 'error')", state)
		}
		if err := servers.ResetState(ctx, client, server.ID, servers.ServerState(state)).ExtractErr(); err != nil {
			return err
		}
	}
	for _, tag := range flagValues(opts, "tag") {
		if err := computetags.Add(ctx, client, server.ID, tag).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func serverUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server unset requires <server>")
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	if boolFlag(opts, "all-properties") {
		if _, err := servers.ResetMetadata(ctx, client, server.ID, servers.MetadataOpts{}).Extract(); err != nil {
			return err
		}
	}
	for _, key := range flagValues(opts, "property") {
		if err := servers.DeleteMetadatum(ctx, client, server.ID, key).ExtractErr(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "description") {
		update := serverUpdateOpts{Description: valueStringPtr("")}
		if _, err := servers.Update(ctx, client, server.ID, update).Extract(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "all-tags") {
		if err := computetags.DeleteAll(ctx, client, server.ID).ExtractErr(); err != nil {
			return err
		}
	}
	for _, tag := range flagValues(opts, "tag") {
		if err := computetags.Delete(ctx, client, server.ID, tag).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func serverCreate(ctx context.Context, stdout io.Writer, opts *Options, computeClient *gophercloud.ServiceClient, imageClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server create requires <server-name>")
	}
	if flagValue(opts, "flavor") == "" {
		return fmt.Errorf("argument --flavor is required")
	}
	flavor, err := findFlavor(ctx, computeClient, flagValue(opts, "flavor"))
	if err != nil {
		return err
	}
	metadata, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	networks, networkMode, err := serverCreateNetworks(ctx, opts, networkClient)
	if err != nil {
		return err
	}
	blockDevices, imageID, err := serverCreateBootSource(ctx, opts, imageClient, volumeClient)
	if err != nil {
		return err
	}
	if imageID == "" && len(blockDevices) == 0 {
		return fmt.Errorf("server create requires --image, --image-property, --volume, --snapshot, or --block-device")
	}
	personality, err := serverPersonality(flagValues(opts, "file"))
	if err != nil {
		return err
	}
	var userData []byte
	if userDataPath := flagValue(opts, "user-data"); userDataPath != "" {
		userData, err = os.ReadFile(expandUserPath(userDataPath))
		if err != nil {
			return err
		}
	}
	createOpts := serverCreateOpts{
		CreateOpts: servers.CreateOpts{
			Name:               args[0],
			ImageRef:           imageID,
			FlavorRef:          flavor.ID,
			SecurityGroups:     flagValues(opts, "security-group"),
			UserData:           userData,
			AvailabilityZone:   flagValue(opts, "availability-zone"),
			Metadata:           metadata,
			Personality:        personality,
			AdminPass:          flagValue(opts, "password"),
			Min:                intFlag(opts, "min"),
			Max:                intFlag(opts, "max"),
			Tags:               flagValues(opts, "tag"),
			Hostname:           flagValue(opts, "hostname"),
			BlockDevice:        blockDevices,
			HypervisorHostname: flagValue(opts, "hypervisor-hostname"),
		},
		Description:              flagValue(opts, "description"),
		Host:                     flagValue(opts, "host"),
		KeyName:                  flagValue(opts, "key-name"),
		TrustedImageCertificates: flagValues(opts, "trusted-image-cert"),
	}
	if boolFlag(opts, "no-security-group") {
		createOpts.SecurityGroups = []string{}
		createOpts.NoSecurityGroups = true
	}
	if networkMode != nil {
		createOpts.Networks = networkMode
	} else if len(networks) > 0 {
		createOpts.Networks = networks
	}
	if flagChanged(opts, "use-config-drive") || flagChanged(opts, "no-config-drive") {
		value := boolFlag(opts, "use-config-drive")
		createOpts.ConfigDrive = &value
	}
	if flagValue(opts, "config-drive") != "" {
		value := true
		createOpts.ConfigDrive = &value
	}
	hints, err := serverSchedulerHints(ctx, opts, computeClient)
	if err != nil {
		return err
	}
	created, err := servers.Create(ctx, computeClient, createOpts, hints).Extract()
	if err != nil {
		return err
	}
	adminPass := created.AdminPass
	if boolFlag(opts, "wait") {
		if err := waitForServerStatus(ctx, stdout, opts, computeClient, created.ID, "Creating", []string{"ACTIVE"}, []string{"ERROR"}); err != nil {
			return err
		}
		created, err = servers.Get(ctx, computeClient, created.ID).Extract()
		if err != nil {
			return err
		}
	}
	showClient := computeClient
	if os.Getenv("OS_COMPUTE_API_VERSION") == "" {
		if withMinimum, err := computeClientWithMinimumMicroversion(ctx, computeClient, "2.96"); err == nil {
			showClient = withMinimum
		}
	}
	if raw, err := computeServerRaw(ctx, showClient, created.ID); err == nil {
		enrichServerRaw(ctx, showClient, raw)
		if adminPass != "" {
			raw["adminPass"] = adminPass
		}
		return renderShowOutput(stdout, opts, serverCreateRawFields(raw, serverNetworkLabelsForPretty(ctx, opts, networkClient)))
	}
	return renderServerShow(stdout, opts, created, nil)
}

func serverImageCreate(ctx context.Context, stdout io.Writer, opts *Options, computeClient *gophercloud.ServiceClient, imageClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server image create requires <server>")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	name := flagValue(opts, "name")
	if name == "" {
		name = server.Name
	}
	metadata, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	result := servers.CreateImage(ctx, computeClient, server.ID, servers.CreateImageOpts{Name: name, Metadata: metadata})
	imageID, err := result.ExtractImageID()
	if err != nil {
		return err
	}
	if boolFlag(opts, "wait") && imageClient != nil {
		if err := waitForImageStatus(ctx, stdout, opts, imageClient, imageID, []string{"active"}, []string{"killed", "deleted"}); err != nil {
			return err
		}
	}
	return renderShowOutput(stdout, opts, []outputField{{"id", imageID}})
}

func serverBackupCreate(ctx context.Context, stdout io.Writer, opts *Options, computeClient *gophercloud.ServiceClient, imageClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server backup create requires <server>")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	name := flagValue(opts, "name")
	if name == "" {
		name = server.Name
	}
	backupType := flagValue(opts, "type")
	rotate := intFlag(opts, "rotate")
	if !flagChanged(opts, "rotate") {
		rotate = 1
	}
	var body struct {
		ImageID string `json:"image_id"`
	}
	resp, err := computeClient.Post(ctx, computeClient.ServiceURL("servers", url.PathEscape(server.ID), "action"), map[string]any{"createBackup": map[string]any{
		"name":        name,
		"backup_type": backupType,
		"rotation":    rotate,
	}}, &body, &gophercloud.RequestOpts{OkCodes: []int{http.StatusAccepted}})
	_, header, err := gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	imageID := body.ImageID
	if imageID == "" {
		if location := header.Get("Location"); location != "" {
			if parsed, err := url.Parse(location); err == nil {
				imageID = strings.Trim(strings.TrimPrefix(parsed.Path, "/"), "/")
				if slash := strings.LastIndex(imageID, "/"); slash >= 0 {
					imageID = imageID[slash+1:]
				}
			}
		}
	}
	if boolFlag(opts, "wait") && imageClient != nil && imageID != "" {
		if err := waitForImageStatus(ctx, stdout, opts, imageClient, imageID, []string{"active"}, []string{"killed", "deleted"}); err != nil {
			return err
		}
	}
	return renderShowOutput(stdout, opts, []outputField{{"id", imageID}})
}

func serverAddFixedIP(ctx context.Context, stdout io.Writer, opts *Options, computeClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server add fixed ip requires <server> <network>")
	}
	if flagValue(opts, "tag") != "" {
		computeClient, _ = computeClientWithMinimumMicroversion(ctx, computeClient, "2.49")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	network, err := findNetwork(ctx, networkClient, args[1])
	if err != nil {
		return err
	}
	createOpts := serverAttachInterfaceOpts{NetworkID: network.ID, Tag: flagValue(opts, "tag")}
	if ip := flagValue(opts, "fixed-ip-address"); ip != "" {
		createOpts.FixedIPs = []attachinterfaces.FixedIP{{IPAddress: ip}}
	}
	item, err := attachinterfaces.Create(ctx, computeClient, server.ID, createOpts).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, serverInterfaceFields(item))
}

func serverAddNetwork(ctx context.Context, opts *Options, computeClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server add network requires <server> <network>")
	}
	if flagValue(opts, "tag") != "" {
		computeClient, _ = computeClientWithMinimumMicroversion(ctx, computeClient, "2.49")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	network, err := findNetwork(ctx, networkClient, args[1])
	if err != nil {
		return err
	}
	_, err = attachinterfaces.Create(ctx, computeClient, server.ID, serverAttachInterfaceOpts{NetworkID: network.ID, Tag: flagValue(opts, "tag")}).Extract()
	return err
}

func serverAddPort(ctx context.Context, opts *Options, computeClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server add port requires <server> <port>")
	}
	if flagValue(opts, "tag") != "" {
		computeClient, _ = computeClientWithMinimumMicroversion(ctx, computeClient, "2.49")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	port, err := findPort(ctx, networkClient, args[1])
	if err != nil {
		return err
	}
	_, err = attachinterfaces.Create(ctx, computeClient, server.ID, serverAttachInterfaceOpts{PortID: port.ID, Tag: flagValue(opts, "tag")}).Extract()
	return err
}

func serverRemovePort(ctx context.Context, computeClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server remove port requires <server> <port>")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	port, err := findPort(ctx, networkClient, args[1])
	if err != nil {
		return err
	}
	return attachinterfaces.Delete(ctx, computeClient, server.ID, port.ID).ExtractErr()
}

func serverRemoveNetwork(ctx context.Context, computeClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server remove network requires <server> <network>")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	network, err := findNetwork(ctx, networkClient, args[1])
	if err != nil {
		return err
	}
	page, err := ports.List(networkClient, ports.ListOpts{DeviceID: server.ID, NetworkID: network.ID}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := ports.ExtractPorts(page)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("No ports found for server %s on network %s", server.ID, network.ID)
	}
	for _, port := range items {
		if err := attachinterfaces.Delete(ctx, computeClient, server.ID, port.ID).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func serverAddVolume(ctx context.Context, stdout io.Writer, opts *Options, computeClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server add volume requires <server> <volume>")
	}
	if flagValue(opts, "tag") != "" {
		computeClient, _ = computeClientWithMinimumMicroversion(ctx, computeClient, "2.49")
	}
	if boolFlag(opts, "enable-delete-on-termination") || boolFlag(opts, "disable-delete-on-termination") {
		computeClient, _ = computeClientWithMinimumMicroversion(ctx, computeClient, "2.79")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	volume, err := findVolume(ctx, volumeClient, args[1])
	if err != nil {
		return err
	}
	createOpts := volumeattach.CreateOpts{
		Device:   flagValue(opts, "device"),
		VolumeID: volume.ID,
		Tag:      flagValue(opts, "tag"),
	}
	if boolFlag(opts, "enable-delete-on-termination") {
		createOpts.DeleteOnTermination = true
	}
	item, err := volumeattach.Create(ctx, computeClient, server.ID, createOpts).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, serverVolumeAttachmentFields(item))
}

func serverRemoveVolume(ctx context.Context, computeClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server remove volume requires <server> <volume>")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	volume, err := findVolume(ctx, volumeClient, args[1])
	if err != nil {
		return err
	}
	return volumeattach.Delete(ctx, computeClient, server.ID, volume.ID).ExtractErr()
}

func serverVolumeSet(ctx context.Context, opts *Options, computeClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server volume set requires <server> <volume>")
	}
	if !flagChanged(opts, "delete-on-termination") && !flagChanged(opts, "preserve-on-termination") {
		return nil
	}
	computeClient, err := computeClientWithMinimumMicroversion(ctx, computeClient, "2.85")
	if err != nil {
		return err
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	volume, err := findVolume(ctx, volumeClient, args[1])
	if err != nil {
		return err
	}
	value := boolFlag(opts, "delete-on-termination")
	if boolFlag(opts, "preserve-on-termination") {
		value = false
	}
	requestURL := computeClient.ServiceURL("servers", url.PathEscape(server.ID), "os-volume_attachments", url.PathEscape(volume.ID))
	resp, err := computeClient.Put(ctx, requestURL, map[string]any{
		"volumeAttachment": map[string]any{"delete_on_termination": value},
	}, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return oscHTTPException(err)
}

func serverAddFloatingIP(ctx context.Context, opts *Options, computeClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server add floating ip requires <server> <ip-address>")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	fip, err := findFloatingIP(ctx, networkClient, args[1])
	if err != nil {
		return err
	}
	portID, fixedIP, err := serverPortForFloatingIP(ctx, networkClient, server.ID, flagValue(opts, "fixed-ip-address"))
	if err != nil {
		return err
	}
	update := floatingips.UpdateOpts{PortID: &portID}
	if fixedIP != "" {
		update.FixedIP = fixedIP
	}
	_, err = floatingips.Update(ctx, networkClient, fip.ID, update).Extract()
	return err
}

func serverRemoveFloatingIP(ctx context.Context, computeClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server remove floating ip requires <server> <ip-address>")
	}
	if _, err := findServer(ctx, computeClient, args[0]); err != nil {
		return err
	}
	fip, err := findFloatingIP(ctx, networkClient, args[1])
	if err != nil {
		return err
	}
	empty := ""
	_, err = floatingips.Update(ctx, networkClient, fip.ID, floatingips.UpdateOpts{PortID: &empty}).Extract()
	return err
}

func serverRemoveFixedIP(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server remove fixed ip requires <server> <ip-address>")
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	return serverRawAction(ctx, client, server.ID, "removeFixedIp", map[string]any{"address": args[1]}, nil, http.StatusAccepted, http.StatusNoContent)
}

func serverSecurityGroupAction(ctx context.Context, opts *Options, computeClient *gophercloud.ServiceClient, networkClient *gophercloud.ServiceClient, args []string, action string) error {
	if len(args) < 2 {
		return fmt.Errorf("server security group command requires <server> <security-group>")
	}
	server, err := findServer(ctx, computeClient, args[0])
	if err != nil {
		return err
	}
	for _, value := range args[1:] {
		groupName := value
		if networkClient != nil {
			group, err := findSecurityGroup(ctx, networkClient, value)
			if err == nil && group.Name != "" {
				groupName = group.Name
			}
		}
		if err := serverRawAction(ctx, computeClient, server.ID, action, map[string]any{"name": groupName}, nil, http.StatusAccepted, http.StatusNoContent); err != nil {
			return err
		}
	}
	return nil
}

func serverSimpleRawAction(ctx context.Context, client *gophercloud.ServiceClient, args []string, action string) error {
	if len(args) < 1 {
		return fmt.Errorf("server command requires <server>")
	}
	for _, value := range args {
		server, err := findServer(ctx, client, value)
		if err != nil {
			return err
		}
		if err := serverRawAction(ctx, client, server.ID, action, nil, nil, http.StatusAccepted, http.StatusNoContent); err != nil {
			return err
		}
	}
	return nil
}

func serverMigrationShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server migration show requires <server> <migration>")
	}
	client, err := computeClientWithMinimumMicroversion(ctx, client, "2.24")
	if err != nil {
		return err
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	var body struct {
		Migration map[string]any `json:"migration"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("servers", url.PathEscape(server.ID), "migrations", url.PathEscape(args[1])), &body, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	return renderShowOutput(stdout, opts, sortedFieldsFromMap(body.Migration, false))
}

func serverMigrationDeleteAction(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server migration abort requires <server> <migration>")
	}
	client, err := computeClientWithMinimumMicroversion(ctx, client, "2.24")
	if err != nil {
		return err
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	resp, err := client.Delete(ctx, client.ServiceURL("servers", url.PathEscape(server.ID), "migrations", url.PathEscape(args[1])), &gophercloud.RequestOpts{OkCodes: []int{http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return oscHTTPException(err)
}

func serverMigrationForceComplete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("server migration force complete requires <server> <migration>")
	}
	client, err := computeClientWithMinimumMicroversion(ctx, client, "2.22")
	if err != nil {
		return err
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	resp, err := client.Post(ctx, client.ServiceURL("servers", url.PathEscape(server.ID), "migrations", url.PathEscape(args[1]), "action"), map[string]any{"force_complete": nil}, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return oscHTTPException(err)
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
	uptimeParts := hypervisorUptimeParts{}
	if uptime != nil {
		uptimeParts = parseHypervisorUptime(uptime.Uptime)
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"aggregates", []any{}},
		{"cpu_info", tableValue{Value: map[string]any{}, Table: "None", Pretty: map[string]any{}}},
		{"host_ip", item.HostIP},
		{"host_time", uptimeParts.HostTime},
		{"hypervisor_hostname", item.HypervisorHostname},
		{"hypervisor_type", item.HypervisorType},
		{"hypervisor_version", item.HypervisorVersion},
		{"id", item.ID},
		{"load_average", uptimeParts.LoadAverage},
		{"service_host", item.Service.Host},
		{"service_id", item.Service.ID},
		{"state", item.State},
		{"status", item.Status},
		{"uptime", uptimeParts.Uptime},
		{"users", uptimeParts.Users},
	})
}

type hypervisorUptimeParts struct {
	HostTime    string
	Uptime      string
	Users       string
	LoadAverage string
}

var hypervisorUptimePattern = regexp.MustCompile(`^(\S+)\s+up\s+(.+),\s+(\d+)\s+users?,\s+load average:\s+(.+)$`)

func parseHypervisorUptime(value string) hypervisorUptimeParts {
	text := strings.TrimSpace(value)
	matches := hypervisorUptimePattern.FindStringSubmatch(text)
	if len(matches) != 5 {
		return hypervisorUptimeParts{Uptime: text}
	}
	return hypervisorUptimeParts{
		HostTime:    matches[1],
		Uptime:      matches[2],
		Users:       matches[3],
		LoadAverage: matches[4],
	}
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
	if raw, err := computeFlavorRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, flavorRawFields(raw))
	}
	return renderShowOutput(stdout, opts, flavorFields(item))
}

func computeFlavorRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		Flavor map[string]any `json:"flavor"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("flavors", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Flavor, nil
}

func flavorRawFields(raw map[string]any) []outputField {
	return []outputField{
		{"OS-FLV-DISABLED:disabled", raw["OS-FLV-DISABLED:disabled"]},
		{"OS-FLV-EXT-DATA:ephemeral", rawNumber(raw["OS-FLV-EXT-DATA:ephemeral"])},
		{"access_project_ids", raw["access_project_ids"]},
		{"description", raw["description"]},
		{"disk", rawNumber(raw["disk"])},
		{"id", raw["id"]},
		{"name", raw["name"]},
		{"os-flavor-access:is_public", raw["os-flavor-access:is_public"]},
		{"properties", flavorPropertiesValue(raw["extra_specs"])},
		{"ram", rawNumber(raw["ram"])},
		{"rxtx_factor", flavorRxTxValue(raw["rxtx_factor"])},
		{"swap", flavorSwapValue(raw["swap"])},
		{"vcpus", rawNumber(raw["vcpus"])},
	}
}

func flavorFields(item *flavors.Flavor) []outputField {
	return []outputField{
		{"OS-FLV-DISABLED:disabled", nil},
		{"OS-FLV-EXT-DATA:ephemeral", item.Ephemeral},
		{"access_project_ids", nil},
		{"description", nilIfEmpty(item.Description)},
		{"disk", item.Disk},
		{"id", item.ID},
		{"name", item.Name},
		{"os-flavor-access:is_public", item.IsPublic},
		{"properties", flavorPropertiesValue(item.ExtraSpecs)},
		{"ram", item.RAM},
		{"rxtx_factor", flavorRxTxValue(item.RxTxFactor)},
		{"swap", item.Swap},
		{"vcpus", item.VCPUs},
	}
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
	sort.SliceStable(items, func(i int, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{"ID": item.ID, "Name": prettyImageValue(item.Name), "Status": string(item.Status)})
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
	add("name", prettyImageValue(item.Name), item.Name != "")
	add("owner", item.Owner, item.Owner != "")
	if len(properties) > 0 {
		add("properties", imagePropertiesValue(properties), true)
	}
	add("protected", item.Protected, true)
	add("schema", item.Schema, item.Schema != "")
	add("size", item.SizeBytes, item.SizeBytes > 0 || item.Status == images.ImageStatusActive)
	add("status", string(item.Status), item.Status != "")
	tags := item.Tags
	if tags == nil {
		tags = []string{}
	}
	add("tags", blankEmptyStringListValue(tags), true)
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
		{"image_name", prettyImageValue(item.ImageName)},
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
	return renderShowOutput(stdout, opts, imageCreateFields(item))
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
	request := map[string]any{"name": args[1]}
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
		{"admin_state_up", adminStateValue(raw["admin_state_up"])},
		{"availability_zone_hints", blankEmptyListValue(raw["availability_zone_hints"])},
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
		{"router:external", routerExternalValue(raw["router:external"])},
		{"segments", raw["segments"]},
		{"shared", raw["shared"]},
		{"status", raw["status"]},
		{"subnets", raw["subnets"]},
		{"tags", blankEmptyListValue(raw["tags"])},
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
		addresses := any(item.Addresses)
		if prettyOutput(opts) {
			addresses = prettyAddressList(item.Addresses)
		}
		rows = append(rows, outputRow{
			"ID":          item.ID,
			"Name":        item.Name,
			"Description": item.Description,
			"Project":     item.ProjectID,
			"Addresses":   addresses,
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
		return renderShowOutput(stdout, opts, prettyNetworkIPOutputFields(opts, addressGroupRawFields(raw)))
	}
	return renderShowOutput(stdout, opts, prettyNetworkIPOutputFields(opts, addressGroupFields(item)))
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
			"Alive":             agentAliveValue(item.Alive),
			"State":             adminStateValue(item.AdminStateUp),
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
		{"admin_state_up", adminStateValue(item["admin_state_up"])},
		{"agent_type", item["agent_type"]},
		{"alive", agentAliveValue(item["alive"])},
		{"availability_zone", item["availability_zone"]},
		{"binary", item["binary"]},
		{"configuration", networkAgentConfigurationValue(item["configurations"])},
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

type networkRBACCreateOpts struct {
	Values map[string]any
}

func (opts networkRBACCreateOpts) ToRBACPolicyCreateMap() (map[string]any, error) {
	return map[string]any{"rbac_policy": opts.Values}, nil
}

type networkRBACUpdateOpts struct {
	Values map[string]any
}

func (opts networkRBACUpdateOpts) ToRBACPolicyUpdateMap() (map[string]any, error) {
	return map[string]any{"rbac_policy": opts.Values}, nil
}

func networkRBACCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network rbac create requires <rbac-object>")
	}
	values, err := networkRBACCreateValues(ctx, opts, networkClient, identityClient, args[0])
	if err != nil {
		return err
	}
	result := rbacpolicies.Create(ctx, networkClient, networkRBACCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if raw, ok := networkRBACRawFromBody(result.Body); ok {
		return renderShowOutput(stdout, opts, networkRBACRawFields(raw))
	}
	return renderShowOutput(stdout, opts, networkRBACFields(item))
}

func networkRBACCreateValues(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, objectNameOrID string) (map[string]any, error) {
	objectType := flagValue(opts, "type")
	action := flagValue(opts, "action")
	if objectType == "" {
		return nil, fmt.Errorf("argument --type is required")
	}
	if action == "" {
		return nil, fmt.Errorf("argument --action is required")
	}
	if flagChanged(opts, "target-project") == boolFlag(opts, "target-all-projects") {
		return nil, fmt.Errorf("one of the arguments --target-project --target-all-projects is required")
	}
	objectID, err := networkRBACObjectID(ctx, networkClient, objectType, objectNameOrID)
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		"object_type":   objectType,
		"action":        action,
		"object_id":     objectID,
		"target_tenant": "*",
	}
	if targetProject := flagValue(opts, "target-project"); targetProject != "" {
		project, err := findProjectWithDomain(ctx, identityClient, targetProject, flagValue(opts, "target-project-domain"))
		if err != nil {
			return nil, err
		}
		values["target_tenant"] = project.ID
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return nil, err
		}
		values["project_id"] = project.ID
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

func networkRBACDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network rbac delete requires <rbac-policy> [<rbac-policy> ...]")
	}
	failures := 0
	for _, policyID := range args {
		if err := rbacpolicies.Delete(ctx, client, policyID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d RBAC policies failed to delete.", failures, len(args))
	}
	return nil
}

func networkRBACSet(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network rbac set requires <rbac-policy>")
	}
	values := map[string]any{}
	if targetProject := flagValue(opts, "target-project"); targetProject != "" {
		project, err := findProjectWithDomain(ctx, identityClient, targetProject, flagValue(opts, "target-project-domain"))
		if err != nil {
			return err
		}
		values["target_tenant"] = project.ID
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	for key, value := range extra {
		values[key] = value
	}
	if len(values) == 0 {
		return nil
	}
	_, err = rbacpolicies.Update(ctx, networkClient, args[0], networkRBACUpdateOpts{Values: values}).Extract()
	return err
}

func networkRBACShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network rbac show requires <rbac-policy>")
	}
	item, err := rbacpolicies.Get(ctx, client, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, networkRBACFields(item))
}

func networkRBACObjectID(ctx context.Context, client *gophercloud.ServiceClient, objectType string, value string) (string, error) {
	switch objectType {
	case "network":
		item, err := findNetwork(ctx, client, value)
		if err != nil {
			return "", err
		}
		return item.ID, nil
	case "qos_policy":
		item, err := findNetworkQoSPolicy(ctx, client, value)
		if err != nil {
			return "", err
		}
		return item.ID, nil
	case "security_group":
		item, err := findSecurityGroup(ctx, client, value)
		if err != nil {
			return "", err
		}
		return item.ID, nil
	case "address_scope":
		item, err := findAddressScope(ctx, client, value)
		if err != nil {
			return "", err
		}
		return item.ID, nil
	case "subnetpool":
		item, err := findSubnetPool(ctx, client, value)
		if err != nil {
			return "", err
		}
		return item.ID, nil
	case "address_group":
		item, err := findAddressGroup(ctx, client, value)
		if err != nil {
			return "", err
		}
		return item.ID, nil
	default:
		return "", fmt.Errorf("invalid RBAC object type %q", objectType)
	}
}

func networkRBACRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		RBACPolicy map[string]any `json:"rbac_policy"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.RBACPolicy == nil {
		return nil, false
	}
	return wrapper.RBACPolicy, true
}

func networkRBACRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"action":            true,
		"id":                true,
		"location":          true,
		"name":              true,
		"object_id":         true,
		"object_type":       true,
		"project_id":        true,
		"target_project_id": true,
		"target_tenant":     true,
		"tenant_id":         true,
	}
	fields := []outputField{
		{"action", raw["action"]},
		{"id", raw["id"]},
		{"object_id", raw["object_id"]},
		{"object_type", raw["object_type"]},
		{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		{"target_project_id", firstPresent(raw, "target_project_id", "target_tenant")},
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

func networkRBACFields(item *rbacpolicies.RBACPolicy) []outputField {
	return []outputField{
		{"action", item.Action},
		{"id", item.ID},
		{"object_id", item.ObjectID},
		{"object_type", item.ObjectType},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"target_project_id", item.TargetTenant},
	}
}

type networkSegmentCreateOpts struct {
	Values map[string]any
}

func (opts networkSegmentCreateOpts) ToSegmentCreateMap() (map[string]any, error) {
	return map[string]any{"segment": opts.Values}, nil
}

type networkSegmentUpdateOpts struct {
	Values map[string]any
}

func (opts networkSegmentUpdateOpts) ToSegmentUpdateMap() (map[string]any, error) {
	return map[string]any{"segment": opts.Values}, nil
}

func networkSegmentCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network segment create requires <name>")
	}
	values, err := networkSegmentCreateValues(ctx, opts, client, args[0])
	if err != nil {
		return err
	}
	result := segments.Create(ctx, client, networkSegmentCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if raw, ok := networkSegmentRawFromBody(result.Body); ok {
		return renderShowOutput(stdout, opts, networkSegmentRawFields(raw))
	}
	return renderShowOutput(stdout, opts, networkSegmentFields(item))
}

func networkSegmentCreateValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, name string) (map[string]any, error) {
	if !flagChanged(opts, "network") || flagValue(opts, "network") == "" {
		return nil, fmt.Errorf("the following arguments are required: --network")
	}
	if !flagChanged(opts, "network-type") || flagValue(opts, "network-type") == "" {
		return nil, fmt.Errorf("the following arguments are required: --network-type")
	}
	networkType := flagValue(opts, "network-type")
	if !networkSegmentTypeValid(networkType) {
		return nil, fmt.Errorf("invalid choice: %q (choose from 'flat', 'geneve', 'gre', 'local', 'vlan', 'vxlan')", networkType)
	}
	network, err := findNetwork(ctx, client, flagValue(opts, "network"))
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		"name":         name,
		"network_id":   network.ID,
		"network_type": networkType,
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if flagChanged(opts, "physical-network") {
		values["physical_network"] = flagValue(opts, "physical-network")
	}
	if flagChanged(opts, "segment") {
		values["segmentation_id"] = intFlag(opts, "segment")
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

func networkSegmentDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network segment delete requires <network-segment> [<network-segment> ...]")
	}
	failures := 0
	for _, segmentArg := range args {
		item, err := findNetworkSegment(ctx, client, segmentArg)
		if err != nil {
			failures++
			continue
		}
		if err := segments.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d network segments failed to delete.", failures, len(args))
	}
	return nil
}

func networkSegmentSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network segment set requires <network-segment>")
	}
	item, err := findNetworkSegment(ctx, client, args[0])
	if err != nil {
		return err
	}
	values, err := networkSegmentSetValues(opts)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	_, err = segments.Update(ctx, client, item.ID, networkSegmentUpdateOpts{Values: values}).Extract()
	return err
}

func networkSegmentSetValues(opts *Options) (map[string]any, error) {
	values := map[string]any{}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
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

func networkSegmentTypeValid(networkType string) bool {
	switch networkType {
	case "flat", "geneve", "gre", "local", "vlan", "vxlan":
		return true
	default:
		return false
	}
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
	if raw, err := neutronSegmentRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, networkSegmentRawFields(raw))
	}
	return renderShowOutput(stdout, opts, networkSegmentFields(item))
}

func neutronSegmentRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		Segment map[string]any `json:"segment"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("segments", id), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Segment, nil
}

func networkSegmentRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		Segment map[string]any `json:"segment"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.Segment == nil {
		return nil, false
	}
	return wrapper.Segment, true
}

func networkSegmentRawFields(raw map[string]any) []outputField {
	hidden := map[string]bool{
		"location":  true,
		"tenant_id": true,
	}
	namesByKey := map[string]bool{
		"created_at":       true,
		"description":      true,
		"id":               true,
		"name":             true,
		"network_id":       true,
		"network_type":     true,
		"physical_network": true,
		"revision_number":  true,
		"segmentation_id":  true,
		"updated_at":       true,
	}
	for name := range raw {
		if !hidden[name] {
			namesByKey[name] = true
		}
	}
	names := make([]string, 0, len(namesByKey))
	for name := range namesByKey {
		names = append(names, name)
	}
	sort.Strings(names)
	fields := make([]outputField, 0, len(names))
	for _, name := range names {
		fields = append(fields, outputField{name, networkSegmentRawValue(name, raw[name])})
	}
	return fields
}

func networkSegmentRawValue(name string, value any) any {
	switch name {
	case "revision_number", "segmentation_id":
		return rawNumber(value)
	default:
		return value
	}
}

func networkSegmentFields(item *segments.Segment) []outputField {
	return []outputField{
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
	}
}

type networkTrunkCreateOpts struct {
	Values map[string]any
}

func (opts networkTrunkCreateOpts) ToTrunkCreateMap() (map[string]any, error) {
	return map[string]any{"trunk": opts.Values}, nil
}

type networkTrunkUpdateOpts struct {
	Values map[string]any
}

func (opts networkTrunkUpdateOpts) ToTrunkUpdateMap() (map[string]any, error) {
	return map[string]any{"trunk": opts.Values}, nil
}

type networkTrunkSubportsOpts struct {
	Subports []map[string]any
}

func (opts networkTrunkSubportsOpts) ToTrunkAddSubportsMap() (map[string]any, error) {
	return map[string]any{"sub_ports": opts.Subports}, nil
}

func (opts networkTrunkSubportsOpts) ToTrunkRemoveSubportsMap() (map[string]any, error) {
	return map[string]any{"sub_ports": opts.Subports}, nil
}

func networkTrunkCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network trunk create requires <name>")
	}
	values, err := networkTrunkCreateValues(ctx, opts, networkClient, identityClient, args[0])
	if err != nil {
		return err
	}
	result := trunks.Create(ctx, networkClient, networkTrunkCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if raw, ok := networkTrunkRawFromBody(result.Body); ok {
		return renderShowOutput(stdout, opts, networkTrunkRawFields(raw))
	}
	return renderShowOutput(stdout, opts, networkTrunkFields(item))
}

func networkTrunkCreateValues(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, name string) (map[string]any, error) {
	if !flagChanged(opts, "parent-port") || flagValue(opts, "parent-port") == "" {
		return nil, fmt.Errorf("the following arguments are required: --parent-port")
	}
	if boolFlag(opts, "enable") && boolFlag(opts, "disable") {
		return nil, fmt.Errorf("argument --disable: not allowed with argument --enable")
	}
	parentPort, err := findPort(ctx, networkClient, flagValue(opts, "parent-port"))
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		"name":           name,
		"port_id":        parentPort.ID,
		"admin_state_up": true,
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if boolFlag(opts, "disable") {
		values["admin_state_up"] = false
	}
	if subports, err := networkTrunkParseSubports(ctx, networkClient, flagValues(opts, "subport")); err != nil {
		return nil, err
	} else if len(subports) > 0 {
		values["sub_ports"] = subports
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return nil, err
		}
		values["tenant_id"] = project.ID
	}
	return values, nil
}

func networkTrunkDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network trunk delete requires <trunk> [<trunk> ...]")
	}
	failures := 0
	for _, trunkArg := range args {
		item, err := findNetworkTrunk(ctx, client, trunkArg)
		if err != nil {
			failures++
			continue
		}
		if err := trunks.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d trunks failed to delete.", failures, len(args))
	}
	return nil
}

func networkTrunkSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network trunk set requires <trunk>")
	}
	item, err := findNetworkTrunk(ctx, client, args[0])
	if err != nil {
		return err
	}
	values, err := networkTrunkSetValues(opts)
	if err != nil {
		return err
	}
	if len(values) > 0 {
		if _, err := trunks.Update(ctx, client, item.ID, networkTrunkUpdateOpts{Values: values}).Extract(); err != nil {
			return fmt.Errorf("Failed to set trunk %q: %w", args[0], err)
		}
	}
	subports, err := networkTrunkParseSubports(ctx, client, flagValues(opts, "subport"))
	if err != nil {
		return err
	}
	if len(subports) > 0 {
		if _, err := trunks.AddSubports(ctx, client, item.ID, networkTrunkSubportsOpts{Subports: subports}).Extract(); err != nil {
			return fmt.Errorf("Failed to add subports to trunk %q: %w", args[0], err)
		}
	}
	return nil
}

func networkTrunkSetValues(opts *Options) (map[string]any, error) {
	if boolFlag(opts, "enable") && boolFlag(opts, "disable") {
		return nil, fmt.Errorf("argument --disable: not allowed with argument --enable")
	}
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if boolFlag(opts, "enable") {
		values["admin_state_up"] = true
	}
	if boolFlag(opts, "disable") {
		values["admin_state_up"] = false
	}
	return values, nil
}

func networkTrunkUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network trunk unset requires <trunk>")
	}
	if len(flagValues(opts, "subport")) == 0 {
		return fmt.Errorf("the following arguments are required: --subport")
	}
	item, err := findNetworkTrunk(ctx, client, args[0])
	if err != nil {
		return err
	}
	subports, err := networkTrunkParseRemoveSubports(ctx, client, flagValues(opts, "subport"))
	if err != nil {
		return err
	}
	_, err = trunks.RemoveSubports(ctx, client, item.ID, networkTrunkSubportsOpts{Subports: subports}).Extract()
	return err
}

func networkSubportList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	if !flagChanged(opts, "trunk") || flagValue(opts, "trunk") == "" {
		return fmt.Errorf("the following arguments are required: --trunk")
	}
	item, err := findNetworkTrunk(ctx, client, flagValue(opts, "trunk"))
	if err != nil {
		return err
	}
	subports, err := trunks.GetSubports(ctx, client, item.ID).Extract()
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(subports))
	for _, subport := range subports {
		rows = append(rows, outputRow{
			"Port":              subport.PortID,
			"Segmentation Type": subport.SegmentationType,
			"Segmentation ID":   subport.SegmentationID,
		})
	}
	return renderListOutput(stdout, opts, []string{"Port", "Segmentation Type", "Segmentation ID"}, rows)
}

func networkTrunkParseSubports(ctx context.Context, client *gophercloud.ServiceClient, values []string) ([]map[string]any, error) {
	entries, err := parseKeyValueEntries(values, "subport", "port")
	if err != nil {
		return nil, err
	}
	subports := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		port, err := findPort(ctx, client, entry["port"])
		if err != nil {
			return nil, err
		}
		subport, err := networkTrunkSubportMap(entry, port.ID)
		if err != nil {
			return nil, err
		}
		subports = append(subports, subport)
	}
	return subports, nil
}

func networkTrunkSubportMap(entry map[string]string, portID string) (map[string]any, error) {
	subport := map[string]any{"port_id": portID}
	if raw := entry["segmentation-id"]; raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("Segmentation-id %q is not an integer", raw)
		}
		subport["segmentation_id"] = parsed
	}
	if raw := entry["segmentation-type"]; raw != "" {
		subport["segmentation_type"] = raw
	}
	return subport, nil
}

func networkTrunkParseRemoveSubports(ctx context.Context, client *gophercloud.ServiceClient, values []string) ([]map[string]any, error) {
	subports := make([]map[string]any, 0, len(values))
	for _, value := range values {
		port, err := findPort(ctx, client, value)
		if err != nil {
			return nil, err
		}
		subports = append(subports, map[string]any{"port_id": port.ID})
	}
	return subports, nil
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
		}
		if boolFlag(opts, "long") {
			row["Status"] = item.Status
			row["State"] = adminStateLabel(item.AdminStateUp)
			row["Created At"] = oscTime(item.CreatedAt)
			row["Updated At"] = oscTime(item.UpdatedAt)
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Name", "Parent Port", "Description"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Status", "State", "Created At", "Updated At")
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
	if raw, err := neutronTrunkRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, networkTrunkRawFields(raw))
	}
	return renderShowOutput(stdout, opts, networkTrunkFields(item))
}

func neutronTrunkRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		Trunk map[string]any `json:"trunk"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("trunks", id), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Trunk, nil
}

func networkTrunkRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		Trunk map[string]any `json:"trunk"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.Trunk == nil {
		return nil, false
	}
	return wrapper.Trunk, true
}

func networkTrunkRawFields(raw map[string]any) []outputField {
	hidden := map[string]bool{
		"admin_state_up": true,
		"location":       true,
		"tenant_id":      true,
	}
	namesByKey := map[string]bool{
		"created_at":        true,
		"description":       true,
		"id":                true,
		"is_admin_state_up": true,
		"name":              true,
		"port_id":           true,
		"project_id":        true,
		"revision_number":   true,
		"status":            true,
		"sub_ports":         true,
		"tags":              true,
		"updated_at":        true,
	}
	for name := range raw {
		if !hidden[name] {
			namesByKey[name] = true
		}
	}
	names := make([]string, 0, len(namesByKey))
	for name := range namesByKey {
		names = append(names, name)
	}
	sort.Strings(names)
	fields := make([]outputField, 0, len(names))
	for _, name := range names {
		fields = append(fields, outputField{name, networkTrunkRawValue(raw, name)})
	}
	return fields
}

func networkTrunkRawValue(raw map[string]any, name string) any {
	switch name {
	case "is_admin_state_up":
		return firstPresent(raw, "is_admin_state_up", "admin_state_up")
	case "project_id":
		return firstPresent(raw, "project_id", "tenant_id")
	case "revision_number":
		return rawNumber(raw[name])
	default:
		return raw[name]
	}
}

func networkTrunkFields(item *trunks.Trunk) []outputField {
	return []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"id", item.ID},
		{"is_admin_state_up", item.AdminStateUp},
		{"name", item.Name},
		{"port_id", item.PortID},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"revision_number", item.RevisionNumber},
		{"status", item.Status},
		{"sub_ports", trunkSubports(item.Subports)},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	}
}

func adminStateLabel(value bool) string {
	if value {
		return "UP"
	}
	return "DOWN"
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

type networkQoSPolicyCreateOpts struct {
	Values map[string]any
}

func (opts networkQoSPolicyCreateOpts) ToPolicyCreateMap() (map[string]any, error) {
	return map[string]any{"policy": opts.Values}, nil
}

type networkQoSPolicyUpdateOpts struct {
	Values map[string]any
}

func (opts networkQoSPolicyUpdateOpts) ToPolicyUpdateMap() (map[string]any, error) {
	return map[string]any{"policy": opts.Values}, nil
}

func networkQoSPolicyCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network qos policy create requires <name>")
	}
	values, err := networkQoSPolicyCreateValues(ctx, opts, identityClient, args[0])
	if err != nil {
		return err
	}
	result := qospolicies.Create(ctx, networkClient, networkQoSPolicyCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if raw, ok := networkQoSPolicyRawFromBody(result.Body); ok {
		return renderShowOutput(stdout, opts, networkQoSPolicyRawFields(raw))
	}
	return renderShowOutput(stdout, opts, networkQoSPolicyFields(item))
}

func networkQoSPolicyCreateValues(ctx context.Context, opts *Options, identityClient *gophercloud.ServiceClient, name string) (map[string]any, error) {
	if boolFlag(opts, "share") && boolFlag(opts, "no-share") {
		return nil, fmt.Errorf("argument --no-share: not allowed with argument --share")
	}
	if boolFlag(opts, "default") && boolFlag(opts, "no-default") {
		return nil, fmt.Errorf("argument --no-default: not allowed with argument --default")
	}
	values := map[string]any{"name": name}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if boolFlag(opts, "share") {
		values["shared"] = true
	}
	if boolFlag(opts, "no-share") {
		values["shared"] = false
	}
	if boolFlag(opts, "default") {
		values["is_default"] = true
	}
	if boolFlag(opts, "no-default") {
		values["is_default"] = false
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return nil, err
		}
		values["project_id"] = project.ID
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

func networkQoSPolicyDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network qos policy delete requires <qos-policy> [<qos-policy> ...]")
	}
	failures := 0
	for _, policyArg := range args {
		item, err := findNetworkQoSPolicy(ctx, client, policyArg)
		if err != nil {
			failures++
			continue
		}
		if err := qospolicies.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d QoS policies failed to delete.", failures, len(args))
	}
	return nil
}

func networkQoSPolicySet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network qos policy set requires <qos-policy>")
	}
	if boolFlag(opts, "share") && boolFlag(opts, "no-share") {
		return fmt.Errorf("argument --no-share: not allowed with argument --share")
	}
	if boolFlag(opts, "default") && boolFlag(opts, "no-default") {
		return fmt.Errorf("argument --no-default: not allowed with argument --default")
	}
	item, err := findNetworkQoSPolicy(ctx, client, args[0])
	if err != nil {
		return err
	}
	values, err := networkQoSPolicySetValues(opts)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	_, err = qospolicies.Update(ctx, client, item.ID, networkQoSPolicyUpdateOpts{Values: values}).Extract()
	return err
}

func networkQoSPolicySetValues(opts *Options) (map[string]any, error) {
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if boolFlag(opts, "share") {
		values["shared"] = true
	}
	if boolFlag(opts, "no-share") {
		values["shared"] = false
	}
	if boolFlag(opts, "default") {
		values["is_default"] = true
	}
	if boolFlag(opts, "no-default") {
		values["is_default"] = false
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

func networkQoSPolicyShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network qos policy show requires <qos-policy>")
	}
	item, err := findNetworkQoSPolicy(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, networkQoSPolicyFields(item))
}

func networkQoSPolicyRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		Policy map[string]any `json:"policy"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.Policy == nil {
		return nil, false
	}
	return wrapper.Policy, true
}

func networkQoSPolicyRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"created_at":      true,
		"description":     true,
		"id":              true,
		"is_default":      true,
		"location":        true,
		"name":            true,
		"project_id":      true,
		"revision_number": true,
		"rules":           true,
		"shared":          true,
		"tags":            true,
		"tenant_id":       true,
		"updated_at":      true,
	}
	fields := []outputField{
		{"created_at", raw["created_at"]},
		{"description", raw["description"]},
		{"id", raw["id"]},
		{"is_default", raw["is_default"]},
		{"name", raw["name"]},
		{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		{"revision_number", rawNumber(raw["revision_number"])},
		{"rules", raw["rules"]},
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

func networkQoSPolicyFields(item *qospolicies.Policy) []outputField {
	return []outputField{
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
	}
}

const (
	networkQoSRuleBandwidthLimit      = "bandwidth-limit"
	networkQoSRuleDSCPMarking         = "dscp-marking"
	networkQoSRuleMinimumBandwidth    = "minimum-bandwidth"
	networkQoSRuleMinimumPacketRate   = "minimum-packet-rate"
	networkQoSRuleBandwidthLimitKey   = "bandwidth_limit_rule"
	networkQoSRuleDSCPMarkingKey      = "dscp_marking_rule"
	networkQoSRuleMinimumBandwidthKey = "minimum_bandwidth_rule"
	networkQoSRuleMinimumPacketKey    = "minimum_packet_rate_rule"
)

type networkQoSRuleOpts struct {
	Values map[string]any
}

func (opts networkQoSRuleOpts) ToBandwidthLimitRuleCreateMap() (map[string]any, error) {
	return map[string]any{networkQoSRuleBandwidthLimitKey: opts.Values}, nil
}

func (opts networkQoSRuleOpts) ToBandwidthLimitRuleUpdateMap() (map[string]any, error) {
	return map[string]any{networkQoSRuleBandwidthLimitKey: opts.Values}, nil
}

func (opts networkQoSRuleOpts) ToDSCPMarkingRuleCreateMap() (map[string]any, error) {
	return map[string]any{networkQoSRuleDSCPMarkingKey: opts.Values}, nil
}

func (opts networkQoSRuleOpts) ToDSCPMarkingRuleUpdateMap() (map[string]any, error) {
	return map[string]any{networkQoSRuleDSCPMarkingKey: opts.Values}, nil
}

func (opts networkQoSRuleOpts) ToMinimumBandwidthRuleCreateMap() (map[string]any, error) {
	return map[string]any{networkQoSRuleMinimumBandwidthKey: opts.Values}, nil
}

func (opts networkQoSRuleOpts) ToMinimumBandwidthRuleUpdateMap() (map[string]any, error) {
	return map[string]any{networkQoSRuleMinimumBandwidthKey: opts.Values}, nil
}

func networkQoSRuleCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network qos rule create requires <qos-policy>")
	}
	ruleType, values, err := networkQoSRuleCreateValues(opts)
	if err != nil {
		return err
	}
	policy, err := findNetworkQoSPolicy(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("Failed to create Network QoS rule: %w", err)
	}
	raw, err := networkQoSRuleCreateRaw(ctx, client, policy.ID, ruleType, values)
	if err != nil {
		return fmt.Errorf("Failed to create Network QoS rule: %w", err)
	}
	return renderShowOutput(stdout, opts, networkQoSRuleRawFields(raw, ruleType))
}

func networkQoSRuleDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network qos rule delete requires <qos-policy> <rule-id>")
	}
	policy, err := findNetworkQoSPolicy(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("Failed to delete Network QoS rule ID %q: %w", args[1], err)
	}
	ruleType, err := networkQoSRuleTypeFromPolicy(policy, args[1])
	if err != nil {
		return fmt.Errorf("Failed to delete Network QoS rule ID %q: %w", args[1], err)
	}
	if err := networkQoSRuleDeleteRaw(ctx, client, policy.ID, args[1], ruleType); err != nil {
		return fmt.Errorf("Failed to delete Network QoS rule ID %q: %w", args[1], err)
	}
	return nil
}

func networkQoSRuleList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network qos rule list requires <qos-policy>")
	}
	policy, err := findNetworkQoSPolicy(ctx, client, args[0])
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(policy.Rules))
	for _, item := range policy.Rules {
		rows = append(rows, outputRow{
			"ID":              item["id"],
			"QoS Policy ID":   policy.ID,
			"Type":            item["type"],
			"Max Kbps":        rawNumber(item["max_kbps"]),
			"Max Burst Kbits": rawNumber(item["max_burst_kbps"]),
			"Min Kbps":        rawNumber(item["min_kbps"]),
			"Min Kpps":        rawNumber(item["min_kpps"]),
			"DSCP mark":       rawNumber(item["dscp_mark"]),
			"Direction":       item["direction"],
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "QoS Policy ID", "Type", "Max Kbps", "Max Burst Kbits", "Min Kbps", "Min Kpps", "DSCP mark", "Direction"}, rows)
}

func networkQoSRuleSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network qos rule set requires <qos-policy> <rule-id>")
	}
	policy, err := findNetworkQoSPolicy(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("Failed to set Network QoS rule ID %q: %w", args[1], err)
	}
	ruleType, err := networkQoSRuleTypeFromPolicy(policy, args[1])
	if err != nil {
		return fmt.Errorf("Failed to set Network QoS rule ID %q: %w", args[1], err)
	}
	values, err := networkQoSRuleValues(opts, ruleType, false)
	if err != nil {
		return fmt.Errorf("Failed to set Network QoS rule ID %q: %w", args[1], err)
	}
	if len(values) == 0 {
		return nil
	}
	if _, err := networkQoSRuleUpdateRaw(ctx, client, policy.ID, args[1], ruleType, values); err != nil {
		return fmt.Errorf("Failed to set Network QoS rule ID %q: %w", args[1], err)
	}
	return nil
}

func networkQoSRuleShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network qos rule show requires <qos-policy> <rule-id>")
	}
	policy, err := findNetworkQoSPolicy(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("Failed to show Network QoS rule ID %q: %w", args[1], err)
	}
	ruleType, err := networkQoSRuleTypeFromPolicy(policy, args[1])
	if err != nil {
		return fmt.Errorf("Failed to show Network QoS rule ID %q: %w", args[1], err)
	}
	raw, err := networkQoSRuleGetRaw(ctx, client, policy.ID, args[1], ruleType)
	if err != nil {
		return fmt.Errorf("Failed to show Network QoS rule ID %q: %w", args[1], err)
	}
	return renderShowOutput(stdout, opts, networkQoSRuleRawFields(raw, ruleType))
}

func networkQoSRuleCreateValues(opts *Options) (string, map[string]any, error) {
	if !flagChanged(opts, "type") || strings.TrimSpace(flagValue(opts, "type")) == "" {
		return "", nil, fmt.Errorf("the following arguments are required: --type")
	}
	ruleType := strings.TrimSpace(flagValue(opts, "type"))
	if !networkQoSRuleTypeValid(ruleType) {
		return "", nil, fmt.Errorf("invalid choice: %q (choose from 'minimum-bandwidth', 'minimum-packet-rate', 'dscp-marking', 'bandwidth-limit')", ruleType)
	}
	values, err := networkQoSRuleValues(opts, ruleType, true)
	return ruleType, values, err
}

func networkQoSRuleValues(opts *Options, ruleType string, isCreate bool) (map[string]any, error) {
	if err := validateNetworkQoSRuleDirection(opts, ruleType); err != nil {
		return nil, err
	}
	values := map[string]any{}
	if flagChanged(opts, "max-kbps") {
		values["max_kbps"] = intFlag(opts, "max-kbps")
	}
	if flagChanged(opts, "max-burst-kbits") {
		values["max_burst_kbps"] = intFlag(opts, "max-burst-kbits")
	}
	if flagChanged(opts, "dscp-mark") {
		values["dscp_mark"] = intFlag(opts, "dscp-mark")
	}
	if flagChanged(opts, "min-kbps") {
		values["min_kbps"] = intFlag(opts, "min-kbps")
	}
	if flagChanged(opts, "min-kpps") {
		values["min_kpps"] = intFlag(opts, "min-kpps")
	}
	if boolFlag(opts, "ingress") {
		values["direction"] = "ingress"
	}
	if boolFlag(opts, "egress") {
		values["direction"] = "egress"
	}
	if boolFlag(opts, "any") {
		values["direction"] = "any"
	}
	if err := checkNetworkQoSRuleTypeParameters(values, ruleType, isCreate); err != nil {
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

func validateNetworkQoSRuleDirection(opts *Options, ruleType string) error {
	var directions []string
	for _, name := range []string{"ingress", "egress", "any"} {
		if boolFlag(opts, name) {
			directions = append(directions, name)
		}
	}
	if len(directions) > 1 {
		return fmt.Errorf("argument --%s: not allowed with argument --%s", directions[1], directions[0])
	}
	if boolFlag(opts, "any") && ruleType != networkQoSRuleMinimumPacketRate {
		return fmt.Errorf("Direction \"any\" can only be used with %s rule type", networkQoSRuleMinimumPacketRate)
	}
	return nil
}

func checkNetworkQoSRuleTypeParameters(values map[string]any, ruleType string, isCreate bool) error {
	required := networkQoSRuleMandatoryParameters(ruleType)
	if isCreate {
		var missing []string
		for _, name := range required {
			if _, ok := values[name]; !ok {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			sort.Strings(required)
			return fmt.Errorf("\"Create\" rule command for type %q requires arguments: %s", ruleType, strings.Join(required, ", "))
		}
	}
	typeParams := map[string]bool{}
	for _, name := range append(networkQoSRuleMandatoryParameters(ruleType), networkQoSRuleOptionalParameters(ruleType)...) {
		typeParams[name] = true
	}
	for _, otherType := range []string{networkQoSRuleMinimumBandwidth, networkQoSRuleMinimumPacketRate, networkQoSRuleDSCPMarking, networkQoSRuleBandwidthLimit} {
		if otherType == ruleType {
			continue
		}
		for _, name := range networkQoSRuleMandatoryParameters(otherType) {
			if typeParams[name] {
				continue
			}
			if _, ok := values[name]; ok {
				allowed := make([]string, 0, len(typeParams))
				for allowedName := range typeParams {
					allowed = append(allowed, allowedName)
				}
				sort.Strings(allowed)
				return fmt.Errorf("Rule type %q only requires arguments: %s", ruleType, strings.Join(allowed, ", "))
			}
		}
	}
	return nil
}

func networkQoSRuleMandatoryParameters(ruleType string) []string {
	switch ruleType {
	case networkQoSRuleMinimumBandwidth:
		return []string{"min_kbps", "direction"}
	case networkQoSRuleMinimumPacketRate:
		return []string{"min_kpps", "direction"}
	case networkQoSRuleDSCPMarking:
		return []string{"dscp_mark"}
	case networkQoSRuleBandwidthLimit:
		return []string{"max_kbps"}
	default:
		return nil
	}
}

func networkQoSRuleOptionalParameters(ruleType string) []string {
	switch ruleType {
	case networkQoSRuleBandwidthLimit:
		return []string{"direction", "max_burst_kbps"}
	default:
		return nil
	}
}

func networkQoSRuleTypeValid(ruleType string) bool {
	switch ruleType {
	case networkQoSRuleMinimumBandwidth, networkQoSRuleMinimumPacketRate, networkQoSRuleDSCPMarking, networkQoSRuleBandwidthLimit:
		return true
	default:
		return false
	}
}

func networkQoSRuleTypeFromPolicy(policy *qospolicies.Policy, ruleID string) (string, error) {
	for _, rule := range policy.Rules {
		if fmt.Sprint(rule["id"]) != ruleID {
			continue
		}
		ruleType := strings.ReplaceAll(fmt.Sprint(rule["type"]), "_", "-")
		if !networkQoSRuleTypeValid(ruleType) {
			return "", fmt.Errorf("unsupported QoS rule type %q", rule["type"])
		}
		return ruleType, nil
	}
	return "", fmt.Errorf("Rule ID %s not found", ruleID)
}

func networkQoSRuleCreateRaw(ctx context.Context, client *gophercloud.ServiceClient, policyID string, ruleType string, values map[string]any) (map[string]any, error) {
	opts := networkQoSRuleOpts{Values: values}
	switch ruleType {
	case networkQoSRuleBandwidthLimit:
		result := qosrules.CreateBandwidthLimitRule(ctx, client, policyID, opts)
		if result.Err != nil {
			return nil, result.Err
		}
		return networkQoSRuleRawFromBody(result.Body, networkQoSRuleBandwidthLimitKey)
	case networkQoSRuleDSCPMarking:
		result := qosrules.CreateDSCPMarkingRule(ctx, client, policyID, opts)
		if result.Err != nil {
			return nil, result.Err
		}
		return networkQoSRuleRawFromBody(result.Body, networkQoSRuleDSCPMarkingKey)
	case networkQoSRuleMinimumBandwidth:
		result := qosrules.CreateMinimumBandwidthRule(ctx, client, policyID, opts)
		if result.Err != nil {
			return nil, result.Err
		}
		return networkQoSRuleRawFromBody(result.Body, networkQoSRuleMinimumBandwidthKey)
	case networkQoSRuleMinimumPacketRate:
		return networkQoSRuleRESTMutate(ctx, client, http.MethodPost, policyID, "", ruleType, values, 201)
	default:
		return nil, fmt.Errorf("unsupported QoS rule type %q", ruleType)
	}
}

func networkQoSRuleGetRaw(ctx context.Context, client *gophercloud.ServiceClient, policyID string, ruleID string, ruleType string) (map[string]any, error) {
	resourceKey, _, ok := networkQoSRuleResourceKeys(ruleType)
	if !ok {
		return nil, fmt.Errorf("unsupported QoS rule type %q", ruleType)
	}
	var body map[string]any
	resp, err := client.Get(ctx, networkQoSRuleURL(client, policyID, ruleID, ruleType), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return networkQoSRuleRawFromBody(body, resourceKey)
}

func networkQoSRuleUpdateRaw(ctx context.Context, client *gophercloud.ServiceClient, policyID string, ruleID string, ruleType string, values map[string]any) (map[string]any, error) {
	opts := networkQoSRuleOpts{Values: values}
	switch ruleType {
	case networkQoSRuleBandwidthLimit:
		result := qosrules.UpdateBandwidthLimitRule(ctx, client, policyID, ruleID, opts)
		if result.Err != nil {
			return nil, result.Err
		}
		return networkQoSRuleRawFromBody(result.Body, networkQoSRuleBandwidthLimitKey)
	case networkQoSRuleDSCPMarking:
		result := qosrules.UpdateDSCPMarkingRule(ctx, client, policyID, ruleID, opts)
		if result.Err != nil {
			return nil, result.Err
		}
		return networkQoSRuleRawFromBody(result.Body, networkQoSRuleDSCPMarkingKey)
	case networkQoSRuleMinimumBandwidth:
		result := qosrules.UpdateMinimumBandwidthRule(ctx, client, policyID, ruleID, opts)
		if result.Err != nil {
			return nil, result.Err
		}
		return networkQoSRuleRawFromBody(result.Body, networkQoSRuleMinimumBandwidthKey)
	case networkQoSRuleMinimumPacketRate:
		return networkQoSRuleRESTMutate(ctx, client, http.MethodPut, policyID, ruleID, ruleType, values, 200)
	default:
		return nil, fmt.Errorf("unsupported QoS rule type %q", ruleType)
	}
}

func networkQoSRuleDeleteRaw(ctx context.Context, client *gophercloud.ServiceClient, policyID string, ruleID string, ruleType string) error {
	switch ruleType {
	case networkQoSRuleBandwidthLimit:
		return qosrules.DeleteBandwidthLimitRule(ctx, client, policyID, ruleID).ExtractErr()
	case networkQoSRuleDSCPMarking:
		return qosrules.DeleteDSCPMarkingRule(ctx, client, policyID, ruleID).ExtractErr()
	case networkQoSRuleMinimumBandwidth:
		return qosrules.DeleteMinimumBandwidthRule(ctx, client, policyID, ruleID).ExtractErr()
	case networkQoSRuleMinimumPacketRate:
		resp, err := client.Delete(ctx, networkQoSRuleURL(client, policyID, ruleID, ruleType), nil)
		_, _, err = gophercloud.ParseResponse(resp, err)
		return err
	default:
		return fmt.Errorf("unsupported QoS rule type %q", ruleType)
	}
}

func networkQoSRuleRESTMutate(ctx context.Context, client *gophercloud.ServiceClient, method string, policyID string, ruleID string, ruleType string, values map[string]any, okCode int) (map[string]any, error) {
	resourceKey, _, ok := networkQoSRuleResourceKeys(ruleType)
	if !ok {
		return nil, fmt.Errorf("unsupported QoS rule type %q", ruleType)
	}
	requestBody := map[string]any{resourceKey: values}
	var responseBody map[string]any
	requestURL := networkQoSRuleURL(client, policyID, ruleID, ruleType)
	requestOpts := &gophercloud.RequestOpts{OkCodes: []int{okCode}}
	var resp *http.Response
	var err error
	switch method {
	case http.MethodPost:
		resp, err = client.Post(ctx, requestURL, requestBody, &responseBody, requestOpts)
	case http.MethodPut:
		resp, err = client.Put(ctx, requestURL, requestBody, &responseBody, requestOpts)
	default:
		return nil, fmt.Errorf("unsupported QoS rule method %q", method)
	}
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return networkQoSRuleRawFromBody(responseBody, resourceKey)
}

func networkQoSRuleRawFromBody(body any, resourceKey string) (map[string]any, error) {
	var wrapper map[string]map[string]any
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, err
	}
	raw := wrapper[resourceKey]
	if raw == nil {
		return nil, fmt.Errorf("missing %s response body", resourceKey)
	}
	return raw, nil
}

func networkQoSRuleRawFields(raw map[string]any, ruleType string) []outputField {
	hidden := map[string]bool{
		"location":      true,
		"name":          true,
		"qos_policy_id": true,
		"tenant_id":     true,
	}
	namesByKey := map[string]bool{}
	for _, name := range networkQoSRuleBaseFieldNames(ruleType) {
		namesByKey[name] = true
	}
	for name := range raw {
		if !hidden[name] {
			namesByKey[name] = true
		}
	}
	names := make([]string, 0, len(namesByKey))
	for name := range namesByKey {
		names = append(names, name)
	}
	sort.Strings(names)
	fields := make([]outputField, 0, len(names))
	for _, name := range names {
		fields = append(fields, outputField{name, networkQoSRuleRawValue(name, raw[name])})
	}
	return fields
}

func networkQoSRuleBaseFieldNames(ruleType string) []string {
	switch ruleType {
	case networkQoSRuleBandwidthLimit:
		return []string{"direction", "id", "max_burst_kbps", "max_kbps"}
	case networkQoSRuleDSCPMarking:
		return []string{"dscp_mark", "id"}
	case networkQoSRuleMinimumBandwidth:
		return []string{"direction", "id", "min_kbps"}
	case networkQoSRuleMinimumPacketRate:
		return []string{"direction", "id", "min_kpps"}
	default:
		return []string{"id"}
	}
}

func networkQoSRuleRawValue(name string, value any) any {
	switch name {
	case "dscp_mark", "max_burst_kbps", "max_kbps", "min_kbps", "min_kpps":
		return rawNumber(value)
	default:
		return value
	}
}

func networkQoSRuleResourceKeys(ruleType string) (string, string, bool) {
	switch ruleType {
	case networkQoSRuleBandwidthLimit:
		return networkQoSRuleBandwidthLimitKey, "bandwidth_limit_rules", true
	case networkQoSRuleDSCPMarking:
		return networkQoSRuleDSCPMarkingKey, "dscp_marking_rules", true
	case networkQoSRuleMinimumBandwidth:
		return networkQoSRuleMinimumBandwidthKey, "minimum_bandwidth_rules", true
	case networkQoSRuleMinimumPacketRate:
		return networkQoSRuleMinimumPacketKey, "minimum_packet_rate_rules", true
	default:
		return "", "", false
	}
}

func networkQoSRuleURL(client *gophercloud.ServiceClient, policyID string, ruleID string, ruleType string) string {
	_, resourcesKey, _ := networkQoSRuleResourceKeys(ruleType)
	if ruleID == "" {
		return client.ServiceURL("qos/policies", policyID, resourcesKey)
	}
	return client.ServiceURL("qos/policies", policyID, resourcesKey, ruleID)
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

type routerCreateOpts struct {
	Values map[string]any
}

func (opts routerCreateOpts) ToRouterCreateMap() (map[string]any, error) {
	return map[string]any{"router": opts.Values}, nil
}

type routerUpdateOpts struct {
	Values map[string]any
}

func (opts routerUpdateOpts) ToRouterUpdateMap() (map[string]any, error) {
	return map[string]any{"router": opts.Values}, nil
}

func routerCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("router create requires <name>")
	}
	if boolFlag(opts, "no-tag") && len(flagValues(opts, "tag")) > 0 {
		return fmt.Errorf("argument --no-tag: not allowed with argument --tag")
	}
	values, err := routerCreateValues(ctx, opts, networkClient, identityClient, args[0])
	if err != nil {
		return err
	}
	result := routers.Create(ctx, networkClient, routerCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	tags := flagValues(opts, "tag")
	if len(tags) > 0 {
		target, err := setNeutronResourceTags(ctx, networkClient, "routers", item.ID, item.Tags, tags, boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
		item.Tags = target
	}
	if raw, ok := routerRawFromBody(result.Body); ok {
		if len(tags) > 0 {
			raw["tags"] = item.Tags
		}
		return renderShowOutput(stdout, opts, routerRawFields(raw, nil))
	}
	return renderShowOutput(stdout, opts, routerFields(item, nil))
}

func routerCreateValues(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, name string) (map[string]any, error) {
	values := map[string]any{"name": name, "admin_state_up": true}
	if flagChanged(opts, "enable") && flagChanged(opts, "disable") {
		return nil, fmt.Errorf("argument --disable: not allowed with argument --enable")
	}
	if boolFlag(opts, "disable") {
		values["admin_state_up"] = false
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return nil, err
		}
		values["project_id"] = project.ID
	}
	if err := routerApplyMutableValues(ctx, opts, networkClient, values, true, nil); err != nil {
		return nil, err
	}
	if flavor := flagValue(opts, "flavor"); flavor != "" {
		values["flavor_id"] = flavor
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

func routerDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("router delete requires <router> [<router> ...]")
	}
	failures := 0
	for _, routerArg := range args {
		item, err := findRouter(ctx, client, routerArg)
		if err != nil {
			failures++
			continue
		}
		if err := routers.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d routers failed to delete.", failures, len(args))
	}
	return nil
}

func routerAddPort(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("router add port requires <router> <port>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	port, err := findPort(ctx, client, args[1])
	if err != nil {
		return err
	}
	_, err = routers.AddInterface(ctx, client, router.ID, routers.AddInterfaceOpts{PortID: port.ID}).Extract()
	return err
}

func routerAddGateway(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("router add gateway requires <router> <network>")
	}
	if err := requireExternalGatewayMultihoming(ctx, client); err != nil {
		return err
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	network, err := findNetwork(ctx, client, args[1])
	if err != nil {
		return err
	}
	gateway := map[string]any{"network_id": network.ID}
	fixedIPs, err := routerFixedIPs(ctx, client, flagValues(opts, "fixed-ip"))
	if err != nil {
		return err
	}
	if len(fixedIPs) > 0 {
		gateway["external_fixed_ips"] = fixedIPs
	}
	raw, err := neutronRouterAction(ctx, client, router.ID, "add_external_gateways", map[string]any{
		"external_gateways": []map[string]any{gateway},
	})
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, routerRawFields(raw, nil))
}

func routerAddRoute(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("router add route requires <router>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	routes, err := parseRouterRoutes(flagValues(opts, "route"))
	if err != nil {
		return err
	}
	raw, err := neutronRouterAction(ctx, client, router.ID, "add_extraroutes", map[string]any{"routes": routes})
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, routerRawFields(raw, nil))
}

func routerAddSubnet(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("router add subnet requires <router> <subnet>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	subnet, err := findSubnet(ctx, client, args[1])
	if err != nil {
		return err
	}
	_, err = routers.AddInterface(ctx, client, router.ID, routers.AddInterfaceOpts{SubnetID: subnet.ID}).Extract()
	return err
}

func routerRemovePort(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("router remove port requires <router> <port>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	port, err := findPort(ctx, client, args[1])
	if err != nil {
		return err
	}
	_, err = routers.RemoveInterface(ctx, client, router.ID, routers.RemoveInterfaceOpts{PortID: port.ID}).Extract()
	return err
}

func routerRemoveGateway(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("router remove gateway requires <router> <network>")
	}
	if err := requireExternalGatewayMultihoming(ctx, client); err != nil {
		return err
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	network, err := findNetwork(ctx, client, args[1])
	if err != nil {
		return err
	}
	gateway := map[string]any{"network_id": network.ID}
	fixedIPs, err := routerFixedIPs(ctx, client, flagValues(opts, "fixed-ip"))
	if err != nil {
		return err
	}
	if len(fixedIPs) > 0 {
		gateway["external_fixed_ips"] = fixedIPs
	}
	raw, err := neutronRouterAction(ctx, client, router.ID, "remove_external_gateways", map[string]any{
		"external_gateways": []map[string]any{gateway},
	})
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, routerRawFields(raw, nil))
}

func routerRemoveRoute(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("router remove route requires <router>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	routes, err := parseRouterRoutes(flagValues(opts, "route"))
	if err != nil {
		return err
	}
	raw, err := neutronRouterAction(ctx, client, router.ID, "remove_extraroutes", map[string]any{"routes": routes})
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, routerRawFields(raw, nil))
}

func routerRemoveSubnet(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("router remove subnet requires <router> <subnet>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	subnet, err := findSubnet(ctx, client, args[1])
	if err != nil {
		return err
	}
	_, err = routers.RemoveInterface(ctx, client, router.ID, routers.RemoveInterfaceOpts{SubnetID: subnet.ID}).Extract()
	return err
}

func requireExternalGatewayMultihoming(ctx context.Context, client *gophercloud.ServiceClient) error {
	if _, err := findNetworkExtension(ctx, client, "external-gateway-multihoming"); err == nil {
		return nil
	}
	return fmt.Errorf("The external-gateway-multihoming extension is not enabled at the Neutron side.")
}

func neutronRouterAction(ctx context.Context, client *gophercloud.ServiceClient, routerID string, action string, values map[string]any) (map[string]any, error) {
	var body struct {
		Router map[string]any `json:"router"`
	}
	resp, err := client.Put(ctx, client.ServiceURL("routers", url.PathEscape(routerID), action), map[string]any{"router": values}, &body, &gophercloud.RequestOpts{
		OkCodes: []int{200},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Router, nil
}

func routerSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("router set requires <router>")
	}
	item, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if err := routerApplyMutableValues(ctx, opts, client, values, false, item); err != nil {
		return err
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	for key, value := range extra {
		values[key] = value
	}
	if len(values) > 0 {
		if _, err := routers.Update(ctx, client, item.ID, routerUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "no-tag") || len(flagValues(opts, "tag")) > 0 {
		_, err := setNeutronResourceTags(ctx, client, "routers", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
	}
	return nil
}

func routerUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("router unset requires <router>")
	}
	if len(flagValues(opts, "tag")) > 0 && boolFlag(opts, "all-tag") {
		return fmt.Errorf("argument --all-tag: not allowed with argument --tag")
	}
	item, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	values := map[string]any{}
	if boolFlag(opts, "external-gateway") {
		values["external_gateway_info"] = nil
	}
	if boolFlag(opts, "qos-policy") {
		values["external_gateway_info"] = map[string]any{"qos_policy_id": nil}
	}
	routes, err := parseRouterRoutes(flagValues(opts, "route"))
	if err != nil {
		return err
	}
	if len(routes) > 0 {
		remaining, err := removeMapValues(routerRouteMaps(item.Routes), routes, "route")
		if err != nil {
			return err
		}
		values["routes"] = remaining
	}
	extra, err := parseUnsetExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	for key, value := range extra {
		values[key] = value
	}
	if len(values) > 0 {
		if _, err := routers.Update(ctx, client, item.ID, routerUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	_, err = unsetNeutronResourceTags(ctx, client, "routers", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "all-tag"))
	return err
}

func routerApplyMutableValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, values map[string]any, create bool, existing *routers.Router) error {
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if adminState, err := networkBoolFlag(opts, "enable", "disable"); err != nil {
		return err
	} else if adminState != nil {
		values["admin_state_up"] = *adminState
	}
	if distributed, err := networkBoolFlag(opts, "distributed", "centralized"); err != nil {
		return err
	} else if distributed != nil {
		values["distributed"] = *distributed
	}
	if ha, err := networkBoolFlag(opts, "ha", "no-ha"); err != nil {
		return err
	} else if ha != nil {
		values["ha"] = *ha
	}
	if hints := flagValues(opts, "availability-zone-hint"); len(hints) > 0 {
		values["availability_zone_hints"] = hints
	}
	if err := routerApplyGatewayValues(ctx, opts, client, values); err != nil {
		return err
	}
	if ndp, err := networkBoolFlag(opts, "enable-ndp-proxy", "disable-ndp-proxy"); err != nil {
		return err
	} else if ndp != nil {
		values["enable_ndp_proxy"] = *ndp
	}
	if bfd, err := networkBoolFlag(opts, "enable-default-route-bfd", "disable-default-route-bfd"); err != nil {
		return err
	} else if bfd != nil {
		values["enable_default_route_bfd"] = *bfd
	}
	if ecmp, err := networkBoolFlag(opts, "enable-default-route-ecmp", "disable-default-route-ecmp"); err != nil {
		return err
	} else if ecmp != nil {
		values["enable_default_route_ecmp"] = *ecmp
	}
	routes, err := parseRouterRoutes(flagValues(opts, "route"))
	if err != nil {
		return err
	}
	if len(routes) > 0 {
		merged := append([]map[string]string{}, routes...)
		if !create && !boolFlag(opts, "no-route") && existing != nil {
			merged = append(merged, routerRouteMaps(existing.Routes)...)
		}
		values["routes"] = merged
	} else if !create && boolFlag(opts, "no-route") {
		values["routes"] = []map[string]string{}
	}
	return nil
}

func routerApplyGatewayValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, values map[string]any) error {
	gateways := flagValues(opts, "external-gateway")
	if len(gateways) == 0 {
		if boolFlag(opts, "enable-snat") || boolFlag(opts, "disable-snat") || len(flagValues(opts, "fixed-ip")) > 0 || flagValue(opts, "qos-policy") != "" || boolFlag(opts, "no-qos-policy") {
			values["external_gateway_info"] = map[string]any{}
		}
	} else {
		gatewayInfo := map[string]any{"network_id": resolveNetworkID(ctx, client, gateways[0])}
		values["external_gateway_info"] = gatewayInfo
	}
	gatewayInfo, _ := values["external_gateway_info"].(map[string]any)
	if gatewayInfo == nil {
		return nil
	}
	if snat, err := networkBoolFlag(opts, "enable-snat", "disable-snat"); err != nil {
		return err
	} else if snat != nil {
		gatewayInfo["enable_snat"] = *snat
	}
	fixedIPs, err := routerFixedIPs(ctx, client, flagValues(opts, "fixed-ip"))
	if err != nil {
		return err
	}
	if len(fixedIPs) > 0 {
		gatewayInfo["external_fixed_ips"] = fixedIPs
	}
	if qosPolicy := flagValue(opts, "qos-policy"); qosPolicy != "" {
		policy, err := findNetworkQoSPolicy(ctx, client, qosPolicy)
		if err != nil {
			return err
		}
		gatewayInfo["qos_policy_id"] = policy.ID
	}
	if boolFlag(opts, "no-qos-policy") {
		if flagChanged(opts, "qos-policy") {
			return fmt.Errorf("argument --no-qos-policy: not allowed with argument --qos-policy")
		}
		gatewayInfo["qos_policy_id"] = nil
	}
	return nil
}

func routerFixedIPs(ctx context.Context, client *gophercloud.ServiceClient, values []string) ([]map[string]string, error) {
	entries, err := parseKeyValueEntries(values, "fixed-ip")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if subnetNameOrID := entry["subnet"]; subnetNameOrID != "" {
			subnet, err := findSubnet(ctx, client, subnetNameOrID)
			if err != nil {
				return nil, err
			}
			entry["subnet_id"] = subnet.ID
			delete(entry, "subnet")
		}
		if ipAddress := entry["ip-address"]; ipAddress != "" {
			entry["ip_address"] = ipAddress
			delete(entry, "ip-address")
		}
	}
	return entries, nil
}

func parseRouterRoutes(values []string) ([]map[string]string, error) {
	entries, err := parseKeyValueEntries(values, "route", "destination", "gateway")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		entry["nexthop"] = entry["gateway"]
		delete(entry, "gateway")
	}
	return entries, nil
}

func routerRouteMaps(values []routers.Route) []map[string]string {
	items := make([]map[string]string, 0, len(values))
	for _, value := range values {
		items = append(items, map[string]string{"destination": value.DestinationCIDR, "nexthop": value.NextHop})
	}
	return items
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
			"State":       adminStateValue(item.AdminStateUp),
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
	interfaces := routerInterfacesInfo(ctx, client, item.ID)
	if raw, err := neutronRouterRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, prettyNetworkIPOutputFields(opts, routerRawFields(raw, interfaces)))
	}
	return renderShowOutput(stdout, opts, prettyNetworkIPOutputFields(opts, routerFields(item, interfaces)))
}

func neutronRouterRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		Router map[string]any `json:"router"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("routers", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Router, nil
}

func routerRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		Router map[string]any `json:"router"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.Router == nil {
		return nil, false
	}
	return wrapper.Router, true
}

func routerRawFields(raw map[string]any, interfaces []map[string]string) []outputField {
	fields := []outputField{
		{"admin_state_up", adminStateValue(raw["admin_state_up"])},
		{"availability_zone_hints", raw["availability_zone_hints"]},
		{"availability_zones", raw["availability_zones"]},
		{"created_at", raw["created_at"]},
		{"description", raw["description"]},
		{"distributed", raw["distributed"]},
	}
	if _, ok := raw["enable_default_route_bfd"]; ok {
		fields = append(fields, outputField{"enable_default_route_bfd", raw["enable_default_route_bfd"]})
	}
	if _, ok := raw["enable_default_route_ecmp"]; ok {
		fields = append(fields, outputField{"enable_default_route_ecmp", raw["enable_default_route_ecmp"]})
	}
	fields = append(fields,
		outputField{"enable_ndp_proxy", raw["enable_ndp_proxy"]},
		outputField{"external_gateway_info", routerGatewayTableValue(raw["external_gateway_info"])},
		outputField{"flavor_id", raw["flavor_id"]},
	)
	if _, ok := raw["gw_port_id"]; ok {
		fields = append(fields, outputField{"gw_port_id", raw["gw_port_id"]})
	}
	fields = append(fields, outputField{"ha", raw["ha"]})
	if _, ok := raw["ha_vr_id"]; ok {
		fields = append(fields, outputField{"ha_vr_id", rawNumber(raw["ha_vr_id"])})
	}
	fields = append(fields, outputField{"id", raw["id"]})
	if interfaces != nil {
		fields = append(fields, outputField{"interfaces_info", routerInterfacesTableValue(interfaces)})
	}
	fields = append(fields,
		outputField{"name", raw["name"]},
		outputField{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		outputField{"revision_number", rawNumber(raw["revision_number"])},
		outputField{"routes", blankEmptyListValue(raw["routes"])},
		outputField{"status", raw["status"]},
		outputField{"tags", blankEmptyListValue(raw["tags"])},
		outputField{"updated_at", raw["updated_at"]},
	)
	return fields
}

func routerFields(item *routers.Router, interfaces []map[string]string) []outputField {
	fields := []outputField{
		{"admin_state_up", item.AdminStateUp},
		{"availability_zone_hints", item.AvailabilityZoneHints},
		{"availability_zones", nil},
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"distributed", item.Distributed},
		{"enable_ndp_proxy", nil},
		{"external_gateway_info", item.GatewayInfo},
		{"flavor_id", nil},
		{"ha", nil},
		{"id", item.ID},
	}
	if interfaces != nil {
		fields = append(fields, outputField{"interfaces_info", interfaces})
	}
	fields = append(fields,
		outputField{"name", item.Name},
		outputField{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		outputField{"revision_number", item.RevisionNumber},
		outputField{"routes", routerRouteMaps(item.Routes)},
		outputField{"status", item.Status},
		outputField{"tags", item.Tags},
		outputField{"updated_at", oscTime(item.UpdatedAt)},
	)
	return fields
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
	rows := []map[string]string{}
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
		{"rules", securityGroupRulesValue(firstPresent(raw, "rules", "security_group_rules"))},
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

func securityGroupRulesValue(value any) tableValue {
	items := anySlice(value)
	jsonKeys := []string{"id", "project_id", "tenant_id", "security_group_id", "ethertype", "direction", "protocol", "port_range_min", "port_range_max", "remote_ip_prefix", "remote_address_group_id", "normalized_cidr", "remote_group_id", "standard_attr_id", "belongs_to_default_sg", "description", "tags", "created_at", "updated_at", "revision_number"}
	jsonValues := make([]any, 0, len(items))
	lines := make([]string, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		orderedValues := map[string]any{}
		presentKeys := make([]string, 0, len(jsonKeys))
		for _, key := range jsonKeys {
			if itemValue, ok := itemMap[key]; ok {
				orderedValues[key] = itemValue
				presentKeys = append(presentKeys, key)
			}
		}
		jsonValues = append(jsonValues, orderedJSONObject{keys: presentKeys, values: orderedValues})
		keys := []string{"belongs_to_default_sg", "created_at", "direction", "ethertype", "id", "normalized_cidr", "port_range_min", "port_range_max", "protocol", "remote_address_group_id", "remote_group_id", "remote_ip_prefix", "standard_attr_id", "updated_at"}
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			value, ok := itemMap[key]
			if !ok || value == nil || valueString(value) == "" || valueString(value) == "None" {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s='%s'", key, strings.ReplaceAll(valueString(value), "'", "\\'")))
		}
		lines = append(lines, strings.Join(parts, ", "))
	}
	return tableValue{Value: jsonValues, Table: strings.Join(lines, "\n"), Pretty: value}
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

func volumeList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, computeClient *gophercloud.ServiceClient) error {
	page, err := volumes.List(client, volumes.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := volumes.ExtractVolumes(page)
	if err != nil {
		return err
	}
	serverNames := serverNameMap(ctx, computeClient)
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, volumeListRow(item, serverNames))
	}
	return renderListOutput(stdout, opts, []string{"ID", "Name", "Status", "Size", "Attached to"}, rows)
}

func volumeListRow(item volumes.Volume, serverNames map[string]string) outputRow {
	return outputRow{
		"ID":          item.ID,
		"Name":        item.Name,
		"Status":      item.Status,
		"Size":        item.Size,
		"Attached to": volumeAttachmentValue(item.Attachments, serverNames),
	}
}

func volumeShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume show requires <volume>")
	}
	if client.Microversion == "" {
		if withMinimum, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.65"); err == nil {
			client = withMinimum
		}
	}
	item, err := findVolume(ctx, client, args[0])
	if err != nil {
		return err
	}
	if raw, err := cinderVolumeRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, volumeRawFields(raw))
	}
	return renderShowOutput(stdout, opts, volumeFields(item))
}

func cinderVolumeRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		Volume map[string]any `json:"volume"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("volumes", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Volume, nil
}

func volumeRawFields(raw map[string]any) []outputField {
	attachments := volumeAttachmentValueFromAny(raw["attachments"], nil)
	properties := mapValueOrEmptyAny(raw["metadata"])
	return []outputField{
		{"attachments", attachments},
		{"availability_zone", raw["availability_zone"]},
		{"backup_id", raw["backup_id"]},
		{"bootable", cinderBool(raw["bootable"])},
		{"cluster_name", raw["cluster_name"]},
		{"consumes_quota", raw["consumes_quota"]},
		{"created_at", raw["created_at"]},
		{"description", raw["description"]},
		{"encrypted", raw["encrypted"]},
		{"group_id", firstPresent(raw, "group_id", "consistencygroup_id")},
		{"id", raw["id"]},
		{"multiattach", raw["multiattach"]},
		{"name", prettyVolumeValue(fmt.Sprint(raw["name"]))},
		{"os-vol-host-attr:host", raw["os-vol-host-attr:host"]},
		{"os-vol-mig-status-attr:migstat", raw["os-vol-mig-status-attr:migstat"]},
		{"os-vol-mig-status-attr:name_id", raw["os-vol-mig-status-attr:name_id"]},
		{"os-vol-tenant-attr:tenant_id", raw["os-vol-tenant-attr:tenant_id"]},
		{"properties", mapTableValue(properties, "")},
		{"provider_id", raw["provider_id"]},
		{"replication_status", raw["replication_status"]},
		{"service_uuid", raw["service_uuid"]},
		{"shared_targets", raw["shared_targets"]},
		{"size", rawNumber(raw["size"])},
		{"snapshot_id", raw["snapshot_id"]},
		{"source_volid", raw["source_volid"]},
		{"status", raw["status"]},
		{"type", raw["volume_type"]},
		{"updated_at", raw["updated_at"]},
		{"user_id", raw["user_id"]},
		{"volume_image_metadata", volumeImageMetadataValue(raw["volume_image_metadata"])},
		{"volume_type_id", raw["volume_type_id"]},
	}
}

func volumeFields(item *volumes.Volume) []outputField {
	properties := map[string]any{}
	for key, value := range item.Metadata {
		properties[key] = value
	}
	return []outputField{
		{"attachments", volumeAttachmentValue(item.Attachments, nil)},
		{"availability_zone", item.AvailabilityZone},
		{"backup_id", stringPtrValue(item.BackupID)},
		{"bootable", cinderBool(item.Bootable)},
		{"cluster_name", nil},
		{"consumes_quota", nil},
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"encrypted", item.Encrypted},
		{"group_id", nilIfEmpty(item.ConsistencyGroupID)},
		{"id", item.ID},
		{"multiattach", item.Multiattach},
		{"name", prettyVolumeValue(item.Name)},
		{"os-vol-host-attr:host", nilIfEmpty(item.Host)},
		{"os-vol-mig-status-attr:migstat", nil},
		{"os-vol-mig-status-attr:name_id", nil},
		{"os-vol-tenant-attr:tenant_id", nilIfEmpty(item.TenantID)},
		{"properties", mapTableValue(properties, "")},
		{"provider_id", nil},
		{"replication_status", nilIfEmpty(item.ReplicationStatus)},
		{"service_uuid", nil},
		{"shared_targets", nil},
		{"size", item.Size},
		{"snapshot_id", nilIfEmpty(item.SnapshotID)},
		{"source_volid", nilIfEmpty(item.SourceVolID)},
		{"status", item.Status},
		{"type", item.VolumeType},
		{"updated_at", oscTime(item.UpdatedAt)},
		{"user_id", nilIfEmpty(item.UserID)},
		{"volume_image_metadata", volumeImageMetadataValue(item.VolumeImageMetadata)},
		{"volume_type_id", nil},
	}
}

func volumeCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, imageClient *gophercloud.ServiceClient, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	metadata, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	if remoteValues := flagValues(opts, "remote-source"); len(remoteValues) > 0 {
		remoteSource, err := parseJSONKeyValueMap(remoteValues, "remote-source")
		if err != nil {
			return err
		}
		host := flagValue(opts, "host")
		cluster := flagValue(opts, "cluster")
		if host == "" && cluster == "" {
			return fmt.Errorf("volume create --remote-source requires --host or --cluster")
		}
		minimum := "3.8"
		if cluster != "" {
			minimum = "3.16"
		}
		client, err = blockStorageClientWithMinimumMicroversion(ctx, client, minimum)
		if err != nil {
			return err
		}
		body := map[string]any{
			"volume": map[string]any{
				"host":              host,
				"cluster":           nilIfEmpty(cluster),
				"ref":               remoteSource,
				"name":              nilIfEmpty(name),
				"description":       nilIfEmpty(flagValue(opts, "description")),
				"volume_type":       nilIfEmpty(flagValue(opts, "type")),
				"availability_zone": nilIfEmpty(flagValue(opts, "availability-zone")),
				"metadata":          metadata,
				"bootable":          boolFlag(opts, "bootable"),
			},
		}
		var response struct {
			Volume *volumes.Volume `json:"volume"`
		}
		resp, err := client.Post(ctx, client.ServiceURL("manageable_volumes"), body, &response, &gophercloud.RequestOpts{
			OkCodes: []int{http.StatusOK, http.StatusAccepted},
		})
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
		if response.Volume == nil {
			return fmt.Errorf("response did not contain volume object")
		}
		if boolFlag(opts, "read-only") {
			if err := volumeSetReadOnly(ctx, client, response.Volume.ID, true); err != nil {
				return err
			}
		} else if boolFlag(opts, "read-write") {
			if err := volumeSetReadOnly(ctx, client, response.Volume.ID, false); err != nil {
				return err
			}
		}
		managed, err := volumes.Get(ctx, client, response.Volume.ID).Extract()
		if err == nil {
			response.Volume = managed
		}
		return renderVolumeShow(stdout, opts, response.Volume)
	}
	createOpts := volumes.CreateOpts{
		Size:               intFlag(opts, "size"),
		AvailabilityZone:   flagValue(opts, "availability-zone"),
		ConsistencyGroupID: flagValue(opts, "consistency-group"),
		Description:        flagValue(opts, "description"),
		Metadata:           metadata,
		Name:               name,
		VolumeType:         flagValue(opts, "type"),
	}
	if snapshotValue := flagValue(opts, "snapshot"); snapshotValue != "" {
		snapshot, err := findVolumeSnapshot(ctx, client, snapshotValue)
		if err != nil {
			return err
		}
		createOpts.SnapshotID = snapshot.ID
	}
	if sourceValue := flagValue(opts, "source"); sourceValue != "" {
		source, err := findVolume(ctx, client, sourceValue)
		if err != nil {
			return err
		}
		createOpts.SourceVolID = source.ID
	}
	if backupValue := flagValue(opts, "backup"); backupValue != "" {
		backup, err := findVolumeBackup(ctx, client, backupValue)
		if err != nil {
			return err
		}
		createOpts.BackupID = backup.ID
		client, err = blockStorageClientWithMinimumMicroversion(ctx, client, "3.47")
		if err != nil {
			return err
		}
	}
	if imageValue := flagValue(opts, "image"); imageValue != "" {
		if imageClient == nil {
			return fmt.Errorf("image service is required for volume create --image")
		}
		image, err := findImage(ctx, imageClient, imageValue)
		if err != nil {
			return err
		}
		createOpts.ImageID = image.ID
	}
	if createOpts.Size == 0 && createOpts.SnapshotID == "" && createOpts.SourceVolID == "" && createOpts.BackupID == "" {
		return fmt.Errorf("volume create requires --size unless --snapshot, --source, or --backup is specified")
	}
	hints, err := volumeSchedulerHints(flagValues(opts, "hint"))
	if err != nil {
		return err
	}
	result := volumes.Create(ctx, client, createOpts, hints)
	created, err := result.Extract()
	if err != nil {
		return err
	}
	if boolFlag(opts, "bootable") {
		if err := volumes.SetBootable(ctx, client, created.ID, volumes.BootableOpts{Bootable: true}).ExtractErr(); err != nil {
			return err
		}
	} else if boolFlag(opts, "non-bootable") {
		if err := volumes.SetBootable(ctx, client, created.ID, volumes.BootableOpts{Bootable: false}).ExtractErr(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "read-only") {
		if err := volumeSetReadOnly(ctx, client, created.ID, true); err != nil {
			return err
		}
	} else if boolFlag(opts, "read-write") {
		if err := volumeSetReadOnly(ctx, client, created.ID, false); err != nil {
			return err
		}
	}
	created, err = volumes.Get(ctx, client, created.ID).Extract()
	if err != nil {
		return err
	}
	return renderVolumeShow(stdout, opts, created)
}

func volumeDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume delete requires <volume>")
	}
	failures := 0
	for _, value := range args {
		item, err := findVolume(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		switch {
		case boolFlag(opts, "remote"):
			err = volumes.Unmanage(ctx, client, item.ID).ExtractErr()
		case boolFlag(opts, "force"):
			err = volumes.ForceDelete(ctx, client, item.ID).ExtractErr()
		default:
			err = volumes.Delete(ctx, client, item.ID, volumes.DeleteOpts{Cascade: boolFlag(opts, "cascade") || boolFlag(opts, "purge")}).ExtractErr()
		}
		if err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d volumes failed to delete.", failures, len(args))
	}
	return nil
}

func volumeSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume set requires <volume>")
	}
	item, err := findVolume(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := volumes.UpdateOpts{}
	needsUpdate := false
	if flagChanged(opts, "name") {
		update.Name = valueStringPtr(flagValue(opts, "name"))
		needsUpdate = true
	}
	if flagChanged(opts, "description") {
		update.Description = valueStringPtr(flagValue(opts, "description"))
		needsUpdate = true
	}
	properties, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	if boolFlag(opts, "no-property") {
		update.Metadata = map[string]string{}
		needsUpdate = true
	}
	if len(properties) > 0 {
		metadata := map[string]string{}
		if !boolFlag(opts, "no-property") {
			for key, value := range item.Metadata {
				metadata[key] = value
			}
		}
		for key, value := range properties {
			metadata[key] = value
		}
		update.Metadata = metadata
		needsUpdate = true
	}
	if needsUpdate {
		if _, err := volumes.Update(ctx, client, item.ID, update).Extract(); err != nil {
			return err
		}
	}
	if newSize := intFlag(opts, "size"); newSize > 0 {
		if err := volumes.ExtendSize(ctx, client, item.ID, volumes.ExtendSizeOpts{NewSize: newSize}).ExtractErr(); err != nil {
			return err
		}
	}
	imageProperties, err := parseStringMap(flagValues(opts, "image-property"), "image-property")
	if err != nil {
		return err
	}
	if len(imageProperties) > 0 {
		if err := volumes.SetImageMetadata(ctx, client, item.ID, volumes.ImageMetadataOpts{Metadata: imageProperties}).ExtractErr(); err != nil {
			return err
		}
	}
	if state := flagValue(opts, "state"); state != "" || boolFlag(opts, "attached") || boolFlag(opts, "detached") {
		reset := volumes.ResetStatusOpts{Status: state}
		if boolFlag(opts, "attached") {
			reset.AttachStatus = "attached"
		} else if boolFlag(opts, "detached") {
			reset.AttachStatus = "detached"
		}
		if err := volumes.ResetStatus(ctx, client, item.ID, reset).ExtractErr(); err != nil {
			return err
		}
	}
	if typeValue := flagValue(opts, "type"); typeValue != "" {
		volumeType, err := findVolumeType(ctx, client, typeValue)
		if err != nil {
			return err
		}
		policy := volumes.MigrationPolicy(flagValue(opts, "migration-policy"))
		if policy == "" {
			policy = volumes.MigrationPolicyNever
		}
		if err := volumes.ChangeType(ctx, client, item.ID, volumes.ChangeTypeOpts{NewType: volumeType.ID, MigrationPolicy: policy}).ExtractErr(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "bootable") {
		if err := volumes.SetBootable(ctx, client, item.ID, volumes.BootableOpts{Bootable: true}).ExtractErr(); err != nil {
			return err
		}
	} else if boolFlag(opts, "non-bootable") {
		if err := volumes.SetBootable(ctx, client, item.ID, volumes.BootableOpts{Bootable: false}).ExtractErr(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "read-only") {
		return volumeSetReadOnly(ctx, client, item.ID, true)
	}
	if boolFlag(opts, "read-write") {
		return volumeSetReadOnly(ctx, client, item.ID, false)
	}
	return nil
}

func volumeUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume unset requires <volume>")
	}
	item, err := findVolume(ctx, client, args[0])
	if err != nil {
		return err
	}
	for _, key := range flagValues(opts, "property") {
		if err := blockStorageDeleteMetadataKey(ctx, client, "volumes", item.ID, key); err != nil {
			return err
		}
	}
	for _, key := range flagValues(opts, "image-property") {
		if err := volumeUnsetImageMetadata(ctx, client, item.ID, key); err != nil {
			return err
		}
	}
	return nil
}

func renderVolumeShow(stdout io.Writer, opts *Options, item *volumes.Volume) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"id", item.ID},
		{"name", prettyVolumeValue(item.Name)},
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

func volumeBackupCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume backup create requires <volume>")
	}
	volume, err := findVolume(ctx, client, args[0])
	if err != nil {
		return err
	}
	metadata, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	createOpts := backups.CreateOpts{
		VolumeID:         volume.ID,
		Force:            boolFlag(opts, "force"),
		Name:             flagValue(opts, "name"),
		Description:      flagValue(opts, "description"),
		Metadata:         metadata,
		Container:        flagValue(opts, "container"),
		Incremental:      boolFlag(opts, "incremental"),
		AvailabilityZone: flagValue(opts, "availability-zone"),
	}
	if snapshotValue := flagValue(opts, "snapshot"); snapshotValue != "" {
		snapshot, err := findVolumeSnapshot(ctx, client, snapshotValue)
		if err != nil {
			return err
		}
		createOpts.SnapshotID = snapshot.ID
	}
	created, err := backups.Create(ctx, client, createOpts).Extract()
	if err != nil {
		return err
	}
	return volumeBackupShow(ctx, stdout, opts, client, []string{created.ID})
}

func volumeBackupDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume backup delete requires <backup>")
	}
	failures := 0
	for _, value := range args {
		item, err := findVolumeBackup(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if boolFlag(opts, "force") {
			err = backups.ForceDelete(ctx, client, item.ID).ExtractErr()
		} else {
			err = backups.Delete(ctx, client, item.ID).ExtractErr()
		}
		if err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d volume backups failed to delete.", failures, len(args))
	}
	return nil
}

func volumeBackupRestore(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume backup restore requires <backup>")
	}
	backup, err := findVolumeBackup(ctx, client, args[0])
	if err != nil {
		return err
	}
	restoreOpts := backups.RestoreOpts{}
	if len(args) > 1 {
		if boolFlag(opts, "force") {
			volume, err := findVolume(ctx, client, args[1])
			if err != nil {
				return err
			}
			restoreOpts.VolumeID = volume.ID
		} else {
			if volume, err := findVolume(ctx, client, args[1]); err == nil {
				restoreOpts.VolumeID = volume.ID
			} else {
				restoreOpts.Name = args[1]
			}
		}
	}
	restored, err := backups.RestoreFromBackup(ctx, client, backup.ID, restoreOpts).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"backup_id", restored.BackupID},
		{"volume_id", restored.VolumeID},
		{"volume_name", restored.VolumeName},
	})
}

func volumeBackupSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume backup set requires <backup>")
	}
	item, err := findVolumeBackup(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := backups.UpdateOpts{}
	needsUpdate := false
	if flagChanged(opts, "name") {
		update.Name = valueStringPtr(flagValue(opts, "name"))
		needsUpdate = true
	}
	if flagChanged(opts, "description") {
		update.Description = valueStringPtr(flagValue(opts, "description"))
		needsUpdate = true
	}
	properties, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	if boolFlag(opts, "no-property") {
		update.Metadata = map[string]string{}
		needsUpdate = true
	}
	if len(properties) > 0 {
		metadata := map[string]string{}
		if !boolFlag(opts, "no-property") && item.Metadata != nil {
			for key, value := range *item.Metadata {
				metadata[key] = value
			}
		}
		for key, value := range properties {
			metadata[key] = value
		}
		update.Metadata = metadata
		needsUpdate = true
	}
	if needsUpdate {
		client, err = blockStorageClientWithMinimumMicroversion(ctx, client, "3.9")
		if err != nil {
			return err
		}
		if _, err := backups.Update(ctx, client, item.ID, update).Extract(); err != nil {
			return err
		}
	}
	if state := flagValue(opts, "state"); state != "" {
		if err := backups.ResetStatus(ctx, client, item.ID, backups.ResetStatusOpts{Status: state}).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func volumeBackupUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume backup unset requires <backup>")
	}
	item, err := findVolumeBackup(ctx, client, args[0])
	if err != nil {
		return err
	}
	client, err = blockStorageClientWithMinimumMicroversion(ctx, client, "3.43")
	if err != nil {
		return err
	}
	for _, key := range flagValues(opts, "property") {
		if err := blockStorageDeleteMetadataKey(ctx, client, "backups", item.ID, key); err != nil {
			return err
		}
	}
	return nil
}

func volumeBackupRecordExport(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume backup record export requires <backup>")
	}
	item, err := findVolumeBackup(ctx, client, args[0])
	if err != nil {
		return err
	}
	record, err := backups.Export(ctx, client, item.ID).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"backup_service", record.BackupService},
		{"backup_url", string(record.BackupURL)},
	})
}

func volumeBackupRecordImport(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("volume backup record import requires <backup_service> <backup_metadata>")
	}
	result, err := backups.Import(ctx, client, backups.ImportOpts{BackupService: args[0], BackupURL: []byte(args[1])}).Extract()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"id", result.ID},
		{"name", result.Name},
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

func volumeServiceSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("volume service set requires <host> <service>")
	}
	if flagValue(opts, "disable-reason") != "" && !boolFlag(opts, "disable") {
		return fmt.Errorf("Cannot specify --disable-reason without --disable")
	}
	action := "enable"
	body := map[string]any{
		"host":   args[0],
		"binary": args[1],
	}
	if boolFlag(opts, "disable") {
		action = "disable"
		if reason := flagValue(opts, "disable-reason"); reason != "" {
			action = "disable-log-reason"
			body["disabled_reason"] = reason
		}
	}
	return blockStoragePutAction(ctx, client, client.ServiceURL("os-services", action), body, http.StatusOK, http.StatusAccepted)
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
		{"Attached At", attachmentTimeValue(item.AttachedAt)},
		{"Detached At", attachmentTimeValue(item.DetachedAt)},
		{"Properties", volumeAttachmentPropertiesValue(item.ConnectionInfo)},
	})
}

type volumeAttachmentCreateOpts struct {
	VolumeUUID   string
	InstanceUUID string
	Connector    map[string]any
	Mode         any
	HasMode      bool
}

func (opts volumeAttachmentCreateOpts) ToAttachmentCreateMap() (map[string]any, error) {
	attachment := map[string]any{
		"volume_uuid":   opts.VolumeUUID,
		"instance_uuid": opts.InstanceUUID,
	}
	if opts.Connector != nil {
		attachment["connector"] = opts.Connector
	}
	if opts.HasMode {
		attachment["mode"] = opts.Mode
	}
	return map[string]any{"attachment": attachment}, nil
}

func volumeAttachmentCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, computeClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("volume attachment create requires <volume> <server>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.27")
	if err != nil {
		return err
	}
	if flagValue(opts, "mode") != "" {
		client, err = blockStorageClientWithMinimumMicroversion(ctx, client, "3.54")
		if err != nil {
			return err
		}
	}
	if computeClient == nil {
		return fmt.Errorf("compute service is required for volume attachment create")
	}
	volume, err := findVolume(ctx, client, args[0])
	if err != nil {
		return err
	}
	server, err := findServer(ctx, computeClient, args[1])
	if err != nil {
		return err
	}
	connector := map[string]any(nil)
	if boolFlag(opts, "connect") {
		connector = volumeAttachmentConnector(opts)
	} else if volumeAttachmentConnectorFlagsSet(opts) {
		return fmt.Errorf("You must specify the --connect option for any of the connection-specific options such as --initiator to be valid")
	}
	createOpts := volumeAttachmentCreateOpts{
		VolumeUUID:   volume.ID,
		InstanceUUID: server.ID,
		Connector:    connector,
	}
	if mode := flagValue(opts, "mode"); mode != "" {
		createOpts.HasMode = true
		if mode == "null" {
			createOpts.Mode = nil
		} else {
			createOpts.Mode = mode
		}
	}
	item, err := bsattachments.Create(ctx, client, createOpts).Extract()
	if err != nil {
		return err
	}
	return renderVolumeAttachmentShow(stdout, opts, item)
}

func volumeAttachmentDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume attachment delete requires <attachment>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.27")
	if err != nil {
		return err
	}
	return bsattachments.Delete(ctx, client, args[0]).ExtractErr()
}

func volumeAttachmentSet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume attachment set requires <attachment>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.27")
	if err != nil {
		return err
	}
	item, err := bsattachments.Update(ctx, client, args[0], bsattachments.UpdateOpts{Connector: volumeAttachmentConnector(opts)}).Extract()
	if err != nil {
		return err
	}
	return renderVolumeAttachmentShow(stdout, opts, item)
}

func volumeAttachmentComplete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume attachment complete requires <attachment>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.44")
	if err != nil {
		return err
	}
	return bsattachments.Complete(ctx, client, args[0]).ExtractErr()
}

func renderVolumeAttachmentShow(stdout io.Writer, opts *Options, item *bsattachments.Attachment) error {
	return renderShowOutput(stdout, opts, []outputField{
		{"ID", item.ID},
		{"Volume ID", item.VolumeID},
		{"Instance ID", item.Instance},
		{"Status", item.Status},
		{"Attach Mode", item.AttachMode},
		{"Attached At", attachmentTimeValue(item.AttachedAt)},
		{"Detached At", attachmentTimeValue(item.DetachedAt)},
		{"Properties", volumeAttachmentPropertiesValue(item.ConnectionInfo)},
	})
}

func volumeAttachmentConnector(opts *Options) map[string]any {
	return map[string]any{
		"initiator":  nilIfEmpty(flagValue(opts, "initiator")),
		"ip":         nilIfEmpty(flagValue(opts, "ip")),
		"platform":   nilIfEmpty(flagValue(opts, "platform")),
		"host":       nilIfEmpty(flagValue(opts, "host")),
		"os_type":    nilIfEmpty(flagValue(opts, "os-type")),
		"multipath":  boolFlag(opts, "multipath"),
		"mountpoint": nilIfEmpty(flagValue(opts, "mountpoint")),
	}
}

func volumeAttachmentConnectorFlagsSet(opts *Options) bool {
	for _, name := range []string{"initiator", "ip", "platform", "host", "os-type", "multipath", "mountpoint"} {
		if flagChanged(opts, name) {
			return true
		}
	}
	return false
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

func volumeQoSCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume qos create requires <name>")
	}
	properties, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	consumer := flagValue(opts, "consumer")
	if consumer == "" {
		consumer = "both"
	}
	item, err := bsqos.Create(ctx, client, bsqos.CreateOpts{
		Name:     args[0],
		Consumer: bsqos.QoSConsumer(consumer),
		Specs:    properties,
	}).Extract()
	if err != nil {
		return err
	}
	return renderVolumeQoSShow(ctx, stdout, opts, client, item)
}

func volumeQoSDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume qos delete requires <qos-spec>")
	}
	failures := 0
	for _, value := range args {
		item, err := findVolumeQoS(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if err := bsqos.Delete(ctx, client, item.ID, bsqos.DeleteOpts{Force: boolFlag(opts, "force")}).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d QoS specifications failed to delete.", failures, len(args))
	}
	return nil
}

func volumeQoSSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume qos set requires <qos-spec>")
	}
	item, err := findVolumeQoS(ctx, client, args[0])
	if err != nil {
		return err
	}
	failures := 0
	if boolFlag(opts, "no-property") && len(item.Specs) > 0 {
		keys := make([]string, 0, len(item.Specs))
		for key := range item.Specs {
			keys = append(keys, key)
		}
		if err := bsqos.DeleteKeys(ctx, client, item.ID, bsqos.DeleteKeysOpts(keys)).ExtractErr(); err != nil {
			failures++
		}
	}
	properties, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	if len(properties) > 0 {
		if _, err := bsqos.Update(ctx, client, item.ID, bsqos.UpdateOpts{Specs: properties}).Extract(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("One or more of the set operations failed")
	}
	return nil
}

func volumeQoSShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume qos show requires <qos-spec>")
	}
	item, err := findVolumeQoS(ctx, client, args[0])
	if err != nil {
		return err
	}
	return renderVolumeQoSShow(ctx, stdout, opts, client, item)
}

func volumeQoSUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume qos unset requires <qos-spec>")
	}
	item, err := findVolumeQoS(ctx, client, args[0])
	if err != nil {
		return err
	}
	values := flagValues(opts, "property")
	if len(values) == 0 {
		return nil
	}
	return bsqos.DeleteKeys(ctx, client, item.ID, bsqos.DeleteKeysOpts(values)).ExtractErr()
}

func volumeQoSAssociate(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("volume qos associate requires <qos-spec> <volume-type>")
	}
	qosSpec, err := findVolumeQoS(ctx, client, args[0])
	if err != nil {
		return err
	}
	volumeType, err := findVolumeType(ctx, client, args[1])
	if err != nil {
		return err
	}
	return bsqos.Associate(ctx, client, qosSpec.ID, bsqos.AssociateOpts{VolumeTypeID: volumeType.ID}).ExtractErr()
}

func volumeQoSDisassociate(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume qos disassociate requires <qos-spec>")
	}
	qosSpec, err := findVolumeQoS(ctx, client, args[0])
	if err != nil {
		return err
	}
	if boolFlag(opts, "all") {
		return bsqos.DisassociateAll(ctx, client, qosSpec.ID).ExtractErr()
	}
	volumeTypeValue := flagValue(opts, "volume-type")
	if volumeTypeValue == "" {
		return nil
	}
	volumeType, err := findVolumeType(ctx, client, volumeTypeValue)
	if err != nil {
		return err
	}
	return bsqos.Disassociate(ctx, client, qosSpec.ID, bsqos.DisassociateOpts{VolumeTypeID: volumeType.ID}).ExtractErr()
}

func renderVolumeQoSShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, item *bsqos.QoS) error {
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
		fields = append(fields, outputField{"Metadata", mapTableValue(metadata, "")})
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

func blockStorageClusterSet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("block storage cluster set requires <cluster>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.7", "block storage cluster set")
	if err != nil {
		return err
	}
	if flagValue(opts, "disable-reason") != "" && !boolFlag(opts, "disable") {
		return fmt.Errorf("Cannot specify --disable-reason without --disable")
	}
	binary := flagValue(opts, "binary")
	if binary == "" {
		binary = "cinder-volume"
	}
	action := "enable"
	body := map[string]any{
		"name":   args[0],
		"binary": binary,
	}
	if boolFlag(opts, "disable") {
		action = "disable"
		if reason := flagValue(opts, "disable-reason"); reason != "" {
			body["disabled_reason"] = reason
		}
	}
	var response struct {
		Cluster blockStorageClusterRecord `json:"cluster"`
	}
	resp, err := client.Put(ctx, client.ServiceURL("clusters", action), body, &response, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusOK, http.StatusAccepted},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
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

type blockStorageCleanupServiceRecord struct {
	ID          string `json:"id"`
	ClusterName string `json:"cluster_name"`
	Host        string `json:"host"`
	Binary      string `json:"binary"`
}

func blockStorageCleanup(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.24")
	if err != nil {
		return err
	}
	body := map[string]any{}
	if value := flagValue(opts, "cluster"); value != "" {
		body["cluster_name"] = value
	}
	if value := flagValue(opts, "host"); value != "" {
		body["host"] = value
	}
	if value := flagValue(opts, "binary"); value != "" {
		body["binary"] = value
	}
	if flagChanged(opts, "up") || flagChanged(opts, "down") {
		body["is_up"] = boolFlag(opts, "up")
	}
	if flagChanged(opts, "disabled") || flagChanged(opts, "enabled") {
		body["disabled"] = boolFlag(opts, "disabled")
	}
	if value := flagValue(opts, "resource-id"); value != "" {
		body["resource_id"] = value
	}
	if value := flagValue(opts, "resource-type"); value != "" {
		body["resource_type"] = value
	}
	if value := flagValue(opts, "service-id"); value != "" {
		body["service_id"] = value
	}
	var response struct {
		Cleaning    []blockStorageCleanupServiceRecord `json:"cleaning"`
		Unavailable []blockStorageCleanupServiceRecord `json:"unavailable"`
	}
	resp, err := client.Post(ctx, client.ServiceURL("workers", "cleanup"), body, &response, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusOK, http.StatusAccepted},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	rows := make([]outputRow, 0, len(response.Cleaning)+len(response.Unavailable))
	for _, item := range response.Cleaning {
		rows = append(rows, outputRow{
			"ID":           item.ID,
			"Cluster Name": nilIfEmpty(item.ClusterName),
			"Host":         nilIfEmpty(item.Host),
			"Binary":       nilIfEmpty(item.Binary),
			"Status":       "Cleaning",
		})
	}
	for _, item := range response.Unavailable {
		rows = append(rows, outputRow{
			"ID":           item.ID,
			"Cluster Name": nilIfEmpty(item.ClusterName),
			"Host":         nilIfEmpty(item.Host),
			"Binary":       nilIfEmpty(item.Binary),
			"Status":       "Unavailable",
		})
	}
	return renderListOutput(stdout, opts, []string{"ID", "Cluster Name", "Host", "Binary", "Status"}, rows)
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
			"Filters":  blockStorageResourceFiltersValue(item.Filters),
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
		{"Filters", blockStorageResourceFiltersValue(item.Filters)},
	})
}

func blockStorageResourceFiltersValue(filters []string) tableValue {
	value := append([]string(nil), filters...)
	tableFilters := append([]string(nil), filters...)
	sort.Strings(tableFilters)
	return tableValue{Value: value, Table: strings.Join(tableFilters, ", "), Pretty: tableFilters}
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

func blockStorageLogLevelSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("block storage log level set requires <log-level>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.32")
	if err != nil {
		return err
	}
	return blockStoragePutAction(ctx, client, client.ServiceURL("os-services", "set-log"), map[string]any{
		"level":  strings.ToUpper(args[0]),
		"binary": flagValue(opts, "service"),
		"server": nilIfEmpty(flagValue(opts, "host")),
		"prefix": nilIfEmpty(flagValue(opts, "log-prefix")),
	}, http.StatusOK, http.StatusAccepted)
}

type blockStorageManageableRecord struct {
	Reference       any `json:"reference"`
	Size            any `json:"size"`
	SafeToManage    any `json:"safe_to_manage"`
	ReasonNotSafe   any `json:"reason_not_safe"`
	CinderID        any `json:"cinder_id"`
	ExtraInfo       any `json:"extra_info"`
	SourceReference any `json:"source_reference"`
}

func blockStorageManageableList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string, resource string) error {
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.8")
	if err != nil {
		return err
	}
	host := ""
	if len(args) > 0 {
		host = args[0]
	}
	cluster := flagValue(opts, "cluster")
	if cluster != "" {
		client, err = blockStorageClientWithMinimumMicroversion(ctx, client, "3.17")
		if err != nil {
			return err
		}
	} else if host == "" {
		return fmt.Errorf("block storage %s manageable list requires <host> or --cluster", strings.TrimSuffix(resource, "s"))
	}
	path := "manageable_volumes"
	responseKey := "manageable-volumes"
	if resource == "snapshots" {
		path = "manageable_snapshots"
		responseKey = "manageable-snapshots"
	}
	if boolFlag(opts, "long") {
		path += "/detail"
	}
	query := url.Values{}
	if cluster != "" {
		query.Set("cluster", cluster)
	} else {
		query.Set("host", host)
	}
	if value := flagValue(opts, "marker"); value != "" {
		query.Set("marker", value)
	}
	if value := intFlag(opts, "limit"); value > 0 {
		query.Set("limit", strconv.Itoa(value))
	}
	if value := intFlag(opts, "offset"); value > 0 {
		query.Set("offset", strconv.Itoa(value))
	}
	if value := flagValue(opts, "sort"); value != "" {
		query.Set("sort", value)
	}
	requestURL := client.ServiceURL(strings.Split(path, "/")...)
	if encoded := query.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	var raw map[string][]blockStorageManageableRecord
	resp, err := client.Get(ctx, requestURL, &raw, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	items := raw[responseKey]
	columns := []string{"reference", "size", "safe_to_manage"}
	if resource == "snapshots" {
		columns = []string{"reference", "size", "safe_to_manage", "source_reference"}
	}
	if boolFlag(opts, "long") {
		columns = append(columns, "reason_not_safe", "cinder_id", "extra_info")
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"reference":      blockStorageManageableReferenceValue(item.Reference),
			"size":           item.Size,
			"safe_to_manage": item.SafeToManage,
		}
		if resource == "snapshots" {
			row["source_reference"] = blockStorageManageableReferenceValue(item.SourceReference)
		}
		if boolFlag(opts, "long") {
			row["reason_not_safe"] = item.ReasonNotSafe
			row["cinder_id"] = item.CinderID
			row["extra_info"] = item.ExtraInfo
		}
		rows = append(rows, row)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func blockStorageManageableReferenceValue(value any) tableValue {
	return tableValue{Value: value, Table: pythonRepr(value), Pretty: value}
}

type consistencyGroupRecord struct {
	ID               string   `json:"id"`
	Status           string   `json:"status"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	AvailabilityZone string   `json:"availability_zone"`
	VolumeTypes      []string `json:"volume_types"`
	CreatedAt        string   `json:"created_at"`
}

func consistencyGroupList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	items, err := listConsistencyGroups(ctx, client, boolFlag(opts, "all-projects"))
	if err != nil {
		return err
	}
	columns := []string{"ID", "Status", "Name"}
	if boolFlag(opts, "long") {
		columns = []string{"ID", "Status", "Availability Zone", "Name", "Description", "Volume Types"}
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":     item.ID,
			"Status": item.Status,
			"Name":   item.Name,
		}
		if boolFlag(opts, "long") {
			row["Availability Zone"] = item.AvailabilityZone
			row["Description"] = item.Description
			row["Volume Types"] = item.VolumeTypes
		}
		rows = append(rows, row)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func consistencyGroupShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("consistency group show requires <consistency-group>")
	}
	item, err := findConsistencyGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	raw, err := getBlockStorageResourceMap(ctx, client, client.ServiceURL("consistencygroups", url.PathEscape(item.ID)), "consistencygroup")
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, sortedFieldsFromMap(raw, false))
}

func consistencyGroupCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	volumeTypeValue := flagValue(opts, "volume-type")
	sourceValue := firstNonEmpty(flagValue(opts, "source"), flagValue(opts, "consistency-group-source"))
	snapshotValue := firstNonEmpty(flagValue(opts, "snapshot"), flagValue(opts, "consistency-group-snapshot"))
	if volumeTypeValue == "" && sourceValue == "" && snapshotValue == "" {
		return fmt.Errorf("one of --volume-type, --source, or --snapshot is required")
	}
	var request map[string]any
	requestURL := client.ServiceURL("consistencygroups")
	if volumeTypeValue != "" {
		volumeType, err := findVolumeType(ctx, client, volumeTypeValue)
		if err != nil {
			return err
		}
		request = map[string]any{
			"consistencygroup": map[string]any{
				"name":              nilIfEmpty(name),
				"description":       nilIfEmpty(flagValue(opts, "description")),
				"volume_types":      volumeType.ID,
				"availability_zone": nilIfEmpty(flagValue(opts, "availability-zone")),
				"status":            "creating",
			},
		}
	} else {
		body := map[string]any{
			"name":        nilIfEmpty(name),
			"description": nilIfEmpty(flagValue(opts, "description")),
			"status":      "creating",
		}
		if sourceValue != "" {
			source, err := findConsistencyGroup(ctx, client, sourceValue)
			if err != nil {
				return err
			}
			body["source_cgid"] = source.ID
		}
		if snapshotValue != "" {
			snapshot, err := findConsistencyGroupSnapshot(ctx, client, snapshotValue)
			if err != nil {
				return err
			}
			body["cgsnapshot_id"] = snapshot.ID
		}
		requestURL = client.ServiceURL("consistencygroups", "create_from_src")
		request = map[string]any{"consistencygroup-from-src": body}
	}
	raw, err := postBlockStorageResourceMap(ctx, client, requestURL, request, "consistencygroup")
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, sortedFieldsFromMap(raw, false))
}

func consistencyGroupDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("consistency group delete requires <consistency-group>")
	}
	failures := 0
	for _, value := range args {
		item, err := findConsistencyGroup(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if err := blockStoragePostAction(ctx, client, client.ServiceURL("consistencygroups", url.PathEscape(item.ID), "delete"), map[string]any{
			"consistencygroup": map[string]any{"force": boolFlag(opts, "force")},
		}, http.StatusOK, http.StatusAccepted, http.StatusNoContent); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d consistency groups failed to delete.", failures, len(args))
	}
	return nil
}

func consistencyGroupSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("consistency group set requires <consistency-group>")
	}
	item, err := findConsistencyGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := map[string]any{}
	if flagChanged(opts, "name") {
		update["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		update["description"] = flagValue(opts, "description")
	}
	if len(update) == 0 {
		return nil
	}
	return blockStoragePutAction(ctx, client, client.ServiceURL("consistencygroups", url.PathEscape(item.ID)), map[string]any{"consistencygroup": update}, http.StatusOK, http.StatusAccepted)
}

func consistencyGroupAddRemoveVolume(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string, add bool) error {
	if len(args) < 2 {
		if add {
			return fmt.Errorf("consistency group add volume requires <consistency-group> <volume>")
		}
		return fmt.Errorf("consistency group remove volume requires <consistency-group> <volume>")
	}
	group, err := findConsistencyGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(args)-1)
	failures := 0
	for _, value := range args[1:] {
		volume, err := findVolume(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		ids = append(ids, volume.ID)
	}
	if len(ids) > 0 {
		key := "add_volumes"
		if !add {
			key = "remove_volumes"
		}
		err = blockStoragePutAction(ctx, client, client.ServiceURL("consistencygroups", url.PathEscape(group.ID)), map[string]any{
			"consistencygroup": map[string]any{key: strings.Join(ids, ",")},
		}, http.StatusOK, http.StatusAccepted)
		if err != nil {
			return err
		}
	}
	if failures > 0 {
		action := "add"
		if !add {
			action = "remove"
		}
		return fmt.Errorf("%d of %d volumes failed to %s.", failures, len(args)-1, action)
	}
	return nil
}

func listConsistencyGroups(ctx context.Context, client *gophercloud.ServiceClient, allProjects bool) ([]consistencyGroupRecord, error) {
	requestURL := client.ServiceURL("consistencygroups", "detail")
	query := url.Values{}
	if allProjects {
		query.Set("all_tenants", "True")
	}
	if encoded := query.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	var response struct {
		ConsistencyGroups []consistencyGroupRecord `json:"consistencygroups"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, oscHTTPException(err)
	}
	return response.ConsistencyGroups, nil
}

func findConsistencyGroup(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string) (consistencyGroupRecord, error) {
	items, err := listConsistencyGroups(ctx, client, false)
	if err == nil {
		for _, item := range items {
			if item.ID == nameOrID || item.Name == nameOrID {
				return item, nil
			}
		}
	}
	var response struct {
		ConsistencyGroup consistencyGroupRecord `json:"consistencygroup"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("consistencygroups", url.PathEscape(nameOrID)), &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return consistencyGroupRecord{}, oscHTTPException(err)
	}
	return response.ConsistencyGroup, nil
}

type consistencyGroupSnapshotRecord struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	ConsistencyGroupID string `json:"consistencygroup_id"`
	CreatedAt          string `json:"created_at"`
}

func consistencyGroupSnapshotList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	consistencyGroupID := ""
	if value := flagValue(opts, "consistency-group"); value != "" {
		group, err := findConsistencyGroup(ctx, client, value)
		if err != nil {
			return err
		}
		consistencyGroupID = group.ID
	}
	items, err := listConsistencyGroupSnapshots(ctx, client, boolFlag(opts, "all-projects"), flagValue(opts, "status"), consistencyGroupID)
	if err != nil {
		return err
	}
	columns := []string{"ID", "Status", "Name"}
	if boolFlag(opts, "long") {
		columns = []string{"ID", "Status", "ConsistencyGroup ID", "Name", "Description", "Created At"}
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{
			"ID":     item.ID,
			"Status": item.Status,
			"Name":   item.Name,
		}
		if boolFlag(opts, "long") {
			row["ConsistencyGroup ID"] = item.ConsistencyGroupID
			row["Description"] = item.Description
			row["Created At"] = item.CreatedAt
		}
		rows = append(rows, row)
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func consistencyGroupSnapshotShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("consistency group snapshot show requires <consistency-group-snapshot>")
	}
	item, err := findConsistencyGroupSnapshot(ctx, client, args[0])
	if err != nil {
		return err
	}
	raw, err := getBlockStorageResourceMap(ctx, client, client.ServiceURL("cgsnapshots", url.PathEscape(item.ID)), "cgsnapshot")
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, sortedFieldsFromMap(raw, false))
}

func consistencyGroupSnapshotCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	groupValue := flagValue(opts, "consistency-group")
	if groupValue == "" {
		groupValue = name
	}
	if groupValue == "" {
		return fmt.Errorf("consistency group snapshot create requires <snapshot-name> or --consistency-group")
	}
	group, err := findConsistencyGroup(ctx, client, groupValue)
	if err != nil {
		return err
	}
	request := map[string]any{
		"cgsnapshot": map[string]any{
			"consistencygroup_id": group.ID,
			"name":                nilIfEmpty(name),
			"description":         nilIfEmpty(flagValue(opts, "description")),
			"status":              "creating",
		},
	}
	raw, err := postBlockStorageResourceMap(ctx, client, client.ServiceURL("cgsnapshots"), request, "cgsnapshot")
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, sortedFieldsFromMap(raw, false))
}

func consistencyGroupSnapshotDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("consistency group snapshot delete requires <consistency-group-snapshot>")
	}
	failures := 0
	for _, value := range args {
		item, err := findConsistencyGroupSnapshot(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if err := blockStorageDeleteAction(ctx, client, client.ServiceURL("cgsnapshots", url.PathEscape(item.ID)), http.StatusOK, http.StatusAccepted, http.StatusNoContent); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d consistency group snapshots failed to delete.", failures, len(args))
	}
	return nil
}

func listConsistencyGroupSnapshots(ctx context.Context, client *gophercloud.ServiceClient, allProjects bool, status string, consistencyGroupID string) ([]consistencyGroupSnapshotRecord, error) {
	requestURL := client.ServiceURL("cgsnapshots", "detail")
	query := url.Values{}
	if allProjects {
		query.Set("all_tenants", "True")
	}
	if status != "" {
		query.Set("status", status)
	}
	if consistencyGroupID != "" {
		query.Set("consistencygroup_id", consistencyGroupID)
	}
	if encoded := query.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	var response struct {
		CGSnapshots []consistencyGroupSnapshotRecord `json:"cgsnapshots"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, oscHTTPException(err)
	}
	return response.CGSnapshots, nil
}

func findConsistencyGroupSnapshot(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string) (consistencyGroupSnapshotRecord, error) {
	items, err := listConsistencyGroupSnapshots(ctx, client, false, "", "")
	if err == nil {
		for _, item := range items {
			if item.ID == nameOrID || item.Name == nameOrID {
				return item, nil
			}
		}
	}
	var response struct {
		CGSnapshot consistencyGroupSnapshotRecord `json:"cgsnapshot"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("cgsnapshots", url.PathEscape(nameOrID)), &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return consistencyGroupSnapshotRecord{}, oscHTTPException(err)
	}
	return response.CGSnapshot, nil
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
		{"links", volumeMessageLinksValue(item.Links)},
		{"message_level", item.MessageLevel},
		{"request_id", item.RequestID},
		{"resource_type", item.ResourceType},
		{"resource_uuid", item.ResourceUUID},
		{"user_message", item.UserMessage},
	})
}

func volumeMessageLinksValue(links []volumeMessageLink) tableValue {
	values := make([]any, 0, len(links))
	for _, link := range links {
		values = append(values, orderedJSONObject{
			keys: []string{"rel", "href"},
			values: map[string]any{
				"rel":  link.Rel,
				"href": link.Href,
			},
		})
	}
	return tableValue{Value: values, Table: pythonRepr(values), Pretty: values}
}

func volumeMessageDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume message delete requires <message-id>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.3", "volume message delete")
	if err != nil {
		return err
	}
	failures := 0
	for _, id := range args {
		if err := blockStorageDeleteAction(ctx, client, client.ServiceURL("messages", url.PathEscape(id)), http.StatusOK, http.StatusAccepted, http.StatusNoContent); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d volume messages failed to delete.", failures, len(args))
	}
	return nil
}

func volumeHostSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume host set requires <host-name>")
	}
	action := "thaw"
	if boolFlag(opts, "disable") {
		action = "freeze"
	}
	return blockStoragePutAction(ctx, client, client.ServiceURL("os-services", action), map[string]any{"host": args[0]}, http.StatusOK, http.StatusAccepted)
}

func volumeMigrate(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume migrate requires <volume>")
	}
	host := flagValue(opts, "host")
	if host == "" {
		return fmt.Errorf("volume migrate requires --host <host>")
	}
	item, err := findVolume(ctx, client, args[0])
	if err != nil {
		return err
	}
	return blockStoragePostAction(ctx, client, client.ServiceURL("volumes", url.PathEscape(item.ID), "action"), map[string]any{
		"os-migrate_volume": map[string]any{
			"host":            host,
			"force_host_copy": boolFlag(opts, "force-host-copy"),
			"lock_volume":     boolFlag(opts, "lock-volume"),
		},
	}, http.StatusOK, http.StatusAccepted, http.StatusNoContent)
}

func volumeRevert(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume revert requires <snapshot>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.40")
	if err != nil {
		return err
	}
	snapshot, err := findVolumeSnapshot(ctx, client, args[0])
	if err != nil {
		return err
	}
	return blockStoragePostAction(ctx, client, client.ServiceURL("volumes", url.PathEscape(snapshot.VolumeID), "action"), map[string]any{
		"revert": map[string]any{"snapshot_id": snapshot.ID},
	}, http.StatusOK, http.StatusAccepted, http.StatusNoContent)
}

func volumeBackendCapabilityShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume backend capability show requires <host>")
	}
	raw, err := getBlockStorageResourceMap(ctx, client, client.ServiceURL("capabilities", url.PathEscape(args[0])), "")
	if err != nil {
		return err
	}
	properties, _ := raw["properties"].(map[string]any)
	keys := orderedBackendCapabilityKeys(properties)
	rows := make([]outputRow, 0, len(keys))
	for _, key := range keys {
		property := mapAnyFromRaw(properties[key])
		rows = append(rows, outputRow{
			"Title":       property["title"],
			"Key":         key,
			"Type":        property["type"],
			"Description": property["description"],
		})
	}
	return renderListOutput(stdout, opts, []string{"Title", "Key", "Type", "Description"}, rows)
}

func orderedBackendCapabilityKeys(properties map[string]any) []string {
	preferred := []string{"thin_provisioning", "compression", "qos", "replication_enabled"}
	keys := make([]string, 0, len(properties))
	seen := map[string]bool{}
	for _, key := range preferred {
		if _, ok := properties[key]; ok {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	remaining := make([]string, 0, len(properties)-len(keys))
	for key := range properties {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	return append(keys, remaining...)
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

func volumeGroupCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	volumeGroupTypeValue := flagValue(opts, "volume-group-type")
	volumeTypeValues := append([]string{}, flagValues(opts, "volume-type")...)
	if volumeGroupTypeValue == "" && len(args) > 0 {
		volumeGroupTypeValue = args[0]
		if len(args) > 1 {
			volumeTypeValues = append(volumeTypeValues, args[1:]...)
		}
	}
	sourceGroupValue := flagValue(opts, "source-group")
	groupSnapshotValue := flagValue(opts, "group-snapshot")
	if sourceGroupValue != "" || groupSnapshotValue != "" {
		client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.14", "volume group create [--source-group|--group-snapshot]")
		if err != nil {
			return err
		}
		body := map[string]any{
			"name":        nilIfEmpty(flagValue(opts, "name")),
			"description": nilIfEmpty(flagValue(opts, "description")),
		}
		if sourceGroupValue != "" {
			sourceGroup, err := findVolumeGroup(ctx, client, sourceGroupValue)
			if err != nil {
				return err
			}
			body["source_group_id"] = sourceGroup.ID
		}
		if groupSnapshotValue != "" {
			groupSnapshot, err := findVolumeGroupSnapshot(ctx, client, groupSnapshotValue)
			if err != nil {
				return err
			}
			body["group_snapshot_id"] = groupSnapshot.ID
		}
		var response struct {
			Group volumeGroupRecord `json:"group"`
		}
		resp, err := client.Post(ctx, client.ServiceURL("groups", "action"), map[string]any{"create-from-src": body}, &response, &gophercloud.RequestOpts{
			OkCodes: []int{http.StatusOK, http.StatusAccepted},
		})
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
		group, err := getVolumeGroup(ctx, client, response.Group.ID)
		if err == nil {
			response.Group = group
		}
		return renderVolumeGroupShow(stdout, opts, response.Group)
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.13", "volume group create")
	if err != nil {
		return err
	}
	if volumeGroupTypeValue == "" {
		return fmt.Errorf("--volume-group-type is required unless --source-group or --group-snapshot is specified")
	}
	if len(volumeTypeValues) == 0 {
		return fmt.Errorf("--volume-type is a required argument when creating a group from group type")
	}
	volumeGroupType, err := findVolumeGroupType(ctx, client, volumeGroupTypeValue)
	if err != nil {
		return err
	}
	volumeTypeIDs := make([]string, 0, len(volumeTypeValues))
	for _, value := range volumeTypeValues {
		volumeType, err := findVolumeType(ctx, client, value)
		if err != nil {
			return err
		}
		volumeTypeIDs = append(volumeTypeIDs, volumeType.ID)
	}
	request := map[string]any{
		"group": map[string]any{
			"name":              nilIfEmpty(flagValue(opts, "name")),
			"description":       nilIfEmpty(flagValue(opts, "description")),
			"group_type":        volumeGroupType.ID,
			"volume_types":      volumeTypeIDs,
			"availability_zone": nilIfEmpty(flagValue(opts, "availability-zone")),
		},
	}
	var response struct {
		Group volumeGroupRecord `json:"group"`
	}
	resp, err := client.Post(ctx, client.ServiceURL("groups"), request, &response, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusOK, http.StatusAccepted},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	group, err := getVolumeGroup(ctx, client, response.Group.ID)
	if err == nil {
		response.Group = group
	}
	return renderVolumeGroupShow(stdout, opts, response.Group)
}

func volumeGroupDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group delete requires <group>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.13", "volume group delete")
	if err != nil {
		return err
	}
	group, err := findVolumeGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	return blockStoragePostAction(ctx, client, client.ServiceURL("groups", url.PathEscape(group.ID), "action"), map[string]any{
		"delete": map[string]any{"delete-volumes": boolFlag(opts, "force")},
	}, http.StatusOK, http.StatusAccepted, http.StatusNoContent)
}

func volumeGroupSet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group set requires <group>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.13", "volume group set")
	if err != nil {
		return err
	}
	group, err := findVolumeGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := map[string]any{}
	if flagChanged(opts, "name") {
		update["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		update["description"] = flagValue(opts, "description")
	}
	if len(update) > 0 {
		resp, err := client.Put(ctx, client.ServiceURL("groups", url.PathEscape(group.ID)), map[string]any{"group": update}, nil, &gophercloud.RequestOpts{
			OkCodes: []int{http.StatusOK, http.StatusAccepted},
		})
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
	}
	if boolFlag(opts, "enable-replication") || boolFlag(opts, "disable-replication") {
		client, err = blockStorageClientWithExplicitMinimumMicroversion(client, "3.38", "volume group set --enable-replication/--disable-replication")
		if err != nil {
			return err
		}
		action := "enable_replication"
		if boolFlag(opts, "disable-replication") {
			action = "disable_replication"
		}
		if err := blockStoragePostAction(ctx, client, client.ServiceURL("groups", url.PathEscape(group.ID), "action"), map[string]any{action: map[string]any{}}, http.StatusOK, http.StatusAccepted, http.StatusNoContent); err != nil {
			return err
		}
	}
	group, err = getVolumeGroup(ctx, client, group.ID)
	if err != nil {
		return err
	}
	return renderVolumeGroupShow(stdout, opts, group)
}

func volumeGroupFailover(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group failover requires <group>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.38", "volume group failover")
	if err != nil {
		return err
	}
	group, err := findVolumeGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	allowAttached := boolFlag(opts, "allow-attached-volume")
	if boolFlag(opts, "disallow-attached-volume") {
		allowAttached = false
	}
	return blockStoragePostAction(ctx, client, client.ServiceURL("groups", url.PathEscape(group.ID), "action"), map[string]any{
		"failover_replication": map[string]any{
			"allow_attached_volume": allowAttached,
			"secondary_backend_id":  nilIfEmpty(flagValue(opts, "secondary-backend-id")),
		},
	}, http.StatusOK, http.StatusAccepted, http.StatusNoContent)
}

func renderVolumeGroupShow(stdout io.Writer, opts *Options, item volumeGroupRecord) error {
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
	return getVolumeGroup(ctx, client, nameOrID)
}

func getVolumeGroup(ctx context.Context, client *gophercloud.ServiceClient, id string) (volumeGroupRecord, error) {
	var response struct {
		Group volumeGroupRecord `json:"group"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("groups", url.PathEscape(id)), &response, nil)
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
	CreatedAt   string `json:"created_at"`
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

func volumeGroupSnapshotCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group snapshot create requires <volume_group>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.14")
	if err != nil {
		return err
	}
	group, err := findVolumeGroup(ctx, client, args[0])
	if err != nil {
		return err
	}
	request := map[string]any{
		"group_snapshot": map[string]any{
			"group_id":    group.ID,
			"name":        nilIfEmpty(flagValue(opts, "name")),
			"description": nilIfEmpty(flagValue(opts, "description")),
		},
	}
	var response struct {
		GroupSnapshot volumeGroupSnapshotRecord `json:"group_snapshot"`
	}
	resp, err := client.Post(ctx, client.ServiceURL("group_snapshots"), request, &response, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusOK, http.StatusAccepted},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	item, err := getVolumeGroupSnapshot(ctx, client, response.GroupSnapshot.ID)
	if err == nil {
		response.GroupSnapshot = item
	}
	return renderVolumeGroupSnapshotShow(stdout, opts, response.GroupSnapshot)
}

func volumeGroupSnapshotDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group snapshot delete requires <snapshot>")
	}
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.14")
	if err != nil {
		return err
	}
	failures := 0
	for _, value := range args {
		item, err := findVolumeGroupSnapshot(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if err := blockStorageDeleteAction(ctx, client, client.ServiceURL("group_snapshots", url.PathEscape(item.ID)), http.StatusOK, http.StatusAccepted, http.StatusNoContent); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d volume group snapshots failed to delete.", failures, len(args))
	}
	return nil
}

func renderVolumeGroupSnapshotShow(stdout io.Writer, opts *Options, item volumeGroupSnapshotRecord) error {
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
	return getVolumeGroupSnapshot(ctx, client, nameOrID)
}

func getVolumeGroupSnapshot(ctx context.Context, client *gophercloud.ServiceClient, id string) (volumeGroupSnapshotRecord, error) {
	var response struct {
		GroupSnapshot volumeGroupSnapshotRecord `json:"group_snapshot"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("group_snapshots", url.PathEscape(id)), &response, nil)
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

func volumeGroupTypeCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group type create requires <name>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.11", "volume group type create")
	if err != nil {
		return err
	}
	isPublic := true
	if boolFlag(opts, "private") {
		isPublic = false
	}
	request := map[string]any{
		"group_type": map[string]any{
			"name":        args[0],
			"description": nilIfEmpty(flagValue(opts, "description")),
			"is_public":   isPublic,
		},
	}
	var response struct {
		GroupType volumeGroupTypeRecord `json:"group_type"`
	}
	resp, err := client.Post(ctx, client.ServiceURL("group_types"), request, &response, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusOK, http.StatusAccepted},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return oscHTTPException(err)
	}
	return renderVolumeGroupTypeShow(stdout, opts, response.GroupType)
}

func volumeGroupTypeDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group type delete requires <group_type>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.11", "volume group type delete")
	if err != nil {
		return err
	}
	failures := 0
	for _, value := range args {
		item, err := findVolumeGroupType(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if err := blockStorageDeleteAction(ctx, client, client.ServiceURL("group_types", url.PathEscape(item.ID)), http.StatusOK, http.StatusAccepted, http.StatusNoContent); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d volume group types failed to delete.", failures, len(args))
	}
	return nil
}

func volumeGroupTypeSet(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume group type set requires <group_type>")
	}
	client, err := blockStorageClientWithExplicitMinimumMicroversion(client, "3.11", "volume group type set")
	if err != nil {
		return err
	}
	item, err := findVolumeGroupType(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := map[string]any{}
	if flagChanged(opts, "name") {
		update["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		update["description"] = flagValue(opts, "description")
	}
	if boolFlag(opts, "public") {
		update["is_public"] = true
	} else if boolFlag(opts, "private") {
		update["is_public"] = false
	}
	if len(update) > 0 {
		var response struct {
			GroupType volumeGroupTypeRecord `json:"group_type"`
		}
		resp, err := client.Put(ctx, client.ServiceURL("group_types", url.PathEscape(item.ID)), map[string]any{"group_type": update}, &response, &gophercloud.RequestOpts{
			OkCodes: []int{http.StatusOK, http.StatusAccepted},
		})
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
		item = response.GroupType
	}
	if boolFlag(opts, "no-property") {
		for key := range item.GroupSpecs {
			if err := blockStorageDeleteAction(ctx, client, client.ServiceURL("group_types", url.PathEscape(item.ID), "group_specs", url.PathEscape(key)), http.StatusOK, http.StatusAccepted, http.StatusNoContent); err != nil {
				return err
			}
		}
	}
	properties, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	if len(properties) > 0 {
		if err := blockStoragePostAction(ctx, client, client.ServiceURL("group_types", url.PathEscape(item.ID), "group_specs"), map[string]any{"group_specs": properties}, http.StatusOK, http.StatusAccepted); err != nil {
			return err
		}
	}
	item, err = findVolumeGroupType(ctx, client, item.ID)
	if err != nil {
		return err
	}
	return renderVolumeGroupTypeShow(stdout, opts, item)
}

func renderVolumeGroupTypeShow(stdout io.Writer, opts *Options, item volumeGroupTypeRecord) error {
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

func volumeTransferCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume transfer request create requires <volume>")
	}
	if boolFlag(opts, "no-snapshots") {
		clientWithVersion, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.55")
		if err != nil {
			return err
		}
		client = clientWithVersion
	}
	volume, err := findVolume(ctx, client, args[0])
	if err != nil {
		return err
	}
	body := map[string]any{
		"transfer": map[string]any{
			"volume_id": volume.ID,
			"name":      nilIfEmpty(flagValue(opts, "name")),
		},
	}
	if boolFlag(opts, "no-snapshots") {
		body["transfer"].(map[string]any)["no_snapshots"] = true
	}
	var response struct {
		Transfer transfers.Transfer `json:"transfer"`
	}
	resp, err := client.Post(ctx, client.ServiceURL("volume-transfers"), body, &response, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusAccepted},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		resp, err = client.Post(ctx, client.ServiceURL("os-volume-transfer"), body, &response, &gophercloud.RequestOpts{
			OkCodes: []int{http.StatusAccepted},
		})
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return err
		}
	}
	return renderVolumeTransferShow(stdout, opts, &response.Transfer)
}

func volumeTransferDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume transfer request delete requires <transfer-request>")
	}
	failures := 0
	for _, value := range args {
		item, err := findVolumeTransfer(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if err := transfers.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d volume transfer requests failed to delete.", failures, len(args))
	}
	return nil
}

func volumeTransferAccept(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume transfer request accept requires <transfer-request-id>")
	}
	authKey := flagValue(opts, "auth-key")
	if authKey == "" {
		return fmt.Errorf("volume transfer request accept requires --auth-key <key>")
	}
	item, err := transfers.Accept(ctx, client, args[0], transfers.AcceptOpts{AuthKey: authKey}).Extract()
	if err != nil {
		return err
	}
	return renderVolumeTransferShow(stdout, opts, item)
}

func renderVolumeTransferShow(stdout io.Writer, opts *Options, item *transfers.Transfer) error {
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

type subnetCreateOpts struct {
	Values map[string]any
}

func (opts subnetCreateOpts) ToSubnetCreateMap() (map[string]any, error) {
	return map[string]any{"subnet": opts.Values}, nil
}

type subnetUpdateOpts struct {
	Values map[string]any
}

func (opts subnetUpdateOpts) ToSubnetUpdateMap() (map[string]any, error) {
	return map[string]any{"subnet": opts.Values}, nil
}

func subnetCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet create requires <name>")
	}
	if boolFlag(opts, "no-tag") && len(flagValues(opts, "tag")) > 0 {
		return fmt.Errorf("argument --no-tag: not allowed with argument --tag")
	}
	values, err := subnetCreateValues(ctx, opts, networkClient, identityClient, args[0])
	if err != nil {
		return err
	}
	result := subnets.Create(ctx, networkClient, subnetCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	tags := flagValues(opts, "tag")
	if len(tags) > 0 {
		target, err := setNeutronResourceTags(ctx, networkClient, "subnets", item.ID, item.Tags, tags, boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
		item.Tags = target
	}
	if raw, ok := subnetRawFromBody(result.Body); ok {
		if len(tags) > 0 {
			raw["tags"] = item.Tags
		}
		return renderShowOutput(stdout, opts, subnetRawFields(raw))
	}
	return renderShowOutput(stdout, opts, subnetFields(item))
}

func subnetCreateValues(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, name string) (map[string]any, error) {
	if flagValue(opts, "network") == "" {
		return nil, fmt.Errorf("argument --network is required")
	}
	values := map[string]any{
		"name":       name,
		"network_id": resolveNetworkID(ctx, networkClient, flagValue(opts, "network")),
		"ip_version": 4,
	}
	if flagChanged(opts, "ip-version") {
		ipVersion := intFlag(opts, "ip-version")
		if ipVersion != 4 && ipVersion != 6 {
			return nil, fmt.Errorf("argument --ip-version: invalid choice: %d (choose from 4, 6)", ipVersion)
		}
		values["ip_version"] = ipVersion
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return nil, err
		}
		values["project_id"] = project.ID
	}
	poolChoices := 0
	if flagValue(opts, "subnet-pool") != "" {
		poolChoices++
	}
	if boolFlag(opts, "use-prefix-delegation") {
		poolChoices++
	}
	if boolFlag(opts, "use-default-subnet-pool") {
		poolChoices++
	}
	if poolChoices > 1 {
		return nil, fmt.Errorf("argument --use-default-subnet-pool: not allowed with argument --subnet-pool or --use-prefix-delegation")
	}
	if subnetPool := flagValue(opts, "subnet-pool"); subnetPool != "" {
		item, err := findSubnetPool(ctx, networkClient, subnetPool)
		if err != nil {
			return nil, err
		}
		values["subnetpool_id"] = item.ID
	}
	if boolFlag(opts, "use-prefix-delegation") {
		values["subnetpool_id"] = "prefix_delegation"
	}
	if boolFlag(opts, "use-default-subnet-pool") {
		values["use_default_subnet_pool"] = true
	}
	if prefixLength := flagValue(opts, "prefix-length"); prefixLength != "" {
		parsed, err := strconv.Atoi(prefixLength)
		if err != nil {
			return nil, fmt.Errorf("argument --prefix-length: invalid int value: %q", prefixLength)
		}
		values["prefixlen"] = parsed
	}
	if cidr := flagValue(opts, "subnet-range"); cidr != "" {
		values["cidr"] = cidr
	} else if poolChoices == 0 {
		return nil, fmt.Errorf("argument --subnet-range is required when --subnet-pool is not specified")
	}
	if err := subnetApplyMutableValues(ctx, opts, networkClient, values, true, nil); err != nil {
		return nil, err
	}
	if value := flagValue(opts, "ipv6-ra-mode"); value != "" {
		if !validIPv6Mode(value) {
			return nil, fmt.Errorf("argument --ipv6-ra-mode: invalid choice: %q", value)
		}
		values["ipv6_ra_mode"] = value
	}
	if value := flagValue(opts, "ipv6-address-mode"); value != "" {
		if !validIPv6Mode(value) {
			return nil, fmt.Errorf("argument --ipv6-address-mode: invalid choice: %q", value)
		}
		values["ipv6_address_mode"] = value
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

func subnetDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet delete requires <subnet> [<subnet> ...]")
	}
	failures := 0
	for _, subnetArg := range args {
		item, err := findSubnet(ctx, client, subnetArg)
		if err != nil {
			failures++
			continue
		}
		if err := subnets.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d subnets failed to delete.", failures, len(args))
	}
	return nil
}

func subnetSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet set requires <subnet>")
	}
	item, err := findSubnet(ctx, client, args[0])
	if err != nil {
		return err
	}
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if err := subnetApplyMutableValues(ctx, opts, client, values, false, item); err != nil {
		return err
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	for key, value := range extra {
		values[key] = value
	}
	if len(values) > 0 {
		if _, err := subnets.Update(ctx, client, item.ID, subnetUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "no-tag") || len(flagValues(opts, "tag")) > 0 {
		_, err := setNeutronResourceTags(ctx, client, "subnets", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
	}
	return nil
}

func subnetUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("subnet unset requires <subnet>")
	}
	if len(flagValues(opts, "tag")) > 0 && boolFlag(opts, "all-tag") {
		return fmt.Errorf("argument --all-tag: not allowed with argument --tag")
	}
	item, err := findSubnet(ctx, client, args[0])
	if err != nil {
		return err
	}
	values, err := subnetUnsetValues(opts, item)
	if err != nil {
		return err
	}
	extra, err := parseUnsetExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	for key, value := range extra {
		values[key] = value
	}
	if len(values) > 0 {
		if _, err := subnets.Update(ctx, client, item.ID, subnetUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	_, err = unsetNeutronResourceTags(ctx, client, "subnets", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "all-tag"))
	return err
}

func subnetApplyMutableValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, values map[string]any, create bool, existing *subnets.Subnet) error {
	if segment := flagValue(opts, "network-segment"); segment != "" {
		item, err := findNetworkSegment(ctx, client, segment)
		if err != nil {
			return err
		}
		values["segment_id"] = item.ID
	}
	if flagChanged(opts, "gateway") {
		gateway := strings.ToLower(flagValue(opts, "gateway"))
		if !create && gateway == "auto" {
			return fmt.Errorf("Auto option is not available for Subnet Set. Valid options are <ip-address> or none")
		}
		if gateway == "none" {
			values["gateway_ip"] = nil
		} else if gateway != "auto" {
			values["gateway_ip"] = gateway
		}
	}
	if dhcp, err := networkBoolFlag(opts, "dhcp", "no-dhcp"); err != nil {
		return err
	} else if dhcp != nil {
		values["enable_dhcp"] = *dhcp
	}
	if dnsPublish, err := networkBoolFlag(opts, "dns-publish-fixed-ip", "no-dns-publish-fixed-ip"); err != nil {
		return err
	} else if dnsPublish != nil {
		values["dns_publish_fixed_ip"] = *dnsPublish
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	allocationPools, err := parseSubnetAllocationPools(flagValues(opts, "allocation-pool"))
	if err != nil {
		return err
	}
	if create {
		if len(allocationPools) > 0 {
			values["allocation_pools"] = allocationPools
		}
	} else if len(allocationPools) > 0 {
		merged := append([]map[string]string{}, allocationPools...)
		if !boolFlag(opts, "no-allocation-pool") && existing != nil {
			merged = append(merged, subnetAllocationPoolMaps(existing.AllocationPools)...)
		}
		values["allocation_pools"] = merged
	} else if boolFlag(opts, "no-allocation-pool") {
		values["allocation_pools"] = []map[string]string{}
	}
	if nameservers := flagValues(opts, "dns-nameserver"); len(nameservers) > 0 {
		merged := append([]string{}, nameservers...)
		if !create && !boolFlag(opts, "no-dns-nameservers") && existing != nil {
			merged = append(merged, existing.DNSNameservers...)
		}
		values["dns_nameservers"] = merged
	} else if !create && boolFlag(opts, "no-dns-nameservers") {
		values["dns_nameservers"] = []string{}
	}
	hostRoutes, err := parseSubnetHostRoutes(flagValues(opts, "host-route"))
	if err != nil {
		return err
	}
	if create {
		if len(hostRoutes) > 0 {
			values["host_routes"] = hostRoutes
		}
	} else if len(hostRoutes) > 0 {
		merged := append([]map[string]string{}, hostRoutes...)
		if !boolFlag(opts, "no-host-route") && existing != nil {
			merged = append(merged, subnetHostRouteMaps(existing.HostRoutes)...)
		}
		values["host_routes"] = merged
	} else if boolFlag(opts, "no-host-route") {
		values["host_routes"] = []map[string]string{}
	}
	if serviceTypes := flagValues(opts, "service-type"); len(serviceTypes) > 0 {
		merged := append([]string{}, serviceTypes...)
		if !create && existing != nil {
			merged = append(merged, existing.ServiceTypes...)
		}
		values["service_types"] = merged
	}
	return nil
}

func subnetUnsetValues(opts *Options, item *subnets.Subnet) (map[string]any, error) {
	values := map[string]any{}
	if boolFlag(opts, "gateway") {
		values["gateway_ip"] = nil
	}
	if valuesToRemove := flagValues(opts, "dns-nameserver"); len(valuesToRemove) > 0 {
		remaining, err := removeStringValues(item.DNSNameservers, valuesToRemove, "dns-nameserver")
		if err != nil {
			return nil, err
		}
		values["dns_nameservers"] = remaining
	}
	hostRoutes, err := parseSubnetHostRoutes(flagValues(opts, "host-route"))
	if err != nil {
		return nil, err
	}
	if len(hostRoutes) > 0 {
		remaining, err := removeMapValues(subnetHostRouteMaps(item.HostRoutes), hostRoutes, "host-route")
		if err != nil {
			return nil, err
		}
		values["host_routes"] = remaining
	}
	allocationPools, err := parseSubnetAllocationPools(flagValues(opts, "allocation-pool"))
	if err != nil {
		return nil, err
	}
	if len(allocationPools) > 0 {
		remaining, err := removeMapValues(subnetAllocationPoolMaps(item.AllocationPools), allocationPools, "allocation-pool")
		if err != nil {
			return nil, err
		}
		values["allocation_pools"] = remaining
	}
	if serviceTypes := flagValues(opts, "service-type"); len(serviceTypes) > 0 {
		remaining, err := removeStringValues(item.ServiceTypes, serviceTypes, "service-type")
		if err != nil {
			return nil, err
		}
		values["service_types"] = remaining
	}
	return values, nil
}

func validIPv6Mode(value string) bool {
	switch value {
	case "dhcpv6-stateful", "dhcpv6-stateless", "slaac":
		return true
	default:
		return false
	}
}

func parseSubnetAllocationPools(values []string) ([]map[string]string, error) {
	entries, err := parseKeyValueEntries(values, "allocation-pool", "start", "end")
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func parseSubnetHostRoutes(values []string) ([]map[string]string, error) {
	entries, err := parseKeyValueEntries(values, "host-route", "destination", "gateway")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		entry["nexthop"] = entry["gateway"]
		delete(entry, "gateway")
	}
	return entries, nil
}

func parseKeyValueEntries(values []string, option string, required ...string) ([]map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	entries := make([]map[string]string, 0, len(values))
	for _, value := range values {
		entry := map[string]string{}
		for _, part := range strings.Split(value, ",") {
			key, raw, ok := strings.Cut(part, "=")
			if !ok || strings.TrimSpace(key) == "" {
				return nil, fmt.Errorf("invalid %s %q", option, value)
			}
			entry[strings.TrimSpace(key)] = raw
		}
		for _, key := range required {
			if entry[key] == "" {
				return nil, fmt.Errorf("invalid %s %q, missing %s", option, value, key)
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func subnetAllocationPoolMaps(values []subnets.AllocationPool) []map[string]string {
	items := make([]map[string]string, 0, len(values))
	for _, value := range values {
		items = append(items, map[string]string{"start": value.Start, "end": value.End})
	}
	return items
}

type subnetAllocationPoolOutput struct {
	Start string `json:"start" yaml:"start"`
	End   string `json:"end" yaml:"end"`
}

func subnetAllocationPoolsForOutput(value any) any {
	table := func(items []subnetAllocationPoolOutput) tableValue {
		lines := make([]string, 0, len(items))
		jsonValues := make([]any, 0, len(items))
		for _, item := range items {
			if item.Start != "" && item.End != "" {
				lines = append(lines, item.Start+"-"+item.End)
			}
			jsonValues = append(jsonValues, orderedJSONObject{
				keys:   []string{"start", "end"},
				values: map[string]any{"start": item.Start, "end": item.End},
			})
		}
		return tableValue{Value: jsonValues, Table: strings.Join(lines, "\n"), Pretty: items}
	}
	switch typed := value.(type) {
	case []subnets.AllocationPool:
		items := make([]subnetAllocationPoolOutput, 0, len(typed))
		for _, pool := range typed {
			items = append(items, subnetAllocationPoolOutput{Start: pool.Start, End: pool.End})
		}
		return table(items)
	case []map[string]string:
		items := make([]subnetAllocationPoolOutput, 0, len(typed))
		for _, pool := range typed {
			items = append(items, subnetAllocationPoolOutput{Start: pool["start"], End: pool["end"]})
		}
		return table(items)
	case []any:
		items := make([]subnetAllocationPoolOutput, 0, len(typed))
		for _, item := range typed {
			pool, ok := item.(map[string]any)
			if !ok {
				return value
			}
			items = append(items, subnetAllocationPoolOutput{
				Start: valueString(pool["start"]),
				End:   valueString(pool["end"]),
			})
		}
		return table(items)
	default:
		return value
	}
}

func subnetHostRouteMaps(values []subnets.HostRoute) []map[string]string {
	items := make([]map[string]string, 0, len(values))
	for _, value := range values {
		items = append(items, map[string]string{"destination": value.DestinationCIDR, "nexthop": value.NextHop})
	}
	return items
}

func removeStringValues(existing []string, values []string, option string) ([]string, error) {
	remaining := append([]string{}, existing...)
	for _, value := range values {
		found := false
		for index, existingValue := range remaining {
			if existingValue == value {
				remaining = append(remaining[:index], remaining[index+1:]...)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Subnet does not contain %s %s", option, value)
		}
	}
	return remaining, nil
}

func removeMapValues(existing []map[string]string, values []map[string]string, option string) ([]map[string]string, error) {
	remaining := append([]map[string]string{}, existing...)
	for _, value := range values {
		found := false
		for index, existingValue := range remaining {
			if stringMapEqual(existingValue, value) {
				remaining = append(remaining[:index], remaining[index+1:]...)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Subnet does not contain %s %v", option, value)
		}
	}
	return remaining, nil
}

func stringMapEqual(left map[string]string, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if right[key] != leftValue {
			return false
		}
	}
	return true
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
	if raw, err := neutronSubnetRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, subnetRawFields(raw))
	}
	return renderShowOutput(stdout, opts, subnetFields(item))
}

func neutronSubnetRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		Subnet map[string]any `json:"subnet"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("subnets", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Subnet, nil
}

func subnetRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		Subnet map[string]any `json:"subnet"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.Subnet == nil {
		return nil, false
	}
	return wrapper.Subnet, true
}

func subnetRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"allocation_pools":     true,
		"cidr":                 true,
		"created_at":           true,
		"description":          true,
		"dns_nameservers":      true,
		"dns_publish_fixed_ip": true,
		"enable_dhcp":          true,
		"gateway_ip":           true,
		"host_routes":          true,
		"id":                   true,
		"ip_version":           true,
		"ipv6_address_mode":    true,
		"ipv6_ra_mode":         true,
		"location":             true,
		"name":                 true,
		"network_id":           true,
		"project_id":           true,
		"revision_number":      true,
		"router:external":      true,
		"segment_id":           true,
		"service_types":        true,
		"subnetpool_id":        true,
		"tags":                 true,
		"tenant_id":            true,
		"updated_at":           true,
	}
	fields := []outputField{
		{"allocation_pools", subnetAllocationPoolsForOutput(raw["allocation_pools"])},
		{"cidr", raw["cidr"]},
		{"created_at", raw["created_at"]},
		{"description", raw["description"]},
		{"dns_nameservers", raw["dns_nameservers"]},
		{"dns_publish_fixed_ip", raw["dns_publish_fixed_ip"]},
		{"enable_dhcp", raw["enable_dhcp"]},
		{"gateway_ip", raw["gateway_ip"]},
		{"host_routes", blankEmptyListValue(raw["host_routes"])},
		{"id", raw["id"]},
		{"ip_version", rawNumber(raw["ip_version"])},
		{"ipv6_address_mode", raw["ipv6_address_mode"]},
		{"ipv6_ra_mode", raw["ipv6_ra_mode"]},
		{"name", raw["name"]},
		{"network_id", raw["network_id"]},
		{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		{"revision_number", rawNumber(raw["revision_number"])},
		{"router:external", raw["router:external"]},
		{"segment_id", raw["segment_id"]},
		{"service_types", blankEmptyListValue(raw["service_types"])},
		{"subnetpool_id", raw["subnetpool_id"]},
		{"tags", blankEmptyListValue(raw["tags"])},
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

func subnetFields(item *subnets.Subnet) []outputField {
	return []outputField{
		{"allocation_pools", subnetAllocationPoolsForOutput(item.AllocationPools)},
		{"cidr", item.CIDR},
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"dns_nameservers", item.DNSNameservers},
		{"dns_publish_fixed_ip", item.DNSPublishFixedIP},
		{"enable_dhcp", item.EnableDHCP},
		{"gateway_ip", nilIfEmpty(item.GatewayIP)},
		{"host_routes", blankEmptyListValue(subnetHostRouteMaps(item.HostRoutes))},
		{"id", item.ID},
		{"ip_version", item.IPVersion},
		{"ipv6_address_mode", nilIfEmpty(item.IPv6AddressMode)},
		{"ipv6_ra_mode", nilIfEmpty(item.IPv6RAMode)},
		{"name", item.Name},
		{"network_id", item.NetworkID},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"revision_number", item.RevisionNumber},
		{"router:external", nil},
		{"segment_id", nilIfEmpty(item.SegmentID)},
		{"service_types", blankEmptyStringListValue(item.ServiceTypes)},
		{"subnetpool_id", nilIfEmpty(item.SubnetPoolID)},
		{"tags", blankEmptyStringListValue(item.Tags)},
		{"updated_at", oscTime(item.UpdatedAt)},
	}
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

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
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

type portCreateOpts struct {
	Values map[string]any
}

func (opts portCreateOpts) ToPortCreateMap() (map[string]any, error) {
	return map[string]any{"port": opts.Values}, nil
}

type portUpdateOpts struct {
	Values map[string]any
}

func (opts portUpdateOpts) ToPortUpdateMap() (map[string]any, error) {
	return map[string]any{"port": opts.Values}, nil
}

func portCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("port create requires <name>")
	}
	if flagValue(opts, "network") == "" {
		return fmt.Errorf("argument --network is required")
	}
	if boolFlag(opts, "no-tag") && len(flagValues(opts, "tag")) > 0 {
		return fmt.Errorf("argument --no-tag: not allowed with argument --tag")
	}
	values, err := portCreateValues(ctx, opts, networkClient, identityClient, args[0])
	if err != nil {
		return err
	}
	result := ports.Create(ctx, networkClient, portCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	tags := flagValues(opts, "tag")
	if boolFlag(opts, "no-tag") || len(tags) > 0 {
		target, err := setNeutronResourceTags(ctx, networkClient, "ports", item.ID, item.Tags, tags, boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
		item.Tags = target
	}
	if raw, ok := portRawFromBody(result.Body); ok {
		if boolFlag(opts, "no-tag") || len(tags) > 0 {
			raw["tags"] = item.Tags
		}
		return renderShowOutput(stdout, opts, portRawFields(raw))
	}
	return renderShowOutput(stdout, opts, portFields(item))
}

func portCreateValues(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, name string) (map[string]any, error) {
	network, err := findNetwork(ctx, networkClient, flagValue(opts, "network"))
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		"name":           name,
		"network_id":     network.ID,
		"admin_state_up": true,
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return nil, err
		}
		values["project_id"] = project.ID
	}
	if err := portApplyMutableValues(ctx, opts, networkClient, values, true, nil, nil); err != nil {
		return nil, err
	}
	if boolFlag(opts, "no-fixed-ip") && len(flagValues(opts, "fixed-ip")) > 0 {
		return nil, fmt.Errorf("argument --no-fixed-ip: not allowed with argument --fixed-ip")
	}
	if boolFlag(opts, "no-fixed-ip") {
		values["fixed_ips"] = []map[string]string{}
	}
	if fixedIPs, err := portFixedIPMaps(ctx, networkClient, flagValues(opts, "fixed-ip")); err != nil {
		return nil, err
	} else if len(fixedIPs) > 0 {
		values["fixed_ips"] = fixedIPs
	}
	if boolFlag(opts, "no-security-group") && len(flagValues(opts, "security-group")) > 0 {
		return nil, fmt.Errorf("argument --no-security-group: not allowed with argument --security-group")
	}
	if boolFlag(opts, "no-security-group") {
		values["security_groups"] = []string{}
	} else if securityGroups, err := portSecurityGroupIDs(ctx, networkClient, flagValues(opts, "security-group")); err != nil {
		return nil, err
	} else if len(securityGroups) > 0 {
		values["security_groups"] = securityGroups
	}
	if allowed, err := portAllowedAddressMaps(flagValues(opts, "allowed-address")); err != nil {
		return nil, err
	} else if len(allowed) > 0 {
		values["allowed_address_pairs"] = allowed
	}
	if dhcpOptions, err := portExtraDHCPOptions(flagValues(opts, "extra-dhcp-option")); err != nil {
		return nil, err
	} else if len(dhcpOptions) > 0 {
		values["extra_dhcp_opts"] = dhcpOptions
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

func portDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("port delete requires <port> [<port> ...]")
	}
	failures := 0
	for _, portArg := range args {
		item, err := findPort(ctx, client, portArg)
		if err != nil {
			failures++
			continue
		}
		if err := ports.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d ports failed to delete.", failures, len(args))
	}
	return nil
}

func portSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("port set requires <port>")
	}
	item, err := findPort(ctx, client, args[0])
	if err != nil {
		return err
	}
	raw, _ := neutronPortRaw(ctx, client, item.ID)
	values, err := portSetValues(ctx, opts, client, item, raw)
	if err != nil {
		return err
	}
	if len(values) > 0 {
		if _, err := ports.Update(ctx, client, item.ID, portUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "no-tag") || len(flagValues(opts, "tag")) > 0 {
		_, err := setNeutronResourceTags(ctx, client, "ports", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
	}
	return nil
}

func portSetValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, item *ports.Port, raw map[string]any) (map[string]any, error) {
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if err := portApplyMutableValues(ctx, opts, client, values, false, raw, item); err != nil {
		return nil, err
	}
	fixedIPs, err := portFixedIPMaps(ctx, client, flagValues(opts, "fixed-ip"))
	if err != nil {
		return nil, err
	}
	if boolFlag(opts, "no-fixed-ip") {
		values["fixed_ips"] = []map[string]string{}
	}
	if len(fixedIPs) > 0 {
		merged := fixedIPs
		if !boolFlag(opts, "no-fixed-ip") {
			merged = append(portFixedIPMapsFromTyped(item.FixedIPs), fixedIPs...)
		}
		values["fixed_ips"] = merged
	}
	if boolFlag(opts, "no-security-group") {
		values["security_groups"] = []string{}
	}
	if securityGroups, err := portSecurityGroupIDs(ctx, client, flagValues(opts, "security-group")); err != nil {
		return nil, err
	} else if len(securityGroups) > 0 {
		merged := securityGroups
		if !boolFlag(opts, "no-security-group") {
			merged = append(append([]string{}, item.SecurityGroups...), securityGroups...)
		}
		values["security_groups"] = merged
	}
	if boolFlag(opts, "no-allowed-address") {
		values["allowed_address_pairs"] = []map[string]string{}
	}
	if allowed, err := portAllowedAddressMaps(flagValues(opts, "allowed-address")); err != nil {
		return nil, err
	} else if len(allowed) > 0 {
		merged := allowed
		if !boolFlag(opts, "no-allowed-address") {
			merged = append(portAllowedAddressMapsFromTyped(item.AllowedAddressPairs), allowed...)
		}
		values["allowed_address_pairs"] = merged
	}
	if dhcpOptions, err := portExtraDHCPOptions(flagValues(opts, "extra-dhcp-option")); err != nil {
		return nil, err
	} else if len(dhcpOptions) > 0 {
		values["extra_dhcp_opts"] = dhcpOptions
	}
	if flagChanged(opts, "data-plane-status") {
		status := flagValue(opts, "data-plane-status")
		if status != "ACTIVE" && status != "DOWN" {
			return nil, fmt.Errorf("invalid data plane status %q", status)
		}
		values["data_plane_status"] = status
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

func portApplyMutableValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, values map[string]any, create bool, raw map[string]any, item *ports.Port) error {
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if flagChanged(opts, "device") {
		values["device_id"] = flagValue(opts, "device")
	}
	if flagChanged(opts, "device-owner") {
		values["device_owner"] = flagValue(opts, "device-owner")
	}
	if flagChanged(opts, "mac-address") {
		values["mac_address"] = flagValue(opts, "mac-address")
	}
	if flagChanged(opts, "vnic-type") {
		values["binding:vnic_type"] = flagValue(opts, "vnic-type")
	}
	if flagChanged(opts, "host") {
		values["binding:host_id"] = flagValue(opts, "host")
	}
	if flagChanged(opts, "dns-domain") {
		values["dns_domain"] = flagValue(opts, "dns-domain")
	}
	if flagChanged(opts, "dns-name") {
		values["dns_name"] = flagValue(opts, "dns-name")
	}
	if adminState, err := networkBoolFlag(opts, "enable", "disable"); err != nil {
		return err
	} else if adminState != nil {
		values["admin_state_up"] = *adminState
	} else if create {
		values["admin_state_up"] = true
	}
	if portSecurity, err := networkBoolFlag(opts, "enable-port-security", "disable-port-security"); err != nil {
		return err
	} else if portSecurity != nil {
		values["port_security_enabled"] = *portSecurity
	}
	if uplink, err := networkBoolFlag(opts, "enable-uplink-status-propagation", "disable-uplink-status-propagation"); err != nil {
		return err
	} else if uplink != nil {
		values["propagate_uplink_status"] = *uplink
	}
	numaPolicy, err := portNumaPolicy(opts)
	if err != nil {
		return err
	}
	if numaPolicy != "" {
		values["numa_affinity_policy"] = numaPolicy
	}
	if flagChanged(opts, "device-profile") {
		values["device_profile"] = flagValue(opts, "device-profile")
	}
	if flagChanged(opts, "hardware-offload-type") {
		values["hardware_offload_type"] = flagValue(opts, "hardware-offload-type")
	}
	if boolFlag(opts, "trusted") && boolFlag(opts, "not-trusted") {
		return fmt.Errorf("argument --not-trusted: not allowed with argument --trusted")
	}
	if boolFlag(opts, "trusted") {
		values["trusted"] = true
	}
	if boolFlag(opts, "not-trusted") {
		values["trusted"] = false
	}
	if profileValues := flagValues(opts, "binding-profile"); len(profileValues) > 0 || boolFlag(opts, "no-binding-profile") {
		profile := map[string]any{}
		if !boolFlag(opts, "no-binding-profile") && !create {
			profile = mapAnyFromRaw(raw["binding:profile"])
			if len(profile) == 0 && item != nil {
				profile = map[string]any{}
			}
		}
		parsed, err := parseJSONKeyValueMap(profileValues, "binding-profile")
		if err != nil {
			return err
		}
		for key, value := range parsed {
			profile[key] = value
		}
		values["binding:profile"] = profile
	}
	if hints := flagValues(opts, "hint"); len(hints) > 0 {
		parsed, err := parsePortHints(hints)
		if err != nil {
			return err
		}
		values["hints"] = parsed
	}
	if qosPolicy := flagValue(opts, "qos-policy"); qosPolicy != "" {
		policy, err := findNetworkQoSPolicy(ctx, client, qosPolicy)
		if err != nil {
			return err
		}
		values["qos_policy_id"] = policy.ID
	}
	return nil
}

func portUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("port unset requires <port>")
	}
	if len(flagValues(opts, "tag")) > 0 && boolFlag(opts, "all-tag") {
		return fmt.Errorf("argument --all-tag: not allowed with argument --tag")
	}
	item, err := findPort(ctx, client, args[0])
	if err != nil {
		return err
	}
	raw, _ := neutronPortRaw(ctx, client, item.ID)
	values := map[string]any{}
	if fixedIPs, err := portFixedIPMaps(ctx, client, flagValues(opts, "fixed-ip")); err != nil {
		return err
	} else if len(fixedIPs) > 0 {
		remaining, err := removePortMapValues(portFixedIPMapsFromTyped(item.FixedIPs), fixedIPs, "fixed-ip")
		if err != nil {
			return err
		}
		values["fixed_ips"] = remaining
	}
	if keys := flagValues(opts, "binding-profile"); len(keys) > 0 {
		profile := mapAnyFromRaw(raw["binding:profile"])
		for _, key := range keys {
			if _, ok := profile[key]; !ok {
				return fmt.Errorf("Port does not contain binding-profile %s", key)
			}
			delete(profile, key)
		}
		values["binding:profile"] = profile
	}
	if securityGroups, err := portSecurityGroupIDs(ctx, client, flagValues(opts, "security-group")); err != nil {
		return err
	} else if len(securityGroups) > 0 {
		remaining, err := removePortStringValues(item.SecurityGroups, securityGroups, "security group")
		if err != nil {
			return err
		}
		values["security_groups"] = remaining
	}
	if allowed, err := portAllowedAddressMaps(flagValues(opts, "allowed-address")); err != nil {
		return err
	} else if len(allowed) > 0 {
		remaining, err := removePortMapValues(portAllowedAddressMapsFromTyped(item.AllowedAddressPairs), allowed, "allowed-address-pair")
		if err != nil {
			return err
		}
		values["allowed_address_pairs"] = remaining
	}
	if boolFlag(opts, "qos-policy") {
		values["qos_policy_id"] = nil
	}
	if boolFlag(opts, "data-plane-status") {
		values["data_plane_status"] = nil
	}
	if boolFlag(opts, "numa-policy") {
		values["numa_affinity_policy"] = nil
	}
	if boolFlag(opts, "host") {
		values["binding:host_id"] = nil
	}
	if boolFlag(opts, "hints") {
		values["hints"] = nil
	}
	if boolFlag(opts, "device") {
		values["device_id"] = ""
	}
	if boolFlag(opts, "device-owner") {
		values["device_owner"] = ""
	}
	extra, err := parseUnsetExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	for key, value := range extra {
		values[key] = value
	}
	if len(values) > 0 {
		if _, err := ports.Update(ctx, client, item.ID, portUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	_, err = unsetNeutronResourceTags(ctx, client, "ports", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "all-tag"))
	return err
}

func portNumaPolicy(opts *Options) (string, error) {
	policies := []struct {
		flag  string
		value string
	}{
		{"numa-policy-required", "required"},
		{"numa-policy-preferred", "preferred"},
		{"numa-policy-socket", "socket"},
		{"numa-policy-legacy", "legacy"},
	}
	selected := ""
	for _, policy := range policies {
		if !boolFlag(opts, policy.flag) {
			continue
		}
		if selected != "" {
			return "", fmt.Errorf("NUMA policy options are mutually exclusive")
		}
		selected = policy.value
	}
	return selected, nil
}

func portFixedIPMaps(ctx context.Context, client *gophercloud.ServiceClient, values []string) ([]map[string]string, error) {
	entries, err := parseKeyValueEntries(values, "fixed-ip")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if subnetNameOrID := entry["subnet"]; subnetNameOrID != "" {
			subnet, err := findSubnet(ctx, client, subnetNameOrID)
			if err != nil {
				return nil, err
			}
			entry["subnet_id"] = subnet.ID
			delete(entry, "subnet")
		}
		if ipAddress := entry["ip-address"]; ipAddress != "" {
			entry["ip_address"] = ipAddress
			delete(entry, "ip-address")
		}
	}
	return entries, nil
}

func portFixedIPMapsFromTyped(values []ports.IP) []map[string]string {
	items := make([]map[string]string, 0, len(values))
	for _, value := range values {
		entry := map[string]string{}
		if value.SubnetID != "" {
			entry["subnet_id"] = value.SubnetID
		}
		if value.IPAddress != "" {
			entry["ip_address"] = value.IPAddress
		}
		if len(entry) > 0 {
			items = append(items, entry)
		}
	}
	return items
}

func portSecurityGroupIDs(ctx context.Context, client *gophercloud.ServiceClient, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(values))
	for _, value := range values {
		group, err := findSecurityGroup(ctx, client, value)
		if err != nil {
			return nil, err
		}
		ids = append(ids, group.ID)
	}
	return ids, nil
}

func portAllowedAddressMaps(values []string) ([]map[string]string, error) {
	entries, err := parseKeyValueEntries(values, "allowed-address", "ip-address")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		entry["ip_address"] = entry["ip-address"]
		delete(entry, "ip-address")
		if macAddress := entry["mac-address"]; macAddress != "" {
			entry["mac_address"] = macAddress
			delete(entry, "mac-address")
		}
	}
	return entries, nil
}

func portAllowedAddressMapsFromTyped(values []ports.AddressPair) []map[string]string {
	items := make([]map[string]string, 0, len(values))
	for _, value := range values {
		entry := map[string]string{}
		if value.IPAddress != "" {
			entry["ip_address"] = value.IPAddress
		}
		if value.MACAddress != "" {
			entry["mac_address"] = value.MACAddress
		}
		if len(entry) > 0 {
			items = append(items, entry)
		}
	}
	return items
}

func portExtraDHCPOptions(values []string) ([]map[string]string, error) {
	entries, err := parseKeyValueEntries(values, "extra-dhcp-option", "name")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		entry["opt_name"] = entry["name"]
		delete(entry, "name")
		if value := entry["value"]; value != "" {
			entry["opt_value"] = value
			delete(entry, "value")
		}
		if version := entry["ip-version"]; version != "" {
			entry["ip_version"] = version
			delete(entry, "ip-version")
		}
	}
	return entries, nil
}

func parseJSONKeyValueMap(values []string, option string) (map[string]any, error) {
	result := map[string]any{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if strings.HasPrefix(trimmed, "{") {
			decoded := map[string]any{}
			if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
				return nil, fmt.Errorf("invalid %s JSON %q: %w", option, value, err)
			}
			for key, decodedValue := range decoded {
				result[key] = decodedValue
			}
			continue
		}
		key, raw, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid %s %q, expected <key>=<value> or JSON", option, value)
		}
		result[strings.TrimSpace(key)] = raw
	}
	return result, nil
}

func parsePortHints(values []string) (map[string]any, error) {
	hints, err := parseJSONKeyValueMap(values, "hint")
	if err != nil {
		return nil, err
	}
	if len(hints) == 0 {
		return nil, nil
	}
	if value, ok := hints["ovs-tx-steering"].(string); ok && len(hints) == 1 {
		if value != "thread" && value != "hash" {
			return nil, fmt.Errorf("invalid value to --hint, see --help for valid values")
		}
		return map[string]any{"openvswitch": map[string]any{"other_config": map[string]any{"tx-steering": value}}}, nil
	}
	if openvswitch, ok := hints["openvswitch"].(map[string]any); ok && len(hints) == 1 {
		otherConfig, _ := openvswitch["other_config"].(map[string]any)
		value, _ := otherConfig["tx-steering"].(string)
		if value == "thread" || value == "hash" {
			return hints, nil
		}
	}
	return nil, fmt.Errorf("invalid value to --hint, see --help for valid values")
}

func mapAnyFromRaw(value any) map[string]any {
	result := map[string]any{}
	if typed, ok := value.(map[string]any); ok {
		for key, mapValue := range typed {
			result[key] = mapValue
		}
	}
	return result
}

func removePortStringValues(existing []string, values []string, option string) ([]string, error) {
	remaining := append([]string{}, existing...)
	for _, value := range values {
		found := false
		for index, existingValue := range remaining {
			if existingValue == value {
				remaining = append(remaining[:index], remaining[index+1:]...)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Port does not contain %s %s", option, value)
		}
	}
	return remaining, nil
}

func removePortMapValues(existing []map[string]string, values []map[string]string, option string) ([]map[string]string, error) {
	remaining := append([]map[string]string{}, existing...)
	for _, value := range values {
		found := false
		for index, existingValue := range remaining {
			if stringMapEqual(existingValue, value) {
				remaining = append(remaining[:index], remaining[index+1:]...)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Port does not contain %s %v", option, value)
		}
	}
	return remaining, nil
}

func neutronPortRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		Port map[string]any `json:"port"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("ports", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.Port, nil
}

func portRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		Port map[string]any `json:"port"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.Port == nil {
		return nil, false
	}
	return wrapper.Port, true
}

func portRawFields(raw map[string]any) []outputField {
	return []outputField{
		{"admin_state_up", adminStateValue(raw["admin_state_up"])},
		{"allowed_address_pairs", blankEmptyListValue(raw["allowed_address_pairs"])},
		{"binding_host_id", raw["binding:host_id"]},
		{"binding_profile", blankEmptyMapValue(raw["binding:profile"])},
		{"binding_vif_details", blankEmptyMapValue(raw["binding:vif_details"])},
		{"binding_vif_type", raw["binding:vif_type"]},
		{"binding_vnic_type", raw["binding:vnic_type"]},
		{"created_at", raw["created_at"]},
		{"data_plane_status", raw["data_plane_status"]},
		{"description", raw["description"]},
		{"device_id", raw["device_id"]},
		{"device_owner", raw["device_owner"]},
		{"device_profile", raw["device_profile"]},
		{"dns_assignment", blankEmptyListValue(raw["dns_assignment"])},
		{"dns_domain", raw["dns_domain"]},
		{"dns_name", raw["dns_name"]},
		{"extra_dhcp_opts", blankEmptyListValue(raw["extra_dhcp_opts"])},
		{"fixed_ips", portFixedIPsTableValue(raw["fixed_ips"])},
		{"hardware_offload_type", raw["hardware_offload_type"]},
		{"hints", blankEmptyMapAsStringValue(raw["hints"])},
		{"id", raw["id"]},
		{"ip_allocation", raw["ip_allocation"]},
		{"mac_address", raw["mac_address"]},
		{"name", raw["name"]},
		{"network_id", raw["network_id"]},
		{"numa_affinity_policy", raw["numa_affinity_policy"]},
		{"port_security_enabled", raw["port_security_enabled"]},
		{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		{"propagate_uplink_status", raw["propagate_uplink_status"]},
		{"resource_request", raw["resource_request"]},
		{"revision_number", rawNumber(raw["revision_number"])},
		{"qos_network_policy_id", raw["qos_network_policy_id"]},
		{"qos_policy_id", raw["qos_policy_id"]},
		{"security_group_ids", blankEmptyListValue(raw["security_groups"])},
		{"status", raw["status"]},
		{"tags", blankEmptyListValue(raw["tags"])},
		{"trunk_details", raw["trunk_details"]},
		{"trusted", raw["trusted"]},
		{"updated_at", raw["updated_at"]},
	}
}

func portFields(item *ports.Port) []outputField {
	return []outputField{
		{"admin_state_up", item.AdminStateUp},
		{"allowed_address_pairs", item.AllowedAddressPairs},
		{"binding_host_id", nil},
		{"binding_profile", nil},
		{"binding_vif_details", nil},
		{"binding_vif_type", nil},
		{"binding_vnic_type", nil},
		{"created_at", oscTime(item.CreatedAt)},
		{"data_plane_status", nil},
		{"description", item.Description},
		{"device_id", item.DeviceID},
		{"device_owner", item.DeviceOwner},
		{"device_profile", nil},
		{"dns_assignment", nil},
		{"dns_domain", nil},
		{"dns_name", nil},
		{"extra_dhcp_opts", nil},
		{"fixed_ips", item.FixedIPs},
		{"hardware_offload_type", nil},
		{"hints", nil},
		{"id", item.ID},
		{"ip_allocation", nil},
		{"mac_address", item.MACAddress},
		{"name", item.Name},
		{"network_id", item.NetworkID},
		{"numa_affinity_policy", nil},
		{"port_security_enabled", nil},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"propagate_uplink_status", item.PropagateUplinkStatus},
		{"resource_request", nil},
		{"revision_number", item.RevisionNumber},
		{"qos_network_policy_id", nil},
		{"qos_policy_id", nil},
		{"security_group_ids", item.SecurityGroups},
		{"status", item.Status},
		{"tags", item.Tags},
		{"trunk_details", nil},
		{"trusted", nil},
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
			"Fixed IP Addresses": portFixedIPsValue(item.FixedIPs),
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
	if raw, err := neutronPortRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, prettyNetworkIPOutputFields(opts, portRawFields(raw)))
	}
	return renderShowOutput(stdout, opts, prettyNetworkIPOutputFields(opts, portFields(item)))
}

type floatingIPCreateOpts struct {
	Values map[string]any
}

func (opts floatingIPCreateOpts) ToFloatingIPCreateMap() (map[string]any, error) {
	return map[string]any{"floatingip": opts.Values}, nil
}

type floatingIPUpdateOpts struct {
	Values map[string]any
}

func (opts floatingIPUpdateOpts) ToFloatingIPUpdateMap() (map[string]any, error) {
	return map[string]any{"floatingip": opts.Values}, nil
}

func floatingIPCreate(ctx context.Context, stdout io.Writer, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("floating ip create requires <network>")
	}
	if boolFlag(opts, "no-tag") && len(flagValues(opts, "tag")) > 0 {
		return fmt.Errorf("argument --no-tag: not allowed with argument --tag")
	}
	values, err := floatingIPCreateValues(ctx, opts, networkClient, identityClient, args[0])
	if err != nil {
		return err
	}
	result := floatingips.Create(ctx, networkClient, floatingIPCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	tags := flagValues(opts, "tag")
	if boolFlag(opts, "no-tag") || len(tags) > 0 {
		target, err := setNeutronResourceTags(ctx, networkClient, "floatingips", item.ID, item.Tags, tags, boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
		item.Tags = target
	}
	if raw, ok := floatingIPRawFromBody(result.Body); ok {
		if boolFlag(opts, "no-tag") || len(tags) > 0 {
			raw["tags"] = item.Tags
		}
		return renderShowOutput(stdout, opts, floatingIPRawFields(raw))
	}
	return renderShowOutput(stdout, opts, floatingIPFields(item))
}

func floatingIPCreateValues(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient, identityClient *gophercloud.ServiceClient, networkNameOrID string) (map[string]any, error) {
	network, err := findNetwork(ctx, networkClient, networkNameOrID)
	if err != nil {
		return nil, err
	}
	values := map[string]any{"floating_network_id": network.ID}
	if subnetNameOrID := flagValue(opts, "subnet"); subnetNameOrID != "" {
		subnet, err := findSubnet(ctx, networkClient, subnetNameOrID)
		if err != nil {
			return nil, err
		}
		values["subnet_id"] = subnet.ID
	}
	if portNameOrID := flagValue(opts, "port"); portNameOrID != "" {
		port, err := findPort(ctx, networkClient, portNameOrID)
		if err != nil {
			return nil, err
		}
		values["port_id"] = port.ID
	}
	if flagChanged(opts, "floating-ip-address") {
		values["floating_ip_address"] = flagValue(opts, "floating-ip-address")
	}
	if flagChanged(opts, "fixed-ip-address") {
		values["fixed_ip_address"] = flagValue(opts, "fixed-ip-address")
	}
	if qosPolicy := flagValue(opts, "qos-policy"); qosPolicy != "" {
		policy, err := findNetworkQoSPolicy(ctx, networkClient, qosPolicy)
		if err != nil {
			return nil, err
		}
		values["qos_policy_id"] = policy.ID
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if projectNameOrID := flagValue(opts, "project"); projectNameOrID != "" {
		project, err := findProjectWithDomain(ctx, identityClient, projectNameOrID, flagValue(opts, "project-domain"))
		if err != nil {
			return nil, err
		}
		values["project_id"] = project.ID
	}
	if flagChanged(opts, "dns-domain") {
		values["dns_domain"] = flagValue(opts, "dns-domain")
	}
	if flagChanged(opts, "dns-name") {
		values["dns_name"] = flagValue(opts, "dns-name")
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

func floatingIPDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("floating ip delete requires <floating-ip> [<floating-ip> ...]")
	}
	failures := 0
	for _, floatingIPArg := range args {
		item, err := findFloatingIP(ctx, client, floatingIPArg)
		if err != nil {
			failures++
			continue
		}
		if err := floatingips.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d floating IPs failed to delete.", failures, len(args))
	}
	return nil
}

func floatingIPSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("floating ip set requires <floating-ip>")
	}
	if flagChanged(opts, "qos-policy") && boolFlag(opts, "no-qos-policy") {
		return fmt.Errorf("argument --no-qos-policy: not allowed with argument --qos-policy")
	}
	item, err := findFloatingIP(ctx, client, args[0])
	if err != nil {
		return err
	}
	values, err := floatingIPSetValues(ctx, opts, client)
	if err != nil {
		return err
	}
	if len(values) > 0 {
		if _, err := floatingips.Update(ctx, client, item.ID, floatingIPUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	if boolFlag(opts, "no-tag") || len(flagValues(opts, "tag")) > 0 {
		_, err := setNeutronResourceTags(ctx, client, "floatingips", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "no-tag"))
		if err != nil {
			return err
		}
	}
	return nil
}

func floatingIPSetValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient) (map[string]any, error) {
	values := map[string]any{}
	if portNameOrID := flagValue(opts, "port"); portNameOrID != "" {
		port, err := findPort(ctx, client, portNameOrID)
		if err != nil {
			return nil, err
		}
		values["port_id"] = port.ID
	}
	if flagChanged(opts, "fixed-ip-address") {
		values["fixed_ip_address"] = flagValue(opts, "fixed-ip-address")
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if qosPolicy := flagValue(opts, "qos-policy"); qosPolicy != "" {
		policy, err := findNetworkQoSPolicy(ctx, client, qosPolicy)
		if err != nil {
			return nil, err
		}
		values["qos_policy_id"] = policy.ID
	}
	if boolFlag(opts, "no-qos-policy") {
		values["qos_policy_id"] = nil
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

func floatingIPUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("floating ip unset requires <floating-ip>")
	}
	if len(flagValues(opts, "tag")) > 0 && boolFlag(opts, "all-tag") {
		return fmt.Errorf("argument --all-tag: not allowed with argument --tag")
	}
	item, err := findFloatingIP(ctx, client, args[0])
	if err != nil {
		return err
	}
	values := map[string]any{}
	if boolFlag(opts, "port") {
		values["port_id"] = nil
	}
	if boolFlag(opts, "qos-policy") {
		values["qos_policy_id"] = nil
	}
	extra, err := parseUnsetExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	for key, value := range extra {
		values[key] = value
	}
	if len(values) > 0 {
		if _, err := floatingips.Update(ctx, client, item.ID, floatingIPUpdateOpts{Values: values}).Extract(); err != nil {
			return err
		}
	}
	_, err = unsetNeutronResourceTags(ctx, client, "floatingips", item.ID, item.Tags, flagValues(opts, "tag"), boolFlag(opts, "all-tag"))
	return err
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
			"Fixed IP Address":    nilIfEmpty(item.FixedIP),
			"Port":                nilIfEmpty(item.PortID),
			"Floating Network":    item.FloatingNetworkID,
			"Project":             firstNonEmpty(item.ProjectID, item.TenantID),
		}
		if boolFlag(opts, "long") {
			row["Router"] = item.RouterID
			row["Status"] = item.Status
			row["Tags"] = item.Tags
		}
		rows = append(rows, row)
	}
	columns := []string{"ID", "Floating IP Address", "Fixed IP Address", "Port", "Floating Network", "Project"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Router", "Status", "Tags")
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
	if raw, err := neutronFloatingIPRaw(ctx, client, item.ID); err == nil {
		return renderShowOutput(stdout, opts, floatingIPRawFields(raw))
	}
	return renderShowOutput(stdout, opts, floatingIPFields(item))
}

func neutronFloatingIPRaw(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	var body struct {
		FloatingIP map[string]any `json:"floatingip"`
	}
	resp, err := client.Get(ctx, client.ServiceURL("floatingips", url.PathEscape(id)), &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.FloatingIP, nil
}

func floatingIPRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		FloatingIP map[string]any `json:"floatingip"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.FloatingIP == nil {
		return nil, false
	}
	return wrapper.FloatingIP, true
}

func floatingIPRawFields(raw map[string]any) []outputField {
	return []outputField{
		{"created_at", raw["created_at"]},
		{"description", raw["description"]},
		{"dns_domain", raw["dns_domain"]},
		{"dns_name", raw["dns_name"]},
		{"fixed_ip_address", raw["fixed_ip_address"]},
		{"floating_ip_address", raw["floating_ip_address"]},
		{"floating_network_id", raw["floating_network_id"]},
		{"id", raw["id"]},
		{"name", raw["floating_ip_address"]},
		{"port_details", firstNonNil(raw["port_details"], map[string]any{})},
		{"port_id", raw["port_id"]},
		{"project_id", firstPresent(raw, "project_id", "tenant_id")},
		{"qos_policy_id", raw["qos_policy_id"]},
		{"revision_number", rawNumber(raw["revision_number"])},
		{"router_id", raw["router_id"]},
		{"status", raw["status"]},
		{"subnet_id", raw["subnet_id"]},
		{"tags", raw["tags"]},
		{"updated_at", raw["updated_at"]},
	}
}

func floatingIPFields(item *floatingips.FloatingIP) []outputField {
	return []outputField{
		{"created_at", oscTime(item.CreatedAt)},
		{"description", item.Description},
		{"dns_domain", nil},
		{"dns_name", nil},
		{"fixed_ip_address", nilIfEmpty(item.FixedIP)},
		{"floating_ip_address", item.FloatingIP},
		{"floating_network_id", item.FloatingNetworkID},
		{"id", item.ID},
		{"name", item.FloatingIP},
		{"port_details", map[string]any{}},
		{"port_id", nilIfEmpty(item.PortID)},
		{"project_id", firstNonEmpty(item.ProjectID, item.TenantID)},
		{"qos_policy_id", nil},
		{"revision_number", item.RevisionNumber},
		{"router_id", nilIfEmpty(item.RouterID)},
		{"status", item.Status},
		{"subnet_id", nil},
		{"tags", item.Tags},
		{"updated_at", oscTime(item.UpdatedAt)},
	}
}

func floatingIPPoolList() error {
	return fmt.Errorf("Floating ip pool operations are only available for Compute v2 network.")
}

type floatingIPPortForwardingCreateOpts struct {
	Values map[string]any
}

func (opts floatingIPPortForwardingCreateOpts) ToPortForwardingCreateMap() (map[string]any, error) {
	return map[string]any{"port_forwarding": opts.Values}, nil
}

type floatingIPPortForwardingUpdateOpts struct {
	Values map[string]any
}

func (opts floatingIPPortForwardingUpdateOpts) ToPortForwardingUpdateMap() (map[string]any, error) {
	return map[string]any{"port_forwarding": opts.Values}, nil
}

func floatingIPPortForwardingCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("floating ip port forwarding create requires <floating-ip>")
	}
	values, err := floatingIPPortForwardingCreateValues(ctx, opts, client)
	if err != nil {
		return err
	}
	floatingIP, err := findFloatingIP(ctx, client, args[0])
	if err != nil {
		return err
	}
	result := portforwarding.Create(ctx, client, floatingIP.ID, floatingIPPortForwardingCreateOpts{Values: values})
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if raw, ok := floatingIPPortForwardingRawFromBody(result.Body); ok {
		return renderShowOutput(stdout, opts, floatingIPPortForwardingRawFields(raw))
	}
	return renderShowOutput(stdout, opts, floatingIPPortForwardingFields(item))
}

func floatingIPPortForwardingCreateValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient) (map[string]any, error) {
	required := []string{"internal-ip-address", "port", "internal-protocol-port", "external-protocol-port", "protocol"}
	for _, flagName := range required {
		if !flagChanged(opts, flagName) || flagValue(opts, flagName) == "" {
			return nil, fmt.Errorf("argument --%s is required", flagName)
		}
	}
	values := map[string]any{}
	port, err := findPort(ctx, client, flagValue(opts, "port"))
	if err != nil {
		return nil, err
	}
	values["internal_port_id"] = port.ID
	values["internal_ip_address"] = flagValue(opts, "internal-ip-address")
	values["protocol"] = flagValue(opts, "protocol")
	if err := floatingIPPortForwardingApplyPortRanges(values, flagValue(opts, "internal-protocol-port"), flagValue(opts, "external-protocol-port")); err != nil {
		return nil, err
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
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

func floatingIPPortForwardingDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("floating ip port forwarding delete requires <floating-ip> <port-forwarding-id> [<port-forwarding-id> ...]")
	}
	floatingIP, err := findFloatingIP(ctx, client, args[0])
	if err != nil {
		return err
	}
	failures := 0
	for _, portForwardingID := range args[1:] {
		if err := portforwarding.Delete(ctx, client, floatingIP.ID, portForwardingID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d Port forwarding failed to delete.", failures, len(args)-1)
	}
	return nil
}

func floatingIPPortForwardingList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("floating ip port forwarding list requires <floating-ip>")
	}
	floatingIP, err := findFloatingIP(ctx, client, args[0])
	if err != nil {
		return err
	}
	values := map[string]any{}
	if portNameOrID := flagValue(opts, "port"); portNameOrID != "" {
		port, err := findPort(ctx, client, portNameOrID)
		if err != nil {
			return err
		}
		values["internal_port_id"] = port.ID
	}
	if externalPort := flagValue(opts, "external-protocol-port"); externalPort != "" {
		if strings.Contains(externalPort, ":") {
			if _, err := parsePortRange(externalPort, "external-protocol-port"); err != nil {
				return err
			}
			values["external_port_range"] = externalPort
		} else {
			parsed, err := parseProtocolPort(externalPort, "external-protocol-port")
			if err != nil {
				return err
			}
			values["external_port"] = parsed
		}
	}
	if protocol := flagValue(opts, "protocol"); protocol != "" {
		values["protocol"] = protocol
	}
	items, err := neutronFloatingIPPortForwardingRawList(ctx, client, floatingIP.ID, values)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"ID":                  item["id"],
			"Internal Port ID":    item["internal_port_id"],
			"Internal IP Address": item["internal_ip_address"],
			"Internal Port":       rawNumber(item["internal_port"]),
			"Internal Port Range": item["internal_port_range"],
			"External Port":       rawNumber(item["external_port"]),
			"External Port Range": item["external_port_range"],
			"Protocol":            item["protocol"],
			"Description":         item["description"],
		})
	}
	columns := []string{"ID", "Internal Port ID", "Internal IP Address", "Internal Port", "Internal Port Range", "External Port", "External Port Range", "Protocol", "Description"}
	return renderListOutput(stdout, opts, columns, rows)
}

func floatingIPPortForwardingSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("floating ip port forwarding set requires <floating-ip> <port-forwarding-id>")
	}
	floatingIP, err := findFloatingIP(ctx, client, args[0])
	if err != nil {
		return err
	}
	values, err := floatingIPPortForwardingSetValues(ctx, opts, client)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	_, err = portforwarding.Update(ctx, client, floatingIP.ID, args[1], floatingIPPortForwardingUpdateOpts{Values: values}).Extract()
	return err
}

func floatingIPPortForwardingSetValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient) (map[string]any, error) {
	values := map[string]any{}
	if portNameOrID := flagValue(opts, "port"); portNameOrID != "" {
		port, err := findPort(ctx, client, portNameOrID)
		if err != nil {
			return nil, err
		}
		values["internal_port_id"] = port.ID
	}
	if flagChanged(opts, "internal-ip-address") {
		values["internal_ip_address"] = flagValue(opts, "internal-ip-address")
	}
	if flagChanged(opts, "internal-protocol-port") || flagChanged(opts, "external-protocol-port") {
		if err := floatingIPPortForwardingApplyPortRanges(values, flagValue(opts, "internal-protocol-port"), flagValue(opts, "external-protocol-port")); err != nil {
			return nil, err
		}
	}
	if flagChanged(opts, "protocol") {
		values["protocol"] = flagValue(opts, "protocol")
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
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

func floatingIPPortForwardingShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("floating ip port forwarding show requires <floating-ip> <port-forwarding-id>")
	}
	floatingIP, err := findFloatingIP(ctx, client, args[0])
	if err != nil {
		return err
	}
	result := portforwarding.Get(ctx, client, floatingIP.ID, args[1])
	item, err := result.Extract()
	if err != nil {
		return err
	}
	if raw, ok := floatingIPPortForwardingRawFromBody(result.Body); ok {
		return renderShowOutput(stdout, opts, floatingIPPortForwardingRawFields(raw))
	}
	return renderShowOutput(stdout, opts, floatingIPPortForwardingFields(item))
}

func floatingIPPortForwardingApplyPortRanges(values map[string]any, internalValue string, externalValue string) error {
	internalPorts, err := parseOptionalPortRange(internalValue, "internal-protocol-port")
	if err != nil {
		return err
	}
	externalPorts, err := parseOptionalPortRange(externalValue, "external-protocol-port")
	if err != nil {
		return err
	}
	if err := validateFloatingIPPortForwardingRanges(internalPorts, externalPorts); err != nil {
		return err
	}
	if internalValue != "" {
		if len(internalPorts) == 2 {
			values["internal_port_range"] = internalValue
		} else {
			values["internal_port"] = internalPorts[0]
		}
	}
	if externalValue != "" {
		if len(externalPorts) == 2 {
			values["external_port_range"] = externalValue
		} else {
			values["external_port"] = externalPorts[0]
		}
	}
	return nil
}

func validateFloatingIPPortForwardingRanges(internalPorts []int, externalPorts []int) error {
	internalDiff, err := portRangeDiff(internalPorts)
	if err != nil {
		return err
	}
	externalDiff, err := portRangeDiff(externalPorts)
	if err != nil {
		return err
	}
	if internalDiff != 0 && internalDiff != externalDiff {
		return fmt.Errorf("The relation between internal and external ports does not match the pattern 1:N and N:N")
	}
	return nil
}

func parseOptionalPortRange(value string, option string) ([]int, error) {
	if value == "" {
		return nil, nil
	}
	if strings.Contains(value, ":") {
		return parsePortRange(value, option)
	}
	parsed, err := parseProtocolPort(value, option)
	if err != nil {
		return nil, err
	}
	return []int{parsed}, nil
}

func parsePortRange(value string, option string) ([]int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid --%s %q", option, value)
	}
	start, err := parseProtocolPort(parts[0], option)
	if err != nil {
		return nil, err
	}
	end, err := parseProtocolPort(parts[1], option)
	if err != nil {
		return nil, err
	}
	return []int{start, end}, nil
}

func parseProtocolPort(value string, option string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid --%s %q", option, value)
	}
	if parsed <= 0 || parsed > 65535 {
		return 0, fmt.Errorf("The port number range is <1-65535>")
	}
	return parsed, nil
}

func portRangeDiff(ports []int) (int, error) {
	if len(ports) == 0 {
		return 0, nil
	}
	diff := ports[len(ports)-1] - ports[0]
	if diff < 0 {
		return 0, fmt.Errorf("The last number in port range must be greater or equal to the first")
	}
	return diff, nil
}

func nilIfZero(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func floatingIPPortForwardingRawFromBody(body any) (map[string]any, bool) {
	var wrapper struct {
		PortForwarding map[string]any `json:"port_forwarding"`
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, false
	}
	if err := json.Unmarshal(encoded, &wrapper); err != nil {
		return nil, false
	}
	if wrapper.PortForwarding == nil {
		return nil, false
	}
	return wrapper.PortForwarding, true
}

func neutronFloatingIPPortForwardingRawList(ctx context.Context, client *gophercloud.ServiceClient, floatingIPID string, values map[string]any) ([]map[string]any, error) {
	var body struct {
		PortForwardings []map[string]any `json:"port_forwardings"`
	}
	target := client.ServiceURL("floatingips", url.PathEscape(floatingIPID), "port_forwardings") + floatingIPPortForwardingQuery(values)
	resp, err := client.Get(ctx, target, &body, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return body.PortForwardings, nil
}

func floatingIPPortForwardingQuery(values map[string]any) string {
	query := url.Values{}
	for key, value := range values {
		if value == nil {
			continue
		}
		query.Set(key, fmt.Sprint(value))
	}
	encoded := query.Encode()
	if encoded == "" {
		return ""
	}
	return "?" + encoded
}

func floatingIPPortForwardingRawFields(raw map[string]any) []outputField {
	known := map[string]bool{
		"description":         true,
		"external_port":       true,
		"external_port_range": true,
		"id":                  true,
		"internal_ip_address": true,
		"internal_port":       true,
		"internal_port_id":    true,
		"internal_port_range": true,
		"location":            true,
		"protocol":            true,
		"tenant_id":           true,
	}
	fields := []outputField{
		{"description", raw["description"]},
		{"external_port", rawNumber(raw["external_port"])},
		{"external_port_range", raw["external_port_range"]},
		{"id", raw["id"]},
		{"internal_ip_address", raw["internal_ip_address"]},
		{"internal_port", rawNumber(raw["internal_port"])},
		{"internal_port_id", raw["internal_port_id"]},
		{"internal_port_range", raw["internal_port_range"]},
		{"protocol", raw["protocol"]},
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

func floatingIPPortForwardingFields(item *portforwarding.PortForwarding) []outputField {
	return []outputField{
		{"description", item.Description},
		{"external_port", nilIfZero(item.ExternalPort)},
		{"external_port_range", nil},
		{"id", item.ID},
		{"internal_ip_address", item.InternalIPAddress},
		{"internal_port", nilIfZero(item.InternalPort)},
		{"internal_port_id", item.InternalPortID},
		{"internal_port_range", nil},
		{"protocol", item.Protocol},
	}
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
		{"subnet_ip_availability", ipAvailabilitySubnetsValue(item.SubnetIPAvailabilities)},
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

func ipAvailabilitySubnetsValue(items []networkipavailabilities.SubnetIPAvailability) tableValue {
	values := ipAvailabilitySubnets(items)
	lines := make([]string, 0, len(values))
	for _, item := range values {
		lines = append(lines, fmt.Sprintf(
			"cidr='%s', ip_version='%s', subnet_id='%s', subnet_name='%s', total_ips='%s', used_ips='%s'",
			item.CIDR,
			valueString(item.IPVersion),
			item.SubnetID,
			item.SubnetName,
			valueString(item.TotalIPs),
			valueString(item.UsedIPs),
		))
	}
	return tableValue{Value: values, Table: strings.Join(lines, "\n"), Pretty: values}
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
	accessProjectIDs, err := volumeTypeAccessProjectIDs(ctx, client, item)
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"access_project_ids", accessProjectIDs},
		{"description", item.Description},
		{"id", item.ID},
		{"is_public", volumeTypeIsPublic(*item)},
		{"name", item.Name},
		{"properties", volumeTypePropertiesValue(item.ExtraSpecs)},
		{"qos_specs_id", nilIfEmpty(item.QosSpecID)},
	})
}

func volumeTypeCreate(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume type create requires <name>")
	}
	if flagValue(opts, "project") != "" && !boolFlag(opts, "private") {
		return fmt.Errorf("--project is only allowed with --private")
	}
	properties, err := volumeTypeProperties(opts)
	if err != nil {
		return err
	}
	createOpts := volumetypes.CreateOpts{
		Name:        args[0],
		Description: flagValue(opts, "description"),
		ExtraSpecs:  properties,
	}
	if boolFlag(opts, "public") {
		createOpts.IsPublic = valueBoolPtr(true)
	} else if boolFlag(opts, "private") {
		createOpts.IsPublic = valueBoolPtr(false)
	}
	item, err := volumetypes.Create(ctx, client, createOpts).Extract()
	if err != nil {
		return err
	}
	if projectValue := flagValue(opts, "project"); projectValue != "" {
		projectID, err := volumeTypeProjectID(ctx, opts, clients, projectValue)
		if err != nil {
			return err
		}
		if err := volumetypes.AddAccess(ctx, client, item.ID, volumetypes.AddAccessOpts{Project: projectID}).ExtractErr(); err != nil {
			return err
		}
	}
	encryption, err := volumeTypeCreateEncryption(ctx, opts, client, item.ID)
	if err != nil {
		return err
	}
	fields := volumeTypeFields(item)
	if encryption != nil {
		fields = append(fields, outputField{"encryption", volumeTypeEncryptionMap(encryption)})
	}
	return renderShowOutput(stdout, opts, fields)
}

func volumeTypeDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume type delete requires <volume-type>")
	}
	failures := 0
	for _, value := range args {
		item, err := findVolumeType(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if err := volumetypes.Delete(ctx, client, item.ID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d volume types failed to delete.", failures, len(args))
	}
	return nil
}

func volumeTypeSet(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume type set requires <volume-type>")
	}
	item, err := findVolumeType(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := volumetypes.UpdateOpts{}
	needsUpdate := false
	if flagChanged(opts, "name") {
		update.Name = valueStringPtr(flagValue(opts, "name"))
		needsUpdate = true
	}
	if flagChanged(opts, "description") {
		update.Description = valueStringPtr(flagValue(opts, "description"))
		needsUpdate = true
	}
	if boolFlag(opts, "public") {
		update.IsPublic = valueBoolPtr(true)
		needsUpdate = true
	} else if boolFlag(opts, "private") {
		update.IsPublic = valueBoolPtr(false)
		needsUpdate = true
	}
	if needsUpdate {
		if _, err := volumetypes.Update(ctx, client, item.ID, update).Extract(); err != nil {
			return err
		}
	}
	properties, err := volumeTypeProperties(opts)
	if err != nil {
		return err
	}
	if len(properties) > 0 {
		if _, err := volumetypes.CreateExtraSpecs(ctx, client, item.ID, volumetypes.ExtraSpecsOpts(properties)).Extract(); err != nil {
			return err
		}
	}
	if projectValue := flagValue(opts, "project"); projectValue != "" {
		projectID, err := volumeTypeProjectID(ctx, opts, clients, projectValue)
		if err != nil {
			return err
		}
		if err := volumetypes.AddAccess(ctx, client, item.ID, volumetypes.AddAccessOpts{Project: projectID}).ExtractErr(); err != nil {
			return err
		}
	}
	if volumeTypeEncryptionFlagsSet(opts) {
		if err := volumeTypeSetEncryption(ctx, opts, client, item.ID); err != nil {
			return err
		}
	}
	return nil
}

func volumeTypeUnset(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume type unset requires <volume-type>")
	}
	item, err := findVolumeType(ctx, client, args[0])
	if err != nil {
		return err
	}
	failures := 0
	for _, key := range flagValues(opts, "property") {
		if err := volumetypes.DeleteExtraSpec(ctx, client, item.ID, key).ExtractErr(); err != nil {
			failures++
		}
	}
	if projectValue := flagValue(opts, "project"); projectValue != "" {
		projectID, err := volumeTypeProjectID(ctx, opts, clients, projectValue)
		if err != nil {
			failures++
		} else if err := volumetypes.RemoveAccess(ctx, client, item.ID, volumetypes.RemoveAccessOpts{Project: projectID}).ExtractErr(); err != nil {
			failures++
		}
	}
	if boolFlag(opts, "encryption-type") {
		encryption, err := volumetypes.GetEncryption(ctx, client, item.ID).Extract()
		if err != nil {
			failures++
		} else if err := volumetypes.DeleteEncryption(ctx, client, item.ID, encryption.EncryptionID).ExtractErr(); err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("Command Failed: One or more of the operations failed")
	}
	return nil
}

func volumeTypeFields(item *volumetypes.VolumeType) []outputField {
	return []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"description", item.Description},
		{"is_public", volumeTypeIsPublic(*item)},
		{"properties", item.ExtraSpecs},
		{"qos_specs_id", item.QosSpecID},
	}
}

func volumeTypeProperties(opts *Options) (map[string]string, error) {
	properties, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return nil, err
	}
	if properties == nil {
		properties = map[string]string{}
	}
	if boolFlag(opts, "multiattach") {
		properties["multiattach"] = "<is> True"
	}
	if boolFlag(opts, "cacheable") {
		properties["cacheable"] = "<is> True"
	}
	if boolFlag(opts, "replicated") {
		properties["replication_enabled"] = "<is> True"
	}
	if availabilityZones := flagValues(opts, "availability-zone"); len(availabilityZones) > 0 {
		properties["RESKEY:availability_zones"] = strings.Join(availabilityZones, ",")
	}
	if len(properties) == 0 {
		return nil, nil
	}
	return properties, nil
}

func volumeTypeProjectID(ctx context.Context, opts *Options, clients *openStackClients, value string) (string, error) {
	identityClient, err := clients.identityV3()
	if err != nil {
		return "", err
	}
	project, err := findProjectWithDomain(ctx, identityClient, value, flagValue(opts, "project-domain"))
	if err != nil {
		return "", err
	}
	return project.ID, nil
}

func volumeTypeCreateEncryption(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, volumeTypeID string) (*volumetypes.EncryptionType, error) {
	if !volumeTypeEncryptionFlagsSet(opts) {
		return nil, nil
	}
	if flagValue(opts, "encryption-provider") == "" {
		return nil, fmt.Errorf("'--encryption-provider' should be specified while creating a new encryption type")
	}
	controlLocation := flagValue(opts, "encryption-control-location")
	if controlLocation == "" {
		controlLocation = "front-end"
	}
	return volumetypes.CreateEncryption(ctx, client, volumeTypeID, volumetypes.CreateEncryptionOpts{
		Provider:        flagValue(opts, "encryption-provider"),
		Cipher:          flagValue(opts, "encryption-cipher"),
		KeySize:         intFlag(opts, "encryption-key-size"),
		ControlLocation: controlLocation,
	}).Extract()
}

func volumeTypeSetEncryption(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, volumeTypeID string) error {
	encryption, err := volumetypes.GetEncryption(ctx, client, volumeTypeID).Extract()
	if err != nil {
		if flagValue(opts, "encryption-provider") == "" {
			return fmt.Errorf("'--encryption-provider' should be specified while creating a new encryption type")
		}
		_, createErr := volumeTypeCreateEncryption(ctx, opts, client, volumeTypeID)
		return createErr
	}
	_, err = volumetypes.UpdateEncryption(ctx, client, volumeTypeID, encryption.EncryptionID, volumetypes.UpdateEncryptionOpts{
		Provider:        flagValue(opts, "encryption-provider"),
		Cipher:          flagValue(opts, "encryption-cipher"),
		KeySize:         intFlag(opts, "encryption-key-size"),
		ControlLocation: flagValue(opts, "encryption-control-location"),
	}).Extract()
	return err
}

func volumeTypeEncryptionFlagsSet(opts *Options) bool {
	for _, name := range []string{"encryption-provider", "encryption-cipher", "encryption-key-size", "encryption-control-location"} {
		if flagChanged(opts, name) {
			return true
		}
	}
	return false
}

func volumeTypeEncryptionMap(item *volumetypes.EncryptionType) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	return map[string]any{
		"cipher":           item.Cipher,
		"control_location": item.ControlLocation,
		"encryption_id":    item.EncryptionID,
		"key_size":         item.KeySize,
		"provider":         item.Provider,
	}
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

func volumeSnapshotCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume snapshot create requires <snapshot-name>")
	}
	volumeValue := flagValue(opts, "volume")
	if volumeValue == "" {
		volumeValue = args[0]
	}
	volume, err := findVolume(ctx, client, volumeValue)
	if err != nil {
		return err
	}
	metadata, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	if remoteValues := flagValues(opts, "remote-source"); len(remoteValues) > 0 {
		remoteSource, err := parseJSONKeyValueMap(remoteValues, "remote-source")
		if err != nil {
			return err
		}
		client, err = blockStorageClientWithMinimumMicroversion(ctx, client, "3.8")
		if err != nil {
			return err
		}
		body := map[string]any{
			"snapshot": map[string]any{
				"volume_id":   volume.ID,
				"ref":         remoteSource,
				"name":        args[0],
				"description": nilIfEmpty(flagValue(opts, "description")),
				"metadata":    metadata,
			},
		}
		var response struct {
			Snapshot *snapshots.Snapshot `json:"snapshot"`
		}
		resp, err := client.Post(ctx, client.ServiceURL("manageable_snapshots"), body, &response, &gophercloud.RequestOpts{
			OkCodes: []int{http.StatusOK, http.StatusAccepted},
		})
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			return oscHTTPException(err)
		}
		if response.Snapshot == nil {
			return fmt.Errorf("response did not contain snapshot object")
		}
		return volumeSnapshotShow(ctx, stdout, opts, client, []string{response.Snapshot.ID})
	}
	created, err := snapshots.Create(ctx, client, snapshots.CreateOpts{
		VolumeID:    volume.ID,
		Force:       boolFlag(opts, "force"),
		Name:        args[0],
		Description: flagValue(opts, "description"),
		Metadata:    metadata,
	}).Extract()
	if err != nil {
		return err
	}
	return volumeSnapshotShow(ctx, stdout, opts, client, []string{created.ID})
}

func volumeSnapshotDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume snapshot delete requires <snapshot>")
	}
	failures := 0
	for _, value := range args {
		item, err := findVolumeSnapshot(ctx, client, value)
		if err != nil {
			failures++
			continue
		}
		if boolFlag(opts, "remote") {
			err = snapshotUnmanage(ctx, client, item.ID)
		} else if boolFlag(opts, "force") {
			err = snapshots.ForceDelete(ctx, client, item.ID).ExtractErr()
		} else {
			err = snapshots.Delete(ctx, client, item.ID).ExtractErr()
		}
		if err != nil {
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d snapshots failed to delete.", failures, len(args))
	}
	return nil
}

func volumeSnapshotSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume snapshot set requires <snapshot>")
	}
	item, err := findVolumeSnapshot(ctx, client, args[0])
	if err != nil {
		return err
	}
	update := snapshots.UpdateOpts{}
	needsUpdate := false
	if flagChanged(opts, "name") {
		update.Name = valueStringPtr(flagValue(opts, "name"))
		needsUpdate = true
	}
	if flagChanged(opts, "description") {
		update.Description = valueStringPtr(flagValue(opts, "description"))
		needsUpdate = true
	}
	if needsUpdate {
		if _, err := snapshots.Update(ctx, client, item.ID, update).Extract(); err != nil {
			return err
		}
	}
	properties, err := parseStringMap(flagValues(opts, "property"), "property")
	if err != nil {
		return err
	}
	if boolFlag(opts, "no-property") || len(properties) > 0 {
		metadata := map[string]any{}
		if !boolFlag(opts, "no-property") {
			for key, value := range item.Metadata {
				metadata[key] = value
			}
		}
		for key, value := range properties {
			metadata[key] = value
		}
		if _, err := snapshots.UpdateMetadata(ctx, client, item.ID, snapshots.UpdateMetadataOpts{Metadata: metadata}).ExtractMetadata(); err != nil {
			return err
		}
	}
	if state := flagValue(opts, "state"); state != "" {
		if err := snapshots.ResetStatus(ctx, client, item.ID, snapshots.ResetStatusOpts{Status: state}).ExtractErr(); err != nil {
			return err
		}
	}
	return nil
}

func volumeSnapshotUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("volume snapshot unset requires <snapshot>")
	}
	item, err := findVolumeSnapshot(ctx, client, args[0])
	if err != nil {
		return err
	}
	for _, key := range flagValues(opts, "property") {
		if err := blockStorageDeleteMetadataKey(ctx, client, "snapshots", item.ID, key); err != nil {
			return err
		}
	}
	return nil
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
	if name := serverImageName(image, imageNames); name != "" {
		return name
	}
	if id, ok := image["id"].(string); ok && id != "" {
		return id
	}
	return image
}

func serverImageShowString(image map[string]any, imageNames map[string]string) string {
	if len(image) == 0 {
		return "N/A (booted from volume)"
	}
	id := stringValue(image["id"])
	name := serverImageName(image, imageNames)
	switch {
	case name != "" && id != "":
		return fmt.Sprintf("%s (%s)", name, id)
	case name != "":
		return name
	case id != "":
		return id
	default:
		return valueString(image)
	}
}

func serverImageName(image map[string]any, imageNames map[string]string) string {
	id := stringValue(image["id"])
	if id != "" && imageNames != nil {
		if name := imageNames[id]; name != "" {
			return name
		}
	}
	if name := stringValue(image["name"]); name != "" {
		return name
	}
	if name := serverImageNameFromProperties(image); name != "" {
		return name
	}
	return ""
}

func serverImageNameFromProperties(image map[string]any) string {
	properties, ok := image["properties"].(map[string]any)
	if !ok {
		return ""
	}
	object := stringValue(properties["owner_specified.openstack.object"])
	if object == "" {
		return ""
	}
	object = strings.TrimRight(object, "/")
	if index := strings.LastIndex(object, "/"); index >= 0 && index < len(object)-1 {
		return object[index+1:]
	}
	return object
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
	networks := serverNetworksInAPISliceOrder(addresses)
	for _, values := range networks {
		sort.Strings(values)
	}
	return networks
}

type serverNetworkLabel struct {
	Prefix netip.Prefix
	Name   string
}

func serverNetworkLabelsForPretty(ctx context.Context, opts *Options, client *gophercloud.ServiceClient) []serverNetworkLabel {
	if !prettyOutput(opts) || client == nil {
		return nil
	}
	return serverNetworkLabels(ctx, client)
}

func serverNetworkLabels(ctx context.Context, client *gophercloud.ServiceClient) []serverNetworkLabel {
	networkPage, err := networks.List(client, networks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil
	}
	networkItems, err := networks.ExtractNetworks(networkPage)
	if err != nil {
		return nil
	}
	networkNames := make(map[string]string, len(networkItems))
	for _, item := range networkItems {
		networkNames[item.ID] = item.Name
	}

	subnetPage, err := subnets.List(client, subnets.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil
	}
	subnetItems, err := subnets.ExtractSubnets(subnetPage)
	if err != nil {
		return nil
	}
	labels := make([]serverNetworkLabel, 0, len(subnetItems))
	for _, item := range subnetItems {
		prefix, err := netip.ParsePrefix(item.CIDR)
		if err != nil {
			continue
		}
		name := firstNonEmpty(networkNames[item.NetworkID], item.NetworkID)
		if name == "" {
			continue
		}
		labels = append(labels, serverNetworkLabel{Prefix: prefix, Name: name})
	}
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Prefix.Bits() > labels[j].Prefix.Bits()
	})
	return labels
}

func serverPrettyNetworkAddresses(addresses map[string]any, labels []serverNetworkLabel) prettyNetworkAddresses {
	return prettyNetworkAddresses(relabelServerNetworks(serverNetworks(addresses), labels))
}

func relabelServerNetworks(values map[string][]string, labels []serverNetworkLabel) map[string][]string {
	if len(labels) == 0 {
		return values
	}
	relabeled := make(map[string][]string, len(values))
	for originalName, addresses := range values {
		for _, address := range addresses {
			name := serverNetworkNameForAddress(address, labels)
			if name == "" {
				name = originalName
			}
			relabeled[name] = append(relabeled[name], address)
		}
	}
	for _, addresses := range relabeled {
		sort.Strings(addresses)
	}
	return relabeled
}

func serverNetworkNameForAddress(address string, labels []serverNetworkLabel) string {
	addr, err := netip.ParseAddr(address)
	if err != nil {
		return ""
	}
	for _, label := range labels {
		if label.Prefix.Contains(addr) {
			return label.Name
		}
	}
	return ""
}

func serverNetworksInAPISliceOrder(addresses map[string]any) map[string][]string {
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
	return networks
}

func serverNetworksSummary(addresses map[string]any) string {
	networks := serverNetworks(addresses)
	names := sortedKeysFromStringSlices(networks)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s=%s", name, strings.Join(networks[name], ", ")))
	}
	return strings.Join(parts, "; ")
}

type prettyNetworkAddresses map[string][]string

func (addresses prettyNetworkAddresses) PrettyString() string {
	names := sortedKeysFromStringSlices(addresses)
	if len(names) == 0 {
		return "None"
	}
	lines := make([]string, 0, len(addresses))
	for _, name := range names {
		values := append([]string(nil), addresses[name]...)
		sort.Strings(values)
		if len(values) == 0 {
			lines = append(lines, name+": None")
			continue
		}
		for _, value := range values {
			lines = append(lines, name+": "+value)
		}
	}
	return strings.Join(lines, "\n")
}

type prettyAddressList []string

func (addresses prettyAddressList) PrettyString() string {
	values := make([]string, 0, len(addresses))
	for _, value := range addresses {
		value = strings.TrimSpace(value)
		if value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return "None"
	}
	sort.Strings(values)
	return strings.Join(values, "\n")
}

func sortedKeysFromStringSlices(values map[string][]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type prettyPortFixedIPAddresses []ports.IP

func (fixedIPs prettyPortFixedIPAddresses) PrettyString() string {
	values := make([]string, 0, len(fixedIPs))
	for _, fixedIP := range fixedIPs {
		switch {
		case strings.TrimSpace(fixedIP.IPAddress) != "":
			values = append(values, strings.TrimSpace(fixedIP.IPAddress))
		case strings.TrimSpace(fixedIP.SubnetID) != "":
			values = append(values, "subnet: "+strings.TrimSpace(fixedIP.SubnetID))
		}
	}
	if len(values) == 0 {
		return "None"
	}
	sort.Strings(values)
	return strings.Join(values, "\n")
}

type prettyIPMapList struct {
	items       []map[string]string
	primaryKeys []string
}

func (list prettyIPMapList) PrettyString() string {
	lines := prettyIPMapLines(list.items, list.primaryKeys...)
	if len(lines) == 0 {
		return "None"
	}
	return strings.Join(lines, "\n")
}

func prettyNetworkIPOutputFields(opts *Options, fields []outputField) []outputField {
	if !prettyOutput(opts) {
		return fields
	}
	formatted := append([]outputField(nil), fields...)
	for index := range formatted {
		formatted[index].Value = prettyNetworkIPFieldValue(formatted[index].Name, formatted[index].Value)
	}
	return formatted
}

func prettyNetworkIPFieldValue(name string, value any) any {
	switch name {
	case "addresses":
		if values, ok := stringSliceFromAny(value); ok {
			return prettyAddressList(values)
		}
	case "allowed_address_pairs", "fixed_ips", "interfaces_info":
		if values, ok := prettyIPMapsFromAny(value); ok {
			return prettyIPMapList{
				items:       values,
				primaryKeys: []string{"ip_address", "fixed_ip_address", "floating_ip_address"},
			}
		}
	case "external_gateway_info":
		if gateway, ok := prettyRouterGatewayInfoFromAny(value); ok {
			return gateway
		}
	}
	return value
}

type prettyRouterGatewayInfo map[string]any

func (gateway prettyRouterGatewayInfo) PrettyString() string {
	if len(gateway) == 0 {
		return "{}"
	}

	seen := map[string]bool{}
	lines := []string{}
	for _, key := range []string{"network_id", "enable_snat", "external_fixed_ips", "qos_policy_id"} {
		if value, ok := gateway[key]; ok {
			lines = appendPrettyRouterGatewayEntry(lines, key, value)
			seen[key] = true
		}
	}
	var extraKeys []string
	for key := range gateway {
		if !seen[key] {
			extraKeys = append(extraKeys, key)
		}
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		lines = appendPrettyRouterGatewayEntry(lines, key, gateway[key])
	}
	if len(lines) == 0 {
		return "{}"
	}
	return strings.Join(lines, "\n")
}

func appendPrettyRouterGatewayEntry(lines []string, key string, value any) []string {
	if key == "external_fixed_ips" {
		if values, ok := prettyIPMapsFromAny(value); ok {
			lines = append(lines, "external fixed IPs:")
			childLines := prettyIPMapLines(values, "ip_address")
			if len(childLines) == 0 {
				return append(lines, "  None")
			}
			for _, line := range childLines {
				lines = append(lines, "  "+line)
			}
			return lines
		}
	}
	if childLines, ok := prettyStructuredLines(value, 2); ok {
		if len(childLines) == 1 && strings.TrimSpace(childLines[0]) == "{}" {
			return append(lines, prettyNetworkIPLabel(key)+": {}")
		}
		if len(childLines) == 1 && strings.TrimSpace(childLines[0]) == "[]" {
			return append(lines, prettyNetworkIPLabel(key)+": []")
		}
		lines = append(lines, prettyNetworkIPLabel(key)+":")
		return append(lines, childLines...)
	}
	return append(lines, prettyNetworkIPLabel(key)+": "+prettyScalarString(value))
}

func prettyRouterGatewayInfoFromAny(value any) (prettyRouterGatewayInfo, bool) {
	if value == nil {
		return nil, false
	}
	switch typed := value.(type) {
	case map[string]any:
		return prettyRouterGatewayInfo(typed), true
	case map[string]string:
		values := map[string]any{}
		for key, item := range typed {
			values[key] = item
		}
		return prettyRouterGatewayInfo(values), true
	case routers.GatewayInfo:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, false
		}
		var values map[string]any
		if err := json.Unmarshal(encoded, &values); err != nil {
			return nil, false
		}
		if values == nil {
			return nil, false
		}
		return prettyRouterGatewayInfo(values), true
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, false
		}
		var values map[string]any
		if err := json.Unmarshal(encoded, &values); err != nil {
			return nil, false
		}
		if values == nil {
			return nil, false
		}
		return prettyRouterGatewayInfo(values), true
	}
}

func prettyIPMapLines(items []map[string]string, primaryKeys ...string) []string {
	sorted := sortedPrettyIPMaps(items)
	lines := make([]string, 0, len(sorted)*2)
	for _, item := range sorted {
		lines = append(lines, prettyIPMapEntryLines(item, primaryKeys...)...)
	}
	return lines
}

func prettyIPMapEntryLines(item map[string]string, primaryKeys ...string) []string {
	primaryKey := ""
	for _, key := range primaryKeys {
		if strings.TrimSpace(item[key]) != "" {
			primaryKey = key
			break
		}
	}

	lines := []string{}
	if primaryKey != "" {
		lines = append(lines, strings.TrimSpace(item[primaryKey]))
	}
	for _, key := range sortedKeys(item) {
		if key == primaryKey {
			continue
		}
		value := strings.TrimSpace(item[key])
		if value == "" {
			continue
		}
		prefix := ""
		if len(lines) > 0 {
			prefix = "  "
		}
		lines = append(lines, prefix+prettyNetworkIPLabel(key)+": "+value)
	}
	return lines
}

func sortedPrettyIPMaps(items []map[string]string) []map[string]string {
	values := make([]map[string]string, 0, len(items))
	for _, item := range items {
		copied := map[string]string{}
		for key, value := range item {
			value = strings.TrimSpace(value)
			if value != "" && value != "None" {
				copied[key] = value
			}
		}
		if len(copied) > 0 {
			values = append(values, copied)
		}
	}
	sort.SliceStable(values, func(left int, right int) bool {
		return prettyIPMapSortKey(values[left]) < prettyIPMapSortKey(values[right])
	})
	return values
}

func prettyIPMapSortKey(item map[string]string) string {
	parts := make([]string, 0, len(item))
	for _, key := range sortedKeys(item) {
		parts = append(parts, key+"="+item[key])
	}
	return strings.Join(parts, "\x00")
}

func prettyNetworkIPLabel(key string) string {
	switch key {
	case "fixed_ip_address":
		return "fixed ip"
	case "floating_ip_address":
		return "floating ip"
	case "ip_address":
		return "ip"
	case "mac_address":
		return "mac"
	case "network_id":
		return "network"
	case "port_id":
		return "port"
	case "subnet_id":
		return "subnet"
	default:
		return strings.ReplaceAll(key, "_", " ")
	}
}

func prettyIPMapsFromAny(value any) ([]map[string]string, bool) {
	switch typed := value.(type) {
	case []ports.IP:
		return portFixedIPMapsFromTyped(typed), true
	case []ports.AddressPair:
		return portAllowedAddressMapsFromTyped(typed), true
	case []routers.ExternalFixedIP:
		values := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			entry := map[string]string{}
			if item.IPAddress != "" {
				entry["ip_address"] = item.IPAddress
			}
			if item.SubnetID != "" {
				entry["subnet_id"] = item.SubnetID
			}
			if len(entry) > 0 {
				values = append(values, entry)
			}
		}
		return values, true
	case []map[string]string:
		return cloneStringMaps(typed), true
	case []map[string]any:
		return stringMapsFromAnyMaps(typed), true
	case []any:
		return stringMapsFromAnySlice(typed)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, false
		}
		var maps []map[string]any
		if err := json.Unmarshal(encoded, &maps); err != nil {
			return nil, false
		}
		return stringMapsFromAnyMaps(maps), true
	}
}

func cloneStringMaps(values []map[string]string) []map[string]string {
	cloned := make([]map[string]string, 0, len(values))
	for _, value := range values {
		item := map[string]string{}
		for key, raw := range value {
			if strings.TrimSpace(raw) != "" {
				item[key] = raw
			}
		}
		if len(item) > 0 {
			cloned = append(cloned, item)
		}
	}
	return cloned
}

func stringMapsFromAnySlice(values []any) ([]map[string]string, bool) {
	items := make([]map[string]string, 0, len(values))
	for _, value := range values {
		item, ok := stringMapFromAny(value)
		if !ok {
			return nil, false
		}
		if len(item) > 0 {
			items = append(items, item)
		}
	}
	return items, true
}

func stringMapsFromAnyMaps(values []map[string]any) []map[string]string {
	items := make([]map[string]string, 0, len(values))
	for _, value := range values {
		item := stringMapFromAnyMap(value)
		if len(item) > 0 {
			items = append(items, item)
		}
	}
	return items
}

func stringMapFromAny(value any) (map[string]string, bool) {
	switch typed := value.(type) {
	case map[string]string:
		return cloneStringMap(typed), true
	case map[string]any:
		return stringMapFromAnyMap(typed), true
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, false
		}
		var item map[string]any
		if err := json.Unmarshal(encoded, &item); err != nil {
			return nil, false
		}
		return stringMapFromAnyMap(item), true
	}
}

func cloneStringMap(value map[string]string) map[string]string {
	item := map[string]string{}
	for key, raw := range value {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			item[key] = raw
		}
	}
	return item
}

func stringMapFromAnyMap(value map[string]any) map[string]string {
	item := map[string]string{}
	for key, raw := range value {
		text := strings.TrimSpace(valueString(raw))
		if text != "" && text != "None" {
			item[key] = text
		}
	}
	return item
}

func stringSliceFromAny(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			values = append(values, text)
		}
		return values, true
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, false
		}
		var values []string
		if err := json.Unmarshal(encoded, &values); err != nil {
			return nil, false
		}
		return values, true
	}
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

func serverNameMap(ctx context.Context, client *gophercloud.ServiceClient) map[string]string {
	names := map[string]string{}
	if client == nil {
		return names
	}
	page, err := servers.List(client, servers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return names
	}
	items, err := extractServers(page)
	if err != nil {
		return names
	}
	for _, item := range items {
		if item.ID != "" && item.Name != "" {
			names[item.ID] = item.Name
		}
	}
	return names
}

func volumeAttachments(attachments []volumes.Attachment) []map[string]any {
	values := make([]map[string]any, 0, len(attachments))
	for _, item := range attachments {
		values = append(values, map[string]any{
			"id":            prettyVolumeValue(item.ID),
			"attachment_id": item.AttachmentID,
			"volume_id":     prettyVolumeValue(item.VolumeID),
			"server_id":     item.ServerID,
			"host_name":     item.HostName,
			"device":        item.Device,
			"attached_at":   item.AttachedAt.Format("2006-01-02T15:04:05.000000"),
		})
	}
	return values
}

func volumeAttachmentValue(attachments []volumes.Attachment, serverNames map[string]string) tableValue {
	jsonValues := make([]any, 0, len(attachments))
	prettyValues := make([]map[string]any, 0, len(attachments))
	tableLines := make([]string, 0, len(attachments))
	for _, item := range attachments {
		jsonValues = append(jsonValues, orderedJSONObject{
			keys: []string{"id", "attachment_id", "volume_id", "server_id", "host_name", "device", "attached_at"},
			values: map[string]any{
				"id":            item.ID,
				"attachment_id": item.AttachmentID,
				"volume_id":     item.VolumeID,
				"server_id":     item.ServerID,
				"host_name":     item.HostName,
				"device":        item.Device,
				"attached_at":   item.AttachedAt.Format("2006-01-02T15:04:05.000000"),
			},
		})
		prettyValues = append(prettyValues, map[string]any{
			"id":            prettyVolumeValue(item.ID),
			"attachment_id": item.AttachmentID,
			"volume_id":     prettyVolumeValue(item.VolumeID),
			"server_id":     item.ServerID,
			"host_name":     item.HostName,
			"device":        item.Device,
			"attached_at":   item.AttachedAt.Format("2006-01-02T15:04:05.000000"),
		})
		if serverNames != nil && item.ServerID != "" && item.Device != "" {
			server := item.ServerID
			if name := serverNames[item.ServerID]; name != "" {
				server = name
			}
			tableLines = append(tableLines, fmt.Sprintf("Attached to %s on %s ", server, item.Device))
		}
	}
	return tableValue{
		Value:  jsonValues,
		Table:  strings.Join(tableLines, "\n"),
		Pretty: prettyValues,
	}
}

func volumeAttachmentValueFromAny(value any, serverNames map[string]string) tableValue {
	items := anySlice(value)
	jsonValues := make([]any, 0, len(items))
	prettyValues := make([]map[string]any, 0, len(items))
	tableLines := make([]string, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ordered := orderedJSONObject{
			keys: []string{"id", "attachment_id", "volume_id", "server_id", "host_name", "device", "attached_at"},
			values: map[string]any{
				"id":            itemMap["id"],
				"attachment_id": itemMap["attachment_id"],
				"volume_id":     itemMap["volume_id"],
				"server_id":     itemMap["server_id"],
				"host_name":     itemMap["host_name"],
				"device":        itemMap["device"],
				"attached_at":   itemMap["attached_at"],
			},
		}
		jsonValues = append(jsonValues, ordered)
		prettyValues = append(prettyValues, ordered.values)
		serverID := strings.TrimSpace(fmt.Sprint(itemMap["server_id"]))
		device := strings.TrimSpace(fmt.Sprint(itemMap["device"]))
		if serverNames != nil && serverID != "" && serverID != "<nil>" && device != "" && device != "<nil>" {
			server := serverID
			if name := serverNames[serverID]; name != "" {
				server = name
			}
			tableLines = append(tableLines, fmt.Sprintf("Attached to %s on %s ", server, device))
		}
	}
	if len(tableLines) == 0 {
		return tableValue{Value: jsonValues, Table: pythonRepr(jsonValues), Pretty: prettyValues}
	}
	return tableValue{Value: jsonValues, Table: strings.Join(tableLines, "\n"), Pretty: prettyValues}
}

func mapValueOrEmptyAny(value any) map[string]any {
	switch typed := value.(type) {
	case nil:
		return map[string]any{}
	case map[string]any:
		return typed
	case map[string]string:
		values := make(map[string]any, len(typed))
		for key, value := range typed {
			values[key] = value
		}
		return values
	default:
		return map[string]any{}
	}
}

func mapTableValue(value any, emptyTable string) tableValue {
	values := mapValueOrEmptyAny(value)
	table := pythonRepr(values)
	if len(values) == 0 {
		table = emptyTable
	}
	return tableValue{Value: values, Table: table, Pretty: values}
}

func networkAgentConfigurationValue(value any) tableValue {
	values := mapValueOrEmptyAny(value)
	if len(values) == 0 {
		return tableValue{Value: values, Table: "{}", Pretty: values}
	}
	keys := []string{
		"arp_responder_enabled",
		"baremetal_smartnic",
		"bridge_mappings",
		"datapath_type",
		"devices",
		"enable_distributed_routing",
		"extensions",
		"in_distributed_mode",
		"integration_bridge",
		"l2_population",
		"log_agent_heartbeats",
		"ovs_capabilities",
		"ovs_hybrid_plug",
		"resource_provider_bandwidths",
		"resource_provider_hypervisors",
		"resource_provider_inventory_defaults",
		"resource_provider_packet_processing_inventory_defaults",
		"resource_provider_packet_processing_with_direction",
		"resource_provider_packet_processing_without_direction",
		"tunnel_types",
		"tunneling_ip",
		"vhostuser_socket_dir",
	}
	jsonValues := make(map[string]any, len(values))
	tableValues := make(map[string]any, len(values))
	for key, item := range values {
		jsonValues[key] = item
		tableValues[key] = item
	}
	if item, ok := tableValues["ovs_capabilities"]; ok {
		ordered := orderedMapFromKeys(mapValueOrEmptyAny(item), []string{"datapath_types", "iface_types"})
		jsonValues["ovs_capabilities"] = ordered
		tableValues["ovs_capabilities"] = ordered
	}
	if item, ok := tableValues["resource_provider_hypervisors"]; ok {
		ordered := orderedMapFromKeys(mapValueOrEmptyAny(item), []string{"rp_tunnelled", "br-ex"})
		jsonValues["resource_provider_hypervisors"] = ordered
		tableValues["resource_provider_hypervisors"] = ordered
	}
	if item, ok := tableValues["resource_provider_inventory_defaults"]; ok {
		ordered := networkAgentInventoryDefaultsValue(item)
		jsonValues["resource_provider_inventory_defaults"] = ordered
		tableValues["resource_provider_inventory_defaults"] = ordered
	}
	if item, ok := tableValues["resource_provider_packet_processing_inventory_defaults"]; ok {
		ordered := networkAgentInventoryDefaultsValue(item)
		jsonValues["resource_provider_packet_processing_inventory_defaults"] = ordered
		tableValues["resource_provider_packet_processing_inventory_defaults"] = ordered
	}
	return tableValue{
		Value:  orderedMapFromKeys(jsonValues, keys),
		Table:  pythonDictRepr(tableValues, keys),
		Pretty: values,
	}
}

func networkAgentInventoryDefaultsValue(value any) orderedJSONObject {
	values := mapValueOrEmptyAny(value)
	if item, ok := values["allocation_ratio"]; ok {
		if numeric, ok := decimalPythonNumber(item); ok {
			values["allocation_ratio"] = numeric
		}
	}
	return orderedMapFromKeys(values, []string{"allocation_ratio", "min_unit", "step_size", "reserved"})
}

func decimalPythonNumber(value any) (json.Number, bool) {
	switch typed := value.(type) {
	case float64:
		return decimalJSONNumber(typed), true
	case float32:
		return decimalJSONNumber(float64(typed)), true
	default:
		return "", false
	}
}

func volumeImageMetadataValue(value any) tableValue {
	values := mapValueOrEmptyAny(value)
	keys := []string{"signature_verified", "owner_specified.openstack.md5", "owner_specified.openstack.object", "owner_specified.openstack.sha256", "image_id", "image_name", "checksum", "container_format", "disk_format", "min_disk", "min_ram", "size"}
	table := pythonDictRepr(values, keys)
	if len(values) == 0 {
		table = ""
	}
	return tableValue{Value: orderedMapFromKeys(values, keys), Table: table, Pretty: values}
}

func imagePropertiesValue(value map[string]any) tableValue {
	keys := sortedMapKeys(value)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s='%s'", key, strings.ReplaceAll(valueString(value[key]), "'", "\\'")))
	}
	return tableValue{Value: orderedMapFromKeys(value, []string{"os_hidden", "os_hash_algo", "os_hash_value"}), Table: strings.Join(parts, ", "), Pretty: value}
}

func flavorPropertiesValue(value any) tableValue {
	values := mapValueOrEmptyAny(value)
	table := pythonRepr(values)
	if len(values) == 0 {
		table = ""
	}
	return tableValue{Value: values, Table: table, Pretty: values}
}

func volumeTypePropertiesValue(value map[string]string) tableValue {
	values := make(map[string]any, len(value))
	for key, item := range value {
		values[key] = item
	}
	parts := make([]string, 0, len(values))
	for _, key := range sortedMapKeys(values) {
		parts = append(parts, fmt.Sprintf("%s='%s'", key, strings.ReplaceAll(valueString(values[key]), "'", "\\'")))
	}
	return tableValue{Value: values, Table: strings.Join(parts, ", "), Pretty: values}
}

func volumeTypeAccessProjectIDs(ctx context.Context, client *gophercloud.ServiceClient, item *volumetypes.VolumeType) (any, error) {
	if volumeTypeIsPublic(*item) {
		return nil, nil
	}
	page, err := volumetypes.ListAccesses(client, item.ID).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	accesses, err := volumetypes.ExtractAccesses(page)
	if err != nil {
		return nil, err
	}
	values := make([]string, 0, len(accesses))
	for _, access := range accesses {
		values = append(values, access.ProjectID)
	}
	sort.Strings(values)
	return values, nil
}

func attachmentTimeValue(value time.Time) tableValue {
	if value.IsZero() {
		return tableValue{Value: "", Table: ""}
	}
	text := value.UTC().Format("2006-01-02T15:04:05.000000")
	return tableValue{Value: text, Table: text}
}

func volumeAttachmentPropertiesValue(value map[string]any) tableValue {
	jsonKeys := []string{
		"target_discovered",
		"target_portal",
		"target_iqn",
		"target_lun",
		"volume_id",
		"auth_method",
		"auth_username",
		"auth_password",
		"encrypted",
		"qos_specs",
		"access_mode",
		"cacheable",
		"driver_volume_type",
		"attachment_id",
		"enforce_multipath",
	}
	tableKeys := sortedMapKeys(value)
	parts := make([]string, 0, len(tableKeys))
	for _, key := range tableKeys {
		item := value[key]
		if item == nil {
			parts = append(parts, key+"=")
			continue
		}
		parts = append(parts, fmt.Sprintf("%s='%s'", key, strings.ReplaceAll(attachmentPropertyString(item), "'", "\\'")))
	}
	return tableValue{
		Value:  orderedMapFromKeys(value, jsonKeys),
		Table:  strings.Join(parts, ", "),
		Pretty: value,
	}
}

func attachmentPropertyString(value any) string {
	switch typed := value.(type) {
	case bool:
		if typed {
			return "True"
		}
		return "False"
	default:
		return valueString(value)
	}
}

func flavorRxTxValue(value any) any {
	switch typed := value.(type) {
	case float64:
		text := decimalJSONNumber(typed)
		return tableValue{Value: text, Table: string(text)}
	case float32:
		text := decimalJSONNumber(float64(typed))
		return tableValue{Value: text, Table: string(text)}
	default:
		return value
	}
}

func decimalJSONNumber(value float64) json.Number {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	return json.Number(text)
}

func flavorSwapValue(value any) any {
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return 0
		}
		return rawNumber(typed)
	default:
		return rawNumber(value)
	}
}

func blankEmptyListValue(value any) any {
	if value == nil {
		return tableValue{Value: []any{}, Table: ""}
	}
	reflected := reflect.ValueOf(value)
	if reflected.IsValid() && reflected.Kind() == reflect.Slice && reflected.Len() == 0 {
		return tableValue{Value: value, Table: ""}
	}
	return value
}

func blankEmptyMapValue(value any) any {
	if value == nil {
		return tableValue{Value: map[string]any{}, Table: ""}
	}
	reflected := reflect.ValueOf(value)
	if reflected.IsValid() && reflected.Kind() == reflect.Map && reflected.Len() == 0 {
		return tableValue{Value: value, Table: ""}
	}
	return value
}

func blankEmptyMapAsStringValue(value any) any {
	if value == nil {
		return tableValue{Value: "", Table: ""}
	}
	reflected := reflect.ValueOf(value)
	if reflected.IsValid() && reflected.Kind() == reflect.Map && reflected.Len() == 0 {
		return tableValue{Value: "", Table: ""}
	}
	return value
}

func blankEmptyStringListValue(value []string) any {
	if len(value) == 0 {
		return tableValue{Value: value, Table: ""}
	}
	return value
}

func adminStateValue(value any) any {
	switch typed := value.(type) {
	case bool:
		if typed {
			return tableValue{Value: typed, Table: "UP"}
		}
		return tableValue{Value: typed, Table: "DOWN"}
	default:
		return value
	}
}

func agentAliveValue(value any) any {
	switch typed := value.(type) {
	case bool:
		if typed {
			return tableValue{Value: typed, Table: ":-)"}
		}
		return tableValue{Value: typed, Table: "XXX"}
	default:
		return value
	}
}

func routerExternalValue(value any) any {
	switch typed := value.(type) {
	case bool:
		if typed {
			return tableValue{Value: typed, Table: "External"}
		}
		return tableValue{Value: typed, Table: "Internal"}
	default:
		return value
	}
}

func routerGatewayTableValue(value any) any {
	gateway, ok := value.(map[string]any)
	if !ok || len(gateway) == 0 {
		return value
	}
	values := map[string]any{}
	keys := []string{"network_id", "external_fixed_ips", "enable_snat", "qos_policy_id"}
	for _, key := range keys {
		if item, ok := gateway[key]; ok {
			if key == "external_fixed_ips" {
				item = orderedIPMapList(item, []string{"subnet_id", "ip_address"})
			}
			values[key] = item
		}
	}
	ordered := orderedMapFromKeys(values, keys)
	return tableValue{Value: ordered, Table: jsonLikeRepr(ordered), Pretty: value}
}

func routerInterfacesTableValue(value []map[string]string) tableValue {
	values := make([]any, 0, len(value))
	for _, item := range value {
		values = append(values, orderedJSONObject{
			keys: []string{"port_id", "ip_address", "subnet_id"},
			values: map[string]any{
				"port_id":    item["port_id"],
				"ip_address": item["ip_address"],
				"subnet_id":  item["subnet_id"],
			},
		})
	}
	return tableValue{Value: values, Table: jsonLikeRepr(values), Pretty: value}
}

func portFixedIPsTableValue(value any) tableValue {
	values := orderedIPMapList(value, []string{"subnet_id", "ip_address"})
	lines := make([]string, 0, len(values))
	for _, item := range values {
		ip := strings.TrimSpace(fmt.Sprint(item.values["ip_address"]))
		subnet := strings.TrimSpace(fmt.Sprint(item.values["subnet_id"]))
		switch {
		case ip != "" && ip != "<nil>" && subnet != "" && subnet != "<nil>":
			lines = append(lines, fmt.Sprintf("ip_address='%s', subnet_id='%s'", ip, subnet))
		case ip != "" && ip != "<nil>":
			lines = append(lines, fmt.Sprintf("ip_address='%s'", ip))
		case subnet != "" && subnet != "<nil>":
			lines = append(lines, fmt.Sprintf("subnet_id='%s'", subnet))
		}
	}
	anyValues := make([]any, len(values))
	for i, item := range values {
		anyValues[i] = item
	}
	return tableValue{Value: anyValues, Table: strings.Join(lines, "\n"), Pretty: value}
}

func portFixedIPsValue(fixedIPs []ports.IP) tableValue {
	tableLines := portFixedIPs(fixedIPs)
	jsonValues := make([]any, 0, len(fixedIPs))
	for _, item := range fixedIPs {
		values := map[string]any{}
		keys := []string{"subnet_id", "ip_address"}
		values["subnet_id"] = item.SubnetID
		values["ip_address"] = item.IPAddress
		jsonValues = append(jsonValues, orderedJSONObject{keys: keys, values: values})
	}
	return tableValue{
		Value:  jsonValues,
		Table:  strings.Join(tableLines, ", "),
		Pretty: prettyPortFixedIPAddresses(fixedIPs),
	}
}

func orderedIPMapList(value any, keys []string) []orderedJSONObject {
	items := anySlice(value)
	values := make([]orderedJSONObject, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		orderedValues := map[string]any{}
		for _, key := range keys {
			orderedValues[key] = itemMap[key]
		}
		values = append(values, orderedJSONObject{keys: keys, values: orderedValues})
	}
	return values
}

func cinderBool(value any) any {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		if strings.EqualFold(typed, "true") {
			return true
		}
		if strings.EqualFold(typed, "false") {
			return false
		}
	}
	return value
}

func jsonLikeRepr(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		encoded, _ := json.Marshal(typed)
		return string(encoded)
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64:
		encoded, _ := json.Marshal(typed)
		return string(encoded)
	case orderedJSONObject:
		parts := make([]string, 0, len(typed.keys))
		for _, key := range typed.keys {
			value, ok := typed.values[key]
			if !ok {
				continue
			}
			encodedKey, _ := json.Marshal(key)
			parts = append(parts, fmt.Sprintf("%s: %s", encodedKey, jsonLikeRepr(value)))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case []orderedJSONObject:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, jsonLikeRepr(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, jsonLikeRepr(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return valueString(typed)
		}
		return string(encoded)
	}
}

func orderedMapFromKeys(values map[string]any, preferred []string) orderedJSONObject {
	keys := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, key := range preferred {
		if _, ok := values[key]; ok {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	for _, key := range sortedMapKeys(values) {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	return orderedJSONObject{keys: keys, values: values}
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func pythonDictRepr(values map[string]any, preferred []string) string {
	if len(values) == 0 {
		return "{}"
	}
	ordered := orderedMapFromKeys(values, preferred)
	parts := make([]string, 0, len(ordered.keys))
	for _, key := range ordered.keys {
		parts = append(parts, fmt.Sprintf("'%s': %s", key, pythonRepr(ordered.values[key])))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func pythonRepr(value any) string {
	switch typed := value.(type) {
	case nil:
		return "None"
	case string:
		return "'" + strings.ReplaceAll(typed, "'", "\\'") + "'"
	case prettyVolumeValue:
		return pythonRepr(string(typed))
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64:
		return valueString(typed)
	case orderedJSONObject:
		parts := make([]string, 0, len(typed.keys))
		for _, key := range typed.keys {
			parts = append(parts, fmt.Sprintf("'%s': %s", key, pythonRepr(typed.values[key])))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case map[string]any:
		return pythonDictRepr(typed, nil)
	case map[string]string:
		values := make(map[string]any, len(typed))
		for key, value := range typed {
			values[key] = value
		}
		return pythonDictRepr(values, nil)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, pythonRepr(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		reflected := reflect.ValueOf(value)
		if reflected.IsValid() && reflected.Kind() == reflect.Slice {
			parts := make([]string, 0, reflected.Len())
			for i := 0; i < reflected.Len(); i++ {
				parts = append(parts, pythonRepr(reflected.Index(i).Interface()))
			}
			return "[" + strings.Join(parts, ", ") + "]"
		}
		return valueString(value)
	}
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
			values = append(values, fmt.Sprintf("ip_address='%s', subnet_id='%s'", item.IPAddress, item.SubnetID))
		case item.IPAddress != "":
			values = append(values, fmt.Sprintf("ip_address='%s'", item.IPAddress))
		case item.SubnetID != "":
			values = append(values, fmt.Sprintf("subnet_id='%s'", item.SubnetID))
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

type serverCreateOpts struct {
	servers.CreateOpts
	Description              string
	Host                     string
	KeyName                  string
	NoSecurityGroups         bool
	TrustedImageCertificates []string
}

func (opts serverCreateOpts) ToServerCreateMap() (map[string]any, error) {
	body, err := opts.CreateOpts.ToServerCreateMap()
	if err != nil {
		return nil, err
	}
	serverBody, _ := body["server"].(map[string]any)
	if opts.Description != "" {
		serverBody["description"] = opts.Description
	}
	if opts.Host != "" {
		serverBody["host"] = opts.Host
	}
	if opts.KeyName != "" {
		serverBody["key_name"] = opts.KeyName
	}
	if opts.NoSecurityGroups {
		serverBody["security_groups"] = []any{}
	}
	if len(opts.TrustedImageCertificates) > 0 {
		serverBody["trusted_image_certificates"] = opts.TrustedImageCertificates
	}
	return body, nil
}

type serverRebuildOpts struct {
	servers.RebuildOpts
	Description              string
	KeyName                  string
	UnsetKeyName             bool
	UserData                 []byte
	UnsetUserData            bool
	TrustedImageCertificates []string
	UnsetTrustedImageCerts   bool
	Hostname                 string
	ReimageBootVolume        bool
	NoReimageBootVolume      bool
}

func (opts serverRebuildOpts) ToServerRebuildMap() (map[string]any, error) {
	body, err := opts.RebuildOpts.ToServerRebuildMap()
	if err != nil {
		return nil, err
	}
	rebuildBody, _ := body["rebuild"].(map[string]any)
	if opts.Description != "" {
		rebuildBody["description"] = opts.Description
	}
	if opts.KeyName != "" {
		rebuildBody["key_name"] = opts.KeyName
	}
	if opts.UnsetKeyName {
		rebuildBody["key_name"] = nil
	}
	if opts.UserData != nil {
		rebuildBody["user_data"] = base64.StdEncoding.EncodeToString(opts.UserData)
	}
	if opts.UnsetUserData {
		rebuildBody["user_data"] = nil
	}
	if len(opts.TrustedImageCertificates) > 0 {
		rebuildBody["trusted_image_certificates"] = opts.TrustedImageCertificates
	}
	if opts.UnsetTrustedImageCerts {
		rebuildBody["trusted_image_certificates"] = nil
	}
	if opts.Hostname != "" {
		rebuildBody["hostname"] = opts.Hostname
	}
	if opts.ReimageBootVolume {
		rebuildBody["reimage_boot_volume"] = true
	}
	if opts.NoReimageBootVolume {
		rebuildBody["reimage_boot_volume"] = false
	}
	return body, nil
}

type serverUpdateOpts struct {
	Name        string
	Hostname    *string
	Description *string
}

func (opts serverUpdateOpts) ToServerUpdateMap() (map[string]any, error) {
	values := map[string]any{}
	if opts.Name != "" {
		values["name"] = opts.Name
	}
	if opts.Hostname != nil {
		values["hostname"] = *opts.Hostname
	}
	if opts.Description != nil {
		values["description"] = *opts.Description
	}
	return map[string]any{"server": values}, nil
}

func (opts serverUpdateOpts) hasValues() bool {
	return opts.Name != "" || opts.Hostname != nil || opts.Description != nil
}

type serverAttachInterfaceOpts struct {
	PortID    string                     `json:"port_id,omitempty"`
	NetworkID string                     `json:"net_id,omitempty"`
	FixedIPs  []attachinterfaces.FixedIP `json:"fixed_ips,omitempty"`
	Tag       string                     `json:"tag,omitempty"`
}

func (opts serverAttachInterfaceOpts) ToAttachInterfacesCreateMap() (map[string]any, error) {
	return gophercloud.BuildRequestBody(opts, "interfaceAttachment")
}

func renderServerShow(stdout io.Writer, opts *Options, item *servers.Server, imageNames map[string]string) error {
	addresses := any(serverNetworksSummary(item.Addresses))
	if prettyOutput(opts) {
		addresses = prettyNetworkAddresses(serverNetworks(item.Addresses))
	}
	fields := []outputField{
		{"id", item.ID},
		{"name", item.Name},
		{"status", item.Status},
		{"project_id", item.TenantID},
		{"user_id", item.UserID},
		{"created", item.Created},
		{"updated", item.Updated},
		{"image", serverImage(item.Image, imageNames)},
		{"flavor", serverFlavor(item.Flavor, nil)},
		{"addresses", addresses},
		{"metadata", item.Metadata},
		{"key_name", nilIfEmpty(item.KeyName)},
	}
	if item.AdminPass != "" {
		fields = append(fields, outputField{"adminPass", item.AdminPass})
	}
	return renderShowOutput(stdout, opts, fields)
}

func serverInterfaceFields(item *attachinterfaces.Interface) []outputField {
	if item == nil {
		return nil
	}
	return []outputField{
		{"fixed_ips", item.FixedIPs},
		{"mac_addr", item.MACAddr},
		{"net_id", item.NetID},
		{"port_id", item.PortID},
		{"port_state", item.PortState},
	}
}

func serverVolumeAttachmentFields(item *volumeattach.VolumeAttachment) []outputField {
	if item == nil {
		return nil
	}
	return []outputField{
		{"device", item.Device},
		{"id", item.ID},
		{"serverId", item.ServerID},
		{"volumeId", item.VolumeID},
		{"tag", stringPtrValue(item.Tag)},
		{"delete_on_termination", boolPtrValue(item.DeleteOnTermination)},
	}
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

func serverRawAction(ctx context.Context, client *gophercloud.ServiceClient, serverID string, action string, payload any, out any, okCodes ...int) error {
	if len(okCodes) == 0 {
		okCodes = []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}
	}
	body := map[string]any{action: payload}
	resp, err := client.Post(ctx, client.ServiceURL("servers", url.PathEscape(serverID), "action"), body, out, &gophercloud.RequestOpts{OkCodes: okCodes})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return oscHTTPException(err)
}

func serverDeleteAdminPassword(ctx context.Context, client *gophercloud.ServiceClient, serverID string) error {
	resp, err := client.Delete(ctx, client.ServiceURL("servers", url.PathEscape(serverID), "os-server-password"), &gophercloud.RequestOpts{OkCodes: []int{http.StatusNoContent, http.StatusAccepted}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return oscHTTPException(err)
}

func findServerForServerCommand(ctx context.Context, client *gophercloud.ServiceClient, value string, allProjects bool) (*servers.Server, error) {
	result := servers.Get(ctx, client, value)
	if result.Err == nil {
		return result.Extract()
	}
	page, err := servers.List(client, servers.ListOpts{Name: value, AllTenants: allProjects}).AllPages(ctx)
	if err != nil {
		return nil, result.Err
	}
	items, err := extractServers(page)
	if err != nil {
		return nil, err
	}
	return singleByName(value, items, func(item servers.Server) string { return item.Name })
}

var serverStatusPollInterval = time.Second

func waitForServerStatus(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, serverID string, actionLabel string, targets []string, failures []string) error {
	if actionLabel == "" {
		actionLabel = "Waiting"
	}
	deadline := time.Now().Add(30 * time.Minute)
	target := upperStringSet(targets)
	failed := upperStringSet(failures)
	prettyProgress := 0.0
	prettyProgressLineOpen := false
	for {
		raw, err := computeServerRaw(ctx, client, serverID)
		if err != nil {
			if prettyProgressLineOpen {
				_ = finishPrettyProgressLine(stdout)
			}
			return err
		}
		status := strings.ToUpper(stringValue(raw["status"]))
		if target[status] && serverRawTaskStateCleared(raw) {
			if prettyOutput(opts) {
				_ = renderPrettyProgressAnimated(stdout, opts, "Complete", prettyProgress, 1, true)
			}
			return nil
		}
		if failed[status] {
			if prettyProgressLineOpen {
				_ = finishPrettyProgressLine(stdout)
			}
			return fmt.Errorf("server %s entered %s status", serverID, status)
		}
		if time.Now().After(deadline) {
			if prettyProgressLineOpen {
				_ = finishPrettyProgressLine(stdout)
			}
			return fmt.Errorf("timed out waiting for server %s", serverID)
		}
		if prettyOutput(opts) {
			previousProgress := prettyProgress
			progress, _ := intFromAny(raw["progress"])
			prettyProgress = nextPrettyWaitProgress(progress, prettyProgress)
			_ = renderPrettyProgressAnimated(stdout, opts, actionLabel, previousProgress, prettyProgress, false)
			prettyProgressLineOpen = true
		}
		select {
		case <-ctx.Done():
			if prettyProgressLineOpen {
				_ = finishPrettyProgressLine(stdout)
			}
			return ctx.Err()
		case <-time.After(serverStatusPollInterval):
		}
	}
}

func serverRawTaskStateCleared(raw map[string]any) bool {
	value, ok := raw["OS-EXT-STS:task_state"]
	return !ok || emptyServerShowValue(value)
}

func nextPrettyWaitProgress(reportedPercent int, current float64) float64 {
	next := current + 0.05
	if reported := float64(reportedPercent) / 100; reported > next {
		next = reported
	}
	if next <= 0 {
		next = 0.05
	}
	if next >= 1 {
		return 0.95
	}
	return next
}

func renderWaitComplete(stdout io.Writer, opts *Options) {
	if prettyOutput(opts) {
		return
	}
	fmt.Fprintln(stdout, "Complete")
}

func waitForServerGone(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, serverID string) error {
	deadline := time.Now().Add(30 * time.Minute)
	prettyProgress := 0.0
	prettyProgressLineOpen := false
	for {
		result := servers.Get(ctx, client, serverID)
		if result.Err != nil {
			if codeErr, ok := unexpectedResponseCode(result.Err); ok && codeErr.Actual == http.StatusNotFound {
				if prettyOutput(opts) {
					_ = renderPrettyProgressAnimated(stdout, opts, "Deleted", prettyProgress, 1, true)
				}
				return nil
			}
			if prettyProgressLineOpen {
				_ = finishPrettyProgressLine(stdout)
			}
			return result.Err
		}
		if time.Now().After(deadline) {
			if prettyProgressLineOpen {
				_ = finishPrettyProgressLine(stdout)
			}
			return fmt.Errorf("timed out waiting for server %s delete", serverID)
		}
		if prettyOutput(opts) {
			previousProgress := prettyProgress
			prettyProgress = nextPrettyWaitProgress(0, prettyProgress)
			_ = renderPrettyProgressAnimated(stdout, opts, "Deleting", previousProgress, prettyProgress, false)
			prettyProgressLineOpen = true
		}
		select {
		case <-ctx.Done():
			if prettyProgressLineOpen {
				_ = finishPrettyProgressLine(stdout)
			}
			return ctx.Err()
		case <-time.After(serverStatusPollInterval):
		}
	}
}

func waitForImageStatus(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, imageID string, targets []string, failures []string) error {
	deadline := time.Now().Add(30 * time.Minute)
	target := lowerStringSet(targets)
	failed := lowerStringSet(failures)
	for {
		image, err := images.Get(ctx, client, imageID).Extract()
		if err != nil {
			return err
		}
		status := strings.ToLower(string(image.Status))
		if target[status] {
			if prettyOutput(opts) {
				_ = renderPrettyProgress(stdout, opts, "complete", 1)
			}
			return nil
		}
		if failed[status] {
			return fmt.Errorf("image %s entered %s status", imageID, image.Status)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for image %s", imageID)
		}
		if prettyOutput(opts) {
			_ = renderPrettyProgress(stdout, opts, "waiting", 0.5)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func upperStringSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[strings.ToUpper(value)] = true
	}
	return result
}

func lowerStringSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[strings.ToLower(value)] = true
	}
	return result
}

func parseStringMap(values []string, option string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := map[string]string{}
	for _, value := range values {
		key, raw, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid %s %q, expected <key>=<value>", option, value)
		}
		result[strings.TrimSpace(key)] = raw
	}
	return result, nil
}

func volumeSchedulerHints(values []string) (volumes.SchedulerHintOptsBuilder, error) {
	if len(values) == 0 {
		return nil, nil
	}
	hints := volumes.SchedulerHintOpts{AdditionalProperties: map[string]any{}}
	for _, value := range values {
		key, raw, ok := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid hint %q, expected <key>=<value>", value)
		}
		switch key {
		case "same_host":
			hints.SameHost = append(hints.SameHost, raw)
		case "different_host":
			hints.DifferentHost = append(hints.DifferentHost, raw)
		case "local_to_instance":
			hints.LocalToInstance = raw
		case "query":
			hints.Query = raw
		default:
			hints.AdditionalProperties[key] = raw
		}
	}
	if len(hints.AdditionalProperties) == 0 {
		hints.AdditionalProperties = nil
	}
	return hints, nil
}

func blockStoragePutAction(ctx context.Context, client *gophercloud.ServiceClient, requestURL string, body any, okCodes ...int) error {
	if len(okCodes) == 0 {
		okCodes = []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}
	}
	resp, err := client.Put(ctx, requestURL, body, nil, &gophercloud.RequestOpts{OkCodes: okCodes})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return oscHTTPException(err)
}

func blockStoragePostAction(ctx context.Context, client *gophercloud.ServiceClient, requestURL string, body any, okCodes ...int) error {
	if len(okCodes) == 0 {
		okCodes = []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}
	}
	resp, err := client.Post(ctx, requestURL, body, nil, &gophercloud.RequestOpts{OkCodes: okCodes})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return oscHTTPException(err)
}

func getBlockStorageResourceMap(ctx context.Context, client *gophercloud.ServiceClient, requestURL string, key string) (map[string]any, error) {
	response := map[string]any{}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, oscHTTPException(err)
	}
	if key == "" {
		return response, nil
	}
	item, ok := response[key].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("response did not contain %q object", key)
	}
	return item, nil
}

func postBlockStorageResourceMap(ctx context.Context, client *gophercloud.ServiceClient, requestURL string, body any, key string) (map[string]any, error) {
	response := map[string]any{}
	resp, err := client.Post(ctx, requestURL, body, &response, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, oscHTTPException(err)
	}
	if key == "" {
		return response, nil
	}
	item, ok := response[key].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("response did not contain %q object", key)
	}
	return item, nil
}

func blockStorageDeleteAction(ctx context.Context, client *gophercloud.ServiceClient, requestURL string, okCodes ...int) error {
	if len(okCodes) == 0 {
		okCodes = []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}
	}
	resp, err := client.Delete(ctx, requestURL, &gophercloud.RequestOpts{OkCodes: okCodes})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return oscHTTPException(err)
}

func blockStorageDeleteMetadataKey(ctx context.Context, client *gophercloud.ServiceClient, resource string, id string, key string) error {
	return blockStorageDeleteAction(ctx, client, client.ServiceURL(resource, url.PathEscape(id), "metadata", url.PathEscape(key)), http.StatusOK, http.StatusAccepted, http.StatusNoContent)
}

func volumeSetReadOnly(ctx context.Context, client *gophercloud.ServiceClient, id string, readOnly bool) error {
	return blockStoragePostAction(ctx, client, client.ServiceURL("volumes", url.PathEscape(id), "action"), map[string]any{
		"os-update_readonly_flag": map[string]any{"readonly": readOnly},
	}, http.StatusOK, http.StatusAccepted, http.StatusNoContent)
}

func volumeUnsetImageMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string, key string) error {
	return blockStoragePostAction(ctx, client, client.ServiceURL("volumes", url.PathEscape(id), "action"), map[string]any{
		"os-unset_image_metadata": key,
	}, http.StatusOK, http.StatusAccepted, http.StatusNoContent)
}

func snapshotUnmanage(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
	return blockStoragePostAction(ctx, client, client.ServiceURL("snapshots", url.PathEscape(id), "action"), map[string]any{
		"os-unmanage": nil,
	}, http.StatusOK, http.StatusAccepted, http.StatusNoContent)
}

func valueStringPtr(value string) *string {
	return &value
}

func valueBoolPtr(value bool) *bool {
	return &value
}

func serverCreateNetworks(ctx context.Context, opts *Options, networkClient *gophercloud.ServiceClient) ([]servers.Network, any, error) {
	if boolFlag(opts, "no-network") {
		return nil, "none", nil
	}
	if boolFlag(opts, "auto-network") {
		return nil, "auto", nil
	}
	var result []servers.Network
	for _, value := range flagValues(opts, "network") {
		networkID := value
		if networkClient != nil {
			if network, err := findNetwork(ctx, networkClient, value); err == nil {
				networkID = network.ID
			}
		}
		result = append(result, servers.Network{UUID: networkID})
	}
	for _, value := range flagValues(opts, "port") {
		portID := value
		if networkClient != nil {
			if port, err := findPort(ctx, networkClient, value); err == nil {
				portID = port.ID
			}
		}
		result = append(result, servers.Network{Port: portID})
	}
	for _, value := range flagValues(opts, "nic") {
		trimmed := strings.TrimSpace(value)
		if trimmed == "none" || trimmed == "auto" {
			return nil, trimmed, nil
		}
		entry, err := parseSingleKeyValueEntry(value, "nic")
		if err != nil {
			return nil, nil, err
		}
		network := servers.Network{UUID: entry["net-id"], Port: entry["port-id"], Tag: entry["tag"]}
		if network.UUID != "" && networkClient != nil {
			if resolved, err := findNetwork(ctx, networkClient, network.UUID); err == nil {
				network.UUID = resolved.ID
			}
		}
		if network.Port != "" && networkClient != nil {
			if resolved, err := findPort(ctx, networkClient, network.Port); err == nil {
				network.Port = resolved.ID
			}
		}
		if ip := entry["v4-fixed-ip"]; ip != "" {
			network.FixedIP = ip
		}
		if ip := entry["v6-fixed-ip"]; ip != "" {
			network.FixedIP = ip
		}
		result = append(result, network)
	}
	return result, nil, nil
}

func serverCreateBootSource(ctx context.Context, opts *Options, imageClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient) ([]servers.BlockDevice, string, error) {
	var blockDevices []servers.BlockDevice
	imageID := ""
	var err error
	if imageValue := flagValue(opts, "image"); imageValue != "" {
		imageID, err = resolveServerImageID(ctx, imageClient, imageValue)
		if err != nil {
			return nil, "", err
		}
	}
	if imageProperty := flagValue(opts, "image-property"); imageProperty != "" {
		imageID, err = resolveServerImageByProperty(ctx, imageClient, imageProperty)
		if err != nil {
			return nil, "", err
		}
	}
	if flagChanged(opts, "boot-from-volume") {
		if imageID == "" {
			return nil, "", fmt.Errorf("--boot-from-volume requires --image or --image-property")
		}
		blockDevices = append(blockDevices, servers.BlockDevice{
			SourceType:          servers.SourceImage,
			DestinationType:     servers.DestinationVolume,
			UUID:                imageID,
			BootIndex:           0,
			VolumeSize:          intFlag(opts, "boot-from-volume"),
			DeleteOnTermination: false,
		})
		imageID = ""
	}
	if volumeValue := flagValue(opts, "volume"); volumeValue != "" {
		volumeID := volumeValue
		if volumeClient != nil {
			volume, err := findVolume(ctx, volumeClient, volumeValue)
			if err != nil {
				return nil, "", err
			}
			volumeID = volume.ID
		}
		blockDevices = append(blockDevices, servers.BlockDevice{SourceType: servers.SourceVolume, DestinationType: servers.DestinationVolume, UUID: volumeID, BootIndex: 0})
		imageID = ""
	}
	if snapshotValue := flagValue(opts, "snapshot"); snapshotValue != "" {
		snapshotID := snapshotValue
		if volumeClient != nil {
			snapshot, err := findVolumeSnapshot(ctx, volumeClient, snapshotValue)
			if err != nil {
				return nil, "", err
			}
			snapshotID = snapshot.ID
		}
		blockDevices = append(blockDevices, servers.BlockDevice{SourceType: servers.SourceSnapshot, DestinationType: servers.DestinationVolume, UUID: snapshotID, BootIndex: 0})
		imageID = ""
	}
	for _, value := range flagValues(opts, "block-device") {
		devices, err := serverBlockDevicesFromValue(value)
		if err != nil {
			return nil, "", err
		}
		blockDevices = append(blockDevices, devices...)
	}
	for _, value := range flagValues(opts, "block-device-mapping") {
		device, err := serverBlockDeviceMapping(value)
		if err != nil {
			return nil, "", err
		}
		blockDevices = append(blockDevices, device)
	}
	if flagChanged(opts, "swap") {
		blockDevices = append(blockDevices, servers.BlockDevice{SourceType: servers.SourceBlank, DestinationType: servers.DestinationLocal, GuestFormat: "swap", VolumeSize: intFlag(opts, "swap"), BootIndex: -1})
	}
	for _, value := range flagValues(opts, "ephemeral") {
		device, err := serverEphemeralBlockDevice(value)
		if err != nil {
			return nil, "", err
		}
		blockDevices = append(blockDevices, device)
	}
	return blockDevices, imageID, nil
}

func resolveServerImageID(ctx context.Context, imageClient *gophercloud.ServiceClient, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if imageClient == nil {
		return value, nil
	}
	image, err := findImage(ctx, imageClient, value)
	if err != nil {
		return "", err
	}
	return image.ID, nil
}

func resolveServerImageByProperty(ctx context.Context, imageClient *gophercloud.ServiceClient, property string) (string, error) {
	if imageClient == nil {
		return "", fmt.Errorf("--image-property requires image service access")
	}
	key, value, ok := strings.Cut(property, "=")
	if !ok || strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("invalid image property %q, expected <key>=<value>", property)
	}
	page, err := images.List(imageClient, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		return "", err
	}
	items, err := images.ExtractImages(page)
	if err != nil {
		return "", err
	}
	var matches []images.Image
	for _, item := range items {
		if valueString(item.Properties[strings.TrimSpace(key)]) == value {
			matches = append(matches, item)
		}
	}
	item, err := singleMatch(property, matches)
	if err != nil {
		return "", err
	}
	return item.ID, nil
}

func serverPersonality(values []string) (servers.Personality, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := servers.Personality{}
	for _, value := range values {
		dest, source, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(dest) == "" || strings.TrimSpace(source) == "" {
			return nil, fmt.Errorf("invalid file %q, expected <dest>=<source>", value)
		}
		contents, err := os.ReadFile(expandUserPath(source))
		if err != nil {
			return nil, err
		}
		result = append(result, &servers.File{Path: dest, Contents: contents})
	}
	return result, nil
}

func serverSchedulerHints(ctx context.Context, opts *Options, client *gophercloud.ServiceClient) (servers.SchedulerHintOpts, error) {
	values, err := parseJSONKeyValueMap(flagValues(opts, "hint"), "hint")
	if err != nil {
		return servers.SchedulerHintOpts{}, err
	}
	hints := servers.SchedulerHintOpts{AdditionalProperties: values}
	if groupValue := flagValue(opts, "server-group"); groupValue != "" {
		group, err := findServerGroup(ctx, client, groupValue)
		if err != nil {
			return hints, err
		}
		hints.Group = group.ID
	}
	return hints, nil
}

func parseSingleKeyValueEntry(value string, option string) (map[string]string, error) {
	entries, err := parseKeyValueEntries([]string{value}, option)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return map[string]string{}, nil
	}
	return entries[0], nil
}

func serverBlockDevicesFromValue(value string) ([]servers.BlockDevice, error) {
	trimmed := strings.TrimSpace(value)
	var raw any
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
			return nil, err
		}
	} else if data, err := os.ReadFile(expandUserPath(trimmed)); err == nil {
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
	} else {
		entry, err := parseSingleKeyValueEntry(value, "block-device")
		if err != nil {
			return nil, err
		}
		return []servers.BlockDevice{serverBlockDeviceFromMap(entry)}, nil
	}
	switch typed := raw.(type) {
	case []any:
		devices := make([]servers.BlockDevice, 0, len(typed))
		for _, item := range typed {
			devices = append(devices, serverBlockDeviceFromAnyMap(mapAnyFromRaw(item)))
		}
		return devices, nil
	case map[string]any:
		return []servers.BlockDevice{serverBlockDeviceFromAnyMap(typed)}, nil
	default:
		return nil, fmt.Errorf("invalid block device %q", value)
	}
}

func serverBlockDeviceFromMap(entry map[string]string) servers.BlockDevice {
	anyEntry := map[string]any{}
	for key, value := range entry {
		anyEntry[key] = value
	}
	return serverBlockDeviceFromAnyMap(anyEntry)
}

func serverBlockDeviceFromAnyMap(entry map[string]any) servers.BlockDevice {
	device := servers.BlockDevice{
		UUID:            valueString(entry["uuid"]),
		SourceType:      servers.SourceType(valueString(entry["source_type"])),
		DestinationType: servers.DestinationType(valueString(entry["destination_type"])),
		GuestFormat:     valueString(entry["guest_format"]),
		DeviceType:      valueString(entry["device_type"]),
		DiskBus:         valueString(entry["disk_bus"]),
		VolumeType:      valueString(entry["volume_type"]),
		Tag:             valueString(entry["tag"]),
	}
	if value := valueString(entry["boot_index"]); value != "" {
		device.BootIndex, _ = strconv.Atoi(value)
	}
	if value := valueString(entry["volume_size"]); value != "" {
		device.VolumeSize, _ = strconv.Atoi(value)
	}
	if value := valueString(entry["delete_on_termination"]); value != "" {
		device.DeleteOnTermination = truthyString(value)
	}
	if device.SourceType == "" {
		device.SourceType = servers.SourceVolume
	}
	if device.DestinationType == "" {
		device.DestinationType = servers.DestinationVolume
	}
	return device
}

func serverBlockDeviceMapping(value string) (servers.BlockDevice, error) {
	_, mapping, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(mapping) == "" {
		return servers.BlockDevice{}, fmt.Errorf("invalid block device mapping %q", value)
	}
	parts := strings.Split(mapping, ":")
	if len(parts) < 1 || strings.TrimSpace(parts[0]) == "" {
		return servers.BlockDevice{}, fmt.Errorf("invalid block device mapping %q", value)
	}
	device := servers.BlockDevice{UUID: parts[0], SourceType: servers.SourceVolume, DestinationType: servers.DestinationVolume, BootIndex: -1}
	if len(parts) > 1 && parts[1] != "" {
		device.SourceType = servers.SourceType(parts[1])
	}
	if len(parts) > 2 && parts[2] != "" {
		device.VolumeSize, _ = strconv.Atoi(parts[2])
	}
	if len(parts) > 3 && parts[3] != "" {
		device.DeleteOnTermination = truthyString(parts[3])
	}
	return device, nil
}

func serverEphemeralBlockDevice(value string) (servers.BlockDevice, error) {
	entry, err := parseSingleKeyValueEntry(value, "ephemeral")
	if err != nil {
		return servers.BlockDevice{}, err
	}
	size, _ := strconv.Atoi(entry["size"])
	return servers.BlockDevice{SourceType: servers.SourceBlank, DestinationType: servers.DestinationLocal, GuestFormat: entry["format"], VolumeSize: size, BootIndex: -1}, nil
}

func truthyString(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func serverPortForFloatingIP(ctx context.Context, networkClient *gophercloud.ServiceClient, serverID string, fixedIP string) (string, string, error) {
	opts := ports.ListOpts{DeviceID: serverID}
	if fixedIP != "" {
		opts.FixedIPs = []ports.FixedIPOpts{{IPAddress: fixedIP}}
	}
	page, err := ports.List(networkClient, opts).AllPages(ctx)
	if err != nil {
		return "", "", err
	}
	items, err := ports.ExtractPorts(page)
	if err != nil {
		return "", "", err
	}
	if len(items) == 0 {
		return "", "", fmt.Errorf("No ports found for server %s", serverID)
	}
	for _, port := range items {
		if fixedIP == "" {
			if len(port.FixedIPs) > 0 {
				return port.ID, port.FixedIPs[0].IPAddress, nil
			}
			return port.ID, "", nil
		}
		for _, ip := range port.FixedIPs {
			if ip.IPAddress == fixedIP {
				return port.ID, fixedIP, nil
			}
		}
	}
	return "", "", fmt.Errorf("No port found for server %s fixed IP %s", serverID, fixedIP)
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
