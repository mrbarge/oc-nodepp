package structs

import "k8s.io/apimachinery/pkg/api/resource"

type ResourceMetric struct {
	Allocatable resource.Quantity
	Utilization resource.Quantity
}
