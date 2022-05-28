package outputter

import (
	"fmt"
	"io"
	"nodepp/internal/structs"

	"github.com/jedib0t/go-pretty/v6/table"

	"nodepp/internal/consts"
)

type Outputter struct {
	ShowUsage   bool
	NodeMetrics *structs.ClusterData
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

func (o *Outputter) Print() {
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	rowConfigAutoMerge := table.RowConfig{AutoMerge: false}
	t.AppendHeader(o.makeHeaderRow(), rowConfigAutoMerge)
	t.AppendFooter(table.Row{
		"",
	})
	o.NodeMetrics.SortByRole()
	for _, node := range o.NodeMetrics.Nodes {
		rows := o.makeRows(node)
		for _, r := range rows {
			t.AppendRow(r, rowConfigAutoMerge)
		}
	}
	fmt.Println(t.Render())
}

func (o *Outputter) PrintRow(w io.Writer) {
}

func (o *Outputter) makeHeaderRow() table.Row {
	r := table.Row{
		tableHeader.ready,
		tableHeader.nodeName,
		tableHeader.machineName,
		tableHeader.internalIP,
		tableHeader.nodeRole,
		tableHeader.status,
	}
	if o.ShowUsage {
		r = append(r, tableHeader.cpu, tableHeader.memory)
	}
	return r
}

func (o *Outputter) makeRows(n *structs.NodeData) []table.Row {

	numRows := n.NumRows()
	fields := make([]table.Row, numRows)

	// First row
	var row table.Row

	// Ready
	if !n.Ready {
		row = append(row, fmt.Sprintf("%c", consts.EMOJI_SIREN))
	} else {
		row = append(row, "")
	}

	// Node name
	if n.NodeName == "" {
		row = append(row, fmt.Sprintf("%c", consts.EMOJI_QUESTION))
	} else {
		row = append(row, n.NodeName)
	}

	// Machine name
	row = append(row, n.MachineName)

	// IP
	row = append(row, n.InternalIP)

	// Role
	if len(n.Roles) > 0 {
		row = append(row, makeRoleValue(n.Roles))
	} else {
		row = append(row, "")
	}

	// Status
	var status string
	if n.Updating {
		status += fmt.Sprintf("%c", consts.EMOJI_WRENCH)
	}
	if n.Cordoned {
		status += fmt.Sprintf("%c", consts.EMOJI_ROADBLOCK)
	}
	switch n.MachinePhase {
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
	if o.ShowUsage {
		// Show utilization and allocatable in first row
		if n.Cpu != nil {
			utilFraction := float64(n.Cpu.Utilization.MilliValue()) / float64(n.Cpu.Allocatable.MilliValue()) * 100
			cpuval := fmt.Sprintf("%vm (%d%%)", n.Cpu.Utilization.MilliValue(), int64(utilFraction))
			if utilFraction > 90 {
				cpuval += string(consts.EMOJI_FIRE)
			}
			row = append(row, cpuval)
		} else {
			row = append(row, "")
		}
		if n.Memory != nil {
			utilFraction := float64(n.Memory.Utilization.MilliValue()) / float64(n.Memory.Allocatable.MilliValue()) * 100
			memval := fmt.Sprintf("%vMi (%d%%)", n.Memory.Utilization.Value()/(1024*1024), int64(utilFraction))
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

func (o *Outputter) PrintKeys() {
	fmt.Printf("%c  Master Node\t%c  Infra Node\t%c  Worker Node\t%c  Missing Node\n",
		consts.EMOJI_BUILDING, consts.EMOJI_BRICK, consts.EMOJI_WORKER, consts.EMOJI_QUESTION)
	fmt.Printf("%c  Not Ready\t%c  Cordoned\t%c  Updating\t%c  Failed\t%c  Deleting\t%c  Provisioning\t%c  Resource is hot\n\n",
		consts.EMOJI_SIREN, consts.EMOJI_ROADBLOCK, consts.EMOJI_WRENCH,
		consts.EMOJI_CROSS, consts.EMOJI_WASTE, consts.EMOJI_UPARROW, consts.EMOJI_FIRE)
}
