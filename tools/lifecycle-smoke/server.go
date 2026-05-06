package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type serverCleanup struct {
	name string
	env  map[string]string
	args []string
}

type serverLifecycle struct {
	cloud       string
	diagnostics *lifecycleDiagnostics
	cleanups    []serverCleanup
}

func runServerLifecycle(cloud string, prefix string) (diagnostics lifecycleDiagnostics, err error) {
	id := uniqueID(prefix)
	diagnostics = lifecycleDiagnostics{
		ID:        id,
		Cloud:     cloud,
		Resource:  "server",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Fixtures: map[string]string{
			"resource_prefix": id,
			"suite":           "server",
		},
	}
	run := &serverLifecycle{cloud: cloud, diagnostics: &diagnostics}
	defer func() {
		if recovered := recover(); recovered != nil {
			if failure, ok := recovered.(serverLifecycleFailure); ok {
				err = failure
				return
			}
			panic(recovered)
		}
	}()

	image := run.must("preflight image list", "image", "list", "-f", "json")
	run.recordImageFixture(image.Stdout)
	flavor := run.must("preflight flavor list", "flavor", "list", "-f", "json")
	run.recordFlavorFixtures(flavor.Stdout)
	network := run.must("preflight network list", "network", "list", "-f", "json")
	run.recordNetworkFixtures(network.Stdout)
	run.must("preflight server list", "server", "list", "-f", "json")
	if diagnostics.Fixtures["image_id"] == "" || diagnostics.Fixtures["flavor_id"] == "" || diagnostics.Fixtures["network_id"] == "" {
		return diagnostics, fmt.Errorf("server lifecycle requires image, flavor, and network fixtures")
	}

	publicKey, cleanupPublicKey, err := writePublicKey()
	if err != nil {
		return diagnostics, err
	}
	defer cleanupPublicKey()
	keyName := id + "-keypair"
	run.must("create keypair", "keypair", "create", "--public-key", publicKey, keyName, "-f", "json")
	diagnostics.Fixtures["keypair_name"] = keyName
	run.addCleanup("cleanup keypair", "keypair", "delete", keyName)

	serverGroupID := ""
	group := run.optional("create server group", "server", "group", "create", "--policy", "anti-affinity", id+"-server-group", "-f", "json")
	if group.ExitCode == 0 {
		serverGroupID = jsonStringField(group.Stdout, "id", "ID")
		if serverGroupID != "" {
			diagnostics.Fixtures["server_group_id"] = serverGroupID
			run.addCleanup("cleanup server group", "server", "group", "delete", serverGroupID)
			run.must("show server group", "server", "group", "show", serverGroupID, "-f", "json")
		}
	}

	serverName := id + "-server"
	createArgs := []string{
		"server", "create",
		"--flavor", diagnostics.Fixtures["flavor_id"],
		"--image", diagnostics.Fixtures["image_id"],
		"--network", diagnostics.Fixtures["network_id"],
		"--key-name", keyName,
		"--property", "golang_osc_test=" + id,
		"--description", "golang-osc lifecycle server",
		"--wait",
	}
	createArgs = append(createArgs, serverName, "-f", "json")
	created := run.must("create server", createArgs...)
	serverID := jsonStringField(created.Stdout, "id", "ID")
	if serverID == "" {
		_ = run.cleanupAll()
		return diagnostics, fmt.Errorf("create server did not return an id")
	}
	diagnostics.Fixtures["server_id"] = serverID
	run.addCleanup("cleanup server", "server", "delete", "--wait", serverID)

	oracleServerName := id + "-oracle-server"
	oracleCreateArgs := []string{
		"server", "create",
		"--flavor", diagnostics.Fixtures["flavor_id"],
		"--image", diagnostics.Fixtures["image_id"],
		"--network", diagnostics.Fixtures["network_id"],
		"--key-name", keyName,
		"--property", "golang_osc_test=" + id,
		"--description", "golang-osc lifecycle server",
		"--wait",
		oracleServerName,
		"-f", "json",
	}
	oracleCreated := runOracleCLI(cloud, nil, oracleCreateArgs...)
	oracleServerID := jsonStringField(oracleCreated.Stdout, "id", "ID")
	if oracleCreated.ExitCode == 0 && oracleServerID != "" {
		diagnostics.Fixtures["oracle_server_id"] = oracleServerID
		run.addCleanup("cleanup oracle server", "server", "delete", "--wait", oracleServerID)
	}
	createParityReplacements := appendPairedValues(nil,
		pairedValue("<server-name>", serverName, oracleServerName),
		pairedValue("<server-id>", serverID, oracleServerID),
	)
	createParity := compareStepResults(created, oracleCreated, createParityReplacements)
	createParity.Name = "oracle parity server create json"
	diagnostics.Steps = append(diagnostics.Steps, createParity)
	if createParity.Error != "" {
		_ = run.cleanupAll()
		return diagnostics, fmt.Errorf("server lifecycle failed at oracle parity server create json")
	}

	serverParityReplacements := appendPairedValues(createParityReplacements,
		pairedValue("<renamed-server-name>", id+"-server-renamed", id+"-oracle-server-renamed"),
	)
	run.mustWaitStatus("wait server active after create", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"ACTIVE"}, 5*time.Minute)
	run.mustWaitStatus("wait oracle server active after create", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"ACTIVE"}, 5*time.Minute)
	run.must("show server", "server", "show", serverID, "-f", "json")
	run.must("list servers", "server", "list", "-f", "json")
	run.mustOracle("oracle parity server show json", nil, "server", "show", serverID, "-f", "json")
	run.mustOracle("oracle parity server show table", nil, "server", "show", serverID)
	run.eventReadLifecycle(serverID)

	run.mustOraclePair("oracle parity server set output", nil,
		[]string{"server", "set", "--name", id + "-server-renamed", "--description", "golang-osc lifecycle server updated", "--property", "phase=set", "--tag", "golang-osc-lifecycle", serverID},
		[]string{"server", "set", "--name", id + "-oracle-server-renamed", "--description", "golang-osc lifecycle server updated", "--property", "phase=set", "--tag", "golang-osc-lifecycle", oracleServerID},
		serverParityReplacements,
	)
	run.mustOraclePair("oracle parity server unset output", nil,
		[]string{"server", "unset", "--property", "phase", "--tag", "golang-osc-lifecycle", "--description", serverID},
		[]string{"server", "unset", "--property", "phase", "--tag", "golang-osc-lifecycle", "--description", oracleServerID},
		serverParityReplacements,
	)
	run.mustOraclePair("oracle parity server lock output", nil,
		[]string{"server", "lock", serverID},
		[]string{"server", "lock", oracleServerID},
		serverParityReplacements,
	)
	run.mustOraclePair("oracle parity server unlock output", nil,
		[]string{"server", "unlock", serverID},
		[]string{"server", "unlock", oracleServerID},
		serverParityReplacements,
	)
	run.mustOraclePair("oracle parity server reboot output", nil,
		[]string{"server", "reboot", "--hard", "--wait", serverID},
		[]string{"server", "reboot", "--hard", "--wait", oracleServerID},
		serverParityReplacements,
	)
	run.mustWaitStatus("wait server active after reboot", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"ACTIVE"}, 3*time.Minute)
	run.mustWaitStatus("wait oracle server active after reboot", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"ACTIVE"}, 3*time.Minute)
	run.mustOraclePair("oracle parity server stop output", nil,
		[]string{"server", "stop", serverID},
		[]string{"server", "stop", oracleServerID},
		serverParityReplacements,
	)
	run.mustWaitStatus("wait server shutoff", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"SHUTOFF"}, 3*time.Minute)
	run.mustWaitStatus("wait oracle server shutoff", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"SHUTOFF"}, 3*time.Minute)
	run.mustOraclePair("oracle parity server start output", nil,
		[]string{"server", "start", serverID},
		[]string{"server", "start", oracleServerID},
		serverParityReplacements,
	)
	run.mustWaitStatus("wait server active after start", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"ACTIVE"}, 3*time.Minute)
	run.mustWaitStatus("wait oracle server active after start", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"ACTIVE"}, 3*time.Minute)

	run.rebuildLifecycle(serverID, oracleServerID, serverParityReplacements)
	run.pauseLifecycle(serverID, oracleServerID, serverParityReplacements)
	run.suspendLifecycle(serverID, oracleServerID, serverParityReplacements)
	run.rescueLifecycle(serverID, oracleServerID, serverParityReplacements)
	run.resizeLifecycle(serverID)
	run.serverVolumeLifecycle(id, serverID, oracleServerID, serverParityReplacements)
	run.serverSecurityGroupLifecycle(id, serverID, oracleServerID, serverParityReplacements)
	run.serverPortLifecycle(id, serverID, oracleServerID, serverParityReplacements)
	run.serverImageLifecycle(id, serverID)
	run.shelveLifecycle(serverID)

	run.recordRiskSkipped("server add/remove floating ip", "requires a currently available external network or floating IP fixture; this suite did not find a safe fixture on cloud6")
	run.recordRiskSkipped("server add/remove fixed ip", "legacy fixed-IP action needs a second safe network/IP fixture and can alter guest network reachability")
	run.recordRiskSkipped("server migrate and migration actions", "migration changes scheduler or host placement and needs a dedicated multi-host compute fixture")
	run.recordRiskSkipped("server evacuate", "evacuation is an admin recovery operation for failed hosts and is not safe as a routine lifecycle action")
	run.recordRiskSkipped("server dump create", "triggering a crash dump deliberately crashes the disposable guest and needs an isolated crash-test fixture")

	run.mustOraclePair("oracle parity server delete output", nil,
		[]string{"server", "delete", "--wait", serverID},
		[]string{"server", "delete", "--wait", oracleServerID},
		serverParityReplacements,
	)
	run.dropCleanup("cleanup server")
	run.dropCleanup("cleanup oracle server")
	run.mustWaitDeleted("wait server deleted", []string{"server", "show", serverID, "-f", "json"}, 3*time.Minute)
	run.mustWaitDeleted("wait oracle server deleted", []string{"server", "show", oracleServerID, "-f", "json"}, 3*time.Minute)
	if err := run.cleanupAll(); err != nil {
		diagnostics.CleanupRequired = true
		return diagnostics, err
	}
	diagnostics.CleanupRequired = false
	return diagnostics, nil
}

