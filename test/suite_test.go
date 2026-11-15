package test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/ude-p/scale0/api/v1alpha1"
	"github.com/ude-p/scale0/internal/controller"
)

var s0Resource = v1alpha1.GroupVersion.WithResource("s0s")

func newReconciler(t *testing.T, objects ...*v1alpha1.S0) (*controller.Reconciler, kubernetes.Interface) {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	dynamicObjects := make([]runtime.Object, 0, len(objects))
	for _, object := range objects {
		dynamicObjects = append(dynamicObjects, toUnstructured(t, object))
	}

	kubeClient := kubefake.NewSimpleClientset()
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{s0Resource: "S0List"},
		dynamicObjects...,
	)

	return controller.New(kubeClient, dynamicClient), kubeClient
}

func toUnstructured(t *testing.T, app *v1alpha1.S0) *unstructured.Unstructured {
	t.Helper()

	object := app.DeepCopy()
	object.TypeMeta = metav1.TypeMeta{APIVersion: v1alpha1.GroupVersion.String(), Kind: "S0"}

	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		t.Fatalf("to unstructured: %v", err)
	}
	return &unstructured.Unstructured{Object: content}
}

func sampleApp() *v1alpha1.S0 {
	return &v1alpha1.S0{
		Name:      "simplywhot",
		Namespace: "default",
		UID:       "s0-uid",
		Spec: v1alpha1.S0Spec{
			Workload: map[string]v1alpha1.Workload{
				"server": {
					Image:    "nginx:1.27",
					Replicas: 2,
					Ports:    []int{8080},
					Mounts:   map[string]string{"data": "/data"},
				},
			},
			Volume: map[string]string{"data": "1Gi"},
		},
	}
}
