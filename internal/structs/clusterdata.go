package structs

import "sort"

type ClusterData struct {
	Nodes []*NodeData
}

// GetNode returns a node with the given node name or machine name
func (c *ClusterData) GetNode(name string) *NodeData {
	// match on nodename or machinename
	for _, n := range c.Nodes {
		if n.NodeName == name || n.MachineName == name {
			return n
		}
	}
	return nil
}

// SortByRole sorts the cluster's nodes by their leading role
func (c *ClusterData) SortByRole() {
	sort.Slice(c.Nodes, func(i, j int) bool {
		if len(c.Nodes[i].Roles) == 0 {
			return false
		}
		if len(c.Nodes[j].Roles) == 0 {
			return true
		}
		return roleSortOrder(c.Nodes[i].Roles[0]) < roleSortOrder(c.Nodes[j].Roles[0])
	})
}

// roleSortOrder decides the order for sorting roles
func roleSortOrder(r string) int {
	switch r {
	case "master":
		return 0
	case "infra":
		return 1
	case "worker":
		return 2
	default:
		return 3
	}
}
