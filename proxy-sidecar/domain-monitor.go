package main

import (
	"fmt"
	"time"

	libvirt "github.com/libvirt/libvirt-go"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	virtcli "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type libvirtEvent struct {
	Domain     string
	Event      *libvirt.DomainEventLifecycle
	AgentEvent *libvirt.DomainEventAgentLifecycle
}

// share process namespace between containers in same pod
// https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/
func createLibvirtConnection() virtcli.Connection {
	libvirtURI := "qemu:///system"
	domainConn, err := virtcli.NewConnection(libvirtURI, "", "", 10*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to libvirtd: %v", err))
	}

	return domainConn
}

func subscribeDomainEvent(domainConn virtcli.Connection, name string, namespace string, uid types.UID) chan watch.Event {
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

	domainEventLifecycleCallback := func(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
		log.Log.Infof("DomainLifecycle event %d with reason %d received", event.Event, event.Detail)
		name, err := d.GetName()
		if err != nil {
			log.Log.Reason(err).Info("Could not determine name of libvirt domain in event callback.")
			return
		}

		apiDomain := util.NewDomainFromName(name, uid)
		if apiDomain == nil {
			log.Log.Errorf("Could not craete api Domain.")
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
