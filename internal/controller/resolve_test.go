package controller

import (
	"reflect"
	"testing"

	"github.com/ude-p/scale0/api/v1alpha1"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*v1alpha1.S0)
		check   func(*testing.T, v1alpha1.S0)
		wantErr string
	}{
		{
			name:   "no overrides",
			mutate: func(*v1alpha1.S0) {},
			check: func(t *testing.T, app v1alpha1.S0) {
				if app.Spec.Overrides != nil {
					t.Errorf("overrides = %v, want nil", app.Spec.Overrides)
				}
				if got := app.Spec.Workload["server"].Image; got != "company/server:latest" {
					t.Errorf("image = %q", got)
				}
			},
		},
		{
			name: "override for another namespace is ignored",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"staging": {
						Workload: map[string]v1alpha1.Workload{
							"server": {Replicas: 5},
						},
					},
				}
			},
			check: func(t *testing.T, app v1alpha1.S0) {
				if got := app.Spec.Workload["server"].Replicas; got != 0 {
					t.Errorf("replicas = %d, want 0", got)
				}
			},
		},
		{
			name: "partial merge keeps base fields",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"production": {
						Workload: map[string]v1alpha1.Workload{
							"server": {Replicas: 1},
						},
					},
				}
			},
			check: func(t *testing.T, app v1alpha1.S0) {
				server := app.Spec.Workload["server"]
				if server.Replicas != 1 {
					t.Errorf("replicas = %d, want 1", server.Replicas)
				}
				if server.Image != "company/server:latest" {
					t.Errorf("image = %q, want base image", server.Image)
				}
				if server.Mounts["data"] != "/data" {
					t.Errorf("mounts = %v, want base mounts", server.Mounts)
				}
				if app.Spec.Overrides != nil {
					t.Errorf("overrides = %v, want nil", app.Spec.Overrides)
				}
			},
		},
		{
			name: "override adds a workload",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"production": {
						Workload: map[string]v1alpha1.Workload{
							"worker": {Image: "company/worker:latest"},
						},
					},
				}
			},
			check: func(t *testing.T, app v1alpha1.S0) {
				if len(app.Spec.Workload) != 2 {
					t.Fatalf("workloads = %v, want server and worker", app.Spec.Workload)
				}
				if got := app.Spec.Workload["worker"].Image; got != "company/worker:latest" {
					t.Errorf("worker image = %q", got)
				}
				if _, ok := app.Spec.Workload["server"]; !ok {
					t.Error("server workload missing")
				}
			},
		},
		{
			name: "config values merge per key",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Config = map[string]v1alpha1.Config{
					"app": {"A": "1", "B": "2"},
				}
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"production": {
						Config: map[string]v1alpha1.Config{
							"app": {"B": "3"},
						},
					},
				}
			},
			check: func(t *testing.T, app v1alpha1.S0) {
				want := v1alpha1.Config{"A": "1", "B": "3"}
				if got := app.Spec.Config["app"]; !reflect.DeepEqual(got, want) {
					t.Errorf("config = %v, want %v", got, want)
				}
			},
		},
		{
			name: "volume size is replaced",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"production": {
						Volume: map[string]string{"data": "1Gi"},
					},
				}
			},
			check: func(t *testing.T, app v1alpha1.S0) {
				if got := app.Spec.Volume["data"]; got != "1Gi" {
					t.Errorf("size = %q, want 1Gi", got)
				}
			},
		},
		{
			name: "invalid override volume fails",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"production": {
						Volume: map[string]string{"data": "lots"},
					},
				}
			},
			wantErr: `volume.data: invalid size "lots"`,
		},
		{
			name: "unknown mount from override fails",
			mutate: func(app *v1alpha1.S0) {
				app.Spec.Overrides = map[string]v1alpha1.Environment{
					"production": {
						Workload: map[string]v1alpha1.Workload{
							"server": {Mounts: map[string]string{"nope": "/nope"}},
						},
					},
				}
			},
			wantErr: `workload.server: unknown mount "nope"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := validApplication()
			tt.mutate(&app)

			resolved, err := Resolve(app)

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("got error %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, resolved)
			}
		})
	}
}
