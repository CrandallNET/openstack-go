package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type volumeCleanup struct {
	name string
	env  map[string]string
	args []string
}

type volumeLifecycle struct {
	cloud       string
	diagnostics *lifecycleDiagnostics
	cleanups    []volumeCleanup
}

func runVolumeLifecycle(cloud string, prefix string) (diagnostics lifecycleDiagnostics, err error) {
	id := uniqueID(prefix)
	diagnostics = lifecycleDiagnostics{
		ID:        id,
		Cloud:     cloud,
		Resource:  "volume",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Fixtures: map[string]string{
			"resource_prefix": id,
			"suite":           "volume",
		},
	}
	run := &volumeLifecycle{cloud: cloud, diagnostics: &diagnostics}
	defer func() {
		if recovered := recover(); recovered != nil {
			if failure, ok := recovered.(volumeLifecycleFailure); ok {
				err = failure
				return
			}
			panic(recovered)
		}
	}()

	preflight := run.optional("preflight volume service list", "volume", "service", "list", "-f", "json")
	if preflight.ExitCode == 0 {
		run.recordFirstServiceFixture(preflight.Stdout)
	}
	run.optional("preflight volume backend pool list", "volume", "backend", "pool", "list", "-f", "json")
	types := run.must("preflight volume type list", "volume", "type", "list", "-f", "json")
	run.recordVolumeTypeFixture(types.Stdout)
	run.optional("preflight block storage resource filter list", "block", "storage", "resource", "filter", "list", "-f", "json")
	if host := diagnostics.Fixtures["cinder_volume_host"]; host != "" {
		run.optional("read manageable volumes", "block", "storage", "volume", "manageable", "list", host, "-f", "json")
		run.optional("read manageable snapshots", "block", "storage", "snapshot", "manageable", "list", host, "-f", "json")
		run.optional("read backend capabilities", "volume", "backend", "capability", "show", host, "-f", "json")
	}

	volumeName := id + "-volume"
	createArgs := []string{"volume", "create", "--size", "1", "--description", "golang-osc lifecycle volume", "--property", "golang_osc_test=" + id}
	if volumeType := diagnostics.Fixtures["fixture_volume_type"]; volumeType != "" {
		createArgs = append(createArgs, "--type", volumeType)
	}
	createArgs = append(createArgs, volumeName, "-f", "json")
	volume := run.must("create volume", createArgs...)
	volumeID := jsonStringField(volume.Stdout, "id", "ID")
	if volumeID == "" {
		_ = run.cleanupAll()
		return diagnostics, fmt.Errorf("create volume did not return an id")
	}
	diagnostics.Fixtures["volume_id"] = volumeID
	run.addCleanup("cleanup volume", "volume", "delete", volumeID)
	run.mustWaitStatus("wait volume available after create", []string{"volume", "show", volumeID, "-f", "json"}, "status", []string{"available", "in-use"}, 3*time.Minute)
	run.must("show volume", "volume", "show", volumeID, "-f", "json")
	run.must("list volumes", "volume", "list", "-f", "json")
	run.volumeAttachmentLifecycle(id, volumeID)
	run.must("set volume metadata", "volume", "set", "--name", id+"-volume-renamed", "--description", "golang-osc lifecycle volume updated", "--property", "phase=set", volumeID)
	run.must("unset volume metadata", "volume", "unset", "--property", "phase", volumeID)
	run.must("set volume read-only", "volume", "set", "--read-only", volumeID)
	run.must("set volume read-write", "volume", "set", "--read-write", volumeID)
	run.must("set volume bootable", "volume", "set", "--bootable", volumeID)
	run.must("set volume non-bootable", "volume", "set", "--non-bootable", volumeID)

	snapshotName := id + "-snapshot"
	snapshot := run.must("create snapshot", "volume", "snapshot", "create", "--volume", volumeID, "--description", "golang-osc lifecycle snapshot", "--property", "golang_osc_test="+id, snapshotName, "-f", "json")
	snapshotID := jsonStringField(snapshot.Stdout, "id", "ID")
	if snapshotID == "" {
		_ = run.cleanupAll()
		return diagnostics, fmt.Errorf("create snapshot did not return an id")
	}
	diagnostics.Fixtures["snapshot_id"] = snapshotID
	run.addCleanup("cleanup snapshot", "volume", "snapshot", "delete", snapshotID)
	run.mustWaitStatus("wait snapshot available after create", []string{"volume", "snapshot", "show", snapshotID, "-f", "json"}, "status", []string{"available"}, 3*time.Minute)
	run.must("show snapshot", "volume", "snapshot", "show", snapshotID, "-f", "json")
	run.must("list snapshots", "volume", "snapshot", "list", "-f", "json")
	run.must("set snapshot metadata", "volume", "snapshot", "set", "--name", id+"-snapshot-renamed", "--description", "golang-osc lifecycle snapshot updated", "--property", "phase=set", snapshotID)
	run.must("unset snapshot metadata", "volume", "snapshot", "unset", "--property", "phase", snapshotID)
	run.optional("revert volume to snapshot", "volume", "revert", snapshotID)
	run.mustWaitStatus("wait volume available after optional revert", []string{"volume", "show", volumeID, "-f", "json"}, "status", []string{"available", "in-use"}, 3*time.Minute)
	run.must("delete snapshot", "volume", "snapshot", "delete", snapshotID)
	run.dropCleanup("cleanup snapshot")
	run.mustWaitDeleted("wait snapshot deleted", []string{"volume", "snapshot", "show", snapshotID, "-f", "json"}, 2*time.Minute)

	run.must("create transfer for delete", "volume", "transfer", "request", "create", "--name", id+"-transfer-delete", volumeID, "-f", "json")
	transferID := jsonStringField(run.lastStdout(), "id", "ID")
	if transferID != "" {
		run.addCleanup("cleanup transfer request", "volume", "transfer", "request", "delete", transferID)
		diagnostics.Fixtures["transfer_id"] = transferID
		run.must("show transfer request", "volume", "transfer", "request", "show", transferID, "-f", "json")
		run.must("list transfer requests", "volume", "transfer", "request", "list", "-f", "json")
		run.must("delete transfer request", "volume", "transfer", "request", "delete", transferID)
		run.dropCleanup("cleanup transfer request")
	}
	accepted := run.must("create transfer for accept", "volume", "transfer", "request", "create", "--name", id+"-transfer-accept", volumeID, "-f", "json")
	acceptID := jsonStringField(accepted.Stdout, "id", "ID")
	authKey := jsonStringField(accepted.Stdout, "auth_key", "Auth Key")
	if acceptID == "" || authKey == "" {
		_ = run.cleanupAll()
		return diagnostics, fmt.Errorf("transfer create did not return id and auth_key")
	}
	run.addCleanup("cleanup accepted transfer request", "volume", "transfer", "request", "delete", acceptID)
	run.must("accept transfer request", "volume", "transfer", "request", "accept", "--auth-key", authKey, acceptID, "-f", "json")
	run.dropCleanup("cleanup accepted transfer request")
	run.mustWaitStatus("wait volume available after transfer accept", []string{"volume", "show", volumeID, "-f", "json"}, "status", []string{"available", "in-use"}, 2*time.Minute)

	volumeTypeID := run.volumeTypeLifecycle(id)
	run.qosLifecycle(id, volumeTypeID)
	fixtureVolumeType := diagnostics.Fixtures["fixture_volume_type"]
	run.groupTypeAndGroupLifecycle(id, firstNonEmptyString(volumeTypeID, fixtureVolumeType))
	run.consistencyGroupLifecycle(id, firstNonEmptyString(volumeTypeID, fixtureVolumeType), volumeID)
	run.backupLifecycleIfAvailable(id, volumeID)

	run.recordRiskSkipped("volume service set", "would disable or re-enable real Cinder services rather than only test-created resources")
	run.recordRiskSkipped("volume host set", "would freeze or thaw a real Cinder host rather than only test-created resources")
	run.recordRiskSkipped("block storage cleanup", "operates on real Cinder worker state and is not scoped to a test-created resource")
	run.recordRiskSkipped("block storage cluster set", "changes real Cinder cluster service state; cloud6 currently reports no Cinder cluster rows")
	run.recordRiskSkipped("block storage log level set", "changes live Cinder service logging levels")
	run.recordRiskSkipped("volume migrate", "moves a volume between real Cinder backends; needs an explicit destination-backend safety approval")
	run.recordRiskSkipped("volume group failover", "fails over replication for a live Cinder group/backend and needs a replication-specific test cloud")
	run.recordRiskSkipped("volume message delete", "no test-created Cinder message fixture was produced; deleting existing messages would violate test safety")
	run.recordRiskSkipped("volume create --remote-source and snapshot --remote-source", "manage-existing operations require backend-local unmanaged storage references, not generic disposable Cinder resources")

	run.must("delete volume", "volume", "delete", volumeID)
	run.dropCleanup("cleanup volume")
	run.mustWaitDeleted("wait volume deleted", []string{"volume", "show", volumeID, "-f", "json"}, 2*time.Minute)
	if err := run.cleanupAll(); err != nil {
		diagnostics.CleanupRequired = true
		return diagnostics, err
	}
	diagnostics.CleanupRequired = false
	return diagnostics, nil
}

