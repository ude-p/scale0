package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/ude-p/scale0/internal/controller"
)

func main() {
	config, err := restConfig()
	if err != nil {
		klog.Error(err, "unable to load kubernetes configuration")
		os.Exit(1)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Error(err, "unable to create kubernetes client")
		os.Exit(1)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Error(err, "unable to create dynamic client")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := controller.New(kubeClient, dynamicClient).Run(ctx); err != nil {
		klog.Error(err, "operator stopped")
		os.Exit(1)
	}
}

func restConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if errors.Is(err, rest.ErrNotInCluster) {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
	}
	return config, err
}
