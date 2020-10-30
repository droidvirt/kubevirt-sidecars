package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"sort"
	"strings"
	"testing"

	v1 "kubevirt.io/client-go/api/v1"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	fakeLoaderPath = "/tmp/OVMF_CODE.fd"
	fakeNVRamPath  = "/tmp/OVMF_VARS.fd"
)

func TestOSType(t *testing.T) {
	domainSpec := domainSchema.DomainSpec{
		OS: domainSchema.OS{
			Type: domainSchema.OSType{
				Arch:    "x86_64",
				OS:      "hvm",
				Machine: "pc-q35-3.0",
			},
		},
	}
	domainSpecXML, err := xml.Marshal(domainSpec)
	if err != nil {
		t.Errorf("Failed to marshal JSON")
	}

	vmi := new(v1.VirtualMachineInstance)
	annotations := map[string]string{
		loaderPath: fakeLoaderPath,
		nvramPath:  fakeNVRamPath,
	}

	vmi.SetAnnotations(annotations)

	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		t.Errorf("Failed to marshal JSON")
	}

	params := hooksV1alpha1.OnDefineDomainParams{domainSpecXML, vmiJSON}

	ctx := context.TODO()

	server := new(v1alpha1Server)
	result, err := server.OnDefineDomain(ctx, &params)
	if err != nil {
		t.Errorf("Failed to invoke OnDefineDomain")
	}

	updateDomainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(result.GetDomainXML(), &updateDomainSpec)
	if err != nil {
		t.Errorf("Failed to unmarshal the domain spec")
	}

	if updateDomainSpec.OS.BootLoader == nil || updateDomainSpec.OS.BootLoader.Path != fakeLoaderPath {
		t.Errorf("Unexpected boot loader")
	}

	if updateDomainSpec.OS.NVRam == nil || updateDomainSpec.OS.NVRam.NVRam != fakeNVRamPath {
		t.Errorf("Unexpected nvram")
	}
}

type callBackClient struct {
	SocketPath          string
	Version             string
	subsribedHookPoints []*hooksInfo.HookPoint
}

func TestDefineVncGraphics(t *testing.T) {
	domainSpec := domainSchema.DomainSpec{}
	domainSpecXML, err := xml.Marshal(domainSpec)
	if err != nil {
		t.Errorf("Failed to marshal JSON")
	}

	vmi := new(v1.VirtualMachineInstance)
	annotations := map[string]string{
		vncPort: "5900",
	}

	vmi.SetAnnotations(annotations)

	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		t.Errorf("Failed to marshal JSON")
	}

	params := hooksV1alpha1.OnDefineDomainParams{domainSpecXML, vmiJSON}

	ctx := context.TODO()

	server := new(v1alpha1Server)
	result, err := server.OnDefineDomain(ctx, &params)
	if err != nil {
		t.Errorf("Failed to invoke OnDefineDomain")
	}

	updateDomainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(result.GetDomainXML(), &updateDomainSpec)
	t.Logf("%+v", updateDomainSpec.Devices.Graphics[0])

	if err != nil {
		t.Errorf("Failed to unmarshal the domain spec")
	}

	if updateDomainSpec.Devices.Graphics[0].Port != 5900 {
		t.Errorf("Unexpected graphics type")
	}

}

func TestSort(t *testing.T) {
	callbacksPerHookPoint := make(map[string][]*callBackClient)
	callbacksPerHookPoint["OnDefineDomain"] = append(callbacksPerHookPoint["OnDefineDomain"], &callBackClient{
		SocketPath: "/tmp/hook1.sock",
		Version:    "v1alpha1",
		subsribedHookPoints: []*hooksInfo.HookPoint{
			&hooksInfo.HookPoint{
				Name:     "OnDefineDomain",
				Priority: 0,
			},
		},
	})

	callbacksPerHookPoint["OnDefine"] = append(callbacksPerHookPoint["OnDefine"], &callBackClient{
		SocketPath: "/tmp/hook2.sock",
		Version:    "v1alpha1",
		subsribedHookPoints: []*hooksInfo.HookPoint{
			&hooksInfo.HookPoint{
				Name:     "OnDefine",
				Priority: 0,
			},
		},
	})

	sortCallbacksPerHookPoint(t, callbacksPerHookPoint)
}

func sortCallbacksPerHookPoint(t *testing.T, callbacksPerHookPoint map[string][]*callBackClient) {
	for _, callbacks := range callbacksPerHookPoint {
		sort.Slice(callbacks, func(i, j int) bool {
			return strings.Compare(callbacks[i].SocketPath, callbacks[j].SocketPath) < 0
		})
		for _, callback := range callbacks {
			sort.Slice(callback.subsribedHookPoints, func(i, j int) bool {
				t.Logf("i=%d, j=%d", i, j)
				if callback.subsribedHookPoints[i].Priority == callback.subsribedHookPoints[j].Priority {
					return strings.Compare(callback.subsribedHookPoints[i].Name, callback.subsribedHookPoints[j].Name) < 0
				} else {
					return callback.subsribedHookPoints[i].Priority > callback.subsribedHookPoints[j].Priority
				}
			})
		}
	}
}
