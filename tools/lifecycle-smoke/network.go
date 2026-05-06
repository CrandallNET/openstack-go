package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type networkCleanup struct {
	name string
	env  map[string]string
	args []string
}

type networkLifecycle struct {
	cloud       string
	diagnostics *lifecycleDiagnostics
	cleanups    []networkCleanup
}

func runNetworkLifecycle(cloud string, prefix string) (diagnostics lifecycleDiagnostics, err error) {
	id := uniqueID(prefix)
	diagnostics = lifecycleDiagnostics{
		ID:        id,
		Cloud:     cloud,
		Resource:  "network",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Fixtures: map[string]string{
			"resource_prefix": id,
			"suite":           "network",
		},
	}
	run := &networkLifecycle{cloud: cloud, diagnostics: &diagnostics}
	defer func() {
		if recovered := recover(); recovered != nil {
			if failure, ok := recovered.(networkLifecycleFailure); ok {
				err = failure
				return
			}
			panic(recovered)
		}
	}()

	run.mustOracle("oracle parity network list json", nil, "network", "list", "-f", "json")
	networkID, oracleNetworkID, subnetID, oracleSubnetID, replacements := run.networkSubnetLifecycle(id)
	run.securityGroupLifecycle(id, replacements)
	run.portLifecycle(id, networkID, oracleNetworkID, replacements)
	run.routerLifecycle(id, networkID, oracleNetworkID, subnetID, oracleSubnetID, replacements)
	run.addressLifecycle(id)
	run.optionalExtensionLifecycles()

	run.mustOraclePair("oracle parity subnet delete output", nil,
		[]string{"subnet", "delete", subnetID},
		[]string{"subnet", "delete", oracleSubnetID},
		replacements,
	)
	run.dropCleanup("cleanup subnet")
	run.dropCleanup("cleanup oracle subnet")
	run.mustOraclePair("oracle parity network delete output", nil,
		[]string{"network", "delete", networkID},
		[]string{"network", "delete", oracleNetworkID},
		replacements,
	)
	run.dropCleanup("cleanup network")
	run.dropCleanup("cleanup oracle network")
	if err := run.cleanupAll(); err != nil {
		diagnostics.CleanupRequired = true
		return diagnostics, err
	}
	diagnostics.CleanupRequired = false
	return diagnostics, nil
}

