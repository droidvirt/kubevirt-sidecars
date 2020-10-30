package main

const (
	converterType    = "converter.droidvirt.io/type"
	vncPort          = "vnc.droidvirt.io/port"
	vncWebsocketPort = "websocket.vnc.droidvirt.io/port"
	diskNames        = "disk.droidvirt.io/names" // split name by comma
	diskDriver       = "disk.droidvirt.io/driverType"
	loaderPath       = "loader.osx-kvm.io/path"
	nvramPath        = "nvram.osx-kvm.io/path"
)

type ConverterType string

const (
	BoardConverter       ConverterType = "board"
	VncConverter         ConverterType = "vnc"
	DiskDriverConverter  ConverterType = "disk-driver"
	BootLoaderConverter  ConverterType = "boot-loader"
	NICModelConverter    ConverterType = "nic-model"
	InputDeviceConverter ConverterType = "input-device"
)
