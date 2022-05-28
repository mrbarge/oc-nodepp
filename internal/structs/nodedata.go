package structs

import (
	"github.com/openshift/api/machine/v1beta1"
	v1 "k8s.io/api/core/v1"
	"nodepp/internal/consts"
	"strings"
)

type NodeData struct {
	NodeName     string
	MachineName  string
	MachinePhase string
	InternalIP   string
	Roles        []string
	Updating     bool
	Missing      bool
	Cordoned     bool
	Ready        bool
	Cpu          *ResourceMetric
	Memory       *ResourceMetric
}

func (n *NodeData) NumRows() int {
	// we don't have a need to extend a single node to multiple rows yet
	maxRows := 1
	return maxRows
}

func NewFromNode(node *v1.Node) (*NodeData, error) {
	nodeData := new(NodeData)
	nodeData.NodeName = node.Name

	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			nodeData.InternalIP = addr.Address
			break
		}
	}

	annotations := node.GetAnnotations()
	if machine, ok := annotations[consts.Annotation_Machine]; ok {
		machineName := strings.SplitAfter(machine, "/")[1]
		nodeData.MachineName = machineName
		//dp.getMachine(machineName)
	}
	nodeData.Cordoned = node.Spec.Unschedulable
	if currentConfig, ok := annotations[consts.Annotation_MachineCurrentConfig]; ok {
		if desiredConfig, ok := annotations[consts.Annotation_MachineDesiredConfig]; ok {
			if currentConfig != desiredConfig {
				nodeData.Updating = true
			}
		}
	}
	nodeData.Roles = make([]string, 0)
	labels := node.GetLabels()
	for _, l := range []string{consts.Label_MasterNodeRole, consts.Label_InfraNodeRole, consts.Label_WorkerNodeRole} {
		if _, ok := labels[l]; ok {
			nodeData.Roles = append(nodeData.Roles, strings.SplitAfter(l, "/")[1])
		}
	}
	for _, c := range node.Status.Conditions {
		if c.Type == v1.NodeReady && c.Status == v1.ConditionTrue {
			nodeData.Ready = true
		}
	}
	nodeData.Cpu = &ResourceMetric{
		Allocatable: node.Status.Allocatable.Cpu().DeepCopy(),
	}
	nodeData.Memory = &ResourceMetric{
		Allocatable: node.Status.Allocatable.Memory().DeepCopy(),
	}

	return nodeData, nil
}

func NewFromMachine(machine *v1beta1.Machine) (*NodeData, error) {
	nodeData := new(NodeData)

	// set the machine name
	nodeData.MachineName = machine.Name

	// get the node name
	if machine.Status.NodeRef != nil && machine.Status.NodeRef.Kind == "Node" {
		nodeData.NodeName = machine.Status.NodeRef.Name
	}

	// set the machine phase
	if machine.Status.Phase != nil {
		nodeData.MachinePhase = *machine.Status.Phase
	}

	// that's all we care about for now
	return nodeData, nil
}
