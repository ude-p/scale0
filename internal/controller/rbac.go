package controller

// Checking whether other controllers are installed: read-only on CRDs.
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

// +kubebuilder:rbac:groups=apps.s0,resources=s0s,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apps.s0,resources=s0s/status,verbs=get;update;patch

// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;configmaps;secrets;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
