### Communication
#### DomainManager (gRPC)
* subscribe vmi update, send command to virt-launcher's libvirtd
* path: `/var/run/kubevirt/sockets/{vmi_uid}_sock`
* Client: virt-handler ("kubevirt.io/kubevirt/pkg/virt-handler/cmd-client")
* Server: virt-launcher ("kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server")
* `k8s-client.informer -> virt-handler.VirtualMachineController -> virt-handler.cmd-client -> virt-launcher.cmd-server`
* Method(client):
```
SyncVirtualMachine(vmi *v1.VirtualMachineInstance) error
SyncMigrationTarget(vmi *v1.VirtualMachineInstance) error
ShutdownVirtualMachine(vmi *v1.VirtualMachineInstance) error
KillVirtualMachine(vmi *v1.VirtualMachineInstance) error
MigrateVirtualMachine(vmi *v1.VirtualMachineInstance) error
DeleteDomain(vmi *v1.VirtualMachineInstance) error
GetDomain() (*api.Domain, bool, error)
Ping() error
Close()
```

#### Watchdog (Text File)
* path: `/var/run/kubevirt/watchdog-files/{namespace}_{vmi_name}`
* content: {vmi_uid}
* ticker
> virt-launcher update file
> if libvirtd.exit || virtlog.exit || DomainManager.exit, then virt-launcher stop ticker
> virt-handler check file update time
> if libvirtd.domain.destroy, then virt-handler delete watchdog file

#### Domain-notify (RPC)
* send libvirt event to virt-handler
* path: `/var/run/kubevirt/domain-notify.sock`
* Server: virt-handler ("kubevirt.io/kubevirt/pkg/virt-handler/notify-server")
* Client: virt-launcher ("kubevirt.io/kubevirt/pkg/virt-launcher/notify-client")
* Method:
```
DomainEvent(args *DomainEventArgs)
type DomainEventArgs struct {
	DomainJSON string
	StatusJSON string
	EventType  string // ADDED, MODIFIED, DELETED, ERROR
}

// v0.14.0 support event: PV size too small 
K8sEvent(event k8sv1.Event)  
```