func (run *networkLifecycle) networkSubnetLifecycle(id string) (string, string, string, string, []parityReplacement) {
	goNetwork := id + "-net"
	oracleNetwork := id + "-oracle-net"
	replacements := []parityReplacement{pairedValue("<network-name>", goNetwork, oracleNetwork)}
	run.addCleanup("cleanup network", "network", "delete", goNetwork)
	run.addCleanup("cleanup oracle network", "network", "delete", oracleNetwork)
	network := run.mustOraclePair("oracle parity network create output", nil,
		[]string{"network", "create", "--description", "golang-osc lifecycle network", "--tag", "golang-osc-lifecycle", goNetwork, "-f", "json"},
		[]string{"network", "create", "--description", "golang-osc lifecycle network", "--tag", "golang-osc-lifecycle", oracleNetwork, "-f", "json"},
		replacements,
	)
	networkID := jsonStringField(network.Stdout, "id", "ID")
	oracleNetworkID := jsonStringField(network.OracleStdout, "id", "ID")
	if networkID == "" || oracleNetworkID == "" {
		_ = run.cleanupAll()
		panic(networkLifecycleFailure{name: "network create did not return paired IDs"})
	}
	run.diagnostics.Fixtures["network_id"] = networkID
	run.diagnostics.Fixtures["oracle_network_id"] = oracleNetworkID
	replacements = appendPairedValues(replacements, pairedValue("<network-id>", networkID, oracleNetworkID))
	run.mustOracle("oracle parity network show json", nil, "network", "show", networkID, "-f", "json")
	run.mustOraclePair("oracle parity network set disable output", nil,
		[]string{"network", "set", "--disable", networkID},
		[]string{"network", "set", "--disable", oracleNetworkID},
		replacements,
	)
	run.mustOraclePair("oracle parity network set enable output", nil,
		[]string{"network", "set", "--enable", networkID},
		[]string{"network", "set", "--enable", oracleNetworkID},
		replacements,
	)
	run.mustOraclePair("oracle parity network unset tag output", nil,
		[]string{"network", "unset", "--tag", "golang-osc-lifecycle", networkID},
		[]string{"network", "unset", "--tag", "golang-osc-lifecycle", oracleNetworkID},
		replacements,
	)

	goSubnet := id + "-subnet"
	oracleSubnet := id + "-oracle-subnet"
	goCIDR := lifecycleCIDR(id, 0)
	oracleCIDR := lifecycleCIDR(id, 1)
	replacements = appendPairedValues(replacements,
		pairedValue("<subnet-name>", goSubnet, oracleSubnet),
		pairedValue("<subnet-cidr>", goCIDR, oracleCIDR),
	)
	run.addCleanup("cleanup subnet", "subnet", "delete", goSubnet)
	run.addCleanup("cleanup oracle subnet", "subnet", "delete", oracleSubnet)
	subnet := run.mustOraclePair("oracle parity subnet create output", nil,
		[]string{"subnet", "create", "--network", networkID, "--subnet-range", goCIDR, "--dns-nameserver", "1.1.1.1", "--description", "golang-osc lifecycle subnet", "--tag", "golang-osc-lifecycle", goSubnet, "-f", "json"},
		[]string{"subnet", "create", "--network", oracleNetworkID, "--subnet-range", oracleCIDR, "--dns-nameserver", "1.1.1.1", "--description", "golang-osc lifecycle subnet", "--tag", "golang-osc-lifecycle", oracleSubnet, "-f", "json"},
		replacements,
	)
	subnetID := jsonStringField(subnet.Stdout, "id", "ID")
	oracleSubnetID := jsonStringField(subnet.OracleStdout, "id", "ID")
	if subnetID == "" || oracleSubnetID == "" {
		_ = run.cleanupAll()
		panic(networkLifecycleFailure{name: "subnet create did not return paired IDs"})
	}
	run.diagnostics.Fixtures["subnet_id"] = subnetID
	run.diagnostics.Fixtures["oracle_subnet_id"] = oracleSubnetID
	replacements = appendPairedValues(replacements, pairedValue("<subnet-id>", subnetID, oracleSubnetID))
	run.mustOracle("oracle parity subnet show json", nil, "subnet", "show", subnetID, "-f", "json")
	run.mustOraclePair("oracle parity subnet set output", nil,
		[]string{"subnet", "set", "--description", "golang-osc lifecycle subnet updated", "--dns-nameserver", "8.8.8.8", subnetID},
		[]string{"subnet", "set", "--description", "golang-osc lifecycle subnet updated", "--dns-nameserver", "8.8.8.8", oracleSubnetID},
		replacements,
	)
	run.mustOraclePair("oracle parity subnet unset output", nil,
		[]string{"subnet", "unset", "--dns-nameserver", "8.8.8.8", "--tag", "golang-osc-lifecycle", subnetID},
		[]string{"subnet", "unset", "--dns-nameserver", "8.8.8.8", "--tag", "golang-osc-lifecycle", oracleSubnetID},
		replacements,
	)
	return networkID, oracleNetworkID, subnetID, oracleSubnetID, replacements
}

