package main

import (
	"fmt"
	"kubevirt.io/client-go/log"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"strconv"
	"strings"
)

const (
	vncBindAddress    = "0.0.0.0"
	defaultDiskDriver = "qcow2"
)

func convertVNCOptions(annotations map[string]string, domainSpec *domainSchema.DomainSpec) {
	if vncPortStr, found := annotations[vncPortAnnotation]; found {
		vncPort, err := strconv.ParseInt(vncPortStr, 10, 32)
		if err != nil || vncPort < 5900 {
			log.Log.Errorf("Invalid VNC Port: %s", vncPortStr)
			return
		}

		if wsPortStr, found := annotations[vncWebsocketPortAnnotation]; !found {
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
	if diskNames, found := annotations[diskNamesAnnotation]; found {
		driverType := annotations[diskDriverAnnotation]
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
						log.Log.Infof("After Change: %+v", domainSpec.Devices.Disks[idx].Driver)
						break
					}
				}
			}
		}
	}
}

func addQEMUArgs(annotations map[string]string, domainSpec *domainSchema.DomainSpec) {
	if qemuArgs, found := annotations[qemuArgsAnnotation]; found {
		args := []domainSchema.Arg{}
		for _, arg := range strings.Split(qemuArgs, ";") {
			args = append(args, domainSchema.Arg{
					Value: arg,
			})
		}
		if domainSpec.QEMUCmd == nil {
			domainSpec.QEMUCmd = &domainSchema.Commandline{}
		}
		if domainSpec.QEMUCmd.QEMUArg == nil {
			domainSpec.QEMUCmd.QEMUArg = args
		} else {
			domainSpec.QEMUCmd.QEMUArg = append(domainSpec.QEMUCmd.QEMUArg, args...)
		}
	}
}