func (run *serverLifecycle) eventReadLifecycle(serverID string) {
	events := run.optional("list server events", "server", "event", "list", serverID, "-f", "json")
	if events.ExitCode != 0 {
		return
	}
	for _, row := range jsonRows(events.Stdout) {
		requestID := jsonRowString(row, "Request ID", "request_id", "request-id")
		if requestID != "" {
			run.optional("show server event", "server", "event", "show", serverID, requestID, "-f", "json")
			return
		}
	}
	run.skip("show server event", "server event list did not return a request id")
}

func (run *serverLifecycle) rebuildLifecycle(serverID string, oracleServerID string, replacements []parityReplacement) {
	result := run.optionalOraclePair("oracle parity server rebuild output", nil,
		[]string{"server", "rebuild", "--image", run.diagnostics.Fixtures["image_id"], "--description", "golang-osc lifecycle rebuilt", "--wait", serverID, "-f", "json"},
		[]string{"server", "rebuild", "--image", run.diagnostics.Fixtures["image_id"], "--description", "golang-osc lifecycle rebuilt", "--wait", oracleServerID, "-f", "json"},
		replacements,
	)
	if result.ExitCode == 0 && result.Error == "" {
		run.mustWaitStatus("wait server active after rebuild", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"ACTIVE"}, 5*time.Minute)
		run.mustWaitStatus("wait oracle server active after rebuild", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"ACTIVE"}, 5*time.Minute)
	}
}

