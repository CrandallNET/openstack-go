package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/spf13/cobra"
)

type neutronExtraResource struct {
	Singular string
	Plural   string
	Path     []string
	Columns  []string
	Keys     []string
	Hidden   map[string]bool
}

func isNeutronExtrasCommand(path string) bool {
	_, ok := neutronExtraCommands[path]
	return ok
}

var neutronExtraCommands = map[string]struct{}{
	"default security group rule create":     {},
	"default security group rule delete":     {},
	"default security group rule list":       {},
	"default security group rule show":       {},
	"local ip association create":            {},
	"local ip association delete":            {},
	"local ip association list":              {},
	"local ip create":                        {},
	"local ip delete":                        {},
	"local ip list":                          {},
	"local ip set":                           {},
	"local ip show":                          {},
	"network agent add network":              {},
	"network agent add router":               {},
	"network agent delete":                   {},
	"network agent remove network":           {},
	"network agent remove router":            {},
	"network agent set":                      {},
	"network auto allocated topology create": {},
	"network auto allocated topology delete": {},
	"network flavor add profile":             {},
	"network flavor create":                  {},
	"network flavor delete":                  {},
	"network flavor list":                    {},
	"network flavor profile create":          {},
	"network flavor profile delete":          {},
	"network flavor profile list":            {},
	"network flavor profile set":             {},
	"network flavor profile show":            {},
	"network flavor remove profile":          {},
	"network flavor set":                     {},
	"network flavor show":                    {},
	"network l3 conntrack helper create":     {},
	"network l3 conntrack helper delete":     {},
	"network l3 conntrack helper list":       {},
	"network l3 conntrack helper set":        {},
	"network l3 conntrack helper show":       {},
	"network meter create":                   {},
	"network meter delete":                   {},
	"network meter list":                     {},
	"network meter rule create":              {},
	"network meter rule delete":              {},
	"network meter rule list":                {},
	"network meter rule show":                {},
	"network meter show":                     {},
	"network segment range create":           {},
	"network segment range delete":           {},
	"network segment range list":             {},
	"network segment range set":              {},
	"network segment range show":             {},
	"router ndp proxy create":                {},
	"router ndp proxy delete":                {},
	"router ndp proxy list":                  {},
	"router ndp proxy set":                   {},
	"router ndp proxy show":                  {},
	"tap flow create":                        {},
	"tap flow delete":                        {},
	"tap flow list":                          {},
	"tap flow show":                          {},
	"tap flow update":                        {},
	"tap mirror create":                      {},
	"tap mirror delete":                      {},
	"tap mirror list":                        {},
	"tap mirror show":                        {},
	"tap mirror update":                      {},
	"tap service create":                     {},
	"tap service delete":                     {},
	"tap service list":                       {},
	"tap service show":                       {},
	"tap service update":                     {},
}

func runNeutronExtras(path string, stdout io.Writer, opts *Options) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		clients, err := newOpenStackClients(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := clients.networkV2()
		if err != nil {
			return err
		}

		switch path {
		case "default security group rule create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronDefaultSecurityGroupRuleSpec(), defaultSecurityGroupRuleValues(opts, args))
		case "default security group rule delete":
			return neutronExtraDelete(cmd.Context(), client, neutronDefaultSecurityGroupRuleSpec(), args)
		case "default security group rule list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronDefaultSecurityGroupRuleSpec(), defaultSecurityGroupRuleQuery(opts))
		case "default security group rule show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronDefaultSecurityGroupRuleSpec(), args)
		case "local ip create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronLocalIPSpec(), localIPValues(cmd.Context(), opts, clients, client, args))
		case "local ip delete":
			return neutronExtraDelete(cmd.Context(), client, neutronLocalIPSpec(), args)
		case "local ip list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronLocalIPSpec(), localIPQuery(cmd.Context(), opts, clients, client))
		case "local ip set":
			return neutronExtraSet(cmd.Context(), opts, clients, client, neutronLocalIPSpec(), args, localIPSetValues(opts))
		case "local ip show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronLocalIPSpec(), args)
		case "local ip association create":
			return localIPAssociationCreate(cmd.Context(), stdout, opts, client, args)
		case "local ip association delete":
			return localIPAssociationDelete(cmd.Context(), client, args)
		case "local ip association list":
			return localIPAssociationList(cmd.Context(), stdout, opts, client, args)
		case "network agent add network":
			return networkAgentAddNetwork(cmd.Context(), opts, client, args)
		case "network agent add router":
			return networkAgentAddRouter(cmd.Context(), opts, client, args)
		case "network agent delete":
			return neutronExtraDelete(ctxNoLookup(cmd.Context()), client, neutronAgentSpec(), args)
		case "network agent remove network":
			return networkAgentRemoveNetwork(cmd.Context(), opts, client, args)
		case "network agent remove router":
			return networkAgentRemoveRouter(cmd.Context(), opts, client, args)
		case "network agent set":
			return neutronExtraSet(cmd.Context(), opts, clients, client, neutronAgentSpec(), args, networkAgentSetValues(opts))
		case "network auto allocated topology create":
			return autoAllocatedTopologyCreate(cmd.Context(), stdout, opts, clients, client)
		case "network auto allocated topology delete":
			return autoAllocatedTopologyDelete(cmd.Context(), opts, clients, client)
		case "network flavor add profile":
			return networkFlavorProfileAssociation(cmd.Context(), client, args, true)
		case "network flavor create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronFlavorSpec(), networkFlavorValues(cmd.Context(), opts, clients, args))
		case "network flavor delete":
			return neutronExtraDelete(cmd.Context(), client, neutronFlavorSpec(), args)
		case "network flavor list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronFlavorSpec(), nil)
		case "network flavor profile create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronServiceProfileSpec(), networkFlavorProfileValues(opts))
		case "network flavor profile delete":
			return neutronExtraDelete(cmd.Context(), client, neutronServiceProfileSpec(), args)
		case "network flavor profile list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronServiceProfileSpec(), nil)
		case "network flavor profile set":
			return neutronExtraSet(cmd.Context(), opts, clients, client, neutronServiceProfileSpec(), args, networkFlavorProfileValues(opts))
		case "network flavor profile show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronServiceProfileSpec(), args)
		case "network flavor remove profile":
			return networkFlavorProfileAssociation(cmd.Context(), client, args, false)
		case "network flavor set":
			return neutronExtraSet(cmd.Context(), opts, clients, client, neutronFlavorSpec(), args, networkFlavorSetValues(opts))
		case "network flavor show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronFlavorSpec(), args)
		case "network l3 conntrack helper create":
			return conntrackHelperCreate(cmd.Context(), stdout, opts, client, args)
		case "network l3 conntrack helper delete":
			return conntrackHelperDelete(cmd.Context(), client, args)
		case "network l3 conntrack helper list":
			return conntrackHelperList(cmd.Context(), stdout, opts, client, args)
		case "network l3 conntrack helper set":
			return conntrackHelperSet(cmd.Context(), opts, client, args)
		case "network l3 conntrack helper show":
			return conntrackHelperShow(cmd.Context(), stdout, opts, client, args)
		case "network meter create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronMeterSpec(), meterValues(cmd.Context(), opts, clients, args))
		case "network meter delete":
			return neutronExtraDelete(cmd.Context(), client, neutronMeterSpec(), args)
		case "network meter list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronMeterSpec(), nil)
		case "network meter show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronMeterSpec(), args)
		case "network meter rule create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronMeterRuleSpec(), meterRuleValues(cmd.Context(), opts, clients, client, args))
		case "network meter rule delete":
			return neutronExtraDelete(ctxNoLookup(cmd.Context()), client, neutronMeterRuleSpec(), args)
		case "network meter rule list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronMeterRuleSpec(), nil)
		case "network meter rule show":
			return neutronExtraShow(ctxNoLookup(cmd.Context()), stdout, opts, client, neutronMeterRuleSpec(), args)
		case "network segment range create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronSegmentRangeSpec(), segmentRangeValues(cmd.Context(), opts, clients, args))
		case "network segment range delete":
			return neutronExtraDelete(cmd.Context(), client, neutronSegmentRangeSpec(), args)
		case "network segment range list":
			return segmentRangeList(cmd.Context(), stdout, opts, clients, client)
		case "network segment range set":
			return neutronExtraSet(cmd.Context(), opts, clients, client, neutronSegmentRangeSpec(), args, segmentRangeSetValues(opts))
		case "network segment range show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronSegmentRangeSpec(), args)
		case "router ndp proxy create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronNDPProxySpec(), ndpProxyValues(cmd.Context(), opts, client, args))
		case "router ndp proxy delete":
			return neutronExtraDelete(cmd.Context(), client, neutronNDPProxySpec(), args)
		case "router ndp proxy list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronNDPProxySpec(), ndpProxyQuery(cmd.Context(), opts, clients, client))
		case "router ndp proxy set":
			return neutronExtraSet(cmd.Context(), opts, clients, client, neutronNDPProxySpec(), args, ndpProxySetValues(opts))
		case "router ndp proxy show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronNDPProxySpec(), args)
		case "tap service create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronTapServiceSpec(), tapServiceValues(cmd.Context(), opts, clients, client))
		case "tap service delete":
			return neutronExtraDelete(cmd.Context(), client, neutronTapServiceSpec(), args)
		case "tap service list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronTapServiceSpec(), projectOnlyQuery(cmd.Context(), opts, clients))
		case "tap service show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronTapServiceSpec(), args)
		case "tap service update":
			return neutronExtraUpdateShow(cmd.Context(), stdout, opts, clients, client, neutronTapServiceSpec(), args, tapUpdateValues(opts))
		case "tap flow create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronTapFlowSpec(), tapFlowValues(cmd.Context(), opts, clients, client))
		case "tap flow delete":
			return neutronExtraDelete(cmd.Context(), client, neutronTapFlowSpec(), args)
		case "tap flow list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronTapFlowSpec(), projectOnlyQuery(cmd.Context(), opts, clients))
		case "tap flow show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronTapFlowSpec(), args)
		case "tap flow update":
			return neutronExtraUpdateShow(cmd.Context(), stdout, opts, clients, client, neutronTapFlowSpec(), args, tapUpdateValues(opts))
		case "tap mirror create":
			return neutronExtraCreate(cmd.Context(), stdout, opts, clients, client, neutronTapMirrorSpec(), tapMirrorValues(cmd.Context(), opts, clients, client))
		case "tap mirror delete":
			return neutronExtraDelete(cmd.Context(), client, neutronTapMirrorSpec(), args)
		case "tap mirror list":
			return neutronExtraList(cmd.Context(), stdout, opts, clients, client, neutronTapMirrorSpec(), projectOnlyQuery(cmd.Context(), opts, clients))
		case "tap mirror show":
			return neutronExtraShow(cmd.Context(), stdout, opts, client, neutronTapMirrorSpec(), args)
		case "tap mirror update":
			return neutronExtraUpdateShow(cmd.Context(), stdout, opts, clients, client, neutronTapMirrorSpec(), args, tapUpdateValues(opts))
		default:
			return fmt.Errorf("unsupported neutron extras command %q", path)
		}
	}
}