func (run *networkLifecycle) securityGroupLifecycle(id string, replacements []parityReplacement) {
	goName := id + "-sg"
	oracleName := id + "-oracle-sg"
	sgReplacements := appendPairedValues(append([]parityReplacement(nil), replacements...), pairedValue("<security-group-name>", goName, oracleName))
	run.addCleanup("cleanup security group", "security", "group", "delete", goName)
	run.addCleanup("cleanup oracle security group", "security", "group", "delete", oracleName)
	group := run.mustOraclePair("oracle parity security group create output", nil,
		[]string{"security", "group", "create", "--description", "golang-osc lifecycle security group", "--tag", "golang-osc-lifecycle", goName, "-f", "json"},
		[]string{"security", "group", "create", "--description", "golang-osc lifecycle security group", "--tag", "golang-osc-lifecycle", oracleName, "-f", "json"},
		sgReplacements,
	)
	groupID := jsonStringField(group.Stdout, "id", "ID")
	oracleGroupID := jsonStringField(group.OracleStdout, "id", "ID")
	if groupID == "" || oracleGroupID == "" {
		_ = run.cleanupAll()
		panic(networkLifecycleFailure{name: "security group create did not return paired IDs"})
	}
	sgReplacements = appendPairedValues(sgReplacements, pairedValue("<security-group-id>", groupID, oracleGroupID))
	run.mustOracle("oracle parity security group show json", nil, "security", "group", "show", groupID, "-f", "json")
	rule := run.mustOraclePair("oracle parity security group rule create output", nil,
		[]string{"security", "group", "rule", "create", "--protocol", "tcp", "--dst-port", "2222", "--remote-ip", "192.0.2.0/24", "--description", "golang-osc lifecycle rule", groupID, "-f", "json"},
		[]string{"security", "group", "rule", "create", "--protocol", "tcp", "--dst-port", "2222", "--remote-ip", "192.0.2.0/24", "--description", "golang-osc lifecycle rule", oracleGroupID, "-f", "json"},
		sgReplacements,
	)
	ruleID := jsonStringField(rule.Stdout, "id", "ID")
	oracleRuleID := jsonStringField(rule.OracleStdout, "id", "ID")
	if ruleID != "" && oracleRuleID != "" {
		run.addCleanup("cleanup security group rule", "security", "group", "rule", "delete", ruleID)
		run.addCleanup("cleanup oracle security group rule", "security", "group", "rule", "delete", oracleRuleID)
		sgReplacements = appendPairedValues(sgReplacements, pairedValue("<security-group-rule-id>", ruleID, oracleRuleID))
		run.mustOraclePair("oracle parity security group rule delete output", nil,
			[]string{"security", "group", "rule", "delete", ruleID},
			[]string{"security", "group", "rule", "delete", oracleRuleID},
			sgReplacements,
		)
		run.dropCleanup("cleanup security group rule")
		run.dropCleanup("cleanup oracle security group rule")
	}
	run.mustOraclePair("oracle parity security group set output", nil,
		[]string{"security", "group", "set", "--description", "golang-osc lifecycle security group updated", groupID},
		[]string{"security", "group", "set", "--description", "golang-osc lifecycle security group updated", oracleGroupID},
		sgReplacements,
	)
	run.mustOraclePair("oracle parity security group unset output", nil,
		[]string{"security", "group", "unset", "--tag", "golang-osc-lifecycle", groupID},
		[]string{"security", "group", "unset", "--tag", "golang-osc-lifecycle", oracleGroupID},
		sgReplacements,
	)
	run.mustOraclePair("oracle parity security group delete output", nil,
		[]string{"security", "group", "delete", groupID},
		[]string{"security", "group", "delete", oracleGroupID},
		sgReplacements,
	)
	run.dropCleanup("cleanup security group")
	run.dropCleanup("cleanup oracle security group")
}

func (run *networkLifecycle) portLifecycle(id string, networkID string, oracleNetworkID string, replacements []parityReplacement) {
	goName := id + "-port"
	oracleName := id + "-oracle-port"
	portReplacements := appendPairedValues(append([]parityReplacement(nil), replacements...), pairedValue("<port-name>", goName, oracleName))
	run.addCleanup("cleanup port", "port", "delete", goName)
	run.addCleanup("cleanup oracle port", "port", "delete", oracleName)
	port := run.mustOraclePair("oracle parity port create output", nil,
		[]string{"port", "create", "--network", networkID, "--description", "golang-osc lifecycle port", "--tag", "golang-osc-lifecycle", goName, "-f", "json"},
		[]string{"port", "create", "--network", oracleNetworkID, "--description", "golang-osc lifecycle port", "--tag", "golang-osc-lifecycle", oracleName, "-f", "json"},
		portReplacements,
	)
	portID := jsonStringField(port.Stdout, "id", "ID")
	oraclePortID := jsonStringField(port.OracleStdout, "id", "ID")
	if portID == "" || oraclePortID == "" {
		_ = run.cleanupAll()
		panic(networkLifecycleFailure{name: "port create did not return paired IDs"})
	}
	run.diagnostics.Fixtures["port_id"] = portID
	run.diagnostics.Fixtures["oracle_port_id"] = oraclePortID
	portReplacements = appendPairedValues(portReplacements, pairedValue("<port-id>", portID, oraclePortID))
	run.mustOracle("oracle parity port show json", nil, "port", "show", portID, "-f", "json")
	run.mustOraclePair("oracle parity port set output", nil,
		[]string{"port", "set", "--description", "golang-osc lifecycle port updated", "--disable", portID},
		[]string{"port", "set", "--description", "golang-osc lifecycle port updated", "--disable", oraclePortID},
		portReplacements,
	)
	run.mustOraclePair("oracle parity port enable output", nil,
		[]string{"port", "set", "--enable", portID},
		[]string{"port", "set", "--enable", oraclePortID},
		portReplacements,
	)
	run.mustOraclePair("oracle parity port unset output", nil,
		[]string{"port", "unset", "--tag", "golang-osc-lifecycle", portID},
		[]string{"port", "unset", "--tag", "golang-osc-lifecycle", oraclePortID},
		portReplacements,
	)
	run.mustOraclePair("oracle parity port delete output", nil,
		[]string{"port", "delete", portID},
		[]string{"port", "delete", oraclePortID},
		portReplacements,
	)
	run.dropCleanup("cleanup port")
	run.dropCleanup("cleanup oracle port")
}