func (run *serverLifecycle) pauseLifecycle(serverID string, oracleServerID string, replacements []parityReplacement) {
	if result := run.optionalOraclePair("oracle parity server pause output", nil,
		[]string{"server", "pause", serverID},
		[]string{"server", "pause", oracleServerID},
		replacements,
	); result.ExitCode == 0 && result.Error == "" {
		run.mustWaitStatus("wait server paused", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"PAUSED"}, 2*time.Minute)
		run.mustWaitStatus("wait oracle server paused", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"PAUSED"}, 2*time.Minute)
		run.mustOraclePair("oracle parity server unpause output", nil,
			[]string{"server", "unpause", serverID},
			[]string{"server", "unpause", oracleServerID},
			replacements,
		)
		run.mustWaitStatus("wait server active after unpause", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"ACTIVE"}, 2*time.Minute)
		run.mustWaitStatus("wait oracle server active after unpause", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"ACTIVE"}, 2*time.Minute)
	}
}

func (run *serverLifecycle) suspendLifecycle(serverID string, oracleServerID string, replacements []parityReplacement) {
	if result := run.optionalOraclePair("oracle parity server suspend output", nil,
		[]string{"server", "suspend", serverID},
		[]string{"server", "suspend", oracleServerID},
		replacements,
	); result.ExitCode == 0 && result.Error == "" {
		run.mustWaitStatus("wait server suspended", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"SUSPENDED"}, 2*time.Minute)
		run.mustWaitStatus("wait oracle server suspended", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"SUSPENDED"}, 2*time.Minute)
		run.mustOraclePair("oracle parity server resume output", nil,
			[]string{"server", "resume", serverID},
			[]string{"server", "resume", oracleServerID},
			replacements,
		)
		run.mustWaitStatus("wait server active after resume", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"ACTIVE"}, 2*time.Minute)
		run.mustWaitStatus("wait oracle server active after resume", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"ACTIVE"}, 2*time.Minute)
	}
}

func (run *serverLifecycle) shelveLifecycle(serverID string) {
	az := ""
	before := run.run("record server availability zone before shelve", "server", "show", serverID, "-f", "json")
	if before.ExitCode == 0 {
		az = jsonStringField(before.Stdout, "OS-EXT-AZ:availability_zone", "availability_zone")
		if az != "" {
			run.diagnostics.Fixtures["server_availability_zone"] = az
		}
	}
	if result := run.optional("shelve server", "server", "shelve", "--wait", serverID); result.ExitCode == 0 {
		shelved, ok := run.waitStatus("wait server shelved", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"SHELVED", "SHELVED_OFFLOADED"}, 5*time.Minute)
		if !ok {
			run.skip("unshelve server", "server did not settle into SHELVED or SHELVED_OFFLOADED before timeout")
			return
		}
		unshelveArgs := []string{"server", "unshelve"}
		status := strings.ToUpper(jsonStringField(shelved.Stdout, "status", "Status"))
		if status == "SHELVED_OFFLOADED" && az != "" {
			unshelveArgs = append(unshelveArgs, "--availability-zone", az)
		}
		unshelveArgs = append(unshelveArgs, serverID)
		if result := run.optional("unshelve server", unshelveArgs...); result.ExitCode == 0 {
			if _, ok := run.waitStatus("wait server active after unshelve", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"ACTIVE", "SHUTOFF"}, 45*time.Second); !ok {
				run.skip("server unshelve completion", "unshelve was accepted but the cloud did not return the server to ACTIVE or SHUTOFF before timeout")
			}
		}
	}
}

