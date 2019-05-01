## MigrationPhase
```
MigrationPhaseUnset = "" 
MigrationPending = "Pending"                 // The migration is accepted by the system
MigrationScheduling = "Scheduling"           // Target pod is being scheduled
MigrationScheduled = "Scheduled"             // Target pod is running
MigrationPreparingTarget = "PreparingTarget" // Target pod is being prepared for migration 
MigrationTargetReady = "TargetReady"         // Target pod is prepared and ready for migration
MigrationRunning = "Running"                 // The migration in progress
MigrationSucceeded = "Succeeded"             // The migration passed
MigrationFailed = "Failed"                   // The migration failed
```

## Step

#### virt-api
* receive and validate `VirtualMachineInstanceMigration` CRD, phase become `MigrationPending` 

#### virt-scheduler
* `MigrationController`'s informer receive event, create target pod, phase become `MigrationScheduling`

```
PodAntiAffinity{
	RequiredDuringSchedulingIgnoredDuringExecution: PodAffinityTerm{
		LabelSelector: {
			MatchLabels: {
				"kubevirt.io/created-by": VMI.UID
			}
		}
	}
} 
```

* wait target pod ready, phase become `MigrationScheduled`
* set target pod's label(labels["kubevirt.io/migrationTargetNodeName"]="nodeName"), phase become `MigrationPreparingTarget`
* wait `MigrationState.TargetNodeAddress` ready, phase become `MigrationTargetReady`
* wait 

#### virt-handler (VirtualMachineController)
* wait labels("kubevirt.io/migrationTargetNodeName") ready
* wait Migration Proxy ready, set migration status:
```
vmi.Status.MigrationState.TargetNodeAddress = {POD_IP}
// from exposed pod port to local libvirt migration port
vmi.Status.MigrationState.TargetDirectMigrationNodePorts = map[int]int{
	{HostListenRandomPort}: 49152,
	{HostListenRandomPort}: 49153,
	{HostListenRandomPort}: 0
}
```

#### virt-launcher
* wait `MigrationState.TargetNodeAddress` ready
* source and target: clean and init `DomainManager` UNIX Domain socket, prepare Migration Proxy
* source: set domain metadata, set host file(127.0.0.1 targetPod)
```
<metadata>
	<kubevirt xmlns="http://kubevirt.io">
		<uid></uid>
		<graceperiod>
			<deletionGracePeriodSeconds>0</deletionGracePeriodSeconds>
		</graceperiod>
		<migration>
			<uid></uid>
			<startTimestamp></startTimestamp>
		</migration>
	</kubevirt>
</metadata>
```
* target:


## Migration Proxy

### Migrate Source UNIX Socket (at source pod's virt-launcher)
* local libvirtd proxy (Right now Libvirt won't let us perform a migration using a unix socket, so we have to create a local host tcp server that forwards the traffic): `127.0.0.1:22222` -> `/var/run/kubevirt/migrationproxy/{UID}-source.sock`
* direct migrate: `127.0.0.1:49152` -> `/var/run/kubevirt/migrationproxy/{UID}-49152-source.sock`
* block migrate: `127.0.0.1:49153` -> `/var/run/kubevirt/migrationproxy/{UID}-49153-source.sock`

### Migrate Target UNIX Socket (at target pod's virt-launcher)
* direct migrate: `/var/run/kubevirt/migrationproxy/{UID}-49152-source.sock` -> `127.0.0.1:49152` 
* block migrate: `/var/run/kubevirt/migrationproxy/{UID}-49153-source.sock` -> `127.0.0.1:49153`

### Expose TCP node port to local UNIX Socket (at target node's virt-handler)
* libvirtd proxy (listen on TCP, forward to UNIX Socket): (node)`0.0.0.0:` -> `/proc/{libvirtd_PID}/root/var/run/libvirt/libvirt-sock`
* direct migrate: (node)`0.0.0.0:{port}` -> `/var/run/kubevirt/migrationproxy/{UID}-49152-source.sock`
* block migrate: (node)`0.0.0.0:{port}` -> `/var/run/kubevirt/migrationproxy/{UID}-49153-source.sock`

### Expose local UNIX Socket to remote TCP node port (at source node's virt-handler)
* libvirtd proxy: `/var/run/kubevirt/migrationproxy/{UID}-source.sock` -> `{TargetNodeIP}:{port}`
* direct migrate: `/var/run/kubevirt/migrationproxy/{UID}-49152-source.sock` -> `{TargetNodeIP}:{port}`
* block migrate: `/var/run/kubevirt/migrationproxy/{UID}-49153-source.sock` -> `{TargetNodeIP}:{port}`



