package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc"

	vmSchema "kubevirt.io/kubevirt/pkg/api/v1"
	hooks "kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	"kubevirt.io/kubevirt/pkg/log"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const vncPortAnnotation = "vnc.droidvirt.io/port"

// split name by comma
const diskNamesAnnotation = "disk.droidvirt.io/names"
const diskDriverAnnotation = "disk.droidvirt.io/driverType"
const hookName = "droidvirt-define-domain"

type infoServer struct{}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: hookName,
		Versions: []string{
			hooksV1alpha1.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			&hooksInfo.HookPoint{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha1Server struct{}

func (s v1alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info("Hook's OnDefineDomain callback method has been called")

	vmiJSON := params.GetVmi()
	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	annotations := vmiSpec.GetAnnotations()

	domainXML := params.GetDomainXML()
	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	if vncPort, found := annotations[vncPortAnnotation]; !found {
		log.Log.Infof("The '%s' attribute was not provided. Not configuring vnc listen port.", vncPortAnnotation)
	} else {
		port, err := strconv.ParseInt(vncPort, 10, 32)
		if err != nil {
			log.Log.Infof("Invalid VNC Port: %s", vncPort)
		} else {
			domainSpec.Devices.Graphics = []domainSchema.Graphics{
				{
					Type: "vnc",
					Port: int32(port),
					Listen: &domainSchema.GraphicsListen{
						Type:    "address",
						Address: "0.0.0.0",
					},
				},
			}
		}
	}

	// change data disk driver type: qcow2
	if diskNames, found := annotations[diskNamesAnnotation]; !found {
		log.Log.Infof("The '%s' attribute was not provided. Not configuring disk names.", diskNamesAnnotation)
	} else {
		driverType := annotations[diskDriverAnnotation]
		if driverType == "" {
			driverType = "qcow2"
		}
		names := strings.Split(diskNames, ",")
		for idx, disk := range domainSpec.Devices.Disks {
			if disk.Alias != nil {
				for _, name := range names {
					if name == disk.Alias.Name {
						domainSpec.Devices.Disks[idx].Driver = &domainSchema.DiskDriver{
							Name: "qemu",
							Type: driverType,
						}
						log.Log.Infof("After Change: %+v", domainSpec.Devices.Disks[idx].Driver)
						break
					}
				}
			}
		}
	}

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %s", err.Error())
		panic(err)
	}

	log.Log.Info("Successfully updated original domain spec with requested attributes")

	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func main() {
	// Start listening on /var/run/kubevirt-hooks/android-x86.sock,
	// and register an infoServer (to expose information about this
	// hook) and a callback server (which does the heavy lifting).
	log.InitializeLogging("droidvirt-hook-sidecar")

	socketPath := hooks.HookSocketsSharedDirectory + "/" + hookName + ".sock"
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, infoServer{})
	hooksV1alpha1.RegisterCallbacksServer(server, v1alpha1Server{})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha1' services on socket %s", socketPath)
	server.Serve(socket)
}
