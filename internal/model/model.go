package model

type Application struct {
	WorkloadSets `yaml:",inline"`
	Resources    Resources              `yaml:"resources" json:"resources"`
	Overrides    map[string]Environment `yaml:"overrides,omitempty" json:"overrides,omitempty"`
}

type WorkloadSets struct {
	Apps     Workloads `yaml:"apps,omitempty" json:"apps,omitempty"`
	Stateful Workloads `yaml:"stateful,omitempty" json:"stateful,omitempty"`
	Jobs     Workloads `yaml:"jobs,omitempty" json:"jobs,omitempty"`
}

func (w WorkloadSets) Groups() map[string]Workloads {
	return map[string]Workloads{
		"apps":     w.Apps,
		"stateful": w.Stateful,
		"jobs":     w.Jobs,
	}
}

type Workload struct {
	Image    string            `yaml:"image" json:"image"`
	Replicas int               `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	CPU      string            `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory   string            `yaml:"memory,omitempty" json:"memory,omitempty"`
	Ports    []int             `yaml:"ports,omitempty" json:"ports,omitempty"`
	Env      map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Command  []string          `yaml:"command,omitempty" json:"command,omitempty"`
	Args     []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Mounts   map[string]string `yaml:"mounts,omitempty" json:"mounts,omitempty"`
	Routes   []Route           `yaml:"routes,omitempty" json:"routes,omitempty"`
}

type Workloads map[string]Workload

type Resources struct {
	Volumes map[string]Volume `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Configs map[string]Config `yaml:"configs,omitempty" json:"configs,omitempty"`
	Secrets map[string]Secret `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

type Volume struct {
	Size string `yaml:"size" json:"size"`
}

type Config struct {
	Values map[string]any `yaml:"values,omitempty" json:"values,omitempty"`
	Files  []string       `yaml:"files,omitempty" json:"files,omitempty"`
}

type Secret struct {
	Env   map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Files []string          `yaml:"files,omitempty" json:"files,omitempty"`
}

type Route struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

type Environment struct {
	WorkloadSets `yaml:",inline"`
	Resources    *Resources `yaml:"resources,omitempty" json:"resources,omitempty"`
}

type Deployment struct {
	ID          string      `yaml:"id" json:"id"`
	Application string      `yaml:"application" json:"application"`
	Environment string      `yaml:"environment" json:"environment"`
	Spec        Application `yaml:"spec" json:"spec"`
}