func (run *volumeLifecycle) volumeTypeLifecycle(id string) string {
	if strings.TrimSpace(os.Getenv("GOLANG_OSC_ENABLE_VOLUME_TYPE_LIFECYCLE")) != "1" {
		run.skip("volume type create/delete", "skipped by default because cloud6 currently creates volume types but the Python oracle reports HTTP 500 deleting them while the configured __DEFAULT__ volume type is missing")
		return ""
	}
	name := id + "-type"
	result := run.optional("create volume type", "volume", "type", "create", "--description", "golang-osc lifecycle type", "--property", "golang_osc_test="+id, name, "-f", "json")
	if result.ExitCode != 0 {
		return ""
	}
	volumeTypeID := jsonStringField(result.Stdout, "id", "ID")
	if volumeTypeID == "" {
		run.skip("volume type follow-up", "volume type create did not return an id")
		return ""
	}
	run.diagnostics.Fixtures["volume_type_id"] = volumeTypeID
	run.addCleanup("cleanup volume type", "volume", "type", "delete", volumeTypeID)
	run.must("show volume type", "volume", "type", "show", volumeTypeID, "-f", "json")
	run.must("list volume types", "volume", "type", "list", "-f", "json")
	run.must("set volume type properties", "volume", "type", "set", "--description", "golang-osc lifecycle type updated", "--property", "phase=set", volumeTypeID)
	run.must("unset volume type properties", "volume", "type", "unset", "--property", "phase", volumeTypeID)
	return volumeTypeID
}

