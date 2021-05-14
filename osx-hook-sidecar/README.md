## How to port MacOS to kubevirt
It works on Mojava and Clover bootloader
### Prepare the bootloader and system disks
* Follow the guide at https://github.com/kholia/OSX-KVM/blob/9c8178c3325a0c8abc20167bcb73feb9de0847a9/README.md
* We need the QEMU img at https://github.com/kholia/OSX-KVM/blob/9c8178c3325a0c8abc20167bcb73feb9de0847a9/macOS-libvirt-NG.xml#L90-L103

### Use custom bootloader config in Libvirt
* Libvirt XML looks like:
```xml
<os>
    <type arch='x86_64' machine='pc-q35-5.1'>hvm</type>
    <loader readonly='yes' secure='no' type='pflash'>/usr/share/OVMF/OVMF_CODE.fd</loader>
    <nvram>/usr/share/OVMF/OVMF_VARS.fd</nvram>
    <smbios mode='sysinfo'/>
</os>
```
* OVMF files:
  * https://github.com/kholia/OSX-KVM/blob/master/OVMF_VARS-1024x768.fd
  * https://github.com/kholia/OSX-KVM/blob/master/OVMF_CODE.fd
* So we need to put OVMF files into compute container of virt-launcher pod, and modify the Libvirt XML kubevirt generated
* Inject OVMF files by MutatingWebhook: https://github.com/lxs137/k8s-sidecar-injector
  * Injector configmap looks like, the `osx-boot-data` PVC contains the OVMF files:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: osx-worker-sidecar
  labels:
    app: k8s-sidecar-injector
data:
  osx-worker-sidecar: |-
    name: osx-worker-sidecar
    volumes:
    - name: ovmf-data
      emptyDir: {}
    - name: ovmf-src-data
      persistentVolumeClaim:
        claimName: osx-boot-data
        readOnly: true
    initContainers:
    - name: ovmf-importer
      image: busybox:latest
      command:
      - cp
      - -r
      - /root/src/
      - /root/boot/ovmf/
      volumeMounts:
      - name: ovmf-data
        mountPath: /root/boot
      - name: ovmf-src-data
        mountPath: /root/src
    volumeMountsInjection:
      containerSelector:
      - compute
      volumeMounts:
      - name: ovmf-data
        mountPath: /usr/share/OVMF
        subPath: ovmf
```
* Modify Libvirt XML by kubevirt hook sidecar: https://github.com/droidvirt/kubevirt-sidecars/blob/b41653ab1a7e16529d51d6385a9ccb55e64198c6/osx-hook-sidecar/converter.go#L17-L31

### Convert NIC model, input devices, etc.
* Finally, my VirtualMachine CR looks like, `osx-clover-autoboot` and `osx-disk-1` PVC contains the QEMU img we got in the first step:
```yaml
apiVersion: kubevirt.io/v1alpha3
kind: VirtualMachine
metadata:
  name: osx-1
spec:
  runStrategy: Always
  template:
    metadata:
      annotations:
        hooks.kubevirt.io/hookSidecars: '[{"image":"registry.cn-shanghai.aliyuncs.com/droidvirt/hook-sidecar:osx-202010062044"}]'
        converter.droidvirt.io/type: 'board,vnc,boot-loader,input-device,nic-model'
        vnc.droidvirt.io/port: '5900'
        loader.osx-kvm.io/path: '/usr/share/OVMF/OVMF_CODE.fd' 
        nvram.osx-kvm.io/path: '/usr/share/OVMF/OVMF_VARS.fd'
      labels:
        vmName: osx-1
        injector.droidvirt.io/request: 'osx-worker-sidecar'
      name: osx-1
    spec:
      hostname: osx-1
      domain:
        cpu:
          model: Penryn
          cores: 4
        devices:
          disks:
          - name: clover
            bootOrder: 1
            disk:
              bus: sata
          - name: data-disk
            bootOrder: 2
            disk:
              bus: sata
          interfaces:
          - name: default
            model: e1000
            masquerade: {}
            ports:
            - name: sshd
              port: 22
        resources:
          requests:
            memory: "4Gi"
      terminationGracePeriodSeconds: 0
      volumes:
      - name: clover
        ephemeral:
          persistentVolumeClaim:
            claimName: osx-clover-autoboot
      - name: data-disk
        persistentVolumeClaim:
          claimName: osx-disk-1
      networks:
      - name: default
        pod:
          vmNetworkCIDR: "192.168.100.0/24"
```


## How to build
### Prepare
* `git clone https://github.com/kubevirt/kubevirt.git`
* Make sure \<kubevirt-dir\> under `$GOPATH/src/kubevirt.io/`
* `git clone https://github.com/droidvirt/kubevirt-sidecars.git <kubevirt-dir>/cmd/droidvirt-sidecar`
### Compile
* `docker run -it -v $GOPATH/src/kubevirt.io:/go/src/kubevirt.io golang:1.11 bash`
* `go build <kubevirt-dir>/cmd/droidvirt-hook-sidecar/<target-sidecar>`
* target binary will under `<kubevirt-dir>`

#### OR
* `cd <kubevirt-dir>`
* `hack/dockerized "./hack/check.sh && KUBEVIRT_VERSION=$VERSION ./hack/build-go.sh install cmd/droidvirt-hook-sidecar/<target-sidecar>"`
* `hack/build-copy-artifacts.sh cmd/droidvirt-hook-sidecar/<target-sidecar>`
* target binary will under `<kubevirt-dir>/_out/cmd/droidvirt-hook-sidecar/<target-sidecar>/`
