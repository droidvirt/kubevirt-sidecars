package main

import (
	"fmt"
	"github.com/libvirt/libvirt-go"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"os"
	"os/signal"
	"syscall"

	"kubevirt.io/kubevirt/cmd/droidvirt-sidecar/proxy-sidecar/monitor"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type cmdArgs struct {
	domainName        string
	domainUID         string
	podNamespace      string
	launcherReadiness string
}

var (
	// https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/
	// domainNameEnv is from metadata.annotations['kubevirt.io/domain']
	// domainUIDEnv is from metadata.labels['kubevirt.io/created-by']
	// podNamespaceEnv is from metadata.namespace
	domainNameEnv   = "DOMAIN_NAME"
	domainUIDEnv    = "DOMAIN_UID"
	podNamespaceEnv = "POD_NAMESPACE"
	args            cmdArgs
)

func init() {
	// must registry the event impl before doing anything else.
	libvirt.EventRegisterDefaultImpl()

	// init command args flags
	args.domainName = os.Getenv(domainNameEnv)
	args.domainUID = os.Getenv(domainUIDEnv)
	args.podNamespace = os.Getenv(podNamespaceEnv)
	args.launcherReadiness = "/var/run/kubevirt-infra/healthy"
}

func markReady(readinessFile string) {
	f, err := os.OpenFile(readinessFile, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	f.Close()
	log.Log.Info("Marked as ready")
}

func waitForDomainUUID(events chan watch.Event, stop chan struct{}) *api.Domain {

	select {
	case e := <-events:
		if e.Type == watch.Deleted {
			return nil
		}
		if e.Object != nil && e.Type == watch.Added {
			domain := e.Object.(*api.Domain)
			log.Log.Infof("Detected domain with UUID %s", domain.Spec.UUID)
			return domain
		}
	case <-stop:
		return nil
	}
	return nil
}

func main() {
	// domain
	name := pflag.String("name", args.domainName, "Name of the VirtualMachineInstance")
	uid := pflag.String("uid", args.domainUID, "UID of the VirtualMachineInstance")
	namespace := pflag.String("namespace", args.podNamespace, "Namespace of the VirtualMachineInstance")
	// readiness
	launcherReadinessFile := pflag.String("launcher-readiness-file", args.launcherReadiness, "Pod looks for this file to determine when virt-launcher is initialized")
	sidecarReadinessFile := pflag.String("sidecar-readiness-file", "/var/run/kubevirt-infra/healthy_sidecar", "Pod looks for this file to determine when proxy sidecar is initialized")
	launcherCheckTimes := pflag.Int("launcher-check-times", 15, "Times of virt-launcher check")
	// socks
	socksServer := pflag.String("socks-server", "192.168.80.211", "socks server address")
	socksPort := pflag.Int("socks-port", 1080, "socks server port")
	socksPassword := pflag.String("socks-password", "password", "socks server address")

	isReady, err := monitor.WaitLauncherReady(*launcherReadinessFile, *launcherCheckTimes)
	if err != nil || !isReady {
		panic(fmt.Errorf("wait libvirt ready error: %s", err))
	}

	domainConn := monitor.CreateLibvirtConnection()
	defer domainConn.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	signalStopChan := make(chan struct{})
	go func() {
		s := <-c
		log.Log.Infof("Received signal %s", s.String())
		close(signalStopChan)
	}()

	startSocksProxy(*socksServer, *socksPort, *socksPassword, signalStopChan)

	events := monitor.SubscribeDomainEvent(domainConn, *name, *namespace, types.UID(*uid))
	markReady(*sidecarReadinessFile)

	for {
		domain := waitForDomainUUID(events, signalStopChan)
		if domain != nil {
			log.Log.Infof("Domain added: %v", domain.Spec)
			err := hackIptables(socksLocalAddress, socksLoadlPort)
			if err != nil {
				log.Log.Reason(err).Errorf("hack iptables error")
			}
		}
	}

}