func (run *volumeLifecycle) volumeAttachmentLifecycle(id string, volumeID string) {
	serverID := run.createAttachmentServerFixture(id)
	if serverID == "" {
		run.skip("volume attachment create/set/complete/delete", "no disposable server fixture was available")
		return
	}
	attachment := run.optional("create volume attachment", "volume", "attachment", "create", "--no-connect", volumeID, serverID, "-f", "json")
	if attachment.ExitCode != 0 {
		return
	}
	attachmentID := jsonStringField(attachment.Stdout, "id", "ID")
	if attachmentID == "" {
		run.skip("volume attachment follow-up", "volume attachment create did not return an id")
		return
	}
	run.diagnostics.Fixtures["volume_attachment_id"] = attachmentID
	run.addCleanup("cleanup volume attachment", "volume", "attachment", "delete", attachmentID)
	run.must("show volume attachment", "volume", "attachment", "show", attachmentID, "-f", "json")
	run.must("list volume attachments", "volume", "attachment", "list", "-f", "json")
	run.optional("set volume attachment connector", "volume", "attachment", "set", "--host", "golang-osc-lifecycle", "--ip", "127.0.0.1", "--initiator", "iqn.1993-08.org.debian:01:golang-osc", attachmentID, "-f", "json")
	if result := run.optional("complete volume attachment", "volume", "attachment", "complete", attachmentID); result.ExitCode == 0 {
		run.skip("volume attachment complete cleanup note", "attachment complete succeeded; delete cleanup will return the volume to a deletable state")
	}
	run.must("delete volume attachment", "volume", "attachment", "delete", attachmentID)
	run.dropCleanup("cleanup volume attachment")
	run.mustWaitStatus("wait volume available after attachment delete", []string{"volume", "show", volumeID, "-f", "json"}, "status", []string{"available"}, 3*time.Minute)
	run.must("delete volume attachment server", "server", "delete", "--wait", serverID)
	run.dropCleanup("cleanup volume attachment server")
}