func (run *networkLifecycle) routerLifecycle(id string, networkID string, oracleNetworkID string, subnetID string, oracleSubnetID string, replacements []parityReplacement) {
	goName := id + "-router"
	oracleName := id + "-oracle-router"
	routerReplacements := appendPairedValues(append([]parityReplacement(nil), replacements...), pairedValue("<router-name>", goName, oracleName))
	run.addCleanup("cleanup router", "router", "delete", goName)
	run.addCleanup("cleanup oracle router", "router", "delete", oracleName)
	router := run.mustOraclePair("oracle parity router create output", nil,
		[]string{"router", "create", "--description", "golang-osc lifecycle router", "--tag", "golang-osc-lifecycle", goName, "-f", "json"},
		[]string{"router", "create", "--description", "golang-osc lifecycle router", "--tag", "golang-osc-lifecycle", oracleName, "-f", "json"},
		routerReplacements,
	)
	routerID := jsonStringField(router.Stdout, "id", "ID")
	oracleRouterID := jsonStringField(router.OracleStdout, "id", "ID")
	if routerID == "" || oracleRouterID == "" {
		_ = run.cleanupAll()
		panic(networkLifecycleFailure{name: "router create did not return paired IDs"})
	}
	run.diagnostics.Fixtures["router_id"] = routerID
	run.diagnostics.Fixtures["oracle_router_id"] = oracleRouterID
	routerReplacements = appendPairedValues(routerReplacements, pairedValue("<router-id>", routerID, oracleRouterID))
	run.mustOracle("oracle parity router show json", nil, "router", "show", routerID, "-f", "json")
	run.mustOraclePair("oracle parity router set output", nil,
		[]string{"router", "set", "--description", "golang-osc lifecycle router updated", routerID},
		[]string{"router", "set", "--description", "golang-osc lifecycle router updated", oracleRouterID},
		routerReplacements,
	)
	run.mustOraclePair("oracle parity router unset output", nil,
		[]string{"router", "unset", "--tag", "golang-osc-lifecycle", routerID},
		[]string{"router", "unset", "--tag", "golang-osc-lifecycle", oracleRouterID},
		routerReplacements,
	)
	portID, oraclePortID := run.routerPortFixture(id, networkID, oracleNetworkID, routerReplacements)
	if portID != "" && oraclePortID != "" {
		run.mustOraclePair("oracle parity router add port output", nil,
			[]string{"router", "add", "port", routerID, portID},
			[]string{"router", "add", "port", oracleRouterID, oraclePortID},
			routerReplacements,
		)
		run.mustOraclePair("oracle parity router remove port output", nil,
			[]string{"router", "remove", "port", routerID, portID},
			[]string{"router", "remove", "port", oracleRouterID, oraclePortID},
			routerReplacements,
		)
		run.mustDeleteOrGone("delete router interface port", "port", "delete", portID)
		run.dropCleanup("cleanup router interface port")
		run.mustDeleteOrGone("delete oracle router interface port", "port", "delete", oraclePortID)
		run.dropCleanup("cleanup oracle router interface port")
	}
	run.mustOraclePair("oracle parity router add subnet output", nil,
		[]string{"router", "add", "subnet", routerID, subnetID},
		[]string{"router", "add", "subnet", oracleRouterID, oracleSubnetID},
		routerReplacements,
	)
	goGateway := lifecycleGateway(id, 0)
	oracleGateway := lifecycleGateway(id, 1)
	routeReplacements := appendPairedValues(routerReplacements, pairedValue("<route-gateway>", goGateway, oracleGateway))
	run.mustOraclePair("oracle parity router add route output", nil,
		[]string{"router", "add", "route", "--route", "destination=203.0.113.0/24,gateway=" + goGateway, routerID, "-f", "json"},
		[]string{"router", "add", "route", "--route", "destination=203.0.113.0/24,gateway=" + oracleGateway, oracleRouterID, "-f", "json"},
		routeReplacements,
	)
	run.mustOraclePair("oracle parity router remove route output", nil,
		[]string{"router", "remove", "route", "--route", "destination=203.0.113.0/24,gateway=" + goGateway, routerID, "-f", "json"},
		[]string{"router", "remove", "route", "--route", "destination=203.0.113.0/24,gateway=" + oracleGateway, oracleRouterID, "-f", "json"},
		routeReplacements,
	)
	run.mustOraclePair("oracle parity router remove subnet output", nil,
		[]string{"router", "remove", "subnet", routerID, subnetID},
		[]string{"router", "remove", "subnet", oracleRouterID, oracleSubnetID},
		routerReplacements,
	)
	run.mustOraclePair("oracle parity router delete output", nil,
		[]string{"router", "delete", routerID},
		[]string{"router", "delete", oracleRouterID},
		routerReplacements,
	)
	run.dropCleanup("cleanup router")
	run.dropCleanup("cleanup oracle router")
}

