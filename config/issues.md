### Ceph RBD device unmap
* `rbd: unmap failed: (16) Device or resource busy`
* solution: `rbd device unmap -o force <dev_path>`

### Pods not scheduled on master
* solution: `kubectl taint nodes <node> node-role.kubernetes.io/master-`