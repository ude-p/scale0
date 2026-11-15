package controller

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/ude-p/scale0/api/v1alpha1"
)

func testApplication() v1alpha1.S0 {
	return v1alpha1.S0{
		Name:      "simplywhot",
		Namespace: "production",
		Spec: v1alpha1.S0Spec{
			Workload: map[string]v1alpha1.Workload{
				"server": {
					Type:     "deployment",
					Image:    "company/server:latest",
					Replicas: 3,
					CPU:      "1",
					Memory:   "512Mi",
					Ports:    []int{8080},
					Env:      map[string]string{"LOG_LEVEL": "info"},
					Command:  []string{"/app/server"},
					Args:     []string{"--port", "8080"},
					Mounts:   map[string]string{"data": "/data"},
				},
				"migrate": {
					Type:    "job",
					Image:   "company/migrate:latest",
					Command: []string{"./migrate", "--up"},
				},
			},
			Volume: map[string]string{"data": "10Gi"},
		},
	}
}

func TestKubernetesObjects(t *testing.T) {
	objects := KubernetesObjects(testApplication())
	if len(objects) != 4 {
		t.Fatalf("got %d objects, want 4", len(objects))
	}

	claim, ok := objects[0].(*corev1.PersistentVolumeClaim)
	if !ok {
		t.Fatalf("objects[0] = %T, want PersistentVolumeClaim", objects[0])
	}
	if claim.Name != "simplywhot-production-data" {
		t.Errorf("claim name = %q", claim.Name)
	}
	if got := claim.Spec.Resources.Requests.Storage().String(); got != "10Gi" {
		t.Errorf("claim size = %q, want 10Gi", got)
	}

	job, ok := objects[1].(*batchv1.Job)
	if !ok {
		t.Fatalf("objects[1] = %T, want Job", objects[1])
	}
	if job.Name != "simplywhot-production-migrate" {
		t.Errorf("job name = %q", job.Name)
	}
	if got := *job.Spec.Parallelism; got != 1 {
		t.Errorf("job parallelism = %d, want default 1", got)
	}
	if job.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyOnFailure {
		t.Errorf("job restart policy = %q", job.Spec.Template.Spec.RestartPolicy)
	}

	app, ok := objects[2].(*appsv1.Deployment)
	if !ok {
		t.Fatalf("objects[2] = %T, want Deployment", objects[2])
	}
	if app.Name != "simplywhot-production-server" {
		t.Errorf("deployment name = %q", app.Name)
	}
	if got := *app.Spec.Replicas; got != 3 {
		t.Errorf("replicas = %d, want 3", got)
	}

	container := app.Spec.Template.Spec.Containers[0]
	if container.Image != "company/server:latest" {
		t.Errorf("image = %q", container.Image)
	}
	if len(container.Command) != 1 || container.Command[0] != "/app/server" {
		t.Errorf("command = %v", container.Command)
	}
	if len(container.Args) != 2 || container.Args[1] != "8080" {
		t.Errorf("args = %v", container.Args)
	}
	if len(container.Env) != 1 || container.Env[0].Name != "LOG_LEVEL" || container.Env[0].Value != "info" {
		t.Errorf("env = %+v", container.Env)
	}
	if len(container.Ports) != 1 || container.Ports[0].ContainerPort != 8080 {
		t.Errorf("ports = %+v", container.Ports)
	}

	requests := container.Resources.Requests
	if got := requests.Cpu().String(); got != "1" {
		t.Errorf("cpu request = %q, want 1", got)
	}
	if got := requests.Memory().String(); got != "512Mi" {
		t.Errorf("memory request = %q, want 512Mi", got)
	}
	if got := container.Resources.Limits.Memory().String(); got != "512Mi" {
		t.Errorf("memory limit = %q", got)
	}

	mounts := container.VolumeMounts
	if len(mounts) != 1 || mounts[0].Name != "data" || mounts[0].MountPath != "/data" {
		t.Errorf("volume mounts = %+v", mounts)
	}
	volumes := app.Spec.Template.Spec.Volumes
	if len(volumes) != 1 || volumes[0].PersistentVolumeClaim == nil {
		t.Fatalf("volumes = %+v", volumes)
	}
	if got := volumes[0].PersistentVolumeClaim.ClaimName; got != "simplywhot-production-data" {
		t.Errorf("claim name = %q", got)
	}

	svc, ok := objects[3].(*corev1.Service)
	if !ok {
		t.Fatalf("objects[3] = %T, want Service", objects[3])
	}
	if svc.Name != "simplywhot-production-server" {
		t.Errorf("service name = %q", svc.Name)
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("service type = %q", svc.Spec.Type)
	}
	if len(svc.Spec.Ports) != 1 || svc.Spec.Ports[0].Port != 8080 {
		t.Errorf("service ports = %+v", svc.Spec.Ports)
	}
	for key, value := range app.Spec.Selector.MatchLabels {
		if svc.Spec.Selector[key] != value {
			t.Errorf("service selector %q = %q, want %q", key, svc.Spec.Selector[key], value)
		}
	}
}

func TestKubernetesObjectsResources(t *testing.T) {
	app := v1alpha1.S0{
		Name:      "simplywhot",
		Namespace: "production",
		Spec: v1alpha1.S0Spec{
			Workload: map[string]v1alpha1.Workload{
				"server": {
					Image: "company/server:latest",
					Mounts: map[string]string{
						"settings": "/settings",
						"creds":    "/creds",
					},
				},
			},
			Config: map[string]v1alpha1.Config{"settings": {"LEVEL": "info"}},
			Secret: map[string]v1alpha1.Secret{"creds": {"TOKEN": "x"}},
		},
	}

	objects := KubernetesObjects(app)
	if len(objects) != 3 {
		t.Fatalf("got %d objects, want 3", len(objects))
	}

	config, ok := objects[0].(*corev1.ConfigMap)
	if !ok {
		t.Fatalf("objects[0] = %T, want ConfigMap", objects[0])
	}
	if config.Data["LEVEL"] != "info" {
		t.Errorf("config data = %v", config.Data)
	}

	secret, ok := objects[1].(*corev1.Secret)
	if !ok {
		t.Fatalf("objects[1] = %T, want Secret", objects[1])
	}
	if secret.StringData["TOKEN"] != "x" {
		t.Errorf("secret data = %v", secret.StringData)
	}

	deployment, ok := objects[2].(*appsv1.Deployment)
	if !ok {
		t.Fatalf("objects[2] = %T, want Deployment", objects[2])
	}
	if got := *deployment.Spec.Replicas; got != 1 {
		t.Errorf("replicas = %d, want 1 for empty type default", got)
	}

	volumes := deployment.Spec.Template.Spec.Volumes
	if len(volumes) != 2 {
		t.Fatalf("volumes = %+v", volumes)
	}
	if volumes[0].Name != "creds" || volumes[0].Secret == nil {
		t.Errorf("volume[0] = %+v, want secret", volumes[0])
	}
	if volumes[1].Name != "settings" || volumes[1].ConfigMap == nil {
		t.Errorf("volume[1] = %+v, want configmap", volumes[1])
	}
}
