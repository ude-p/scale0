package controller

import (
	"fmt"
	"maps"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/ude-p/scale0/api/v1alpha1"
)

const (
	labelName      = "app.kubernetes.io/name"
	labelInstance  = "app.kubernetes.io/instance"
	labelComponent = "app.kubernetes.io/component"
)

type kube struct {
	app  string
	env  string
	spec v1alpha1.S0Spec
}

func KubernetesObjects(app v1alpha1.S0) []runtime.Object {
	r := kube{app: app.Name, env: app.Namespace, spec: app.Spec}

	objects := make([]runtime.Object, 0, 2*len(app.Spec.Workload)+len(app.Spec.Volume)+len(app.Spec.Config)+len(app.Spec.Secret))
	for _, name := range slices.Sorted(maps.Keys(app.Spec.Volume)) {
		objects = append(objects, r.volumeClaim(name, app.Spec.Volume[name]))
	}
	for _, name := range slices.Sorted(maps.Keys(app.Spec.Config)) {
		objects = append(objects, r.configMap(name, app.Spec.Config[name]))
	}
	for _, name := range slices.Sorted(maps.Keys(app.Spec.Secret)) {
		objects = append(objects, r.secret(name, app.Spec.Secret[name]))
	}
	for _, name := range slices.Sorted(maps.Keys(app.Spec.Workload)) {
		w := app.Spec.Workload[name]
		switch w.Type {
		case "job":
			objects = append(objects, r.job(name, w))
		case "stateful":
			objects = append(objects, r.statefulSet(name, w))
		default:
			objects = append(objects, r.deployment(name, w))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(app.Spec.Workload)) {
		if w := app.Spec.Workload[name]; len(w.Ports) > 0 && w.Type != "job" {
			objects = append(objects, r.service(name, w))
		}
	}
	return objects
}

func (r kube) objectName(component string) string {
	return r.app + "-" + r.env + "-" + component
}

func (r kube) labels(component string) map[string]string {
	return map[string]string{
		labelName:      r.app,
		labelInstance:  r.env,
		labelComponent: component,
	}
}

func (r kube) deployment(name string, w v1alpha1.Workload) *appsv1.Deployment {
	return &appsv1.Deployment{
		Name:   r.objectName(name),
		Labels: r.labels(name),
		Spec: appsv1.DeploymentSpec{
			Replicas: new(replicas(w)),
			Selector: &metav1.LabelSelector{MatchLabels: r.labels(name)},
			Template: r.podTemplate(name, w),
		},
	}
}

func (r kube) statefulSet(name string, w v1alpha1.Workload) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		Name:   r.objectName(name),
		Labels: r.labels(name),
		Spec: appsv1.StatefulSetSpec{
			Replicas:    new(replicas(w)),
			Selector:    &metav1.LabelSelector{MatchLabels: r.labels(name)},
			ServiceName: r.objectName(name),
			Template:    r.podTemplate(name, w),
		},
	}
}

func (r kube) job(name string, w v1alpha1.Workload) *batchv1.Job {
	template := r.podTemplate(name, w)
	template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
	return &batchv1.Job{
		Name:   r.objectName(name),
		Labels: r.labels(name),
		Spec: batchv1.JobSpec{
			Parallelism: new(replicas(w)),
			Template:    template,
		},
	}
}

func (r kube) service(name string, w v1alpha1.Workload) *corev1.Service {
	ports := make([]corev1.ServicePort, 0, len(w.Ports))
	for _, p := range w.Ports {
		ports = append(ports, corev1.ServicePort{
			Name:       portName(p),
			Protocol:   corev1.ProtocolTCP,
			Port:       int32(p),
			TargetPort: intstr.FromInt32(int32(p)),
		})
	}
	return &corev1.Service{
		Name:   r.objectName(name),
		Labels: r.labels(name),
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: r.labels(name),
			Ports:    ports,
		},
	}
}

