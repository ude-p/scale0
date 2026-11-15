package controller

import (
	"testing"

	"github.com/ude-p/scale0/api/v1alpha1"
)

func validApplication() v1alpha1.S0 {
	return v1alpha1.S0{
		Name:      "simplywhot",
		Namespace: "production",
		Spec: v1alpha1.S0Spec{
			Workload: map[string]v1alpha1.Workload{
				"server": {
					Type:   "deployment",
					Image:  "company/server:latest",
					Mounts: map[string]string{"data": "/data"},
				},
			},
			Volume: map[string]string{"data": "10Gi"},
		},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*v1alpha1.S0)
		wantErr string
	}{
		{
			name:   "valid",
			mutate: func(*v1alpha1.S0) {},
		},
		{
			name: "invalid volume size",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Volume["data"] = "lots"
			},
			wantErr: `volume.data: invalid size "lots"`,
		},
		{
			name: "unknown mount",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Workload["server"] = v1alpha1.Workload{
					Mounts: map[string]string{"nope": "/nope"},
				}
			},
			wantErr: `workload.server: unknown mount "nope"`,
		},
		{
			name: "empty mount path",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Workload["server"] = v1alpha1.Workload{
					Mounts: map[string]string{"data": ""},
				}
			},
			wantErr: `workload.server: mount "data" path is required`,
		},
		{
			name: "mount from config",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Config = map[string]v1alpha1.Config{"cache": {"ttl": "60"}}
				app.Spec.Workload["server"] = v1alpha1.Workload{
					Mounts: map[string]string{"cache": "/cache"},
				}
			},
		},
		{
			name: "mount from secret",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Secret = map[string]v1alpha1.Secret{"creds": {"token": "x"}}
				app.Spec.Workload["server"] = v1alpha1.Workload{
					Mounts: map[string]string{"creds": "/creds"},
				}
			},
		},
		{
			name: "override mount from override volume",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"staging": {
						Workload: map[string]v1alpha1.Workload{
							"worker": {Mounts: map[string]string{"scratch": "/scratch"}},
						},
						Volume: map[string]string{"scratch": "1Gi"},
					},
				}
			},
		},
		{
			name: "override with unknown mount",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"staging": {
						Workload: map[string]v1alpha1.Workload{
							"worker": {Mounts: map[string]string{"scratch": "/scratch"}},
						},
					},
				}
			},
			wantErr: `overrides.staging: workload.worker: unknown mount "scratch"`,
		},
		{
			name: "override with invalid volume size",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"staging": {Volume: map[string]string{"scratch": "big"}},
				}
			},
			wantErr: `overrides.staging: volume.scratch: invalid size "big"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := validApplication()
			tt.mutate(&app)

			err := Validate(app)

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
