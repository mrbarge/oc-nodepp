package cmd

import (
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"

	"nodepp/internal/consts"
)

type outputter struct {
	showUsage bool
	nm        *clusterData
}

type tableRow struct {
	ready       string
	nodeName    string
	machineName string
	internalIP  string
	nodeRole    string
	status      string
	cpu         string
	memory      string
}

var tableHeader = tableRow{
	ready:       " ",
	nodeName:    "NODE",
	machineName: "MACHINE",
	internalIP:  "INTERNAL IP",
	nodeRole:    "ROLE",
	status:      "STATUS",
	cpu:         "CPU",
	memory:      "MEMORY",
}

func (o *outputter) Print() {
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	rowConfigAutoMerge := table.RowConfig{AutoMerge: false}
	t.AppendHeader(o.makeHeaderRow(), rowConfigAutoMerge)
	t.AppendFooter(table.Row{
		"",
	})
	for _, node := range o.nm.nodes {
		rows := o.makeRows(node)
		for _, r := range rows {
			t.AppendRow(r, rowConfigAutoMerge)
		}
	}
	fmt.Println(t.Render())
}

func (o *outputter) PrintRow(w io.Writer) {
}

func (o *outputter) makeHeaderRow() table.Row {
	r := table.Row{
		tableHeader.ready,
		tableHeader.nodeName,
		tableHeader.machineName,
		tableHeader.internalIP,
		tableHeader.nodeRole,
		tableHeader.status,
	}
	if o.showUsage {
		r = append(r, tableHeader.cpu, tableHeader.memory)
	}
	return r
}

func (o *outputter) makeRows(n *nodeData) []table.Row {

	numRows := n.numRows()
	fields := make([]table.Row, numRows)

	// First row
	var row table.Row

	// Ready
	if !n.ready {
		row = append(row, fmt.Sprintf("%c", consts.EMOJI_SIREN))
	} else {
		row = append(row, "")
	}

	// Node name
	if n.nodeName == "" {
		row = append(row, fmt.Sprintf("%c", consts.EMOJI_QUESTION))
	} else {
		row = append(row, n.nodeName)
	}

	// Machine name
	row = append(row, n.machineName)

	// IP
	row = append(row, n.internalIP)

	// Role
	if len(n.roles) > 0 {
		row = append(row, makeRoleValue(n.roles))
	} else {
		row = append(row, "")
	}

	// Status
	var status string
	if n.updating {
		status += fmt.Sprintf("%c", consts.EMOJI_WRENCH)
	}
	if n.cordoned {
		status += fmt.Sprintf("%c", consts.EMOJI_ROADBLOCK)
	}
	switch n.machinePhase {
	case "Failed":
		status += fmt.Sprintf("%c", consts.EMOJI_CROSS)
	case "Deleting":
		status += fmt.Sprintf("%c", consts.EMOJI_WASTE)
	case "Provisioned":
		status += fmt.Sprintf("%c", consts.EMOJI_UPARROW)
	case "Provisioning":
		status += fmt.Sprintf("%c", consts.EMOJI_UPARROW)
	}
	row = append(row, status)

	// Usage
	if o.showUsage {
		// Show utilization and allocatable in first row
		if n.cpu != nil {
			utilFraction := float64(n.cpu.utilization.MilliValue()) / float64(n.cpu.allocatable.MilliValue()) * 100
			cpuval := fmt.Sprintf("%vm (%d%%)", n.cpu.utilization.MilliValue(), int64(utilFraction))
			if utilFraction > 90 {
				cpuval += string(consts.EMOJI_FIRE)
			}
			row = append(row, cpuval)
		} else {
			row = append(row, "")
		}
		if n.memory != nil {
			utilFraction := float64(n.memory.utilization.MilliValue()) / float64(n.memory.allocatable.MilliValue()) * 100
			memval := fmt.Sprintf("%vMi (%d%%)", n.memory.utilization.Value()/(1024*1024), int64(utilFraction))
			if utilFraction > 90 {
				memval += string(consts.EMOJI_FIRE)
			}
			row = append(row, memval)
		} else {
			row = append(row, "")
		}
	}
	fields = append(fields, row)

	return fields
}

func (n *nodeData) numRows() int {
	// we don't have a need to extend a single node to multiple rows yet
	maxRows := 1
	return maxRows
}

func makeRoleValue(roles []string) string {
	// handle no roles
	if len(roles) == 0 {
		return "-"
	}

	// favour the most appropriate role in order of master->infra->worker
	foundMaster := false
	foundInfra := false
	foundWorker := false
	for _, role := range roles {
		switch role {
		case "master":
			foundMaster = true
		case "infra":
			foundInfra = true
		case "worker":
			foundWorker = true
		}
	}

	if foundMaster {
		return fmt.Sprintf("%c  master", consts.EMOJI_BUILDING)
	}
	if foundInfra {
		return fmt.Sprintf("%c infra", consts.EMOJI_BRICK)
	}
	if foundWorker {
		return fmt.Sprintf("%c worker", consts.EMOJI_WORKER)
	}

	// just return the first role in the list
	return roles[0]
}

func (o *outputter) PrintKeys() {
	fmt.Printf("%c  Master Node\t%c  Infra Node\t%c  Worker Node\t%c  Missing Node\n",
		consts.EMOJI_BUILDING, consts.EMOJI_BRICK, consts.EMOJI_WORKER, consts.EMOJI_QUESTION)
	fmt.Printf("%c  Not Ready\t%c  Cordoned\t%c  Updating\t%c  Failed\t%c  Deleting\t%c  Provisioning\t%c  Resource is hot\n\n",
		consts.EMOJI_SIREN, consts.EMOJI_ROADBLOCK, consts.EMOJI_WRENCH,
		consts.EMOJI_CROSS, consts.EMOJI_WASTE, consts.EMOJI_UPARROW, consts.EMOJI_FIRE)
}
