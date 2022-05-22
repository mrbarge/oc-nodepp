package cmd

import (
	"context"
	"io"
	"nodepp/internal/config"
	"strings"

	"github.com/spf13/cobra"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	mcs "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/openshift/api/machine/v1beta1"
	machinev1 "github.com/openshift/client-go/machine/clientset/versioned/typed/machine/v1beta1"

	"nodepp/internal/consts"
)

const (
	command = `nodepp`

	longDescription = `
This command produces a summary of all machines on the cluster, their
associated node if it exists, and a summary of information about these
resources to help attain a quick at-a-glance view of the cluster's state.
`
	shortDescription = `
Nodes, plus a little more more.
`
)

var (
	showUsage bool
	showKeys  bool
)

type clusterData struct {
	nodes map[string]*nodeData
}

type resourceMetric struct {
	allocatable resource.Quantity
	utilization resource.Quantity
}

type nodeData struct {
	nodeName     string
	machineName  string
	machinePhase string
	internalIP   string
	roles        []string
	updating     bool
	missing      bool
	cordoned     bool
	ready        bool
	cpu          *resourceMetric
	memory       *resourceMetric
}

type nodePPCommand struct {
	out        io.Writer
	f          cmdutil.Factory
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
}

func NewNodePPCommand(streams genericclioptions.IOStreams) *cobra.Command {
	dpcmd := &nodePPCommand{
		out: streams.Out,
	}

	ccmd := &cobra.Command{
		Use:          "kubectl nodepp <node>",
		Short:        shortDescription,
		Long:         longDescription,
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dpcmd.run(args)
		},
	}

	ccmd.PersistentFlags().BoolVarP(&showUsage, config.ShowUsage, "u", true, "Show node resource usage")
	ccmd.PersistentFlags().BoolVarP(&showKeys, config.ShowKeys, "k", true, "Show symbol keys")

	fsets := ccmd.PersistentFlags()
	cfgFlags := genericclioptions.NewConfigFlags(true)
	cfgFlags.AddFlags(fsets)
	matchVersionFlags := cmdutil.NewMatchVersionFlags(cfgFlags)
	matchVersionFlags.AddFlags(fsets)
	dpcmd.f = cmdutil.NewFactory(matchVersionFlags)

	return ccmd
}

func (dp *nodePPCommand) run(args []string) error {
	// Setup clients
	clientset, err := dp.f.KubernetesClientSet()
	if err != nil {
		return err
	}
	dp.clientset = clientset
	rc, err := dp.f.ToRESTConfig()
	if err != nil {
		return err
	}
	dp.restConfig = rc

	// Initialise data store
	cd := new(clusterData)
	cd.nodes = make(map[string]*nodeData, 0)

	// Pull node info
	nodesToProcess := make([]v1.Node, 0)
	if len(args) == 1 {
		node, err := dp.clientset.CoreV1().Nodes().Get(context.Background(), args[0], metav1.GetOptions{})
		if err != nil {
			return err
		}
		nodesToProcess = append(nodesToProcess, *node)
	} else {
		nodes, err := dp.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, node := range nodes.Items {
			nodesToProcess = append(nodesToProcess, node)
		}
	}

	// Process each node
	for _, node := range nodesToProcess {
		nodeData, err := dp.processNode(&node)
		if err != nil {
			return err
		}
		cd.nodes[nodeData.nodeName] = nodeData
	}

	// Process machines
	machines, err := dp.getAllMachines()
	if err != nil {
		return err
	}
	for _, machine := range machines.Items {
		nodeData, err := dp.processMachine(&machine)
		if err != nil {
			return err
		}
		if nodeData.nodeName == "" {
			cd.nodes[nodeData.machineName] = nodeData
		} else {
			// just merge in machine info, if we pulled node info originally
			if _, ok := cd.nodes[nodeData.nodeName]; ok {
				cd.nodes[nodeData.nodeName].machinePhase = nodeData.machinePhase
			}
		}
	}

	// Process node metrics
	if showUsage {
		nodeMetrics, err := dp.getNodeMetrics()
		if err != nil {
			return err
		}
		for _, nm := range nodeMetrics.Items {
			// ignore nodes we never pulled info for originally
			if _, ok := cd.nodes[nm.Name]; !ok {
				continue
			}
			if cd.nodes[nm.Name].cpu != nil {
				cd.nodes[nm.Name].cpu.utilization = nm.Usage.Cpu().DeepCopy()
			}
			if cd.nodes[nm.Name].memory != nil {
				cd.nodes[nm.Name].memory.utilization = nm.Usage.Memory().DeepCopy()
			}
		}
	}

	// Render output
	o := outputter{
		showUsage: showUsage,
		nm:        cd,
	}
	o.Print()
	if showKeys {
		o.PrintKeys()
	}

	return nil
}

func (dp *nodePPCommand) processNode(node *v1.Node) (*nodeData, error) {

	nodeData := new(nodeData)
	nodeData.nodeName = node.Name

	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			nodeData.internalIP = addr.Address
			break
		}
	}

	annotations := node.GetAnnotations()
	if machine, ok := annotations[consts.Annotation_Machine]; ok {
		machineName := strings.SplitAfter(machine, "/")[1]
		nodeData.machineName = machineName
		//dp.getMachine(machineName)
	}
	nodeData.cordoned = node.Spec.Unschedulable
	if currentConfig, ok := annotations[consts.Annotation_MachineCurrentConfig]; ok {
		if desiredConfig, ok := annotations[consts.Annotation_MachineDesiredConfig]; ok {
			if currentConfig != desiredConfig {
				nodeData.updating = true
			}
		}
	}
	nodeData.roles = make([]string, 0)
	labels := node.GetLabels()
	for _, l := range []string{consts.Label_MasterNodeRole, consts.Label_InfraNodeRole, consts.Label_WorkerNodeRole} {
		if _, ok := labels[l]; ok {
			nodeData.roles = append(nodeData.roles, strings.SplitAfter(l, "/")[1])
		}
	}
	for _, c := range node.Status.Conditions {
		if c.Type == v1.NodeReady && c.Status == v1.ConditionTrue {
			nodeData.ready = true
		}
	}
	nodeData.cpu = &resourceMetric{
		allocatable: node.Status.Allocatable.Cpu().DeepCopy(),
	}
	nodeData.memory = &resourceMetric{
		allocatable: node.Status.Allocatable.Memory().DeepCopy(),
	}

	return nodeData, nil
}

func (dp *nodePPCommand) processMachine(machine *v1beta1.Machine) (*nodeData, error) {

	nodeData := new(nodeData)

	// set the machine name
	nodeData.machineName = machine.Name

	// get the node name
	if machine.Status.NodeRef != nil && machine.Status.NodeRef.Kind == "Node" {
		nodeData.nodeName = machine.Status.NodeRef.Name
	}

	// set the machine phase
	nodeData.machinePhase = *machine.Status.Phase

	// that's all we care about for now
	return nodeData, nil
}

func (dp *nodePPCommand) getMachine(name string) error {
	machineClient, err := machinev1.NewForConfig(dp.restConfig)
	_, err = machineClient.Machines(consts.MachineNamespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (dp *nodePPCommand) getAllMachines() (*v1beta1.MachineList, error) {
	machineClient, err := machinev1.NewForConfig(dp.restConfig)
	machines, err := machineClient.Machines(consts.MachineNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return machines, nil
}

func (dp *nodePPCommand) getNodeMetrics() (*metricsv1beta1.NodeMetricsList, error) {
	metricsClient, err := mcs.NewForConfig(dp.restConfig)
	nmList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nmList, nil
}