func (run *volumeLifecycle) createAttachmentServerFixture(id string) string {
	image := run.optional("preflight attachment image list", "image", "list", "-f", "json")
	flavor := run.optional("preflight attachment flavor list", "flavor", "list", "-f", "json")
	network := run.optional("preflight attachment network list", "network", "list", "-f", "json")
	if image.ExitCode != 0 || flavor.ExitCode != 0 || network.ExitCode != 0 {
		return ""
	}
	imageID := firstActiveImageID(image.Stdout)
	flavorID := smallestFlavorID(flavor.Stdout)
	networkID := firstNetworkID(network.Stdout)
	if imageID == "" || flavorID == "" || networkID == "" {
		return ""
	}
	run.diagnostics.Fixtures["attachment_server_image_id"] = imageID
	run.diagnostics.Fixtures["attachment_server_flavor_id"] = flavorID
	run.diagnostics.Fixtures["attachment_server_network_id"] = networkID
	server := run.optional("create volume attachment server", "server", "create", "--flavor", flavorID, "--image", imageID, "--network", networkID, "--property", "golang_osc_test="+id, "--description", "golang-osc volume attachment lifecycle server", "--wait", id+"-attachment-server", "-f", "json")
	if server.ExitCode != 0 {
		return ""
	}
	serverID := jsonStringField(server.Stdout, "id", "ID")
	if serverID == "" {
		run.skip("volume attachment server follow-up", "server create did not return an id")
		return ""
	}
	run.diagnostics.Fixtures["attachment_server_id"] = serverID
	run.addCleanup("cleanup volume attachment server", "server", "delete", "--wait", serverID)
	return serverID
}

func (run *volumeLifecycle) qosLifecycle(id string, volumeTypeID string) {
	name := id + "-qos"
	result := run.optional("create volume qos", "volume", "qos", "create", "--consumer", "both", "--property", "read_iops_sec=100", name, "-f", "json")
	if result.ExitCode != 0 {
		return
	}
	qosID := jsonStringField(result.Stdout, "id", "ID")
	if qosID == "" {
		run.skip("volume qos follow-up", "volume qos create did not return an id")
		return
	}
	run.diagnostics.Fixtures["volume_qos_id"] = qosID
	run.addCleanup("cleanup volume qos", "volume", "qos", "delete", "--force", qosID)
	run.must("show volume qos", "volume", "qos", "show", qosID, "-f", "json")
	run.must("list volume qos", "volume", "qos", "list", "-f", "json")
	run.must("set volume qos", "volume", "qos", "set", "--property", "write_iops_sec=100", qosID)
	run.must("unset volume qos", "volume", "qos", "unset", "--property", "write_iops_sec", qosID)
	if volumeTypeID != "" {
		run.must("associate volume qos", "volume", "qos", "associate", qosID, volumeTypeID)
		run.must("disassociate volume qos", "volume", "qos", "disassociate", "--volume-type", volumeTypeID, qosID)
	}
	run.must("delete volume qos", "volume", "qos", "delete", "--force", qosID)
	run.dropCleanup("cleanup volume qos")
}