func ctxNoLookup(ctx context.Context) context.Context {
	return context.WithValue(ctx, neutronExtraLookupDisabled{}, true)
}

type neutronExtraLookupDisabled struct{}

func neutronDefaultSecurityGroupRuleSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "default_security_group_rule",
		Plural:   "default_security_group_rules",
		Path:     []string{"default-security-group-rules"},
		Columns:  []string{"ID", "IP Protocol", "Ethertype", "IP Range", "Port Range", "Direction", "Remote Security Group", "Remote Address Group", "Used in default Security Group", "Used in custom Security Group"},
		Keys:     []string{"id", "protocol", "ethertype", "ip_range", "port_range", "direction", "remote_group_id", "remote_address_group_id", "used_in_default_sg", "used_in_non_default_sg"},
		Hidden:   map[string]bool{"location": true, "name": true, "revision_number": true},
	}
}

func neutronLocalIPSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "local_ip",
		Plural:   "local_ips",
		Path:     []string{"local_ips"},
		Columns:  []string{"ID", "Name", "Description", "Project", "Local Port ID", "Network", "Local IP address", "IP mode"},
		Keys:     []string{"id", "name", "description", "project_id", "local_port_id", "network_id", "local_ip_address", "ip_mode"},
		Hidden:   map[string]bool{"location": true, "tenant_id": true},
	}
}

func neutronLocalIPAssociationSpec(localIPID string) neutronExtraResource {
	return neutronExtraResource{
		Singular: "port_association",
		Plural:   "port_associations",
		Path:     []string{"local_ips", localIPID, "port_associations"},
		Columns:  []string{"Local IP ID", "Local IP Address", "Fixed port ID", "Fixed IP", "Host"},
		Keys:     []string{"local_ip_id", "local_ip_address", "fixed_port_id", "fixed_ip", "host"},
		Hidden:   map[string]bool{"location": true, "name": true, "id": true, "tenant_id": true},
	}
}

func neutronAgentSpec() neutronExtraResource {
	return neutronExtraResource{Singular: "agent", Plural: "agents", Path: []string{"agents"}, Hidden: map[string]bool{"location": true, "name": true, "tenant_id": true}}
}

func neutronFlavorSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "flavor",
		Plural:   "flavors",
		Path:     []string{"flavors"},
		Columns:  []string{"ID", "Name", "Enabled", "Service Type", "Description"},
		Keys:     []string{"id", "name", "enabled", "service_type", "description"},
		Hidden:   map[string]bool{"location": true, "tenant_id": true},
	}
}

func neutronServiceProfileSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "service_profile",
		Plural:   "service_profiles",
		Path:     []string{"service_profiles"},
		Columns:  []string{"ID", "Driver", "Enabled", "Metainfo", "Description"},
		Keys:     []string{"id", "driver", "enabled", "metainfo", "description"},
		Hidden:   map[string]bool{"location": true, "name": true, "tenant_id": true, "project_id": true},
	}
}

func neutronConntrackHelperSpec(routerID string) neutronExtraResource {
	return neutronExtraResource{
		Singular: "conntrack_helper",
		Plural:   "conntrack_helpers",
		Path:     []string{"routers", routerID, "conntrack_helpers"},
		Columns:  []string{"ID", "Router ID", "Helper", "Protocol", "Port"},
		Keys:     []string{"id", "router_id", "helper", "protocol", "port"},
		Hidden:   map[string]bool{"location": true, "tenant_id": true},
	}
}

func neutronMeterSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "metering_label",
		Plural:   "metering_labels",
		Path:     []string{"metering", "metering-labels"},
		Columns:  []string{"ID", "Name", "Description", "Shared"},
		Keys:     []string{"id", "name", "description", "shared"},
		Hidden:   map[string]bool{"location": true, "tenant_id": true},
	}
}

func neutronMeterRuleSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "metering_label_rule",
		Plural:   "metering_label_rules",
		Path:     []string{"metering", "metering-label-rules"},
		Columns:  []string{"ID", "Excluded", "Direction", "Remote IP Prefix", "Source IP Prefix", "Destination IP Prefix"},
		Keys:     []string{"id", "excluded", "direction", "remote_ip_prefix", "source_ip_prefix", "destination_ip_prefix"},
		Hidden:   map[string]bool{"location": true, "tenant_id": true},
	}
}

func neutronSegmentRangeSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "network_segment_range",
		Plural:   "network_segment_ranges",
		Path:     []string{"network_segment_ranges"},
		Columns:  []string{"ID", "Name", "Default", "Shared", "Project ID", "Network Type", "Physical Network", "Minimum ID", "Maximum ID"},
		Keys:     []string{"id", "name", "default", "shared", "project_id", "network_type", "physical_network", "minimum", "maximum"},
		Hidden:   map[string]bool{"location": true, "tenant_id": true},
	}
}

func neutronNDPProxySpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "ndp_proxy",
		Plural:   "ndp_proxies",
		Path:     []string{"ndp_proxies"},
		Columns:  []string{"ID", "Name", "Router ID", "IP Address", "Project"},
		Keys:     []string{"id", "name", "router_id", "ip_address", "project_id"},
		Hidden:   map[string]bool{"location": true},
	}
}

func neutronTapServiceSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "tap_service",
		Plural:   "tap_services",
		Path:     []string{"taas", "tap_services"},
		Columns:  []string{"ID", "Tenant", "Name", "Port", "Status"},
		Keys:     []string{"id", "tenant_id", "name", "port_id", "status"},
		Hidden:   map[string]bool{"location": true, "tenant_id": true},
	}
}

func neutronTapFlowSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "tap_flow",
		Plural:   "tap_flows",
		Path:     []string{"taas", "tap_flows"},
		Columns:  []string{"ID", "Tenant", "Name", "Status", "source_port", "tap_service_id", "Direction"},
		Keys:     []string{"id", "tenant_id", "name", "status", "source_port", "tap_service_id", "direction"},
		Hidden:   map[string]bool{"location": true, "tenant_id": true},
	}
}

func neutronTapMirrorSpec() neutronExtraResource {
	return neutronExtraResource{
		Singular: "tap_mirror",
		Plural:   "tap_mirrors",
		Path:     []string{"taas", "tap_mirrors"},
		Columns:  []string{"ID", "Tenant", "Name", "Port", "Directions", "Remote IP", "Mirror Type"},
		Keys:     []string{"id", "tenant_id", "name", "port_id", "directions", "remote_ip", "mirror_type"},
		Hidden:   map[string]bool{"location": true, "tenant_id": true},
	}
}

func neutronExtraList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, spec neutronExtraResource, query map[string]any) error {
	if err := errorValue(query); err != nil {
		return err
	}
	items, err := neutronExtraListRaw(ctx, client, spec, valuesQuery(query))
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, neutronExtraRow(item, spec.Keys, spec.Columns))
	}
	return renderListOutput(stdout, opts, spec.Columns, rows)
}

func neutronExtraShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, spec neutronExtraResource, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s show requires <%s>", strings.ReplaceAll(spec.Singular, "_", " "), strings.ReplaceAll(spec.Singular, "_", "-"))
	}
	item, err := neutronExtraFindRaw(ctx, client, spec, args[0])
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, neutronExtraFields(item, spec.Hidden))
}

func neutronExtraCreate(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, spec neutronExtraResource, values map[string]any) error {
	if err := errorValue(values); err != nil {
		return err
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	for key, value := range extra {
		values[key] = value
	}
	var response map[string]any
	resp, err := client.Post(ctx, neutronExtraCollectionURL(client, spec), map[string]any{spec.Singular: values}, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	item, err := neutronExtraExtractObject(response, spec.Singular)
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, neutronExtraFields(item, spec.Hidden))
}

func neutronExtraSet(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, spec neutronExtraResource, args []string, values map[string]any) error {
	if err := errorValue(values); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("%s set requires <%s>", strings.ReplaceAll(spec.Singular, "_", " "), strings.ReplaceAll(spec.Singular, "_", "-"))
	}
	extra, err := parseExtraProperties(flagValues(opts, "extra-property"))
	if err != nil {
		return err
	}
	for key, value := range extra {
		values[key] = value
	}
	item, err := neutronExtraFindRaw(ctx, client, spec, args[0])
	if err != nil {
		return err
	}
	id := valueString(item["id"])
	if id == "" {
		id = args[0]
	}
	resp, err := client.Put(ctx, neutronExtraItemURL(client, spec, id), map[string]any{spec.Singular: values}, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func neutronExtraUpdateShow(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, spec neutronExtraResource, args []string, values map[string]any) error {
	if err := errorValue(values); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("%s update requires <%s>", strings.ReplaceAll(spec.Singular, "_", " "), strings.ReplaceAll(spec.Singular, "_", "-"))
	}
	item, err := neutronExtraFindRaw(ctx, client, spec, args[0])
	if err != nil {
		return err
	}
	id := valueString(item["id"])
	if id == "" {
		id = args[0]
	}
	var response map[string]any
	resp, err := client.Put(ctx, neutronExtraItemURL(client, spec, id), map[string]any{spec.Singular: values}, &response, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	if len(response) == 0 {
		item, err = neutronExtraFindRaw(ctx, client, spec, id)
		if err != nil {
			return err
		}
	} else {
		item, err = neutronExtraExtractObject(response, spec.Singular)
		if err != nil {
			return err
		}
	}
	return renderShowOutput(stdout, opts, neutronExtraFields(item, spec.Hidden))
}

func neutronExtraDelete(ctx context.Context, client *gophercloud.ServiceClient, spec neutronExtraResource, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s delete requires at least one <%s>", strings.ReplaceAll(spec.Singular, "_", " "), strings.ReplaceAll(spec.Singular, "_", "-"))
	}
	var failures []string
	for _, arg := range args {
		id := arg
		if ctx.Value(neutronExtraLookupDisabled{}) == nil {
			item, err := neutronExtraFindRaw(ctx, client, spec, arg)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", arg, err))
				continue
			}
			if found := valueString(item["id"]); found != "" {
				id = found
			}
		}
		resp, err := client.Delete(ctx, neutronExtraItemURL(client, spec, id), &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
		_, _, err = gophercloud.ParseResponse(resp, err)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", arg, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%d of %d deletes failed: %s", len(failures), len(args), strings.Join(failures, "; "))
	}
	return nil
}

