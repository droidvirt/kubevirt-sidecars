package main

import (
	"fmt"
	"strconv"
	"strings"

	"kubevirt.io/client-go/log"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	vncBindAddress    = "0.0.0.0"
	defaultDiskDriver = "qcow2"
)

func addBootLoader(annotations map[string]string, domainSpec *domainSchema.DomainSpec) {
	if loaderPath, found := annotations[loaderPath]; found {
		domainSpec.OS.BootLoader = &domainSchema.Loader{
			Path:     loaderPath,
			ReadOnly: "yes",
			Secure:   "no",
			Type:     "pflash",
		}
	}
	if nvramPath, found := annotations[nvramPath]; found {
		domainSpec.OS.NVRam = &domainSchema.NVRam{
			NVRam: nvramPath,
		}
	}
}

func addInputDevice(domainSpec *domainSchema.DomainSpec) {
	inputDevices := make([]domainSchema.Input, 0)

	inputDevices = append(inputDevices, domainSchema.Input{
		Type: "keyboard",
		Bus:  "ps2",
	})

	inputDevices = append(inputDevices, domainSchema.Input{
		Type: "mouse",
		Bus:  "ps2",
	})

	inputDevices = append(inputDevices, domainSchema.Input{
		Type: "tablet",
		Bus:  "usb",
	})

	inputDevices = append(inputDevices, domainSchema.Input{
		Type: "keyboard",
		Bus:  "usb",
	})

	domainSpec.Devices.Inputs = inputDevices

	for idx, ctrl := range domainSpec.Devices.Controllers {
		if ctrl.Type == "usb" && ctrl.Model == "none" {
			domainSpec.Devices.Controllers = append(domainSpec.Devices.Controllers[:idx], domainSpec.Devices.Controllers[idx+1:]...)
			break
		}
	}

	domainSpec.Devices.Controllers = append(domainSpec.Devices.Controllers, domainSchema.Controller{
		Type:  "usb",
		Index: "0",
		Model: "piix3-uhci",
	})
}

func convertBoardType(domainSpec *domainSchema.DomainSpec) {
	log.Log.Info("Set options in XML 'qemu:commandline'")
	if domainSpec.XmlNS == "" {
		domainSpec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	}

	if domainSpec.QEMUCmd == nil {
		domainSpec.QEMUCmd = &domainSchema.Commandline{}
	}

	if domainSpec.QEMUCmd.QEMUArg == nil {
		domainSpec.QEMUCmd.QEMUArg = make([]domainSchema.Arg, 0)
	}

	args := []string{
		"-device",
		"isa-applesmc,osk=ourhardworkbythesewordsguardedpleasedontsteal(c)AppleComputerInc",
		"-smbios",
		"type=2",
		"-cpu",
		"Penryn,kvm=on,vendor=GenuineIntel,+invtsc,vmware-cpuid-freq=on,+pcid,+ssse3,+sse4.2,+popcnt,+avx,+aes,+xsave,+xsaveopt,check",
	}

	for _, arg := range args {
		domainSpec.QEMUCmd.QEMUArg = append(domainSpec.QEMUCmd.QEMUArg, domainSchema.Arg{
			Value: arg,
		})
	}
}

func addVncQEMUArgs(annotations map[string]string, domainSpec *domainSchema.DomainSpec) {
	var heads uint = 1
	var ram uint = 65536
	var vram uint = 65536
	var vgamem uint = 16384
	domainSpec.Devices.Video = []domainSchema.Video{
		{
			Model: domainSchema.VideoModel{
				Type:   "qxl",
				Heads:  &heads,
				Ram:    &ram,
				VRam:   &vram,
				VGAMem: &vgamem,
			},
		},
	}

	if vncPortStr, found := annotations[vncPort]; found {
		vncPort, err := strconv.ParseInt(vncPortStr, 10, 32)
		if err != nil || vncPort < 5900 {
			log.Log.Errorf("Invalid VNC Port: %s", vncPortStr)
			return
		}

		if wsPortStr, found := annotations[vncWebsocketPort]; !found {
			log.Log.Info("No WebSocket. Set options in XML 'devices.graphics' directly")
			domainSpec.Devices.Graphics = []domainSchema.Graphics{
				{
					Type: "vnc",
					Port: int32(vncPort),
					Listen: &domainSchema.GraphicsListen{
						Type:    "address",
						Address: vncBindAddress,
					},
				},
			}
		} else {
			wsPort, err := strconv.ParseInt(wsPortStr, 10, 32)
			if err != nil || wsPort < 5900 || wsPort == vncPort {
				log.Log.Errorf("Invalid WebSocket Port: %s", wsPortStr)
				return
			}

			log.Log.Info("VNC WebSocket. Set options in XML 'qemu:commandline'")
			if domainSpec.XmlNS == "" {
				domainSpec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
			}
			// not need graphic option
			domainSpec.Devices.Graphics = []domainSchema.Graphics{}

			if domainSpec.QEMUCmd == nil {
				domainSpec.QEMUCmd = &domainSchema.Commandline{}
			}
			if domainSpec.QEMUCmd.QEMUArg == nil {
				domainSpec.QEMUCmd.QEMUArg = make([]domainSchema.Arg, 0)
			}

			domainSpec.QEMUCmd.QEMUArg = append(domainSpec.QEMUCmd.QEMUArg, domainSchema.Arg{
				Value: "-vnc",
			})
			domainSpec.QEMUCmd.QEMUArg = append(domainSpec.QEMUCmd.QEMUArg, domainSchema.Arg{
				Value: fmt.Sprintf("%s:%d,websocket=%d", vncBindAddress, vncPort-5900, wsPort),
			})
		}
	}
}

func convertDiskOptions(annotations map[string]string, domainSpec *domainSchema.DomainSpec) {
	// change data disk driver type: qcow2
	if diskNames, found := annotations[diskNames]; found {
		driverType := annotations[diskDriver]
		if driverType == "" {
			driverType = defaultDiskDriver
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
						break
					}
				}
			}
		}
	}
}

func convertNicModel(domainSpec *domainSchema.DomainSpec) {
	if domainSpec.Devices.Interfaces == nil {
		return
	}

	for _, nicDevice := range domainSpec.Devices.Interfaces {
		if nicDevice.Model != nil && nicDevice.Model.Type != "vmxnet3" {
			nicDevice.Model.Type = "vmxnet3"
		}
	}
}
