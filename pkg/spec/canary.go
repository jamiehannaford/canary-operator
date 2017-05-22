package spec

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/api/v1"
)

type Canary struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata,omitempty"`

	DeploymentName     string
	CanaryImage        string
	RolloutTimespan    int64
	IncreaseRate       string
	InitialCanaryCount int64
	DeleteDeployment   bool
	MonitorProbe       *v1.Probe
}
