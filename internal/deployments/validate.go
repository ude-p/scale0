package deployments

import (
	"errors"
	"fmt"

	"github.com/ude-p/shipl/internal/model"
)

func (s *Service) Validate(req CreateRequest) error {
	if req.Application == "" {
		return errors.New("application is required")
	}
	if req.Environment == "" {
		return errors.New("environment is required")
	}
	if err := validateWorkloadGroups(req.Spec.Groups(), true); err != nil {
		return err
	}
	if err := validateResources(req.Spec.Resources); err != nil {
		return err
	}
	if err := validateOverrides(req.Spec); err != nil {
		return err
	}
	return validateReferences(req.Spec)
}

func validateWorkloadGroups(groups map[string]model.Workloads, requireImage bool) error {
	for kind, workloads := range groups {
		for name, w := range workloads {
			switch {
			case name == "":
				return fmt.Errorf("%s: workload name is required", kind)
			case requireImage && w.Image == "":
				return fmt.Errorf("%s.%s: image is required", kind, name)
			case w.Replicas < 0:
				return fmt.Errorf("%s.%s: replicas must not be negative", kind, name)
			}
			for mount, path := range w.Mounts {
				if mount == "" || path == "" {
					return fmt.Errorf("%s.%s: mount name and path are required", kind, name)
				}
			}
			for _, route := range w.Routes {
				if route.Host == "" {
					return fmt.Errorf("%s.%s: route host is required", kind, name)
				}
				if route.Port < 1 || route.Port > 65535 {
					return fmt.Errorf("%s.%s: route port must be between 1 and 65535", kind, name)
				}
			}
		}
	}
	return nil
}

func validateResources(res model.Resources) error {
	for name, volume := range res.Volumes {
		if name == "" {
			return errors.New("volumes: resource name is required")
		}
		if volume.Size == "" {
			return fmt.Errorf("volumes.%s: size is required", name)
		}
	}
	for name, config := range res.Configs {
		if name == "" {
			return errors.New("configs: resource name is required")
		}
		if len(config.Values) == 0 && len(config.Files) == 0 {
			return fmt.Errorf("configs.%s: values or files are required", name)
		}
	}
	for name, secret := range res.Secrets {
		if name == "" {
			return errors.New("secrets: resource name is required")
		}
		if len(secret.Env) == 0 && len(secret.Files) == 0 {
			return fmt.Errorf("secrets.%s: env or files are required", name)
		}
	}
	return nil
}

func validateOverrides(spec model.Application) error {
	for name, env := range spec.Overrides {
		if name == "" {
			return errors.New("overrides: environment name is required")
		}
		if err := validateWorkloadGroups(env.Groups(), false); err != nil {
			return fmt.Errorf("overrides.%s: %w", name, err)
		}
		if env.Resources != nil {
			if err := validateResources(*env.Resources); err != nil {
				return fmt.Errorf("overrides.%s: %w", name, err)
			}
		}
	}
	return nil
}

func validateReferences(spec model.Application) error {
	base := resourceNames(spec.Resources)
	if err := checkMounts(spec.Groups(), base); err != nil {
		return err
	}

	for name, env := range spec.Overrides {
		names := resourceNames(spec.Resources)
		if env.Resources != nil {
			for resource := range resourceNames(*env.Resources) {
				names[resource] = struct{}{}
			}
		}
		if err := checkMounts(env.Groups(), names); err != nil {
			return fmt.Errorf("overrides.%s: %w", name, err)
		}
	}
	return nil
}

func checkMounts(groups map[string]model.Workloads, resources map[string]struct{}) error {
	for kind, workloads := range groups {
		for name, w := range workloads {
			for mount := range w.Mounts {
				if _, ok := resources[mount]; !ok {
					return fmt.Errorf("%s.%s: unknown mount %q", kind, name, mount)
				}
			}
		}
	}
	return nil
}

func resourceNames(res model.Resources) map[string]struct{} {
	names := map[string]struct{}{}
	for name := range res.Volumes {
		names[name] = struct{}{}
	}
	for name := range res.Configs {
		names[name] = struct{}{}
	}
	for name := range res.Secrets {
		names[name] = struct{}{}
	}
	return names
}