func (run *serverLifecycle) rescueLifecycle(serverID string, oracleServerID string, replacements []parityReplacement) {
	if result := run.optionalOraclePair("oracle parity server rescue output", nil,
		[]string{"server", "rescue", "--image", run.diagnostics.Fixtures["image_id"], serverID, "-f", "json"},
		[]string{"server", "rescue", "--image", run.diagnostics.Fixtures["image_id"], oracleServerID, "-f", "json"},
		replacements,
	); result.ExitCode == 0 && result.Error == "" {
		run.mustWaitStatus("wait server rescue", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"RESCUE"}, 3*time.Minute)
		run.mustWaitStatus("wait oracle server rescue", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"RESCUE"}, 3*time.Minute)
		run.mustOraclePair("oracle parity server unrescue output", nil,
			[]string{"server", "unrescue", serverID},
			[]string{"server", "unrescue", oracleServerID},
			replacements,
		)
		run.mustWaitStatus("wait server active after unrescue", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"ACTIVE"}, 3*time.Minute)
		run.mustWaitStatus("wait oracle server active after unrescue", []string{"server", "show", oracleServerID, "-f", "json"}, "status", []string{"ACTIVE"}, 3*time.Minute)
	}
}

func (run *serverLifecycle) resizeLifecycle(serverID string) {
	alternate := run.diagnostics.Fixtures["alternate_flavor_id"]
	if alternate == "" {
		run.skip("resize server", "no alternate flavor fixture was available")
		return
	}
	if result := run.optional("resize server", "server", "resize", "--flavor", alternate, serverID); result.ExitCode == 0 {
		if _, ok := run.waitStatus("wait server verify resize", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"VERIFY_RESIZE"}, 3*time.Minute); !ok {
			run.skip("server resize completion", "resize was accepted but the cloud did not return the server to VERIFY_RESIZE before timeout")
			return
		}
		run.must("revert server resize", "server", "resize", "revert", serverID)
		run.mustWaitStatus("wait server active after resize revert", []string{"server", "show", serverID, "-f", "json"}, "status", []string{"ACTIVE"}, 5*time.Minute)
	}
}

