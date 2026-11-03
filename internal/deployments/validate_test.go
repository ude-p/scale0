package deployments

import (
	"testing"

	"github.com/ude-p/shipl/internal/model"
)

func validRequest() CreateRequest {
	return CreateRequest{
		Application: "simplywhot",
		Environment: "production",
		Spec: model.Application{
			Apps: model.Workloads{
				"server": {
					Image:  "company/server:latest",
					Mounts: map[string]string{"data": "/data"},
				},
			},
			Resources: model.Resources{
				Volumes: map[string]model.Volume{"data": {Size: "10Gi"}},
			},
		},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*CreateRequest)
		wantErr string
	}{
		{
			name:   "valid",
			mutate: func(*CreateRequest) {},
		},
		{
			name:    "missing application",
			mutate:  func(r *CreateRequest) { r.Application = "" },
			wantErr: "application is required",
		},
		{
			name:    "missing environment",
			mutate:  func(r *CreateRequest) { r.Environment = "" },
			wantErr: "environment is required",
		},
		{
			name:    "missing workload image",
			mutate:  func(r *CreateRequest) { r.Spec.Apps["server"] = model.Workload{} },
			wantErr: "apps.server: image is required",
		},
		{
			name: "negative replicas",
			mutate: func(r *CreateRequest) {
				w := r.Spec.Apps["server"]
				w.Replicas = -1
				r.Spec.Apps["server"] = w
			},
			wantErr: "apps.server: replicas must not be negative",
		},
		{
			name: "missing mount path",
			mutate: func(r *CreateRequest) {
				w := r.Spec.Apps["server"]
				w.Mounts = map[string]string{"data": ""}
				r.Spec.Apps["server"] = w
			},
			wantErr: "apps.server: mount name and path are required",
		},
		{
			name: "route without host",
			mutate: func(r *CreateRequest) {
				w := r.Spec.Apps["server"]
				w.Routes = []model.Route{{Port: 8080}}
				r.Spec.Apps["server"] = w
			},
			wantErr: "apps.server: route host is required",
		},
		{
			name: "route with invalid port",
			mutate: func(r *CreateRequest) {
				w := r.Spec.Apps["server"]
				w.Routes = []model.Route{{Host: "example.com", Port: 70000}}
				r.Spec.Apps["server"] = w
			},
			wantErr: "apps.server: route port must be between 1 and 65535",
		},
		{
			name:    "volume without size",
			mutate:  func(r *CreateRequest) { r.Spec.Resources.Volumes["data"] = model.Volume{} },
			wantErr: "volumes.data: size is required",
		},
		{
			name: "empty config",
			mutate: func(r *CreateRequest) {
				r.Spec.Resources.Configs = map[string]model.Config{"app": {}}
			},
			wantErr: "configs.app: values or files are required",
		},
		{
			name: "empty secret",
			mutate: func(r *CreateRequest) {
				r.Spec.Resources.Secrets = map[string]model.Secret{"token": {}}
			},
			wantErr: "secrets.token: env or files are required",
		},
		{
			name: "unknown mount",
			mutate: func(r *CreateRequest) {
				w := r.Spec.Apps["server"]
				w.Mounts = map[string]string{"missing": "/data"}
				r.Spec.Apps["server"] = w
			},
			wantErr: `apps.server: unknown mount "missing"`,
		},
		{
			name: "valid override",
			mutate: func(r *CreateRequest) {
				r.Spec.Overrides = map[string]model.Environment{
					"production": {
						Apps: model.Workloads{
							"server": {Replicas: 3, Mounts: map[string]string{"scratch": "/scratch"}},
						},
						Resources: &model.Resources{
							Volumes: map[string]model.Volume{"scratch": {Size: "1Gi"}},
						},
					},
				}
			},
		},
		{
			name: "override with negative replicas",
			mutate: func(r *CreateRequest) {
				r.Spec.Overrides = map[string]model.Environment{
					"production": {Apps: model.Workloads{"server": {Replicas: -1}}},
				}
			},
			wantErr: "overrides.production: apps.server: replicas must not be negative",
		},
		{
			name: "override with empty volume",
			mutate: func(r *CreateRequest) {
				r.Spec.Overrides = map[string]model.Environment{
					"production": {
						Resources: &model.Resources{
							Volumes: map[string]model.Volume{"scratch": {}},
						},
					},
				}
			},
			wantErr: "overrides.production: volumes.scratch: size is required",
		},
		{
			name: "override with unknown mount",
			mutate: func(r *CreateRequest) {
				r.Spec.Overrides = map[string]model.Environment{
					"production": {Apps: model.Workloads{"server": {Mounts: map[string]string{"missing": "/data"}}}},
				}
			},
			wantErr: `overrides.production: apps.server: unknown mount "missing"`,
		},
		{
			name: "override without environment name",
			mutate: func(r *CreateRequest) {
				r.Spec.Overrides = map[string]model.Environment{
					"": {Apps: model.Workloads{"server": {Replicas: 3}}},
				}
			},
			wantErr: "overrides: environment name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest()
			tt.mutate(&req)

			err := NewService().Validate(req)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("got error %v, want %q", err, tt.wantErr)
			}
		})
	}
}
