package monitor

import (
	"fmt"
	"time"

	libvirt "github.com/libvirt/libvirt-go"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	virtcli "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type libvirtEvent struct {
	Domain     string
	Event      *libvirt.DomainEventLifecycle
	AgentEvent *libvirt.DomainEventAgentLifecycle
}

// CreateLibvirtConnection :
// share process namespace between containers in same pod
// https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/
func CreateLibvirtConnection() virtcli.Connection {
	libvirtURI := "qemu:///system"
	domainConn, err := virtcli.NewConnection(libvirtURI, "", "", 10*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to libvirtd: %v", err))
	}

	return domainConn
}

func lookupDomainInfo(conn virtcli.Connection, fullName string, uid types.UID) *api.Domain {
	domain := util.NewDomainFromName(fullName, uid)
	if domain == nil {
		log.Log.Errorf("Could not craete api Domain.")
		return nil
	}

	virtDomain, err := conn.LookupDomainByName(fullName)
	if err != nil {
		if !domainerrors.IsNotFound(err) {
			log.Log.Reason(err).Error("Could not fetch the virt Domain.")
			return domain
		}
		domain.SetState(api.NoState, api.ReasonNonExistent)
	} else {
		defer virtDomain.Free()

		// No matter which event, try to fetch the domain xml
		// and the state. If we get a IsNotFound error, that
		// means that the VirtualMachineInstance was removed.
		status, reason, err := virtDomain.GetState()
		if err != nil {
			if !domainerrors.IsNotFound(err) {
				log.Log.Reason(err).Error("Could not fetch the Domain state.")
				return domain
			}
			domain.SetState(api.NoState, api.ReasonNonExistent)
		} else {
			domain.SetState(util.ConvState(status), util.ConvReason(status, reason))
		}

		spec, err := util.GetDomainSpecWithRuntimeInfo(status, virtDomain)
		if err != nil {
			// NOTE: Getting domain metadata for a live-migrating VM isn't allowed
			if !domainerrors.IsNotFound(err) && !domainerrors.IsInvalidOperation(err) {
				log.Log.Reason(err).Error("Could not fetch the Domain specification.")
				return domain
			}
		} else {
			domain.ObjectMeta.UID = spec.Metadata.KubeVirt.UID
		}
		if spec != nil {
			domain.Spec = *spec
		}

		log.Log.Infof("kubevirt domain status: %v(%v):%v(%v)", domain.Status.Status, status, domain.Status.Reason, reason)
	}
	return domain
}

// SubscribeDomainEvent :
// connect to libvirt, register domain event
// watch.Event{
//   Type: watch.Deleted | watch.Added | watch.Modified
//   Object: *api.Domain
// }
func SubscribeDomainEvent(domainConn virtcli.Connection, name string, namespace string, uid types.UID) chan watch.Event {
	go func() {
		for {
			if res := libvirt.EventRunDefaultImpl(); res != nil {
				log.Log.Reason(res).Error("Listening to libvirt events failed, retrying.")
				time.Sleep(time.Second)
			}
		}
	}()

	eventChan := make(chan watch.Event, 10)
	reconnectChan := make(chan bool, 10)
	domainConn.SetReconnectChan(reconnectChan)

	domainFullName := fmt.Sprintf("%s_%s", namespace, name)

	domainEventLifecycleCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
		log.Log.Infof("DomainLifecycle event %d with reason %d received", event.Event, event.Detail)
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info("Get name from EventLifecycleCallback error.")
			return
		} else if name != domainFullName {
			log.Log.Infof("Get name from EventLifecycleCallback not match: '%s' != '%s'", name, domainFullName)
			return
		}

		apiDomain := lookupDomainInfo(domainConn, name, uid)
		if apiDomain == nil {
			log.Log.Errorf("Could not fetch domain info from libvirt.")
			return
		}

		switch apiDomain.Status.Reason {
		case api.ReasonNonExistent:
			eventChan <- watch.Event{Type: watch.Deleted, Object: apiDomain}
		default:
			if event != nil {
				if event.Event == libvirt.DOMAIN_EVENT_DEFINED && libvirt.DomainEventDefinedDetailType(event.Detail) == libvirt.DOMAIN_EVENT_DEFINED_ADDED {
					eventChan <- watch.Event{Type: watch.Added, Object: apiDomain}
				} else if event.Event == libvirt.DOMAIN_EVENT_STARTED && libvirt.DomainEventStartedDetailType(event.Detail) == libvirt.DOMAIN_EVENT_STARTED_MIGRATED {
					eventChan <- watch.Event{Type: watch.Added, Object: apiDomain}
				}
			}
			eventChan <- watch.Event{Type: watch.Modified, Object: apiDomain}
		}
	}

	err := domainConn.DomainEventLifecycleRegister(domainEventLifecycleCallback)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to register event callback with libvirt")
		return nil
	}

	return eventChan
}