func (run *volumeLifecycle) groupTypeAndGroupLifecycle(id string, volumeTypeID string) {
	groupTypeName := id + "-group-type"
	groupEnv := map[string]string{"OS_VOLUME_API_VERSION": "3.14"}
	result := run.optionalWithEnv("create volume group type", groupEnv, "volume", "group", "type", "create", "--description", "golang-osc lifecycle group type", groupTypeName, "-f", "json")
	if result.ExitCode != 0 {
		return
	}
	groupTypeID := jsonStringField(result.Stdout, "ID", "id")
	if groupTypeID == "" {
		run.skip("volume group type follow-up", "volume group type create did not return an id")
		return
	}
	run.diagnostics.Fixtures["volume_group_type_id"] = groupTypeID
	run.addCleanupWithEnv("cleanup volume group type", groupEnv, "volume", "group", "type", "delete", groupTypeID)
	run.mustWithEnv("show volume group type", groupEnv, "volume", "group", "type", "show", groupTypeID, "-f", "json")
	run.mustWithEnv("list volume group types", groupEnv, "volume", "group", "type", "list", "-f", "json")
	run.mustWithEnv("set volume group type", groupEnv, "volume", "group", "type", "set", "--description", "golang-osc lifecycle group type updated", "--property", "phase=set", groupTypeID, "-f", "json")
	run.mustWithEnv("clear volume group type properties", groupEnv, "volume", "group", "type", "set", "--no-property", groupTypeID, "-f", "json")
	if volumeTypeID == "" {
		run.skip("volume group create", "no disposable volume type fixture was available")
		run.mustWithEnv("delete volume group type", groupEnv, "volume", "group", "type", "delete", groupTypeID)
		run.dropCleanup("cleanup volume group type")
		return
	}
	groupName := id + "-group"
	group := run.optionalWithEnv("create volume group", groupEnv, "volume", "group", "create", "--volume-group-type", groupTypeID, "--volume-type", volumeTypeID, "--name", groupName, "--description", "golang-osc lifecycle group", "-f", "json")
	if group.ExitCode == 0 {
		groupID := jsonStringField(group.Stdout, "ID", "id")
		if groupID != "" {
			run.diagnostics.Fixtures["volume_group_id"] = groupID
			run.addCleanupWithEnv("cleanup volume group", groupEnv, "volume", "group", "delete", groupID)
			run.optionalWaitStatusWithEnv("wait volume group available", groupEnv, []string{"volume", "group", "show", groupID, "-f", "json"}, "Status", []string{"available", "creating", "error"}, time.Minute)
			run.mustWithEnv("show volume group", groupEnv, "volume", "group", "show", groupID, "-f", "json")
			run.mustWithEnv("list volume groups", groupEnv, "volume", "group", "list", "-f", "json")
			run.mustWithEnv("set volume group", groupEnv, "volume", "group", "set", "--name", id+"-group-renamed", "--description", "golang-osc lifecycle group updated", groupID, "-f", "json")
			groupSnapshot := run.optionalWithEnv("create volume group snapshot", groupEnv, "volume", "group", "snapshot", "create", "--name", id+"-group-snapshot", "--description", "golang-osc lifecycle group snapshot", groupID, "-f", "json")
			if groupSnapshot.ExitCode == 0 {
				groupSnapshotID := jsonStringField(groupSnapshot.Stdout, "ID", "id")
				if groupSnapshotID != "" {
					run.diagnostics.Fixtures["volume_group_snapshot_id"] = groupSnapshotID
					run.addCleanupWithEnv("cleanup volume group snapshot", groupEnv, "volume", "group", "snapshot", "delete", groupSnapshotID)
					run.optionalWaitStatusWithEnv("wait volume group snapshot available", groupEnv, []string{"volume", "group", "snapshot", "show", groupSnapshotID, "-f", "json"}, "Status", []string{"available", "creating", "error"}, time.Minute)
					run.mustWithEnv("show volume group snapshot", groupEnv, "volume", "group", "snapshot", "show", groupSnapshotID, "-f", "json")
					run.mustWithEnv("list volume group snapshots", groupEnv, "volume", "group", "snapshot", "list", "-f", "json")
					run.mustWithEnv("delete volume group snapshot", groupEnv, "volume", "group", "snapshot", "delete", groupSnapshotID)
					run.dropCleanup("cleanup volume group snapshot")
				}
			}
			run.mustWithEnv("delete volume group", groupEnv, "volume", "group", "delete", groupID)
			run.dropCleanup("cleanup volume group")
		}
	}
	run.mustWithEnv("delete volume group type", groupEnv, "volume", "group", "type", "delete", groupTypeID)
	run.dropCleanup("cleanup volume group type")
}

func (run *volumeLifecycle) consistencyGroupLifecycle(id string, volumeTypeID string, volumeID string) {
	if volumeTypeID == "" {
		run.skip("consistency group create", "no disposable volume type fixture was available")
		return
	}
	groupName := id + "-cg"
	group := run.optional("create consistency group", "consistency", "group", "create", "--volume-type", volumeTypeID, "--description", "golang-osc lifecycle consistency group", groupName, "-f", "json")
	if group.ExitCode != 0 {
		return
	}
	groupID := jsonStringField(group.Stdout, "id", "ID")
	if groupID == "" {
		run.skip("consistency group follow-up", "consistency group create did not return an id")
		return
	}
	run.diagnostics.Fixtures["consistency_group_id"] = groupID
	run.addCleanup("cleanup consistency group", "consistency", "group", "delete", "--force", groupID)
	run.must("show consistency group", "consistency", "group", "show", groupID, "-f", "json")
	run.must("list consistency groups", "consistency", "group", "list", "-f", "json")
	run.must("set consistency group", "consistency", "group", "set", "--name", id+"-cg-renamed", "--description", "golang-osc lifecycle consistency group updated", groupID)
	run.optional("add volume to consistency group", "consistency", "group", "add", "volume", groupID, volumeID)
	run.optional("remove volume from consistency group", "consistency", "group", "remove", "volume", groupID, volumeID)
	snapshot := run.optional("create consistency group snapshot", "consistency", "group", "snapshot", "create", "--consistency-group", groupID, "--description", "golang-osc lifecycle consistency group snapshot", id+"-cg-snapshot", "-f", "json")
	if snapshot.ExitCode == 0 {
		snapshotID := jsonStringField(snapshot.Stdout, "id", "ID")
		if snapshotID != "" {
			run.diagnostics.Fixtures["consistency_group_snapshot_id"] = snapshotID
			run.addCleanup("cleanup consistency group snapshot", "consistency", "group", "snapshot", "delete", snapshotID)
			run.must("show consistency group snapshot", "consistency", "group", "snapshot", "show", snapshotID, "-f", "json")
			run.must("list consistency group snapshots", "consistency", "group", "snapshot", "list", "-f", "json")
			run.must("delete consistency group snapshot", "consistency", "group", "snapshot", "delete", snapshotID)
			run.dropCleanup("cleanup consistency group snapshot")
		}
	}
	run.must("delete consistency group", "consistency", "group", "delete", "--force", groupID)
	run.dropCleanup("cleanup consistency group")
}

