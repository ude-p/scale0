package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/ude-p/scale0/api/v1alpha1"
)

const (
	resyncPeriod = 10 * time.Minute
	workers      = 1
)

var s0Resource = v1alpha1.GroupVersion.WithResource("s0s")

type Reconciler struct {
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
	queue         workqueue.TypedRateLimitingInterface[types.NamespacedName]
}

func New(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) *Reconciler {
	return &Reconciler{
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
		queue:         workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[types.NamespacedName]()),
	}
}

func (r *Reconciler) Run(ctx context.Context) error {
	kubeFactory := informers.NewSharedInformerFactory(r.kubeClient, resyncPeriod)
	s0Factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(r.dynamicClient, resyncPeriod, metav1.NamespaceAll, nil)

	s0s := s0Factory.ForResource(s0Resource).Informer()
	if _, err := s0s.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    r.enqueueS0,
		UpdateFunc: func(_, obj any) { r.enqueueS0(obj) },
		DeleteFunc: r.enqueueS0,
	}); err != nil {
		return fmt.Errorf("add s0 event handler: %w", err)
	}

	deployments := kubeFactory.Apps().V1().Deployments().Informer()
	statefulSets := kubeFactory.Apps().V1().StatefulSets().Informer()
	jobs := kubeFactory.Batch().V1().Jobs().Informer()
	services := kubeFactory.Core().V1().Services().Informer()
	claims := kubeFactory.Core().V1().PersistentVolumeClaims().Informer()
	configMaps := kubeFactory.Core().V1().ConfigMaps().Informer()
	secrets := kubeFactory.Core().V1().Secrets().Informer()

	owned := []cache.SharedIndexInformer{deployments, statefulSets, jobs, services, claims, configMaps, secrets}
	for _, informer := range owned {
		if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    r.enqueueOwner,
			UpdateFunc: func(_, obj any) { r.enqueueOwner(obj) },
			DeleteFunc: r.enqueueOwner,
		}); err != nil {
			return fmt.Errorf("add owned object event handler: %w", err)
		}
	}

	kubeFactory.Start(ctx.Done())
	s0Factory.Start(ctx.Done())

	synced := []cache.InformerSynced{s0s.HasSynced}
	for _, informer := range owned {
		synced = append(synced, informer.HasSynced)
	}
	if !cache.WaitForCacheSync(ctx.Done(), synced...) {
		return errors.New("failed to sync informer caches")
	}

	for range workers {
		go wait.UntilWithContext(ctx, r.work, time.Second)
	}

	<-ctx.Done()
	r.queue.ShutDown()
	return nil
}

func (r *Reconciler) work(ctx context.Context) {
	for r.processNext(ctx) {
	}
}

func (r *Reconciler) processNext(ctx context.Context) bool {
	key, shutdown := r.queue.Get()
	if shutdown {
		return false
	}
	defer r.queue.Done(key)

	if err := r.Reconcile(ctx, key); err != nil {
		klog.FromContext(ctx).Error(err, "reconcile failed", "s0", key.String())
		r.queue.AddRateLimited(key)
		return true
	}

	r.queue.Forget(key)
	return true
}

func (r *Reconciler) Reconcile(ctx context.Context, key types.NamespacedName) error {
	object, err := r.dynamicClient.Resource(s0Resource).Namespace(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get s0 %s: %w", key, err)
	}

	app, err := decode(object)
	if err != nil {
		return err
	}

	resolved, err := Resolve(*app)
	if err != nil {
		return err
	}

	objects := KubernetesObjects(resolved)
	for _, desired := range objects {
		if err := r.apply(ctx, app, desired); err != nil {
			return err
		}
	}

	return r.prune(ctx, app, objects)
}