func (run *networkLifecycle) routerPortFixture(id string, networkID string, oracleNetworkID string, replacements []parityReplacement) (string, string) {
	goName := id + "-router-port"
	oracleName := id + "-oracle-router-port"
	portReplacements := appendPairedValues(append([]parityReplacement(nil), replacements...), pairedValue("<router-port-name>", goName, oracleName))
	run.addCleanup("cleanup router interface port", "port", "delete", goName)
	run.addCleanup("cleanup oracle router interface port", "port", "delete", oracleName)
	port := run.optionalOraclePair("oracle parity router interface port create output", nil,
		[]string{"port", "create", "--network", networkID, "--description", "golang-osc lifecycle router interface port", goName, "-f", "json"},
		[]string{"port", "create", "--network", oracleNetworkID, "--description", "golang-osc lifecycle router interface port", oracleName, "-f", "json"},
		portReplacements,
	)
	if port.ExitCode != 0 || port.Error != "" {
		return "", ""
	}
	portID := jsonStringField(port.Stdout, "id", "ID")
	oraclePortID := jsonStringField(port.OracleStdout, "id", "ID")
	if portID == "" || oraclePortID == "" {
		run.skip("router add/remove port", "router interface port create did not return paired IDs")
		return "", ""
	}
	return portID, oraclePortID
}

