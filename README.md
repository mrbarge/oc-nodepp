# oc-nodepp (nodes++)

A CLI plugin for OpenShift v4 which provides a more detailed view of the state of 
the cluster's nodes and machines.

This plugin originated from the perspective of an OpenShift administrator in order to 
provide an easier at-a-glance view of the cluster's state, where frequently the
following commands are employed:

- `oc get nodes`
- `oc get machines -n openshift-machine-api`
- `oc adm top nodes`

This plugin provides a view that combinations information from all three sources:
- Nodes, and their CPU and memory resource usage.
- Machines associated with nodes, and their provisioning status.
- Highlights for:
  - Machines that do not have associated nodes.
  - Nodes that are NotReady, cordoned, or updating
  - CPU and memory resource usage that exceeds 85%  
 
## Usage

`oc-nodepp` will act as an OpenShift CLI plugin if available in the user's `$PATH`

```bash
# Search across all nodes in the cluster 
oc nodepp

# View a specific node 'node1'
oc nodepp node1

# Don't query for node metrics 
oc nodepp -u=false

# Don't show the symbol key output
oc nodepp -k=false
```