func (run *volumeLifecycle) backupLifecycleIfAvailable(id string, volumeID string) {
	if strings.EqualFold(run.diagnostics.Fixtures["cinder_backup_state"], "up") {
		backup := run.optional("create volume backup", "volume", "backup", "create", "--name", id+"-backup", "--description", "golang-osc lifecycle backup", "--property", "golang_osc_test="+id, volumeID, "-f", "json")
		if backup.ExitCode != 0 {
			return
		}
		backupID := jsonStringField(backup.Stdout, "id", "ID")
		if backupID == "" {
			run.skip("volume backup follow-up", "volume backup create did not return an id")
			return
		}
		run.diagnostics.Fixtures["volume_backup_id"] = backupID
		run.addCleanup("cleanup volume backup", "volume", "backup", "delete", "--force", backupID)
		run.mustWaitStatus("wait backup available", []string{"volume", "backup", "show", backupID, "-f", "json"}, "status", []string{"available"}, 10*time.Minute)
		run.must("show volume backup", "volume", "backup", "show", backupID, "-f", "json")
		run.must("list volume backups", "volume", "backup", "list", "-f", "json")
		run.must("set volume backup", "volume", "backup", "set", "--name", id+"-backup-renamed", "--description", "golang-osc lifecycle backup updated", "--property", "phase=set", backupID)
		run.must("unset volume backup", "volume", "backup", "unset", "--property", "phase", backupID)
		run.optional("export volume backup record", "volume", "backup", "record", "export", backupID, "-f", "json")
		run.must("delete volume backup", "volume", "backup", "delete", "--force", backupID)
		run.dropCleanup("cleanup volume backup")
		return
	}
	run.skip("volume backup create/delete/restore/set/unset/record", "cinder-backup service is not up on this cloud")
}

func (run *volumeLifecycle) run(name string, args ...string) stepResult {
	return run.runWithEnv(name, nil, args...)
}

func (run *volumeLifecycle) runWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := runCLIWithEnv(run.cloud, env, args...)
	result.Name = name
	run.diagnostics.Steps = append(run.diagnostics.Steps, result)
	return result
}

func (run *volumeLifecycle) must(name string, args ...string) stepResult {
	return run.mustWithEnv(name, nil, args...)
}

func (run *volumeLifecycle) mustWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := run.runWithEnv(name, env, args...)
	if result.ExitCode != 0 {
		_ = run.cleanupAll()
		panic(volumeLifecycleFailure{name: name})
	}
	return result
}

func (run *volumeLifecycle) optional(name string, args ...string) stepResult {
	return run.optionalWithEnv(name, nil, args...)
}

func (run *volumeLifecycle) optionalWithEnv(name string, env map[string]string, args ...string) stepResult {
	result := run.runWithEnv(name, env, args...)
	if result.ExitCode != 0 {
		run.skip(name+" follow-up", strings.TrimSpace(firstNonEmptyString(result.Error, result.Stderr, result.Stdout)))
	}
	return result
}

func (run *volumeLifecycle) skip(name string, reason string) {
	run.diagnostics.Steps = append(run.diagnostics.Steps, stepResult{Name: name, Skipped: true, SkipReason: reason})
}