func neutronExtraFindRaw(ctx context.Context, client *gophercloud.ServiceClient, spec neutronExtraResource, value string) (map[string]any, error) {
	var response map[string]any
	resp, err := client.Get(ctx, neutronExtraItemURL(client, spec, value), &response, nil)
	_, _, parsedErr := gophercloud.ParseResponse(resp, err)
	if parsedErr == nil {
		return neutronExtraExtractObject(response, spec.Singular)
	}
	if !isNotFound(parsedErr) {
		return nil, parsedErr
	}
	items, err := neutronExtraListRaw(ctx, client, spec, "")
	if err != nil {
		return nil, err
	}
	var matches []map[string]any
	for _, item := range items {
		if valueString(item["id"]) == value || valueString(item["name"]) == value {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple %s matches found for %q", strings.ReplaceAll(spec.Singular, "_", " "), value)
	}
	return nil, fmt.Errorf("No %s with a name or ID of '%s' exists.", strings.ReplaceAll(spec.Singular, "_", " "), value)
}

func neutronExtraListRaw(ctx context.Context, client *gophercloud.ServiceClient, spec neutronExtraResource, query string) ([]map[string]any, error) {
	requestURL := neutronExtraCollectionURL(client, spec) + query
	var response map[string]any
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	rawItems := anySlice(response[spec.Plural])
	items := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		if item, ok := raw.(map[string]any); ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func neutronExtraExtractObject(response map[string]any, key string) (map[string]any, error) {
	if item, ok := response[key].(map[string]any); ok {
		return item, nil
	}
	return nil, fmt.Errorf("expected %q object in Neutron response", key)
}

func neutronExtraCollectionURL(client *gophercloud.ServiceClient, spec neutronExtraResource) string {
	return client.ServiceURL(escapedSegments(spec.Path)...)
}

func neutronExtraItemURL(client *gophercloud.ServiceClient, spec neutronExtraResource, id string) string {
	segments := append([]string{}, spec.Path...)
	segments = append(segments, id)
	return client.ServiceURL(escapedSegments(segments)...)
}

func escapedSegments(values []string) []string {
	segments := make([]string, len(values))
	for i, value := range values {
		segments[i] = url.PathEscape(value)
	}
	return segments
}

func neutronExtraRow(item map[string]any, keys []string, columns []string) outputRow {
	row := outputRow{}
	for i, column := range columns {
		key := keys[i]
		if key == "port_range" {
			row[column] = neutronPortRangeValue(item)
			continue
		}
		if key == "ip_range" {
			row[column] = neutronDefaultSecurityGroupRuleIPRange(item)
			continue
		}
		row[column] = mapValueOrEmpty(item, key)
	}
	return row
}

func neutronExtraFields(item map[string]any, hidden map[string]bool) []outputField {
	fields := sortedFieldsFromMap(item, false)
	values := fields[:0]
	for _, field := range fields {
		if hidden != nil && hidden[field.Name] {
			continue
		}
		if field.Name == "port_range_min" || field.Name == "port_range_max" {
			continue
		}
		if field.Name == "enabled" {
			field.Name = "is_enabled"
		}
		if field.Name == "shared" {
			field.Name = "is_shared"
		}
		values = append(values, field)
	}
	if _, ok := item["port_range_min"]; ok {
		values = append(values, outputField{"port_range", neutronPortRangeValue(item)})
	}
	sort.SliceStable(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return values
}

func neutronPortRangeValue(item map[string]any) string {
	min := valueString(item["port_range_min"])
	max := valueString(item["port_range_max"])
	if min == "" || min == "None" {
		min = ""
	}
	if max == "" || max == "None" {
		max = ""
	}
	if min == "" && max == "" {
		return ""
	}
	if min == max || max == "" {
		return min
	}
	if min == "" {
		return max
	}
	return min + ":" + max
}

func neutronDefaultSecurityGroupRuleIPRange(item map[string]any) any {
	if value, ok := item["remote_ip_prefix"]; ok && value != nil {
		return value
	}
	switch valueString(item["ethertype"]) {
	case "IPv6":
		return "::/0"
	default:
		return "0.0.0.0/0"
	}
}

func valuesQuery(values map[string]any) string {
	if len(values) == 0 {
		return ""
	}
	query := url.Values{}
	for key, value := range values {
		if value == nil {
			continue
		}
		if err, ok := value.(error); ok {
			query.Set("__error__", err.Error())
			continue
		}
		text := valueString(value)
		if text != "" {
			query.Set(key, text)
		}
	}
	if len(query) == 0 {
		return ""
	}
	return "?" + query.Encode()
}

func errorValue(values map[string]any) error {
	for _, value := range values {
		if err, ok := value.(error); ok {
			return err
		}
	}
	return nil
}

func isNotFound(err error) bool {
	var codeErr gophercloud.ErrUnexpectedResponseCode
	return errors.As(err, &codeErr) && codeErr.Actual == http.StatusNotFound
}

func defaultSecurityGroupRuleValues(opts *Options, args []string) map[string]any {
	values := map[string]any{}
	if protocol := flagValue(opts, "protocol"); protocol != "" && protocol != "any" {
		values["protocol"] = strings.ToLower(protocol)
	}
	if desc := flagValue(opts, "description"); desc != "" {
		values["description"] = desc
	}
	if boolFlag(opts, "egress") {
		values["direction"] = "egress"
	} else {
		values["direction"] = "ingress"
	}
	ethertype := flagValue(opts, "ethertype")
	if ethertype == "" {
		if strings.Contains(valueString(values["protocol"]), "ipv6") {
			ethertype = "IPv6"
		} else {
			ethertype = "IPv4"
		}
	}
	values["ethertype"] = ethertype
	if remoteGroup := flagValue(opts, "remote-group"); remoteGroup != "" {
		values["remote_group_id"] = remoteGroup
	} else if remoteAddressGroup := flagValue(opts, "remote-address-group"); remoteAddressGroup != "" {
		values["remote_address_group_id"] = remoteAddressGroup
	} else if remoteIP := flagValue(opts, "remote-ip"); remoteIP != "" {
		values["remote_ip_prefix"] = remoteIP
	} else if ethertype == "IPv6" {
		values["remote_ip_prefix"] = "::/0"
	} else {
		values["remote_ip_prefix"] = "0.0.0.0/0"
	}
	if dstPort := flagValue(opts, "dst-port"); dstPort != "" {
		ports, err := parseOptionalPortRange(dstPort, "dst-port")
		if err != nil {
			values["__error__"] = err
			return values
		}
		if len(ports) == 1 {
			values["port_range_min"] = ports[0]
			values["port_range_max"] = ports[0]
		} else if len(ports) == 2 {
			values["port_range_min"] = ports[0]
			values["port_range_max"] = ports[1]
		}
	}
	if icmpType := intFlag(opts, "icmp-type"); flagChanged(opts, "icmp-type") {
		values["port_range_min"] = icmpType
	}
	if icmpCode := intFlag(opts, "icmp-code"); flagChanged(opts, "icmp-code") {
		values["port_range_max"] = icmpCode
	}
	values["used_in_default_sg"] = boolFlag(opts, "for-default-sg")
	values["used_in_non_default_sg"] = boolFlag(opts, "for-custom-sg")
	return values
}

func defaultSecurityGroupRuleQuery(opts *Options) map[string]any {
	values := map[string]any{}
	for flag, key := range map[string]string{
		"description":          "description",
		"protocol":             "protocol",
		"ethertype":            "ethertype",
		"remote-ip":            "remote_ip_prefix",
		"remote-group":         "remote_group_id",
		"remote-address-group": "remote_address_group_id",
	} {
		if value := flagValue(opts, flag); value != "" {
			values[key] = value
		}
	}
	if boolFlag(opts, "ingress") {
		values["direction"] = "ingress"
	}
	if boolFlag(opts, "egress") {
		values["direction"] = "egress"
	}
	if boolFlag(opts, "used-in-default-sg") {
		values["used_in_default_sg"] = true
	}
	if boolFlag(opts, "used-in-non-default-sg") {
		values["used_in_non_default_sg"] = true
	}
	return values
}

func localIPValues(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, args []string) map[string]any {
	values := map[string]any{}
	if value := flagValue(opts, "name"); value != "" {
		values["name"] = value
	}
	if value := flagValue(opts, "description"); value != "" {
		values["description"] = value
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	if network := flagValue(opts, "network"); network != "" {
		item, err := findNetwork(ctx, client, network)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["network_id"] = item.ID
	}
	if portValue := flagValue(opts, "local-port"); portValue != "" {
		port, err := findPort(ctx, client, portValue)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["local_port_id"] = port.ID
	}
	if value := flagValue(opts, "local-ip-address"); value != "" {
		values["local_ip_address"] = value
	}
	if value := flagValue(opts, "ip-mode"); value != "" {
		values["ip_mode"] = value
	}
	return values
}

func localIPQuery(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) map[string]any {
	values := map[string]any{}
	if value := flagValue(opts, "name"); value != "" {
		values["name"] = value
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	if network := flagValue(opts, "network"); network != "" {
		item, err := findNetwork(ctx, client, network)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["network_id"] = item.ID
	}
	if portValue := flagValue(opts, "local-port"); portValue != "" {
		port, err := findPort(ctx, client, portValue)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["local_port_id"] = port.ID
	}
	if value := flagValue(opts, "local-ip-address"); value != "" {
		values["local_ip_address"] = value
	}
	if value := flagValue(opts, "ip-mode"); value != "" {
		values["ip_mode"] = value
	}
	return values
}

func localIPSetValues(opts *Options) map[string]any {
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	return values
}

func localIPAssociationCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("local ip association create requires <local-ip> <fixed-port>")
	}
	localIP, err := neutronExtraFindRaw(ctx, client, neutronLocalIPSpec(), args[0])
	if err != nil {
		return err
	}
	port, err := findPort(ctx, client, args[1])
	if err != nil {
		return err
	}
	values := map[string]any{"fixed_port_id": port.ID}
	if fixedIP := flagValue(opts, "fixed-ip"); fixedIP != "" {
		values["fixed_ip"] = fixedIP
	}
	spec := neutronLocalIPAssociationSpec(valueString(localIP["id"]))
	return neutronExtraCreate(ctx, stdout, opts, nil, client, spec, values)
}

func localIPAssociationDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("local ip association delete requires <local-ip> <fixed-port-id> [<fixed-port-id> ...]")
	}
	localIP, err := neutronExtraFindRaw(ctx, client, neutronLocalIPSpec(), args[0])
	if err != nil {
		return err
	}
	return neutronExtraDelete(ctxNoLookup(ctx), client, neutronLocalIPAssociationSpec(valueString(localIP["id"])), args[1:])
}

func localIPAssociationList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("local ip association list requires <local-ip>")
	}
	localIP, err := neutronExtraFindRaw(ctx, client, neutronLocalIPSpec(), args[0])
	if err != nil {
		return err
	}
	values := map[string]any{}
	if fixedPort := flagValue(opts, "fixed-port"); fixedPort != "" {
		port, err := findPort(ctx, client, fixedPort)
		if err != nil {
			return err
		}
		values["fixed_port_id"] = port.ID
	}
	if fixedIP := flagValue(opts, "fixed-ip"); fixedIP != "" {
		values["fixed_ip"] = fixedIP
	}
	if host := flagValue(opts, "host"); host != "" {
		values["host"] = host
	}
	spec := neutronLocalIPAssociationSpec(valueString(localIP["id"]))
	return neutronExtraList(ctx, stdout, opts, nil, client, spec, values)
}