func (run *serverLifecycle) serverVolumeLifecycle(id string, serverID string, oracleServerID string, replacements []parityReplacement) {
	createArgs := []string{"volume", "create", "--size", "1", "--description", "golang-osc server lifecycle attachment volume", "--property", "golang_osc_test=" + id}
	if volumeType := run.diagnostics.Fixtures["volume_type"]; volumeType != "" {
		createArgs = append(createArgs, "--type", volumeType)
	}
	goVolumeName := id + "-server-volume"
	oracleVolumeName := id + "-oracle-server-volume"
	createArgs = append(createArgs, goVolumeName, "-f", "json")
	volume := run.optional("create server attachment volume", createArgs...)
	if volume.ExitCode != 0 {
		return
	}
	volumeID := jsonStringField(volume.Stdout, "id", "ID")
	if volumeID == "" {
		run.skip("server volume follow-up", "volume create did not return an id")
		return
	}
	run.diagnostics.Fixtures["server_volume_id"] = volumeID
	run.addCleanup("cleanup server attachment volume", "volume", "delete", volumeID)
	run.mustWaitStatus("wait attachment volume available", []string{"volume", "show", volumeID, "-f", "json"}, "status", []string{"available"}, 3*time.Minute)

	oracleVolumeArgs := []string{"volume", "create", "--size", "1", "--description", "golang-osc server lifecycle attachment volume", "--property", "golang_osc_test=" + id}
	if volumeType := run.diagnostics.Fixtures["volume_type"]; volumeType != "" {
		oracleVolumeArgs = append(oracleVolumeArgs, "--type", volumeType)
	}
	oracleVolumeArgs = append(oracleVolumeArgs, oracleVolumeName, "-f", "json")
	oracleVolume := run.optional("create oracle server attachment volume", oracleVolumeArgs...)
	if oracleVolume.ExitCode != 0 {
		return
	}
	oracleVolumeID := jsonStringField(oracleVolume.Stdout, "id", "ID")
	if oracleVolumeID == "" {
		run.skip("oracle server volume follow-up", "volume create did not return an id")
		return
	}
	run.diagnostics.Fixtures["oracle_server_volume_id"] = oracleVolumeID
	run.addCleanup("cleanup oracle server attachment volume", "volume", "delete", oracleVolumeID)
	run.mustWaitStatus("wait oracle attachment volume available", []string{"volume", "show", oracleVolumeID, "-f", "json"}, "status", []string{"available"}, 3*time.Minute)
	volumeReplacements := appendPairedValues(append([]parityReplacement(nil), replacements...),
		pairedValue("<volume-id>", volumeID, oracleVolumeID),
		pairedValue("<volume-name>", goVolumeName, oracleVolumeName),
	)

	if result := run.optionalOraclePair("oracle parity server add volume output", nil,
		[]string{"server", "add", "volume", serverID, volumeID, "-f", "json"},
		[]string{"server", "add", "volume", oracleServerID, oracleVolumeID, "-f", "json"},
		volumeReplacements,
	); result.ExitCode == 0 {
		run.mustWaitStatus("wait attachment volume in-use", []string{"volume", "show", volumeID, "-f", "json"}, "status", []string{"in-use"}, 3*time.Minute)
		run.mustWaitStatus("wait oracle attachment volume in-use", []string{"volume", "show", oracleVolumeID, "-f", "json"}, "status", []string{"in-use"}, 3*time.Minute)
		if result.Error != "" {
			run.run("detach server attachment volume after parity failure", "server", "remove", "volume", serverID, volumeID)
			run.run("detach oracle server attachment volume after parity failure", "server", "remove", "volume", oracleServerID, oracleVolumeID)
			_ = run.cleanupAll()
			panic(serverLifecycleFailure{name: "oracle parity server add volume output"})
		}
		run.must("list server volumes", "server", "volume", "list", serverID, "-f", "json")
		run.mustOraclePair("oracle parity server volume set output", nil,
			[]string{"server", "volume", "set", "--preserve-on-termination", serverID, volumeID},
			[]string{"server", "volume", "set", "--preserve-on-termination", oracleServerID, oracleVolumeID},
			volumeReplacements,
		)
		run.mustOraclePair("oracle parity server volume update output", nil,
			[]string{"server", "volume", "update", "--preserve-on-termination", serverID, volumeID},
			[]string{"server", "volume", "update", "--preserve-on-termination", oracleServerID, oracleVolumeID},
			volumeReplacements,
		)
		run.mustOraclePair("oracle parity server remove volume output", nil,
			[]string{"server", "remove", "volume", serverID, volumeID},
			[]string{"server", "remove", "volume", oracleServerID, oracleVolumeID},
			volumeReplacements,
		)
		run.mustWaitStatus("wait attachment volume available after detach", []string{"volume", "show", volumeID, "-f", "json"}, "status", []string{"available"}, 3*time.Minute)
		run.mustWaitStatus("wait oracle attachment volume available after detach", []string{"volume", "show", oracleVolumeID, "-f", "json"}, "status", []string{"available"}, 3*time.Minute)
	}
	run.must("delete server attachment volume", "volume", "delete", volumeID)
	run.dropCleanup("cleanup server attachment volume")
	run.mustWaitDeleted("wait server attachment volume deleted", []string{"volume", "show", volumeID, "-f", "json"}, 2*time.Minute)
	run.must("delete oracle server attachment volume", "volume", "delete", oracleVolumeID)
	run.dropCleanup("cleanup oracle server attachment volume")
	run.mustWaitDeleted("wait oracle server attachment volume deleted", []string{"volume", "show", oracleVolumeID, "-f", "json"}, 2*time.Minute)
}

func (run *serverLifecycle) serverSecurityGroupLifecycle(id string, serverID string, oracleServerID string, replacements []parityReplacement) {
	securityGroup := run.optional("create server security group", "security", "group", "create", "--description", "golang-osc server lifecycle security group", id+"-sg", "-f", "json")
	if securityGroup.ExitCode != 0 {
		return
	}
	securityGroupID := jsonStringField(securityGroup.Stdout, "id", "ID")
	if securityGroupID == "" {
		run.skip("server security group follow-up", "security group create did not return an id")
		return
	}
	run.addCleanup("cleanup server security group", "security", "group", "delete", securityGroupID)
	securityGroupReplacements := appendPairedValues(append([]parityReplacement(nil), replacements...),
		pairedValue("<security-group-id>", securityGroupID, securityGroupID),
	)
	if result := run.optionalOraclePair("oracle parity server add security group output", nil,
		[]string{"server", "add", "security", "group", serverID, securityGroupID},
		[]string{"server", "add", "security", "group", oracleServerID, securityGroupID},
		securityGroupReplacements,
	); result.ExitCode == 0 {
		if result.Error != "" {
			run.run("remove server security group after parity failure", "server", "remove", "security", "group", serverID, securityGroupID)
			run.run("remove oracle server security group after parity failure", "server", "remove", "security", "group", oracleServerID, securityGroupID)
			_ = run.cleanupAll()
			panic(serverLifecycleFailure{name: "oracle parity server add security group output"})
		}
		run.mustOraclePair("oracle parity server remove security group output", nil,
			[]string{"server", "remove", "security", "group", serverID, securityGroupID},
			[]string{"server", "remove", "security", "group", oracleServerID, securityGroupID},
			securityGroupReplacements,
		)
	}
	run.must("delete server security group", "security", "group", "delete", securityGroupID)
	run.dropCleanup("cleanup server security group")
}

