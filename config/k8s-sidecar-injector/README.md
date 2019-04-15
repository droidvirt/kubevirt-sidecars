# Deploy injectro
## Source Code

* Code is Fork From [tumblr/injector](https://github.com/tumblr/k8s-sidecar-injector) to [lxs137/injector](https://github.com/lxs137/k8s-sidecar-injector)
* Determine mutation by pod's labels

## Create TLS first (`cd ./tls`)

### Generating TLS Certs

```bash
$ DEPLOYMENT=droidvirt CLUSTER=PRODUCTION ./new-cluster-injector-cert.rb
```

This will generate all the files necessary for a new CA, and the k8s-sidecar-injector cert!

### MutatingWebhookConfiguration

The MutatingWebhookConfiguration needs to know what `ca.crt` is used to sign the certs used to terminate TLS by the service. So, we need to extract the `caBundle` from your generated certificates in the previous step, and set it in MutatingWebhookConfiguration(../k8s-yaml/mutating-webhook-configuration.yaml)

Keeping with our `DEPLOYMENT=us-east-1` and `CLUSTER=PRODUCTION` example:

```bash
$ cd examples/tls
$ CABUNDLE_BASE64="$(cat $DEPLOYMENT/$CLUSTER/ca.crt |base64|tr -d '\n')"
```

Now, take this data and set it into the mutating webhook config as the `caBundle:` value.

#### Create k8s Secret

```bash
kubectl create secret generic k8s-sidecar-injector \
    --from-file=${DEPLOYMENT}/${CLUSTER}/sidecar-injector.crt \
    --from-file=${DEPLOYMENT}/${CLUSTER}/sidecar-injector.key \
    --dry-run -o yaml |
    kubectl -n ${NAMESPACE} apply -f -
```

## Create k8s resources (`cd ./k8s-yaml`)

```bash
kubectl apply -f clusterrole.yaml
kubectl apply -n ${NAMESPACE} -f clusterrolebinding.yaml
kubectl apply -n ${NAMESPACE} -f serviceaccount.yaml
kubectl apply -n ${NAMESPACE} -f service.yaml
kubectl apply -n ${NAMESPACE} -f deployment.yaml
cat mutating-webhook-configuration.yaml | sed "s/__NAMESPACE__/${NAMESPACE}/" | kubectl apply -f -
# Create sidecar config first
# Example in configmap-sidecar-test.yaml
```

# Prepare injection resources

* namespace: `kubectl label namespace <namespace> sidecar-injector=enabled`

* pod: 
```yaml
...
kind: Pod
metadata:
  labels:
    injector.droidvirt.io/request: <sidecar_name>
... 
```