func (run *networkLifecycle) addressLifecycle(id string) {
	goGroup := id + "-ag"
	oracleGroup := id + "-oracle-ag"
	replacements := []parityReplacement{pairedValue("<address-group-name>", goGroup, oracleGroup)}
	if result := run.optionalOraclePair("oracle parity address group create output", nil,
		[]string{"address", "group", "create", "--description", "golang-osc lifecycle address group", "--address", "192.0.2.10/32", goGroup, "-f", "json"},
		[]string{"address", "group", "create", "--description", "golang-osc lifecycle address group", "--address", "192.0.2.10/32", oracleGroup, "-f", "json"},
		replacements,
	); result.ExitCode == 0 && result.Error == "" {
		groupID := jsonStringField(result.Stdout, "id", "ID")
		oracleGroupID := jsonStringField(result.OracleStdout, "id", "ID")
		run.addCleanup("cleanup address group", "address", "group", "delete", groupID)
		run.addCleanup("cleanup oracle address group", "address", "group", "delete", oracleGroupID)
		replacements = appendPairedValues(replacements, pairedValue("<address-group-id>", groupID, oracleGroupID))
		run.mustOraclePair("oracle parity address group set output", nil,
			[]string{"address", "group", "set", "--description", "golang-osc lifecycle address group updated", "--address", "192.0.2.11/32", groupID},
			[]string{"address", "group", "set", "--description", "golang-osc lifecycle address group updated", "--address", "192.0.2.11/32", oracleGroupID},
			replacements,
		)
		run.mustOraclePair("oracle parity address group unset output", nil,
			[]string{"address", "group", "unset", "--address", "192.0.2.11/32", groupID},
			[]string{"address", "group", "unset", "--address", "192.0.2.11/32", oracleGroupID},
			replacements,
		)
		run.mustOraclePair("oracle parity address group delete output", nil,
			[]string{"address", "group", "delete", groupID},
			[]string{"address", "group", "delete", oracleGroupID},
			replacements,
		)
		run.dropCleanup("cleanup address group")
		run.dropCleanup("cleanup oracle address group")
	}

	goScope := id + "-scope"
	oracleScope := id + "-oracle-scope"
	scopeReplacements := []parityReplacement{pairedValue("<address-scope-name>", goScope, oracleScope)}
	if result := run.optionalOraclePair("oracle parity address scope create output", nil,
		[]string{"address", "scope", "create", "--ip-version", "4", "--no-share", goScope, "-f", "json"},
		[]string{"address", "scope", "create", "--ip-version", "4", "--no-share", oracleScope, "-f", "json"},
		scopeReplacements,
	); result.ExitCode == 0 && result.Error == "" {
		scopeID := jsonStringField(result.Stdout, "id", "ID")
		oracleScopeID := jsonStringField(result.OracleStdout, "id", "ID")
		run.addCleanup("cleanup address scope", "address", "scope", "delete", scopeID)
		run.addCleanup("cleanup oracle address scope", "address", "scope", "delete", oracleScopeID)
		scopeReplacements = appendPairedValues(scopeReplacements, pairedValue("<address-scope-id>", scopeID, oracleScopeID))
		run.mustOraclePair("oracle parity address scope set output", nil,
			[]string{"address", "scope", "set", "--name", id + "-scope-renamed", scopeID},
			[]string{"address", "scope", "set", "--name", id + "-oracle-scope-renamed", oracleScopeID},
			scopeReplacements,
		)
		run.mustOraclePair("oracle parity address scope delete output", nil,
			[]string{"address", "scope", "delete", scopeID},
			[]string{"address", "scope", "delete", oracleScopeID},
			scopeReplacements,
		)
		run.dropCleanup("cleanup address scope")
		run.dropCleanup("cleanup oracle address scope")
	}
}

func (run *networkLifecycle) optionalExtensionLifecycles() {
	run.skip("floating ip create/set/unset/delete", "requires an external network fixture; this lifecycle pass did not discover and reserve one")
	run.skip("floating ip port forwarding create/set/delete", "requires a floating IP and a cloud exposing the port-forwarding extension")
	run.skip("network qos policy/rule create/set/delete", "cloud6 has previously returned Neutron 404 for the QoS policy collection in both Python OSC and the Go CLI")
	run.skip("network segment create/set/delete", "requires provider network privileges or a cloud exposing project-safe segment creation")
	run.skip("network trunk create/set/unset/delete", "requires a cloud exposing the trunk extension and paired disposable parent/subports")
	run.skip("network rbac create/set/delete", "sharing policies can expose resources outside the test project and need an explicit target-project fixture")
	run.skip("subnet pool create/set/unset/delete", "requires non-overlapping address allocation fixtures beyond the directly-created subnet coverage")
}

func lifecycleCIDR(id string, offset int) string {
	sum := 0
	for _, char := range id {
		sum += int(char)
	}
	second := 200 + (sum % 30)
	third := 20 + ((sum + offset) % 180)
	return fmt.Sprintf("10.%d.%d.0/24", second, third)
}

func lifecycleGateway(id string, offset int) string {
	return strings.TrimSuffix(lifecycleCIDR(id, offset), "0/24") + "2"
}

