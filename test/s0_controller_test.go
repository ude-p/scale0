package test

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReconcileCreatesObjects(t *testing.T) {
	ctx := context.Background()
	app := sampleApp()
	reconciler, kube := newReconciler(t, app)

	key := types.NamespacedName{Namespace: app.Namespace, Name: app.Name}
	if err := reconciler.Reconcile(ctx, key); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	deployment, err := kube.AppsV1().Deployments(app.Namespace).Get(ctx, "simplywhot-default-server", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if got := *deployment.Spec.Replicas; got != 2 {
		t.Errorf("replicas = %d, want 2", got)
	}
	if image := deployment.Spec.Template.Spec.Containers[0].Image; image != "nginx:1.27" {
		t.Errorf("image = %q", image)
	}
	if !metav1.IsControlledBy(deployment, app) {
		t.Error("deployment is not owned by the S0")
	}

	service, err := kube.CoreV1().Services(app.Namespace).Get(ctx, "simplywhot-default-server", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get service: %v", err)
	}
	if len(service.Spec.Ports) != 1 || service.Spec.Ports[0].Port != 8080 {
		t.Errorf("service ports = %+v", service.Spec.Ports)
	}

	claim, err := kube.CoreV1().PersistentVolumeClaims(app.Namespace).Get(ctx, "simplywhot-default-data", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get claim: %v", err)
	}
	if got := claim.Spec.Resources.Requests.Storage().String(); got != "1Gi" {
		t.Errorf("claim size = %q, want 1Gi", got)
	}
}
