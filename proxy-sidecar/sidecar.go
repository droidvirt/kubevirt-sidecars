package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	libvirt "github.com/libvirt/libvirt-go"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

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

func waitLauncherReady(readinessFilePath string, checkTimes int) (bool, error) {
	checkCount := 0
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		isExist, err := fileExists(readinessFilePath)
		log.Log.Infof("Try check virt-launcher ready count: %d", checkCount)
		if err != nil || checkCount > checkTimes {
			return false, errors.New("check error or timeout")
		} else if isExist {
			return true, nil
		} else {
			checkCount++
		}
	}
	return false, errors.New("unknown")
}

func markReady(readinessFile string) {
	f, err := os.OpenFile(readinessFile, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	f.Close()
	log.Log.Info("Marked as ready")
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	exists := false

	if err == nil {
		exists = true
	} else if os.IsNotExist(err) {
		err = nil
	}
	return exists, err
}

func waitForDomainUUID(timeout time.Duration, events chan watch.Event, stop chan struct{}) *api.Domain {

	ticker := time.NewTicker(timeout).C
	select {
	case <-ticker:
		log.Log.Errorf("timed out waiting for domain to be defined")
		return nil
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
	name := pflag.String("name", args.domainName, "Name of the VirtualMachineInstance")
	uid := pflag.String("uid", args.domainUID, "UID of the VirtualMachineInstance")
	namespace := pflag.String("namespace", args.podNamespace, "Namespace of the VirtualMachineInstance")
	launcherReadinessFile := pflag.String("launcher-readiness-file", args.launcherReadiness, "Pod looks for this file to determine when virt-launcher is initialized")
	sidecarReadinessFile := pflag.String("sidecar-readiness-file", "/var/run/kubevirt-infra/healthy_sidecar", "Pod looks for this file to determine when proxy sidecar is initialized")
	launcherCheckTimes := pflag.Int("launcher-check-times", 15, "Times of virt-launcher check")
	qemuTimeout := pflag.Duration("qemu-timeout", 3*time.Minute, "Amount of time to wait for qemu")

	isReady, err := waitLauncherReady(*launcherReadinessFile, *launcherCheckTimes)
	if err != nil || !isReady {
		panic(fmt.Errorf("Wait libvirt ready error: %s", err))
	}

	domainConn := createLibvirtConnection()
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

	events := subscribeDomainEvent(domainConn, *name, *namespace, types.UID(*uid))
	markReady(*sidecarReadinessFile)

	for {
		domain := waitForDomainUUID(*qemuTimeout, events, signalStopChan)
		if domain != nil {
			log.Log.Infof("Domain added: %+v", domain)
		}
	}

}
