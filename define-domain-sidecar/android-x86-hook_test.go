package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"testing"

	"kubevirt.io/kubevirt/pkg/api/v1"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func TestDefineVncGraphics(t *testing.T) {
	domainSpec := domainSchema.DomainSpec{}
	domainSpecXML, err := xml.Marshal(domainSpec)
	if err != nil {
		t.Errorf("Failed to marshal JSON")
	}

	vmi := new(v1.VirtualMachineInstance)
	annotations := map[string]string{
		vncPortAnnotation: "5900",
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

func TestDefineVncWebSocket(t *testing.T) {
	domainSpec := domainSchema.DomainSpec{}
	domainSpecXML, err := xml.Marshal(domainSpec)
	if err != nil {
		t.Errorf("Failed to marshal JSON")
	}

	vmi := new(v1.VirtualMachineInstance)
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)

	annotations := map[string]string{
		vncPortAnnotation: "5900",
		vncWebsocketPortAnnotation: "5901",
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
	t.Logf("%+v", updateDomainSpec)

	if err != nil {
		t.Errorf("Failed to unmarshal the domain spec")
	}

	if len(updateDomainSpec.Devices.Graphics) != 0 ||
		updateDomainSpec.QEMUCmd == nil ||
		len(updateDomainSpec.QEMUCmd.QEMUArg) != 2 ||
		updateDomainSpec.QEMUCmd.QEMUArg[0].Value != "-vnc" ||
		updateDomainSpec.QEMUCmd.QEMUArg[1].Value != "0.0.0.0:0,websocket=5901" {
		t.Fail()
	}
}

func TestDefineDiskDriver(t *testing.T) {
	domainSpec := domainSchema.DomainSpec{
		Devices: domainSchema.Devices{
			Disks: []domainSchema.Disk{
				{
					Device: "disk",
					Type:   "file",
					Driver: &domainSchema.DiskDriver{
						Name:  "qemu",
						Type:  "raw",
						Cache: "none",
					},
					Alias: &domainSchema.Alias{
						Name: "test-disk",
					},
				},
			},
		},
	}
	domainSpecXML, err := xml.Marshal(domainSpec)
	if err != nil {
		t.Errorf("Failed to marshal JSON")
	}

	vmi := new(v1.VirtualMachineInstance)
	annotations := map[string]string{
		diskNamesAnnotation:  "test-disk",
		diskDriverAnnotation: "qcow",
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

	if updateDomainSpec.Devices.Disks == nil || len(updateDomainSpec.Devices.Disks) == 0 {
		t.Errorf("Disks not set")
	}

	disk := updateDomainSpec.Devices.Disks[0]
	if disk.Driver == nil {
		t.Errorf("Disk Driver not set")
	}

	if disk.Driver.Name != "qemu" || disk.Driver.Type != "qcow" || disk.Driver.Cache != "" {
		t.Errorf("Disk Driver not change, %+v", disk.Driver)
	}

}