func (run *networkLifecycle) run(name string, args ...string) stepResult {
	return run.runWithEnv(name, nil, args...)
}

func (run *networkLifecycle) runWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := runCLIWithEnv(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	return result
}

func (run *networkLifecycle) mustOracle(name string, env map[string]string, args ...string) stepResult {
	result := compareWithOracle(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(networkLifecycleFailure{name: name})
	}
	return result
}

func (run *networkLifecycle) mustOraclePair(name string, env map[string]string, goArgs []string, oracleArgs []string, replacements []parityReplacement) stepResult {
	result := compareWithOracleArgs(run.cloud, env, goArgs, oracleArgs, replacements)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(networkLifecycleFailure{name: name})
	}
	return result
}

func (run *networkLifecycle) optionalOraclePair(name string, env map[string]string, goArgs []string, oracleArgs []string, replacements []parityReplacement) stepResult {
	goResult := runCLIWithEnv(run.cloud, env, goArgs...)
	if goResult.ExitCode != 0 {
		goResult.Name = name
		run.diagnostics.Steps = append(run.diagnostics.Steps, goResult)
		run.skip(name+" follow-up", strings.TrimSpace(firstNonEmptyString(goResult.Error, goResult.Stderr, goResult.Stdout)))
		return goResult
	}
	result := compareExistingWithOracle(run.cloud, env, goResult, oracleArgs, replacements)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		run.skip(name+" follow-up", strings.TrimSpace(firstNonEmptyString(result.Error, result.OracleStderr, result.Stderr, result.OracleStdout, result.Stdout)))
	}
	return result
}

func (run *networkLifecycle) mustDeleteOrGone(name string, args ...string) stepResult {
	result := run.run(name, args...)
	if result.ExitCode != 0 && !looksDeleted(result) && !run.confirmDeleteTargetGone(name, nil, args) {
		_ = run.cleanupAll()
		panic(networkLifecycleFailure{name: name})
	}
	return result
}

func (run *networkLifecycle) confirmDeleteTargetGone(name string, env map[string]string, args []string) bool {
	deleteIndex := -1
	for i, arg := range args {
		if arg == "delete" {
			deleteIndex = i
			break
		}
	}
	if deleteIndex < 1 || deleteIndex >= len(args)-1 {
		return false
	}
	showArgs := append([]string{}, args[:deleteIndex]...)
	showArgs = append(showArgs, "show")
	showArgs = append(showArgs, args[deleteIndex+1:]...)
	result := run.runWithEnv(name+" confirm gone", env, showArgs...)
	return result.ExitCode != 0 && looksDeleted(result)
}

func (run *networkLifecycle) skip(name string, reason string) {
	run.diagnostics.Steps = append(run.diagnostics.Steps, stepResult{Name: name, Skipped: true, SkipReason: reason})
}

func (run *networkLifecycle) addCleanup(name string, args ...string) {
	run.diagnostics.CleanupRequired = true
	run.cleanups = append(run.cleanups, networkCleanup{name: name, args: args})
}

func (run *networkLifecycle) dropCleanup(name string) {
	for i := len(run.cleanups) - 1; i >= 0; i-- {
		if run.cleanups[i].name == name {
			run.cleanups = append(run.cleanups[:i], run.cleanups[i+1:]...)
			break
		}
	}
	if len(run.cleanups) == 0 {
		run.diagnostics.CleanupRequired = false
	}
}

func (run *networkLifecycle) cleanupAll() error {
	var failures []error
	for i := len(run.cleanups) - 1; i >= 0; i-- {
		cleanup := run.cleanups[i]
		result := run.runWithEnv(cleanup.name, cleanup.env, cleanup.args...)
		if result.ExitCode != 0 && !looksDeleted(result) && !run.confirmDeleteTargetGone(cleanup.name, cleanup.env, cleanup.args) {
			failures = append(failures, fmt.Errorf("%s failed", cleanup.name))
		}
	}
	run.cleanups = nil
	run.diagnostics.CleanupRequired = len(failures) > 0
	return errors.Join(failures...)
}

type networkLifecycleFailure struct {
	name string
}

func (err networkLifecycleFailure) Error() string {
	return "network lifecycle failed at " + err.name
}