func (run *serverLifecycle) serverPortLifecycle(id string, serverID string, oracleServerID string, replacements []parityReplacement) {
	networkID := firstNonEmptyString(run.diagnostics.Fixtures["alternate_network_id"], run.diagnostics.Fixtures["network_id"])
	port := run.optional("create server attach port", "port", "create", "--network", networkID, id+"-port", "-f", "json")
	if port.ExitCode != 0 {
		return
	}
	portID := jsonStringField(port.Stdout, "id", "ID")
	if portID == "" {
		run.skip("server port follow-up", "port create did not return an id")
		return
	}
	run.addCleanup("cleanup server attach port", "port", "delete", portID)
	oraclePort := run.optional("create oracle server attach port", "port", "create", "--network", networkID, id+"-oracle-port", "-f", "json")
	if oraclePort.ExitCode != 0 {
		return
	}
	oraclePortID := jsonStringField(oraclePort.Stdout, "id", "ID")
	if oraclePortID == "" {
		run.skip("oracle server port follow-up", "port create did not return an id")
		return
	}
	run.addCleanup("cleanup oracle server attach port", "port", "delete", oraclePortID)
	portReplacements := appendPairedValues(append([]parityReplacement(nil), replacements...),
		pairedValue("<port-id>", portID, oraclePortID),
		pairedValue("<port-name>", id+"-port", id+"-oracle-port"),
	)
	if result := run.optionalOraclePair("oracle parity server add port output", nil,
		[]string{"server", "add", "port", serverID, portID},
		[]string{"server", "add", "port", oracleServerID, oraclePortID},
		portReplacements,
	); result.ExitCode == 0 {
		if result.Error != "" {
			run.run("remove server port after parity failure", "server", "remove", "port", serverID, portID)
			run.run("remove oracle server port after parity failure", "server", "remove", "port", oracleServerID, oraclePortID)
			_ = run.cleanupAll()
			panic(serverLifecycleFailure{name: "oracle parity server add port output"})
		}
		run.mustOraclePair("oracle parity server remove port output", nil,
			[]string{"server", "remove", "port", serverID, portID},
			[]string{"server", "remove", "port", oracleServerID, oraclePortID},
			portReplacements,
		)
	}
	run.must("delete server attach port", "port", "delete", portID)
	run.dropCleanup("cleanup server attach port")
	run.must("delete oracle server attach port", "port", "delete", oraclePortID)
	run.dropCleanup("cleanup oracle server attach port")

	if alternate := run.diagnostics.Fixtures["alternate_network_id"]; alternate != "" {
		networkReplacements := appendPairedValues(append([]parityReplacement(nil), replacements...),
			pairedValue("<network-id>", alternate, alternate),
		)
		if result := run.optionalOraclePair("oracle parity server add network output", nil,
			[]string{"server", "add", "network", serverID, alternate},
			[]string{"server", "add", "network", oracleServerID, alternate},
			networkReplacements,
		); result.ExitCode == 0 {
			if result.Error != "" {
				run.run("remove server network after parity failure", "server", "remove", "network", serverID, alternate)
				run.run("remove oracle server network after parity failure", "server", "remove", "network", oracleServerID, alternate)
				_ = run.cleanupAll()
				panic(serverLifecycleFailure{name: "oracle parity server add network output"})
			}
			run.mustOraclePair("oracle parity server remove network output", nil,
				[]string{"server", "remove", "network", serverID, alternate},
				[]string{"server", "remove", "network", oracleServerID, alternate},
				networkReplacements,
			)
		}
	} else {
		run.skip("server add/remove network", "no alternate network fixture was available")
	}
}

