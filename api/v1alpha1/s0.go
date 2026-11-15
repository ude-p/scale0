// Package v1alpha1 contains the apps.s0 API types.
// +kubebuilder:object:generate=true
// +groupName=apps.s0
package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// S0 is an application in one environment, stored in the namespace named after
// that environment.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=s0
type S0 struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec S0Spec `json:"spec"`
}

type S0Spec struct {
	// +kubebuilder:validation:MinProperties=1
	Workload map[string]Workload `json:"workload,omitempty"`

	Volume map[string]string `json:"volume,omitempty"`

	Config map[string]Config `json:"config,omitempty"`

	Secret map[string]Secret `json:"secret,omitempty"`

	Overrides map[string]Environment `json:"overrides,omitempty"`
}

type Workload struct {
	// +kubebuilder:validation:Enum=deployment;stateful;job
	// +kubebuilder:default=deployment
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// +kubebuilder:validation:Minimum=0
	Replicas int `json:"replicas,omitempty"`

	// +kubebuilder:validation:Pattern=^([0-9]+(\.[0-9]+)?|[0-9]+m)$
	CPU string `json:"cpu,omitempty"`

	// +kubebuilder:validation:Pattern=^[0-9]+(Mi|Gi|Ti|M|G|T)$
	Memory string `json:"memory,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:Minimum=1
	// +kubebuilder:validation:items:Maximum=65535
	Ports []int `json:"ports,omitempty"`

	Env map[string]string `json:"env,omitempty"`

	Command []string `json:"command,omitempty"`

	Args []string `json:"args,omitempty"`

	// +kubebuilder:validation:MinProperties=1
	Mounts map[string]string `json:"mounts,omitempty"`

	Routes []Route `json:"routes,omitempty"`
}

type Environment struct {
	Workload map[string]Workload `json:"workload,omitempty"`

	Volume map[string]string `json:"volume,omitempty"`

	Config map[string]Config `json:"config,omitempty"`

	Secret map[string]Secret `json:"secret,omitempty"`
}

// +kubebuilder:validation:MinProperties=1
type Config map[string]string

// +kubebuilder:validation:MinProperties=1
type Secret map[string]string

type Route struct {
	// +kubebuilder:validation:MinLength=1
	Host string `json:"host"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int `json:"port"`
}
