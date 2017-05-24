package spec

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/api/v1"
)

const (
	defaultVersion = "0.0.1"

	TPRKind        = "canary"
	TPRKindPlural  = "canaries"
	TPRGroup       = "canaries.jh"
	TPRVersion     = "v1beta1"
	TPRDescription = "Managed canaries"
)

func TPRName() string {
	return fmt.Sprintf("%s.%s", TPRKind, TPRGroup)
}

type Canary struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CanarySpec `json:"spec"`
}

type CanaryList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata,omitempty"`

	Items []Canary `json:"items"`
}

type CanarySpec struct {
	DeploymentName     string
	CanaryImage        string
	RolloutTimespan    int64
	IncreaseRate       string
	InitialCanaryCount int64
	DeleteDeployment   bool
	MonitorProbe       *v1.Probe
}
