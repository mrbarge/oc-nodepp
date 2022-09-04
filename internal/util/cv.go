package util

import (
	"fmt"

	v1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCurrentVersion strings a version as a string and error
func GetCurrentVersion(clusterVersion *v1.ClusterVersion) (string, error) {
	var gotVersion string
	var latestCompletionTime *metav1.Time = nil
	for _, history := range clusterVersion.Status.History {
		if history.State == v1.CompletedUpdate {
			if latestCompletionTime == nil || history.CompletionTime.After(latestCompletionTime.Time) {
				gotVersion = history.Version
				latestCompletionTime = history.CompletionTime
			}
		}
	}

	if len(gotVersion) == 0 {
		return gotVersion, fmt.Errorf("failed to get current version")
	}

	return gotVersion, nil
}