func (r *Reconciler) apply(ctx context.Context, app *v1alpha1.S0, desired runtime.Object) error {
	accessor, err := meta.Accessor(desired)
	if err != nil {
		return fmt.Errorf("access %T: %w", desired, err)
	}
	accessor.SetNamespace(app.Namespace)
	accessor.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(app, s0Kind())})

	namespace := app.Namespace

	switch object := desired.(type) {
	case *appsv1.Deployment:
		deployments := r.kubeClient.AppsV1().Deployments(namespace)
		return upsert(ctx, typedClient[*appsv1.Deployment]{
			get:    deployments.Get,
			create: deployments.Create,
			update: deployments.Update,
		}, object, func(existing, desired *appsv1.Deployment) {
			existing.Spec = desired.Spec
		})
	case *appsv1.StatefulSet:
		statefulSets := r.kubeClient.AppsV1().StatefulSets(namespace)
		return upsert(ctx, typedClient[*appsv1.StatefulSet]{
			get:    statefulSets.Get,
			create: statefulSets.Create,
			update: statefulSets.Update,
		}, object, func(existing, desired *appsv1.StatefulSet) {
			existing.Spec = desired.Spec
		})
	case *batchv1.Job:
		jobs := r.kubeClient.BatchV1().Jobs(namespace)
		return upsert(ctx, typedClient[*batchv1.Job]{
			get:    jobs.Get,
			create: jobs.Create,
			update: jobs.Update,
		}, object, func(existing, desired *batchv1.Job) {
			existing.Spec = desired.Spec
		})
	case *corev1.Service:
		services := r.kubeClient.CoreV1().Services(namespace)
		return upsert(ctx, typedClient[*corev1.Service]{
			get:    services.Get,
			create: services.Create,
			update: services.Update,
		}, object, func(existing, desired *corev1.Service) {
			existing.Spec = desired.Spec
		})
	case *corev1.PersistentVolumeClaim:
		claims := r.kubeClient.CoreV1().PersistentVolumeClaims(namespace)
		return upsert(ctx, typedClient[*corev1.PersistentVolumeClaim]{
			get:    claims.Get,
			create: claims.Create,
			update: claims.Update,
		}, object, func(existing, desired *corev1.PersistentVolumeClaim) {
			existing.Spec = desired.Spec
		})
	case *corev1.ConfigMap:
		configMaps := r.kubeClient.CoreV1().ConfigMaps(namespace)
		return upsert(ctx, typedClient[*corev1.ConfigMap]{
			get:    configMaps.Get,
			create: configMaps.Create,
			update: configMaps.Update,
		}, object, func(existing, desired *corev1.ConfigMap) {
			existing.Data = desired.Data
		})
	case *corev1.Secret:
		secrets := r.kubeClient.CoreV1().Secrets(namespace)
		return upsert(ctx, typedClient[*corev1.Secret]{
			get:    secrets.Get,
			create: secrets.Create,
			update: secrets.Update,
		}, object, func(existing, desired *corev1.Secret) {
			existing.Type = desired.Type
			existing.StringData = desired.StringData
		})
	}

	return fmt.Errorf("unsupported object type %T", desired)
}

type typedClient[T metav1.Object] struct {
	get    func(ctx context.Context, name string, opts metav1.GetOptions) (T, error)
	create func(ctx context.Context, obj T, opts metav1.CreateOptions) (T, error)
	update func(ctx context.Context, obj T, opts metav1.UpdateOptions) (T, error)
}