func networkAgentAddNetwork(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network agent add network requires <agent-id> <network>")
	}
	network, err := findNetwork(ctx, client, args[1])
	if err != nil {
		return err
	}
	if !boolFlag(opts, "dhcp") {
		return nil
	}
	resp, err := client.Post(ctx, client.ServiceURL("agents", url.PathEscape(args[0]), "dhcp-networks"), map[string]any{"network_id": network.ID}, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusCreated, http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func networkAgentRemoveNetwork(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network agent remove network requires <agent-id> <network>")
	}
	network, err := findNetwork(ctx, client, args[1])
	if err != nil {
		return err
	}
	if !boolFlag(opts, "dhcp") {
		return nil
	}
	resp, err := client.Delete(ctx, client.ServiceURL("agents", url.PathEscape(args[0]), "dhcp-networks", url.PathEscape(network.ID)), &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func networkAgentAddRouter(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network agent add router requires <agent-id> <router>")
	}
	router, err := findRouter(ctx, client, args[1])
	if err != nil {
		return err
	}
	if !boolFlag(opts, "l3") {
		return nil
	}
	resp, err := client.Post(ctx, client.ServiceURL("agents", url.PathEscape(args[0]), "l3-routers"), map[string]any{"router_id": router.ID}, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusCreated, http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func networkAgentRemoveRouter(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network agent remove router requires <agent-id> <router>")
	}
	router, err := findRouter(ctx, client, args[1])
	if err != nil {
		return err
	}
	if !boolFlag(opts, "l3") {
		return nil
	}
	resp, err := client.Delete(ctx, client.ServiceURL("agents", url.PathEscape(args[0]), "l3-routers", url.PathEscape(router.ID)), &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func networkAgentSetValues(opts *Options) map[string]any {
	values := map[string]any{}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if boolFlag(opts, "enable") {
		values["admin_state_up"] = true
	}
	if boolFlag(opts, "disable") {
		values["admin_state_up"] = false
	}
	return values
}

func autoAllocatedTopologyCreate(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) error {
	projectID, err := projectIDForAutoTopology(ctx, opts, clients)
	if err != nil {
		return err
	}
	requestURL := client.ServiceURL("auto-allocated-topology", url.PathEscape(projectID))
	if boolFlag(opts, "check-resources") {
		requestURL += "?fields=dry-run"
	}
	var response map[string]any
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}
	if boolFlag(opts, "check-resources") {
		item := map[string]any{"dry_run": mapValueOrEmpty(response, "id", "dry_run")}
		if valueString(item["dry_run"]) == "dry-run=pass" {
			item["dry_run"] = "pass"
		}
		return renderShowOutput(stdout, opts, []outputField{{"dry_run", item["dry_run"]}})
	}
	item, err := neutronExtraExtractObject(response, "auto_allocated_topology")
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, neutronExtraFields(item, map[string]bool{"location": true, "name": true, "tenant_id": true}))
}

func autoAllocatedTopologyDelete(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) error {
	projectID, err := projectIDForAutoTopology(ctx, opts, clients)
	if err != nil {
		return err
	}
	resp, err := client.Delete(ctx, client.ServiceURL("auto-allocated-topology", url.PathEscape(projectID)), &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func projectIDForAutoTopology(ctx context.Context, opts *Options, clients *openStackClients) (string, error) {
	if project := flagValue(opts, "project"); project != "" {
		identity, err := clients.identityV3()
		if err != nil {
			return "", err
		}
		item, err := findProjectWithDomain(ctx, identity, project, flagValue(opts, "project-domain"))
		if err != nil {
			return "", err
		}
		return item.ID, nil
	}
	if clients.AuthOptions.TenantID != "" {
		return clients.AuthOptions.TenantID, nil
	}
	if clients.AuthOptions.Scope.ProjectID != "" {
		return clients.AuthOptions.Scope.ProjectID, nil
	}
	return "current", nil
}

func networkFlavorValues(ctx context.Context, opts *Options, clients *openStackClients, args []string) map[string]any {
	values := map[string]any{}
	if len(args) < 1 {
		values["__error__"] = fmt.Errorf("network flavor create requires <name>")
		return values
	}
	values["name"] = args[0]
	if value := flagValue(opts, "service-type"); value != "" {
		values["service_type"] = value
	}
	if value := flagValue(opts, "description"); value != "" {
		values["description"] = value
	}
	if boolFlag(opts, "enable") {
		values["enabled"] = true
	}
	if boolFlag(opts, "disable") {
		values["enabled"] = false
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	return values
}

func networkFlavorSetValues(opts *Options) map[string]any {
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if boolFlag(opts, "enable") {
		values["enabled"] = true
	}
	if boolFlag(opts, "disable") {
		values["enabled"] = false
	}
	return values
}

func networkFlavorProfileValues(opts *Options) map[string]any {
	values := map[string]any{}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	if flagChanged(opts, "driver") {
		values["driver"] = flagValue(opts, "driver")
	}
	if flagChanged(opts, "metainfo") {
		values["metainfo"] = flagValue(opts, "metainfo")
	}
	if boolFlag(opts, "enable") {
		values["enabled"] = true
	}
	if boolFlag(opts, "disable") {
		values["enabled"] = false
	}
	return values
}

func networkFlavorProfileAssociation(ctx context.Context, client *gophercloud.ServiceClient, args []string, add bool) error {
	if len(args) < 2 {
		return fmt.Errorf("network flavor profile association requires <flavor> <service-profile>")
	}
	flavor, err := neutronExtraFindRaw(ctx, client, neutronFlavorSpec(), args[0])
	if err != nil {
		return err
	}
	profile, err := neutronExtraFindRaw(ctx, client, neutronServiceProfileSpec(), args[1])
	if err != nil {
		return err
	}
	flavorID := valueString(flavor["id"])
	profileID := valueString(profile["id"])
	if add {
		resp, err := client.Post(ctx, client.ServiceURL("flavors", url.PathEscape(flavorID), "service_profiles"), map[string]any{"service_profile": map[string]any{"id": profileID}}, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent}})
		_, _, err = gophercloud.ParseResponse(resp, err)
		return err
	}
	resp, err := client.Delete(ctx, client.ServiceURL("flavors", url.PathEscape(flavorID), "service_profiles", url.PathEscape(profileID)), &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

func conntrackHelperCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network l3 conntrack helper create requires <router>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	values := conntrackHelperValues(opts)
	return neutronExtraCreate(ctx, stdout, opts, nil, client, neutronConntrackHelperSpec(router.ID), values)
}

func conntrackHelperDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network l3 conntrack helper delete requires <router> <conntrack-helper-id> [<conntrack-helper-id> ...]")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	return neutronExtraDelete(ctxNoLookup(ctx), client, neutronConntrackHelperSpec(router.ID), args[1:])
}

func conntrackHelperList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("network l3 conntrack helper list requires <router>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	return neutronExtraList(ctx, stdout, opts, nil, client, neutronConntrackHelperSpec(router.ID), conntrackHelperValues(opts))
}

func conntrackHelperSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network l3 conntrack helper set requires <router> <conntrack-helper-id>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	return neutronExtraSet(ctxNoLookup(ctx), opts, nil, client, neutronConntrackHelperSpec(router.ID), args[1:2], conntrackHelperValues(opts))
}

func conntrackHelperShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("network l3 conntrack helper show requires <router> <conntrack-helper-id>")
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		return err
	}
	return neutronExtraShow(ctxNoLookup(ctx), stdout, opts, client, neutronConntrackHelperSpec(router.ID), args[1:2])
}

