package main

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getKubeClient() (kubernetes.Interface, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, err
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

func getKubeConfig() (*rest.Config, error) {
	kubeconfigPath := getKubeconfigPath()
	
	// Check if kubeconfig exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig not found at %s. Please create the cluster first with: kubevirt-env create-test-cluster", kubeconfigPath)
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Set the context if specified
	kubectlContext := getKubectlContext()
	if kubectlContext != "" {
		// Load kubeconfig
		configLoadingRules := &clientcmd.ClientConfigLoadingRules{
			ExplicitPath: kubeconfigPath,
		}
		configOverrides := &clientcmd.ConfigOverrides{
			CurrentContext: kubectlContext,
		}
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, configOverrides).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig with context: %w", err)
		}
	}

	return config, nil
}