func (r kube) podTemplate(name string, w v1alpha1.Workload) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		Labels: r.labels(name),
		Spec: corev1.PodSpec{
			Containers:    []corev1.Container{r.container(name, w)},
			Volumes:       r.volumes(w),
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}
}

func (r kube) container(name string, w v1alpha1.Workload) corev1.Container {
	return corev1.Container{
		Name:         name,
		Image:        w.Image,
		Command:      w.Command,
		Args:         w.Args,
		Env:          envVars(w.Env),
		Ports:        containerPorts(w.Ports),
		Resources:    resourceRequirements(w),
		VolumeMounts: volumeMounts(w),
	}
}

func (r kube) volumeClaim(name, size string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		Name:   r.objectName(name),
		Labels: r.labels(name),
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(size)},
			},
		},
	}
}

func (r kube) configMap(name string, data v1alpha1.Config) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		Name:   r.objectName(name),
		Labels: r.labels(name),
		Data:   maps.Clone(data),
	}
}

func (r kube) secret(name string, data v1alpha1.Secret) *corev1.Secret {
	return &corev1.Secret{
		Name:       r.objectName(name),
		Labels:     r.labels(name),
		Type:       corev1.SecretTypeOpaque,
		StringData: maps.Clone(data),
	}
}

func (r kube) volumes(w v1alpha1.Workload) []corev1.Volume {
	if len(w.Mounts) == 0 {
		return nil
	}
	volumes := make([]corev1.Volume, 0, len(w.Mounts))
	for _, name := range slices.Sorted(maps.Keys(w.Mounts)) {
		volumes = append(volumes, r.volume(name))
	}
	return volumes
}

func (r kube) volume(name string) corev1.Volume {
	volume := corev1.Volume{Name: name}
	if _, ok := r.spec.Secret[name]; ok {
		volume.Secret = &corev1.SecretVolumeSource{SecretName: r.objectName(name)}
		return volume
	}
	if _, ok := r.spec.Config[name]; ok {
		volume.ConfigMap = &corev1.ConfigMapVolumeSource{Name: r.objectName(name)}
		return volume
	}
	volume.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{ClaimName: r.objectName(name)}
	return volume
}

func volumeMounts(w v1alpha1.Workload) []corev1.VolumeMount {
	mounts := make([]corev1.VolumeMount, 0, len(w.Mounts))
	for _, name := range slices.Sorted(maps.Keys(w.Mounts)) {
		mounts = append(mounts, corev1.VolumeMount{Name: name, MountPath: w.Mounts[name]})
	}
	return mounts
}

func envVars(env map[string]string) []corev1.EnvVar {
	vars := make([]corev1.EnvVar, 0, len(env))
	for _, name := range slices.Sorted(maps.Keys(env)) {
		vars = append(vars, corev1.EnvVar{Name: name, Value: env[name]})
	}
	return vars
}

func containerPorts(ports []int) []corev1.ContainerPort {
	if len(ports) == 0 {
		return nil
	}
	out := make([]corev1.ContainerPort, 0, len(ports))
	for _, p := range ports {
		out = append(out, corev1.ContainerPort{
			Name:          portName(p),
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: int32(p),
		})
	}
	return out
}

func portName(port int) string {
	return fmt.Sprintf("port-%d", port)
}

func resourceRequirements(w v1alpha1.Workload) corev1.ResourceRequirements {
	if w.CPU == "" && w.Memory == "" {
		return corev1.ResourceRequirements{}
	}
	resources := corev1.ResourceList{}
	if w.CPU != "" {
		resources[corev1.ResourceCPU] = resource.MustParse(w.CPU)
	}
	if w.Memory != "" {
		resources[corev1.ResourceMemory] = resource.MustParse(w.Memory)
	}
	return corev1.ResourceRequirements{
		Requests: resources,
		Limits:   resources,
	}
}

func replicas(w v1alpha1.Workload) int32 {
	return int32(max(w.Replicas, 1))
}
