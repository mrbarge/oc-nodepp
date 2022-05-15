package consts

const (
	MachineNamespace                = "openshift-machine-api"
	Annotation_Machine              = "machine.openshift.io/machine"
	Annotation_MachineCurrentConfig = "machineconfiguration.openshift.io/currentConfig"
	Annotation_MachineDesiredConfig = "machineconfiguration.openshift.io/desiredConfig"

	Label_MasterNodeRole = "node-role.kubernetes.io/master"
	Label_WorkerNodeRole = "node-role.kubernetes.io/worker"
	Label_InfraNodeRole  = "node-role.kubernetes.io/infra"
)
