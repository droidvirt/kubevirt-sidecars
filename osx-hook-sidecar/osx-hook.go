package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"google.golang.org/grpc"
	"net"
	"os"
	"strings"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	hookName         = "osx-hook"
)

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
				Priority: 1,
			},
		},
	}, nil
}

type v1alpha1Server struct{}

func (s v1alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info("Hook's OnDefineDomain callback method has been called")

	vmiJSON := params.GetVmi()
	vmiSpec := v1.VirtualMachineInstance{}
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

	converterStr, isExist := annotations[converterType]
	if !isExist {
		return nil, fmt.Errorf("miss converter")
	}
	log.Log.Infof("enable converter: %s", converterStr)

	converters := strings.Split(converterStr, ",")
	for _, converter := range converters {
		switch ConverterType(converter) {
		case BootLoaderConverter:
			addBootLoader(annotations, &domainSpec)
			break
		case BoardConverter:
			convertBoardType(&domainSpec)
			break
		case InputDeviceConverter:
			addInputDevice(&domainSpec)
			break
		case VncConverter:
			addVncQEMUArgs(annotations, &domainSpec)
			break
		case NICModelConverter:
			convertNicModel(&domainSpec)
			break
		case DiskDriverConverter:
			convertDiskOptions(annotations, &domainSpec)
			break
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
	// Start listening on /var/run/kubevirt-hooks/osx-hook.sock,
	// and register an infoServer (to expose information about this
	// hook) and a callback server (which does the heavy lifting).
	log.InitializeLogging("osx-hook-sidecar")

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
