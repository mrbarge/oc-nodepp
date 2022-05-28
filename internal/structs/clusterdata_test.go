package structs

import "testing"

type sortByRoleTest struct {
	arg               ClusterData
	expectedRoleOrder []string
}

var sortByRoleTests = []sortByRoleTest{
	{
		arg: ClusterData{
			Nodes: []*NodeData{
				&NodeData{Roles: []string{"worker"}},
				&NodeData{Roles: []string{"master"}},
				&NodeData{Roles: []string{"infra"}},
			},
		},
		expectedRoleOrder: []string{
			"master", "infra", "worker",
		},
	},
	{
		arg: ClusterData{
			Nodes: []*NodeData{
				&NodeData{Roles: []string{"infra"}},
				&NodeData{Roles: []string{"master"}},
				&NodeData{Roles: []string{"nothing"}},
				&NodeData{Roles: []string{"worker"}},
			},
		},
		expectedRoleOrder: []string{
			"master", "infra", "worker", "nothing",
		},
	},
	{
		arg: ClusterData{
			Nodes: []*NodeData{
				&NodeData{Roles: []string{"infra"}},
				&NodeData{Roles: []string{"master"}},
				&NodeData{Roles: []string{"infra"}},
				&NodeData{Roles: []string{"master"}},
			},
		},
		expectedRoleOrder: []string{
			"master", "master", "infra", "infra",
		},
	},
	{
		arg: ClusterData{
			Nodes: []*NodeData{
				&NodeData{Roles: []string{"infra"}},
				&NodeData{Roles: []string{}},
				&NodeData{Roles: []string{"master"}},
			},
		},
		expectedRoleOrder: []string{
			"master", "infra", "",
		},
	},
}

func TestSortByRole(t *testing.T) {
	for _, test := range sortByRoleTests {
		test.arg.SortByRole()
		for i, r := range test.arg.Nodes {
			if len(r.Roles) > 0 {
				if test.expectedRoleOrder[i] != r.Roles[0] {
					t.Errorf("Role order incorrect")
				}
			} else if test.expectedRoleOrder[i] != "" {
				t.Errorf("Role order incorrect")
			}
		}
	}
}
