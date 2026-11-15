package controller

import (
	"maps"

	"github.com/ude-p/scale0/api/v1alpha1"
)

func Resolve(app v1alpha1.S0) (v1alpha1.S0, error) {
	if env, ok := app.Spec.Overrides[app.Namespace]; ok {
		app.Spec = mergeSpec(app.Spec, env)
	}
	app.Spec.Overrides = nil

	if err := Validate(app); err != nil {
		return v1alpha1.S0{}, err
	}
	return app, nil
}

func mergeSpec(base v1alpha1.S0Spec, env v1alpha1.Environment) v1alpha1.S0Spec {
	base.Workload = mergeWorkloads(base.Workload, env.Workload)
	base.Volume = mergeStringMaps(base.Volume, env.Volume)
	base.Config = mergeConfigs(base.Config, env.Config)
	base.Secret = mergeSecrets(base.Secret, env.Secret)
	return base
}

func mergeWorkloads(base, override map[string]v1alpha1.Workload) map[string]v1alpha1.Workload {
	if override == nil {
		return base
	}
	merged := make(map[string]v1alpha1.Workload, len(base)+len(override))
	maps.Copy(merged, base)
	for name, w := range override {
		if existing, ok := merged[name]; ok {
			merged[name] = mergeWorkload(existing, w)
			continue
		}
		merged[name] = w
	}
	return merged
}

func mergeWorkload(base, override v1alpha1.Workload) v1alpha1.Workload {
	if override.Type != "" {
		base.Type = override.Type
	}
	if override.Image != "" {
		base.Image = override.Image
	}
	if override.Replicas != 0 {
		base.Replicas = override.Replicas
	}
	if override.CPU != "" {
		base.CPU = override.CPU
	}
	if override.Memory != "" {
		base.Memory = override.Memory
	}
	if override.Ports != nil {
		base.Ports = override.Ports
	}
	if override.Env != nil {
		base.Env = mergeStringMaps(base.Env, override.Env)
	}
	if override.Command != nil {
		base.Command = override.Command
	}
	if override.Args != nil {
		base.Args = override.Args
	}
	if override.Mounts != nil {
		base.Mounts = mergeStringMaps(base.Mounts, override.Mounts)
	}
	if override.Routes != nil {
		base.Routes = override.Routes
	}
	return base
}

func mergeConfigs(base, override map[string]v1alpha1.Config) map[string]v1alpha1.Config {
	if override == nil {
		return base
	}
	merged := make(map[string]v1alpha1.Config, len(base)+len(override))
	maps.Copy(merged, base)
	for name, config := range override {
		merged[name] = mergeStringMaps(base[name], config)
	}
	return merged
}

func mergeSecrets(base, override map[string]v1alpha1.Secret) map[string]v1alpha1.Secret {
	if override == nil {
		return base
	}
	merged := make(map[string]v1alpha1.Secret, len(base)+len(override))
	maps.Copy(merged, base)
	for name, secret := range override {
		merged[name] = mergeStringMaps(base[name], secret)
	}
	return merged
}

func mergeStringMaps(base, override map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(override))
	maps.Copy(merged, base)
	maps.Copy(merged, override)
	return merged
}
