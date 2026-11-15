package controller

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/ude-p/scale0/api/v1alpha1"
)

func Validate(app v1alpha1.S0) error {
	if err := validateVolumes(app.Spec.Volume); err != nil {
		return err
	}
	if err := validateMounts(app.Spec.Workload, resourceNames(app.Spec)); err != nil {
		return err
	}

	for name, env := range app.Spec.Overrides {
		if err := validateVolumes(env.Volume); err != nil {
			return fmt.Errorf("overrides.%s: %w", name, err)
		}

		available := resourceNames(app.Spec)
		for resource := range env.Volume {
			available[resource] = struct{}{}
		}
		for resource := range env.Config {
			available[resource] = struct{}{}
		}
		for resource := range env.Secret {
			available[resource] = struct{}{}
		}

		if err := validateMounts(env.Workload, available); err != nil {
			return fmt.Errorf("overrides.%s: %w", name, err)
		}
	}

	return nil
}

func validateVolumes(volumes map[string]string) error {
	for name, size := range volumes {
		if name == "" {
			return errors.New("volume: name is required")
		}
		if _, err := resource.ParseQuantity(size); err != nil {
			return fmt.Errorf("volume.%s: invalid size %q", name, size)
		}
	}
	return nil
}

func validateMounts(workloads map[string]v1alpha1.Workload, available map[string]struct{}) error {
	for workload, w := range workloads {
		for name, path := range w.Mounts {
			if name == "" {
				return fmt.Errorf("workload.%s: mount name is required", workload)
			}
			if path == "" {
				return fmt.Errorf("workload.%s: mount %q path is required", workload, name)
			}
			if _, ok := available[name]; !ok {
				return fmt.Errorf("workload.%s: unknown mount %q", workload, name)
			}
		}
	}
	return nil
}

func resourceNames(spec v1alpha1.S0Spec) map[string]struct{} {
	names := make(map[string]struct{}, len(spec.Volume)+len(spec.Config)+len(spec.Secret))
	for name := range spec.Volume {
		names[name] = struct{}{}
	}
	for name := range spec.Config {
		names[name] = struct{}{}
	}
	for name := range spec.Secret {
		names[name] = struct{}{}
	}
	return names
}