func (run *serverLifecycle) serverImageLifecycle(id string, serverID string) {
	image := run.optional("create server image", "server", "image", "create", "--name", id+"-server-image", "--property", "golang_osc_test="+id, serverID, "-f", "json")
	if image.ExitCode == 0 {
		imageID := jsonStringField(image.Stdout, "id", "ID")
		if imageID != "" {
			run.addCleanup("cleanup server image", "image", "delete", imageID)
			if _, ok := run.waitStatus("wait server image active", []string{"image", "show", imageID, "-f", "json"}, "status", []string{"active"}, 2*time.Minute); !ok {
				run.skip("server image create completion", "image create was accepted but the cloud did not return the image to active before timeout")
			}
			run.mustDeleteOrGone("delete server image", "image", "delete", imageID)
			run.dropCleanup("cleanup server image")
		}
	}
	backup := run.optional("create server backup", "server", "backup", "create", "--name", id+"-server-backup", "--type", "daily", "--rotate", "1", serverID, "-f", "json")
	if backup.ExitCode == 0 {
		backupID := jsonStringField(backup.Stdout, "id", "ID")
		if backupID != "" {
			run.addCleanup("cleanup server backup image", "image", "delete", backupID)
			if _, ok := run.waitStatus("wait server backup image active", []string{"image", "show", backupID, "-f", "json"}, "status", []string{"active"}, 2*time.Minute); !ok {
				run.skip("server backup create completion", "backup create was accepted but the cloud did not return the image to active before timeout")
			}
			run.mustDeleteOrGone("delete server backup image", "image", "delete", backupID)
			run.dropCleanup("cleanup server backup image")
		}
	}
}

func (run *serverLifecycle) run(name string, args ...string) stepResult {
	return run.runWithEnv(name, nil, args...)
}

func (run *serverLifecycle) runWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := runCLIWithEnv(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	return result
}

func (run *serverLifecycle) must(name string, args ...string) stepResult {
	return run.mustWithEnv(name, nil, args...)
}

func (run *serverLifecycle) mustWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := run.runWithEnv(name, env, args...)
	if result.ExitCode != 0 {
		_ = run.cleanupAll()
		panic(serverLifecycleFailure{name: name})
	}
	return result
}

func (run *serverLifecycle) mustDeleteOrGone(name string, args ...string) stepResult {
	result := run.run(name, args...)
	if result.ExitCode != 0 && !looksDeleted(result) {
		_ = run.cleanupAll()
		panic(serverLifecycleFailure{name: name})
	}
	return result
}

func (run *serverLifecycle) optional(name string, args ...string) stepResult {
	return run.optionalWithEnv(name, nil, args...)
}

func (run *serverLifecycle) optionalWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := run.runWithEnv(name, env, args...)
	if result.ExitCode != 0 {
		run.skip(name+" follow-up", strings.TrimSpace(firstNonEmptyString(result.Error, result.Stderr, result.Stdout)))
	}
	return result
}

func (run *serverLifecycle) mustOracle(name string, env map[string]string, args ...string) stepResult {
	result := compareWithOracle(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(serverLifecycleFailure{name: name})
	}
	return result
}

func (run *serverLifecycle) mustOracleExisting(name string, goResult stepResult, env map[string]string, oracleArgs []string, replacements []parityReplacement) stepResult {
	result := compareExistingWithOracle(run.cloud, env, goResult, oracleArgs, replacements)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(serverLifecycleFailure{name: name})
	}
	return result
}

func (run *serverLifecycle) mustOraclePair(name string, env map[string]string, goArgs []string, oracleArgs []string, replacements []parityReplacement) stepResult {
	result := compareWithOracleArgs(run.cloud, env, goArgs, oracleArgs, replacements)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	if result.Error != "" {
		_ = run.cleanupAll()
		panic(serverLifecycleFailure{name: name})
	}
	return result
}

