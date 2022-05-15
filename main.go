package main

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"nodepp/cmd"
	"os"
)

func main() {

	podInspectCmd := cmd.NewNodePPCommand(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := podInspectCmd.Execute(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
