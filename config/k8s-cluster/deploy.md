### init cluster (master)

```bash
sudo kubeadm init --node-name node1 --kubernetes-version v1.13.3 --pod-network-cidr 10.244.0.0/16
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
# flannel pod
kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
```

### reset cluster

```bash
sudo kubeadm reset
# clean iptables rules and chains
sudo iptables -F && sudo iptables -t nat -F && sudo iptables -t mangle -F && sudo iptables -X
sudo iptables -t nat -L | grep KUBE- | awk -F ' ' '{print $2}' | xargs -i sudo iptables -t nat -X {}
```