func (run *serverLifecycle) optionalOraclePair(name string, env map[string]string, goArgs []string, oracleArgs []string, replacements []parityReplacement) stepResult {
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

func (run *serverLifecycle) skip(name string, reason string) {
	run.diagnostics.Steps = append(run.diagnostics.Steps, stepResult{Name: name, Skipped: true, SkipReason: reason})
}

func (run *serverLifecycle) recordRiskSkipped(command string, reason string) {
	run.skip("skip "+command, reason)
}

func (run *serverLifecycle) addCleanup(name string, args ...string) {
	run.addCleanupWithEnv(name, nil, args...)
}

func (run *serverLifecycle) addCleanupWithEnv(name string, env map[string]string, args ...string) {
	run.diagnostics.CleanupRequired = true
	run.cleanups = append(run.cleanups, serverCleanup{name: name, env: env, args: args})
}

func (run *serverLifecycle) dropCleanup(name string) {
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

func (run *serverLifecycle) cleanupAll() error {
	var failures []error
	for i := len(run.cleanups) - 1; i >= 0; i-- {
		cleanup := run.cleanups[i]
		result := run.runWithEnv(cleanup.name, cleanup.env, cleanup.args...)
		if result.ExitCode != 0 && !looksDeleted(result) {
			failures = append(failures, fmt.Errorf("%s failed", cleanup.name))
		}
	}
	run.cleanups = nil
	run.diagnostics.CleanupRequired = len(failures) > 0
	return errors.Join(failures...)
}

func (run *serverLifecycle) mustWaitStatus(name string, args []string, field string, accepted []string, timeout time.Duration) stepResult {
	result, ok := run.waitStatus(name, args, field, accepted, timeout)
	if !ok {
		_ = run.cleanupAll()
		panic(serverLifecycleFailure{name: name})
	}
	return result
}

func (run *serverLifecycle) waitStatus(name string, args []string, field string, accepted []string, timeout time.Duration) (stepResult, bool) {
	deadline := time.Now().Add(timeout)
	var last stepResult
	for {
		last = run.run(name, args...)
		if last.ExitCode == 0 {
			status := strings.ToUpper(jsonStringField(last.Stdout, field, strings.ToLower(field), strings.ToUpper(field)))
			for _, value := range accepted {
				if status == strings.ToUpper(value) && serverLifecycleTaskStateCleared(last.Stdout) {
					return last, true
				}
			}
		}
		if time.Now().After(deadline) {
			return last, false
		}
		time.Sleep(time.Second)
	}
}

func serverLifecycleTaskStateCleared(jsonText string) bool {
	taskState := jsonStringField(jsonText, "OS-EXT-STS:task_state", "task_state")
	return taskState == "" || strings.EqualFold(taskState, "none")
}

func (run *serverLifecycle) mustWaitDeleted(name string, args []string, timeout time.Duration) stepResult {
	deadline := time.Now().Add(timeout)
	var last stepResult
	for {
		last = run.run(name, args...)
		if last.ExitCode != 0 && looksDeleted(last) {
			return last
		}
		if time.Now().After(deadline) {
			_ = run.cleanupAll()
			panic(serverLifecycleFailure{name: name})
		}
		time.Sleep(time.Second)
	}
}

func (run *serverLifecycle) recordImageFixture(jsonText string) {
	rows := jsonRows(jsonText)
	for _, row := range rows {
		status := strings.ToLower(jsonRowString(row, "Status", "status"))
		name := jsonRowString(row, "Name", "name")
		id := jsonRowString(row, "ID", "id")
		if id == "" || status != "active" {
			continue
		}
		if run.diagnostics.Fixtures["image_id"] == "" || strings.Contains(strings.ToLower(name), "cirros") {
			run.diagnostics.Fixtures["image_id"] = id
			run.diagnostics.Fixtures["image_name"] = name
			if strings.Contains(strings.ToLower(name), "cirros") {
				return
			}
		}
	}
}

func (run *serverLifecycle) recordFlavorFixtures(jsonText string) {
	rows := jsonRows(jsonText)
	type flavorCandidate struct {
		id   string
		name string
		ram  int
		disk int
	}
	var candidates []flavorCandidate
	for _, row := range rows {
		id := jsonRowString(row, "ID", "id")
		name := jsonRowString(row, "Name", "name")
		if id == "" {
			continue
		}
		candidates = append(candidates, flavorCandidate{id: id, name: name, ram: jsonRowInt(row, "RAM", "ram"), disk: jsonRowInt(row, "Disk", "disk")})
	}
	if len(candidates) == 0 {
		return
	}
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].ram < candidates[i].ram || (candidates[j].ram == candidates[i].ram && candidates[j].disk < candidates[i].disk) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	run.diagnostics.Fixtures["flavor_id"] = candidates[0].id
	run.diagnostics.Fixtures["flavor_name"] = candidates[0].name
	for _, candidate := range candidates[1:] {
		if candidate.id != candidates[0].id {
			run.diagnostics.Fixtures["alternate_flavor_id"] = candidate.id
			run.diagnostics.Fixtures["alternate_flavor_name"] = candidate.name
			return
		}
	}
}

func (run *serverLifecycle) recordNetworkFixtures(jsonText string) {
	rows := jsonRows(jsonText)
	for _, row := range rows {
		id := jsonRowString(row, "ID", "id")
		name := jsonRowString(row, "Name", "name")
		if id == "" {
			continue
		}
		if run.diagnostics.Fixtures["network_id"] == "" {
			run.diagnostics.Fixtures["network_id"] = id
			run.diagnostics.Fixtures["network_name"] = name
			continue
		}
		if run.diagnostics.Fixtures["alternate_network_id"] == "" {
			run.diagnostics.Fixtures["alternate_network_id"] = id
			run.diagnostics.Fixtures["alternate_network_name"] = name
		}
	}
	types := run.optional("preflight volume type list", "volume", "type", "list", "-f", "json")
	if types.ExitCode == 0 {
		for _, row := range jsonRows(types.Stdout) {
			name := jsonRowString(row, "Name", "name")
			if name == "" {
				continue
			}
			run.diagnostics.Fixtures["volume_type"] = name
			if strings.EqualFold(name, "LVM") {
				return
			}
		}
	}
}

type serverLifecycleFailure struct {
	name string
}

func (err serverLifecycleFailure) Error() string {
	return "server lifecycle failed at " + err.name
}

func compactJSON(jsonText string) string {
	var value any
	if err := json.Unmarshal([]byte(jsonText), &value); err != nil {
		return jsonText
	}
	data, err := json.Marshal(value)
	if err != nil {
		return jsonText
	}
	return string(data)
}