func conntrackHelperValues(opts *Options) map[string]any {
	values := map[string]any{}
	if flagChanged(opts, "helper") {
		values["helper"] = flagValue(opts, "helper")
	}
	if flagChanged(opts, "protocol") {
		values["protocol"] = flagValue(opts, "protocol")
	}
	if flagChanged(opts, "port") {
		values["port"] = intFlag(opts, "port")
	}
	return values
}

func meterValues(ctx context.Context, opts *Options, clients *openStackClients, args []string) map[string]any {
	values := map[string]any{}
	if len(args) < 1 {
		values["__error__"] = fmt.Errorf("network meter create requires <name>")
		return values
	}
	values["name"] = args[0]
	if value := flagValue(opts, "description"); value != "" {
		values["description"] = value
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	if boolFlag(opts, "share") {
		values["shared"] = true
	}
	if boolFlag(opts, "no-share") {
		values["shared"] = false
	}
	return values
}

func meterRuleValues(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient, args []string) map[string]any {
	values := map[string]any{}
	if len(args) < 1 {
		values["__error__"] = fmt.Errorf("network meter rule create requires <meter>")
		return values
	}
	meter, err := neutronExtraFindRaw(ctx, client, neutronMeterSpec(), args[0])
	if err != nil {
		values["__error__"] = err
		return values
	}
	values["metering_label_id"] = valueString(meter["id"])
	if boolFlag(opts, "exclude") {
		values["excluded"] = true
	}
	if boolFlag(opts, "include") {
		values["excluded"] = false
	}
	if boolFlag(opts, "egress") {
		values["direction"] = "egress"
	} else {
		values["direction"] = "ingress"
	}
	for flag, key := range map[string]string{
		"remote-ip-prefix":      "remote_ip_prefix",
		"source-ip-prefix":      "source_ip_prefix",
		"destination-ip-prefix": "destination_ip_prefix",
	} {
		if value := flagValue(opts, flag); value != "" {
			values[key] = value
		}
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	return values
}

func segmentRangeValues(ctx context.Context, opts *Options, clients *openStackClients, args []string) map[string]any {
	values := map[string]any{}
	if len(args) < 1 {
		values["__error__"] = fmt.Errorf("network segment range create requires <name>")
		return values
	}
	values["name"] = args[0]
	if boolFlag(opts, "private") {
		values["shared"] = false
	} else {
		values["shared"] = true
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	if value := flagValue(opts, "network-type"); value != "" {
		values["network_type"] = value
	}
	if value := flagValue(opts, "physical-network"); value != "" {
		values["physical_network"] = value
	}
	if flagChanged(opts, "minimum") {
		values["minimum"] = intFlag(opts, "minimum")
	}
	if flagChanged(opts, "maximum") {
		values["maximum"] = intFlag(opts, "maximum")
	}
	return values
}

func segmentRangeSetValues(opts *Options) map[string]any {
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "minimum") {
		values["minimum"] = intFlag(opts, "minimum")
	}
	if flagChanged(opts, "maximum") {
		values["maximum"] = intFlag(opts, "maximum")
	}
	return values
}

func segmentRangeList(ctx context.Context, stdout io.Writer, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) error {
	spec := neutronSegmentRangeSpec()
	if boolFlag(opts, "long") || boolFlag(opts, "available") || boolFlag(opts, "unavailable") || boolFlag(opts, "used") || boolFlag(opts, "unused") {
		spec.Columns = append(spec.Columns, "Used", "Available")
		spec.Keys = append(spec.Keys, "used", "available")
	}
	items, err := neutronExtraListRaw(ctx, client, spec, "")
	if err != nil {
		return err
	}
	rows := []outputRow{}
	for _, item := range items {
		if !segmentRangeFilter(item, opts) {
			continue
		}
		rows = append(rows, neutronExtraRow(item, spec.Keys, spec.Columns))
	}
	return renderListOutput(stdout, opts, spec.Columns, rows)
}

func segmentRangeFilter(item map[string]any, opts *Options) bool {
	used := len(anySlice(item["used"])) > 0 || len(mapFromAny(item["used"])) > 0
	available := len(anySlice(item["available"])) > 0
	if boolFlag(opts, "used") && !used {
		return false
	}
	if boolFlag(opts, "unused") && used {
		return false
	}
	if boolFlag(opts, "available") && !available {
		return false
	}
	if boolFlag(opts, "unavailable") && available {
		return false
	}
	return true
}

func ndpProxyValues(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) map[string]any {
	values := map[string]any{}
	if len(args) < 1 {
		values["__error__"] = fmt.Errorf("router ndp proxy create requires <router>")
		return values
	}
	router, err := findRouter(ctx, client, args[0])
	if err != nil {
		values["__error__"] = err
		return values
	}
	values["router_id"] = router.ID
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if portValue := flagValue(opts, "port"); portValue != "" {
		port, err := findPort(ctx, client, portValue)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["port_id"] = port.ID
	}
	if value := flagValue(opts, "ip-address"); value != "" {
		values["ip_address"] = value
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	return values
}

func ndpProxyQuery(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) map[string]any {
	values := map[string]any{}
	if routerValue := flagValue(opts, "router"); routerValue != "" {
		router, err := findRouter(ctx, client, routerValue)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["router_id"] = router.ID
	}
	if portValue := flagValue(opts, "port"); portValue != "" {
		port, err := findPort(ctx, client, portValue)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["port_id"] = port.ID
	}
	if value := flagValue(opts, "ip-address"); value != "" {
		values["ip_address"] = value
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	if value := flagValue(opts, "name"); value != "" {
		values["name"] = value
	}
	return values
}

func ndpProxySetValues(opts *Options) map[string]any {
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	return values
}

func tapServiceValues(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) map[string]any {
	values := tapUpdateValues(opts)
	if portValue := flagValue(opts, "port"); portValue != "" {
		port, err := findPort(ctx, client, portValue)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["port_id"] = port.ID
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	return values
}

func tapFlowValues(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) map[string]any {
	values := tapUpdateValues(opts)
	if portValue := flagValue(opts, "port"); portValue != "" {
		port, err := findPort(ctx, client, portValue)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["source_port"] = port.ID
	}
	if tapService := flagValue(opts, "tap-service"); tapService != "" {
		service, err := neutronExtraFindRaw(ctx, client, neutronTapServiceSpec(), tapService)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["tap_service_id"] = valueString(service["id"])
	}
	if value := flagValue(opts, "direction"); value != "" {
		values["direction"] = strings.ToUpper(value)
	}
	if value := flagValue(opts, "vlan-filter"); value != "" {
		values["vlan_filter"] = value
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	return values
}

func tapMirrorValues(ctx context.Context, opts *Options, clients *openStackClients, client *gophercloud.ServiceClient) map[string]any {
	values := tapUpdateValues(opts)
	if portValue := flagValue(opts, "port"); portValue != "" {
		port, err := findPort(ctx, client, portValue)
		if err != nil {
			values["__error__"] = err
			return values
		}
		values["port_id"] = port.ID
	}
	if value := flagValue(opts, "directions"); value != "" {
		values["directions"] = parseDirectionsValue(value)
	}
	if value := flagValue(opts, "remote-ip"); value != "" {
		values["remote_ip"] = value
	}
	if value := flagValue(opts, "mirror-type"); value != "" {
		values["mirror_type"] = value
	}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	return values
}

func tapUpdateValues(opts *Options) map[string]any {
	values := map[string]any{}
	if flagChanged(opts, "name") {
		values["name"] = flagValue(opts, "name")
	}
	if flagChanged(opts, "description") {
		values["description"] = flagValue(opts, "description")
	}
	return values
}

func projectOnlyQuery(ctx context.Context, opts *Options, clients *openStackClients) map[string]any {
	values := map[string]any{}
	if project := projectFlagID(ctx, opts, clients); project != nil {
		values["project_id"] = project
	}
	return values
}

func projectFlagID(ctx context.Context, opts *Options, clients *openStackClients) any {
	project := flagValue(opts, "project")
	if project == "" {
		return nil
	}
	identity, err := clients.identityV3()
	if err != nil {
		return err
	}
	item, err := findProjectWithDomain(ctx, identity, project, flagValue(opts, "project-domain"))
	if err != nil {
		return err
	}
	return item.ID
}

func parseDirectionsValue(value string) any {
	var decoded any
	if json.Unmarshal([]byte(value), &decoded) == nil {
		return decoded
	}
	result := map[string]any{}
	for _, part := range strings.Split(value, ",") {
		key, raw, ok := strings.Cut(part, "=")
		if !ok {
			key, raw, ok = strings.Cut(part, ":")
		}
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		if parsed, err := strconv.Atoi(raw); err == nil {
			result[key] = parsed
		} else {
			result[key] = raw
		}
	}
	if len(result) == 0 {
		return value
	}
	return result
}

func mapFromAny(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func addNeutronExtrasCommandFlags(cmd *cobra.Command, path string) {
	switch path {
	case "default security group rule create":
		neutronExtraFlagExtra(cmd)
		cmd.Flags().String("description", "", "set default security group rule description")
		cmd.Flags().Int("icmp-type", 0, "ICMP type")
		cmd.Flags().Int("icmp-code", 0, "ICMP code")
		cmd.Flags().Bool("ingress", false, "ingress rule")
		cmd.Flags().Bool("egress", false, "egress rule")
		cmd.Flags().String("ethertype", "", "ethertype")
		cmd.Flags().String("remote-ip", "", "remote IP prefix")
		cmd.Flags().String("remote-group", "", "remote security group")
		cmd.Flags().String("remote-address-group", "", "remote address group")
		cmd.Flags().String("dst-port", "", "destination port range")
		cmd.Flags().String("protocol", "", "IP protocol")
		cmd.Flags().Bool("for-default-sg", false, "use for default security groups")
		cmd.Flags().Bool("for-custom-sg", false, "use for custom security groups")
	case "default security group rule list":
		cmd.Flags().String("description", "", "filter by description")
		cmd.Flags().Bool("ingress", false, "ingress rules")
		cmd.Flags().Bool("egress", false, "egress rules")
		cmd.Flags().String("ethertype", "", "filter by ethertype")
		cmd.Flags().String("remote-ip", "", "filter by remote IP prefix")
		cmd.Flags().String("remote-group", "", "filter by remote security group")
		cmd.Flags().String("remote-address-group", "", "filter by remote address group")
		cmd.Flags().String("protocol", "", "filter by protocol")
		cmd.Flags().Bool("used-in-default-sg", false, "filter rules used in default security groups")
		cmd.Flags().Bool("used-in-non-default-sg", false, "filter rules used in non-default security groups")
	case "local ip create":
		cmd.Flags().String("name", "", "local IP name")
		cmd.Flags().String("description", "", "local IP description")
		cmd.Flags().String("network", "", "network")
		cmd.Flags().String("local-port", "", "local port")
		cmd.Flags().String("local-ip-address", "", "local IP address")
		cmd.Flags().String("ip-mode", "", "IP mode")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "local ip list":
		cmd.Flags().String("name", "", "filter by name")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("network", "", "filter by network")
		cmd.Flags().String("local-port", "", "filter by local port")
		cmd.Flags().String("local-ip-address", "", "filter by local IP address")
		cmd.Flags().String("ip-mode", "", "filter by IP mode")
	case "local ip set":
		cmd.Flags().String("name", "", "set local IP name")
		cmd.Flags().String("description", "", "set local IP description")
	case "local ip association create":
		cmd.Flags().String("fixed-ip", "", "fixed IP")
		cmd.Flags().String("project-domain", "", "project domain")
	case "local ip association list":
		cmd.Flags().String("fixed-port", "", "filter by fixed port")
		cmd.Flags().String("fixed-ip", "", "filter by fixed IP")
		cmd.Flags().String("host", "", "filter by host")
		cmd.Flags().String("project-domain", "", "project domain")
	case "network agent add network":
		cmd.Flags().Bool("dhcp", false, "add network to a DHCP agent")
	case "network agent add router":
		cmd.Flags().Bool("l3", false, "add router to an L3 agent")
	case "network agent set":
		cmd.Flags().String("description", "", "set network agent description")
		cmd.Flags().Bool("enable", false, "enable network agent")
		cmd.Flags().Bool("disable", false, "disable network agent")
	case "network auto allocated topology create":
		cmd.Flags().String("project", "", "project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("check-resources", false, "validate requirements")
		cmd.Flags().Bool("or-show", true, "show existing topology")
	case "network auto allocated topology delete":
		cmd.Flags().String("project", "", "project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "network flavor create":
		neutronExtraFlagExtra(cmd)
		cmd.Flags().String("service-type", "", "service type")
		cmd.Flags().String("description", "", "flavor description")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("enable", false, "enable flavor")
		cmd.Flags().Bool("disable", false, "disable flavor")
	case "network flavor set":
		neutronExtraFlagExtra(cmd)
		cmd.Flags().String("name", "", "set flavor name")
		cmd.Flags().String("description", "", "set flavor description")
		cmd.Flags().Bool("enable", false, "enable flavor")
		cmd.Flags().Bool("disable", false, "disable flavor")
	case "network flavor profile create", "network flavor profile set":
		neutronExtraFlagExtra(cmd)
		cmd.Flags().String("description", "", "flavor profile description")
		cmd.Flags().Bool("enable", false, "enable flavor profile")
		cmd.Flags().Bool("disable", false, "disable flavor profile")
		cmd.Flags().String("driver", "", "driver")
		cmd.Flags().String("metainfo", "", "metainfo")
	case "network l3 conntrack helper create", "network l3 conntrack helper set", "network l3 conntrack helper list":
		cmd.Flags().String("helper", "", "conntrack helper module")
		cmd.Flags().String("protocol", "", "network protocol")
		cmd.Flags().Int("port", 0, "network port")
	case "network meter create":
		neutronExtraFlagExtra(cmd)
		cmd.Flags().String("description", "", "meter description")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("share", false, "share meter")
		cmd.Flags().Bool("no-share", false, "do not share meter")
	case "network meter rule create":
		neutronExtraFlagExtra(cmd)
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().Bool("exclude", false, "exclude IP prefix")
		cmd.Flags().Bool("include", false, "include IP prefix")
		cmd.Flags().Bool("ingress", false, "ingress rule")
		cmd.Flags().Bool("egress", false, "egress rule")
		cmd.Flags().String("remote-ip-prefix", "", "remote IP prefix")
		cmd.Flags().String("source-ip-prefix", "", "source IP prefix")
		cmd.Flags().String("destination-ip-prefix", "", "destination IP prefix")
	case "network segment range create":
		neutronExtraFlagExtra(cmd)
		cmd.Flags().Bool("private", false, "private range")
		cmd.Flags().Bool("shared", false, "shared range")
		cmd.Flags().String("project", "", "owner project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("network-type", "", "network type")
		cmd.Flags().String("physical-network", "", "physical network")
		cmd.Flags().Int("minimum", 0, "minimum segmentation ID")
		cmd.Flags().Int("maximum", 0, "maximum segmentation ID")
	case "network segment range list":
		cmd.Flags().Bool("long", false, "list additional fields")
		cmd.Flags().Bool("used", false, "show ranges with used segments")
		cmd.Flags().Bool("unused", false, "show ranges without used segments")
		cmd.Flags().Bool("available", false, "show ranges with available segments")
		cmd.Flags().Bool("unavailable", false, "show ranges without available segments")
	case "network segment range set":
		neutronExtraFlagExtra(cmd)
		cmd.Flags().String("name", "", "set range name")
		cmd.Flags().Int("minimum", 0, "set minimum segmentation ID")
		cmd.Flags().Int("maximum", 0, "set maximum segmentation ID")
	case "router ndp proxy create":
		cmd.Flags().String("name", "", "NDP proxy name")
		cmd.Flags().String("port", "", "port")
		cmd.Flags().String("ip-address", "", "IP address")
		cmd.Flags().String("description", "", "description")
	case "router ndp proxy list":
		cmd.Flags().String("router", "", "filter by router")
		cmd.Flags().String("port", "", "filter by port")
		cmd.Flags().String("ip-address", "", "filter by IP address")
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
		cmd.Flags().String("name", "", "filter by name")
	case "router ndp proxy set":
		cmd.Flags().String("name", "", "set NDP proxy name")
		cmd.Flags().String("description", "", "set NDP proxy description")
	case "tap service create":
		neutronTapCommonCreateFlags(cmd)
		cmd.Flags().String("port", "", "port")
	case "tap flow create":
		neutronTapCommonCreateFlags(cmd)
		cmd.Flags().String("port", "", "source port")
		cmd.Flags().String("tap-service", "", "tap service")
		cmd.Flags().String("direction", "", "direction")
		cmd.Flags().String("vlan-filter", "", "VLAN filter")
	case "tap mirror create":
		neutronTapCommonCreateFlags(cmd)
		cmd.Flags().String("port", "", "port")
		cmd.Flags().String("directions", "", "directions")
		cmd.Flags().String("remote-ip", "", "remote IP")
		cmd.Flags().String("mirror-type", "", "mirror type")
	case "tap service list", "tap flow list", "tap mirror list":
		cmd.Flags().String("project", "", "filter by project")
		cmd.Flags().String("project-domain", "", "project domain")
	case "tap service update", "tap flow update", "tap mirror update":
		cmd.Flags().String("name", "", "set name")
		cmd.Flags().String("description", "", "set description")
	}
}

func neutronExtraFlagExtra(cmd *cobra.Command) {
	cmd.Flags().StringArray("extra-property", nil, "additional property type=<type>,name=<name>,value=<value>")
}

func neutronTapCommonCreateFlags(cmd *cobra.Command) {
	cmd.Flags().String("project", "", "owner project")
	cmd.Flags().String("project-domain", "", "project domain")
	cmd.Flags().String("name", "", "name")
	cmd.Flags().String("description", "", "description")
}
