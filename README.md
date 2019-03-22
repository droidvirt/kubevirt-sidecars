## How to build
### Prepare
* `git clone https://github.com/kubevirt/kubevirt.git`
* Make sure \<kubevirt-dir\> under `$GOPATH/src/kubevirt.io/`
* `git clone https://github.com/lxs137/droidvirt-sidecar.git <kubevirt-dir>/cmd/droidvirt-sidecar`
### Compile
* `docker run -it -v $GOPATH/src/kubevirt.io:/go/src/kubevirt.io golang:1.11 bash`
* `go build <kubevirt-dir>/cmd/droidvirt-hook-sidecar/<target-sidecar>`
* target binary will under `<kubevirt-dir>`

#### OR
* `cd <kubevirt-dir>`
* `hack/dockerized "./hack/check.sh && KUBEVIRT_VERSION=$VERSION ./hack/build-go.sh install cmd/droidvirt-hook-sidecar/<target-sidecar>"`
* `hack/build-copy-artifacts.sh cmd/droidvirt-hook-sidecar/<target-sidecar>`
* target binary will under `<kubevirt-dir>/_out/cmd/droidvirt-hook-sidecar/<target-sidecar>/`
