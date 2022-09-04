package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"io"
	"nodepp/internal/structs"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	mcs "k8s.io/metrics/pkg/client/clientset/versioned"

	oapi "github.com/openshift/api/config/v1"
	"github.com/openshift/api/machine/v1beta1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machinev1 "github.com/openshift/client-go/machine/clientset/versioned/typed/machine/v1beta1"

	"nodepp/internal/config"
	"nodepp/internal/consts"
	"nodepp/internal/outputter"
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
	showUsage     bool
	showKeys      bool
	showVersion   bool
	showOperators bool
)

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
	ccmd.PersistentFlags().BoolVarP(&showVersion, config.ShowVersion, "v", true, "Show cluster version data")
	ccmd.PersistentFlags().BoolVarP(&showOperators, config.ShowOperators, "o", true, "Show cluster operator data")
	ccmd.PersistentFlags().BoolVarP(&showKeys, config.ShowKeys, "k", false, "Show symbol keys")

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
	cd := new(structs.ClusterData)
	cd.Nodes = make([]*structs.NodeData, 0)

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
		nodeData, err := structs.NewFromNode(&node)
		if err != nil {
			return err
		}
		cd.Nodes = append(cd.Nodes, nodeData)
	}

	// Process machines
	machines, err := dp.getAllMachines()
	if err != nil {
		return err
	}
	for _, machine := range machines.Items {
		nodeData, err := structs.NewFromMachine(&machine)
		if err != nil {
			return err
		}
		if nodeData.NodeName == "" {
			cd.Nodes = append(cd.Nodes, nodeData)
		} else {
			// just merge in machine info, if we pulled node info originally
			node := cd.GetNode(nodeData.NodeName)
			if node != nil {
				node.MachinePhase = nodeData.MachinePhase
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
			node := cd.GetNode(nm.Name)
			if node == nil {
				continue
			}
			if node.Cpu != nil {
				node.Cpu.Utilization = nm.Usage.Cpu().DeepCopy()
			}
			if node.Memory != nil {
				node.Memory.Utilization = nm.Usage.Memory().DeepCopy()
			}
		}
	}

	// Process cluster version
	if showVersion {
		cv, err := dp.getClusterVersion()
		if err != nil {
			return err
		}
		cd.Version = cv
	}

	// Process cluster operators
	if showOperators {
		co, err := dp.getClusterOperators()
		if err != nil {
			return err
		}
		cd.ClusterOperators = co
	}

	// Render output
	o := outputter.Outputter{
		ShowUsage:   showUsage,
		NodeMetrics: cd,
	}
	o.Print()
	if showKeys {
		o.PrintKeys()
	}

	return nil
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

func (dp *nodePPCommand) getClusterVersion() (*oapi.ClusterVersion, error) {
	cvClient, err := configclient.NewForConfig(dp.restConfig)
	cv, err := cvClient.ConfigV1().ClusterVersions().Get(context.Background(), "version", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cv, nil
}

func (dp *nodePPCommand) getClusterOperators() (*oapi.ClusterOperatorList, error) {
	cvClient, err := configclient.NewForConfig(dp.restConfig)
	cos, err := cvClient.ConfigV1().ClusterOperators().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return cos, nil
}