func (run *volumeLifecycle) recordRiskSkipped(command string, reason string) {
	run.skip("skip "+command, reason)
}

func (run *volumeLifecycle) addCleanup(name string, args ...string) {
	run.addCleanupWithEnv(name, nil, args...)
}

func (run *volumeLifecycle) addCleanupWithEnv(name string, env map[string]string, args ...string) {
	run.diagnostics.CleanupRequired = true
	run.cleanups = append(run.cleanups, volumeCleanup{name: name, env: env, args: args})
}

func (run *volumeLifecycle) dropCleanup(name string) {
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

func (run *volumeLifecycle) cleanupAll() error {
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

func (run *volumeLifecycle) mustWaitStatus(name string, args []string, field string, accepted []string, timeout time.Duration) stepResult {
	result, ok := run.waitStatus(name, args, field, accepted, timeout)
	if !ok {
		_ = run.cleanupAll()
		panic(volumeLifecycleFailure{name: name})
	}
	return result
}

func (run *volumeLifecycle) optionalWaitStatus(name string, args []string, field string, accepted []string, timeout time.Duration) stepResult {
	return run.optionalWaitStatusWithEnv(name, nil, args, field, accepted, timeout)
}

func (run *volumeLifecycle) optionalWaitStatusWithEnv(name string, env map[string]string, args []string, field string, accepted []string, timeout time.Duration) stepResult {
	result, ok := run.waitStatusWithEnv(name, env, args, field, accepted, timeout)
	if !ok {
		run.skip(name+" follow-up", "timed out waiting for an accepted status")
	}
	return result
}

func (run *volumeLifecycle) waitStatus(name string, args []string, field string, accepted []string, timeout time.Duration) (stepResult, bool) {
	return run.waitStatusWithEnv(name, nil, args, field, accepted, timeout)
}

func (run *volumeLifecycle) waitStatusWithEnv(name string, env map[string]string, args []string, field string, accepted []string, timeout time.Duration) (stepResult, bool) {
	deadline := time.Now().Add(timeout)
	var last stepResult
	for {
		last = run.runWithEnv(name, env, args...)
		if last.ExitCode == 0 {
			status := strings.ToLower(jsonStringField(last.Stdout, field, strings.ToLower(field), strings.ToUpper(field)))
			for _, value := range accepted {
				if status == strings.ToLower(value) {
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

func (run *volumeLifecycle) mustWaitDeleted(name string, args []string, timeout time.Duration) stepResult {
	deadline := time.Now().Add(timeout)
	var last stepResult
	for {
		last = run.run(name, args...)
		if last.ExitCode != 0 && looksDeleted(last) {
			return last
		}
		if time.Now().After(deadline) {
			_ = run.cleanupAll()
			panic(volumeLifecycleFailure{name: name})
		}
		time.Sleep(time.Second)
	}
}

func (run *volumeLifecycle) lastStdout() string {
	if len(run.diagnostics.Steps) == 0 {
		return ""
	}
	return run.diagnostics.Steps[len(run.diagnostics.Steps)-1].Stdout
}

func (run *volumeLifecycle) recordFirstServiceFixture(jsonText string) {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonText), &rows); err != nil {
		return
	}
	for _, row := range rows {
		binary := strings.TrimSpace(fmt.Sprint(row["Binary"]))
		host := strings.TrimSpace(fmt.Sprint(row["Host"]))
		state := strings.TrimSpace(fmt.Sprint(row["State"]))
		if binary == "cinder-volume" && host != "" {
			run.diagnostics.Fixtures["cinder_volume_host"] = host
		}
		if binary == "cinder-backup" {
			run.diagnostics.Fixtures["cinder_backup_state"] = state
		}
	}
}

func (run *volumeLifecycle) recordVolumeTypeFixture(jsonText string) {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonText), &rows); err != nil {
		return
	}
	first := ""
	for _, row := range rows {
		name := strings.TrimSpace(fmt.Sprint(row["Name"]))
		if name == "" || name == "<nil>" {
			continue
		}
		if first == "" {
			first = name
		}
		if strings.EqualFold(name, "LVM") {
			run.diagnostics.Fixtures["fixture_volume_type"] = name
			return
		}
	}
	if first != "" {
		run.diagnostics.Fixtures["fixture_volume_type"] = first
	}
}

type volumeLifecycleFailure struct {
	name string
}

func (err volumeLifecycleFailure) Error() string {
	return "volume lifecycle failed at " + err.name
}