func upsert[T metav1.Object](ctx context.Context, client typedClient[T], desired T, copySpec func(existing, desired T)) error {
	existing, err := client.get(ctx, desired.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = client.create(ctx, desired, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	existing.SetLabels(desired.GetLabels())
	existing.SetOwnerReferences(desired.GetOwnerReferences())
	copySpec(existing, desired)

	_, err = client.update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (r *Reconciler) prune(ctx context.Context, app *v1alpha1.S0, desired []runtime.Object) error {
	want := make(map[string]struct{}, len(desired))
	for _, object := range desired {
		accessor, err := meta.Accessor(object)
		if err != nil {
			return fmt.Errorf("access %T: %w", object, err)
		}
		want[objectKey(object, accessor.GetName())] = struct{}{}
	}

	deployments, err := r.kubeClient.AppsV1().Deployments(app.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}
	for i := range deployments.Items {
		object := &deployments.Items[i]
		if !wantedPrune(object, object, app, want) {
			continue
		}
		if err := r.kubeClient.AppsV1().Deployments(app.Namespace).Delete(ctx, object.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("prune deployment %q: %w", object.Name, err)
		}
	}

	statefulSets, err := r.kubeClient.AppsV1().StatefulSets(app.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list statefulsets: %w", err)
	}
	for i := range statefulSets.Items {
		object := &statefulSets.Items[i]
		if !wantedPrune(object, object, app, want) {
			continue
		}
		if err := r.kubeClient.AppsV1().StatefulSets(app.Namespace).Delete(ctx, object.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("prune statefulset %q: %w", object.Name, err)
		}
	}

	jobs, err := r.kubeClient.BatchV1().Jobs(app.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list jobs: %w", err)
	}
	for i := range jobs.Items {
		object := &jobs.Items[i]
		if !wantedPrune(object, object, app, want) {
			continue
		}
		if err := r.kubeClient.BatchV1().Jobs(app.Namespace).Delete(ctx, object.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("prune job %q: %w", object.Name, err)
		}
	}

	services, err := r.kubeClient.CoreV1().Services(app.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list services: %w", err)
	}
	for i := range services.Items {
		object := &services.Items[i]
		if !wantedPrune(object, object, app, want) {
			continue
		}
		if err := r.kubeClient.CoreV1().Services(app.Namespace).Delete(ctx, object.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("prune service %q: %w", object.Name, err)
		}
	}

	claims, err := r.kubeClient.CoreV1().PersistentVolumeClaims(app.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list persistentvolumeclaims: %w", err)
	}
	for i := range claims.Items {
		object := &claims.Items[i]
		if !wantedPrune(object, object, app, want) {
			continue
		}
		if err := r.kubeClient.CoreV1().PersistentVolumeClaims(app.Namespace).Delete(ctx, object.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("prune persistentvolumeclaim %q: %w", object.Name, err)
		}
	}

	configMaps, err := r.kubeClient.CoreV1().ConfigMaps(app.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list configmaps: %w", err)
	}
	for i := range configMaps.Items {
		object := &configMaps.Items[i]
		if !wantedPrune(object, object, app, want) {
			continue
		}
		if err := r.kubeClient.CoreV1().ConfigMaps(app.Namespace).Delete(ctx, object.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("prune configmap %q: %w", object.Name, err)
		}
	}

	secrets, err := r.kubeClient.CoreV1().Secrets(app.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list secrets: %w", err)
	}
	for i := range secrets.Items {
		object := &secrets.Items[i]
		if !wantedPrune(object, object, app, want) {
			continue
		}
		if err := r.kubeClient.CoreV1().Secrets(app.Namespace).Delete(ctx, object.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("prune secret %q: %w", object.Name, err)
		}
	}

	return nil
}

func wantedPrune(object any, accessor metav1.Object, app *v1alpha1.S0, want map[string]struct{}) bool {
	if _, ok := want[objectKey(object, accessor.GetName())]; ok {
		return false
	}
	return metav1.IsControlledBy(accessor, app)
}

func objectKey(object any, name string) string {
	return fmt.Sprintf("%T/%s", object, name)
}

func decode(object *unstructured.Unstructured) (*v1alpha1.S0, error) {
	var app v1alpha1.S0
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &app); err != nil {
		return nil, fmt.Errorf("decode s0 %s/%s: %w", object.GetNamespace(), object.GetName(), err)
	}
	return &app, nil
}

func s0Kind() schema.GroupVersionKind {
	return v1alpha1.GroupVersion.WithKind("S0")
}

func (r *Reconciler) enqueueS0(obj any) {
	accessor, err := meta.Accessor(unwrap(obj))
	if err != nil {
		return
	}
	r.queue.Add(types.NamespacedName{Namespace: accessor.GetNamespace(), Name: accessor.GetName()})
}

func (r *Reconciler) enqueueOwner(obj any) {
	accessor, err := meta.Accessor(unwrap(obj))
	if err != nil {
		return
	}
	for _, ref := range accessor.GetOwnerReferences() {
		if ref.Kind != "S0" || ref.APIVersion != v1alpha1.GroupVersion.String() {
			continue
		}
		r.queue.Add(types.NamespacedName{Namespace: accessor.GetNamespace(), Name: ref.Name})
	}
}

func unwrap(obj any) any {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		return tombstone.Obj
	}
	return obj
}
