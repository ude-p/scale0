package deployments

import (
	"maps"

	"github.com/ude-p/shipl/internal/model"
)

func (s *Service) Resolve(req CreateRequest) (model.Deployment, error) {
	spec := req.Spec
	if env, ok := spec.Overrides[req.Environment]; ok {
		spec = mergeApplication(spec, env)
	}
	spec.Overrides = nil

	if err := validateWorkloadGroups(spec.Groups(), true); err != nil {
		return model.Deployment{}, err
	}
	if err := validateResources(spec.Resources); err != nil {
		return model.Deployment{}, err
	}
	if err := validateReferences(spec); err != nil {
		return model.Deployment{}, err
	}

	return model.Deployment{
		Application: req.Application,
		Environment: req.Environment,
		Spec:        spec,
	}, nil
}

func mergeApplication(base model.Application, env model.Environment) model.Application {
	base.Apps = mergeWorkloads(base.Apps, env.Apps)
	base.Stateful = mergeWorkloads(base.Stateful, env.Stateful)
	base.Jobs = mergeWorkloads(base.Jobs, env.Jobs)
	if env.Resources != nil {
		base.Resources = mergeResources(base.Resources, *env.Resources)
	}
	return base
}

func mergeWorkloads(base, override model.Workloads) model.Workloads {
	if override == nil {
		return base
	}
	merged := make(model.Workloads, len(base)+len(override))
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

func mergeWorkload(base, override model.Workload) model.Workload {
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

func mergeResources(base, override model.Resources) model.Resources {
	if override.Volumes != nil {
		volumes := make(map[string]model.Volume, len(base.Volumes)+len(override.Volumes))
		maps.Copy(volumes, base.Volumes)
		for name, volume := range override.Volumes {
			if existing, ok := volumes[name]; ok {
				if volume.Size != "" {
					existing.Size = volume.Size
				}
				volumes[name] = existing
				continue
			}
			volumes[name] = volume
		}
		base.Volumes = volumes
	}
	if override.Configs != nil {
		configs := make(map[string]model.Config, len(base.Configs)+len(override.Configs))
		maps.Copy(configs, base.Configs)
		for name, config := range override.Configs {
			if existing, ok := configs[name]; ok {
				configs[name] = mergeConfig(existing, config)
				continue
			}
			configs[name] = config
		}
		base.Configs = configs
	}
	if override.Secrets != nil {
		secrets := make(map[string]model.Secret, len(base.Secrets)+len(override.Secrets))
		maps.Copy(secrets, base.Secrets)
		for name, secret := range override.Secrets {
			if existing, ok := secrets[name]; ok {
				secrets[name] = mergeSecret(existing, secret)
				continue
			}
			secrets[name] = secret
		}
		base.Secrets = secrets
	}
	return base
}

func mergeConfig(base, override model.Config) model.Config {
	if override.Values != nil {
		base.Values = mergeAnyMaps(base.Values, override.Values)
	}
	if override.Files != nil {
		base.Files = override.Files
	}
	return base
}

func mergeSecret(base, override model.Secret) model.Secret {
	if override.Env != nil {
		base.Env = mergeStringMaps(base.Env, override.Env)
	}
	if override.Files != nil {
		base.Files = override.Files
	}
	return base
}

func mergeStringMaps(base, override map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(override))
	maps.Copy(merged, base)
	maps.Copy(merged, override)
	return merged
}

func mergeAnyMaps(base, override map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(override))
	maps.Copy(merged, base)
	for k, v := range override {
		if v == nil {
			delete(merged, k)
			continue
		}
		if baseMap, ok := merged[k].(map[string]any); ok {
			if overrideMap, ok := v.(map[string]any); ok {
				merged[k] = mergeAnyMaps(baseMap, overrideMap)
				continue
			}
		}
		merged[k] = v
	}
	return merged
}
