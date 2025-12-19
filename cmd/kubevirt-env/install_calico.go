package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/spf13/cobra"
)

const (
	calicoVersion    = "v3.29.1"
	calicoManifestURL = "https://raw.githubusercontent.com/projectcalico/calico/%s/manifests/calico.yaml"
)

func newInstallCalicoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-calico",
		Short: "Install Calico CNI",
		Long:  "Install Calico CNI (required for LoadBalancer support in KubeVirt)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return installCalico()
		},
	}

	return cmd
}

func installCalico() error {
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	fmt.Printf("Installing Calico CNI %s...\n", calicoVersion)
	calicoURL := fmt.Sprintf(calicoManifestURL, calicoVersion)

	// Download and apply manifest using client-go
	if err := applyManifestFromURL(dynamicClient, config, calicoURL); err != nil {
		return fmt.Errorf("failed to apply Calico manifest: %w", err)
	}

	fmt.Println("Waiting for Calico to be ready...")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Wait for calico-kube-controllers deployment using client-go
	fmt.Println("Waiting for Calico kube-controllers deployment...")
	if err := waitForDeployment(ctx, clientset, "kube-system", "calico-kube-controllers"); err != nil {
		fmt.Printf("Warning: Calico kube-controllers may not be fully ready: %v\n", err)
	}

	// Wait for calico-node daemonset using client-go
	fmt.Println("Waiting for Calico node daemonset pods to be ready...")
	if err := waitForDaemonset(ctx, clientset, "kube-system", "calico-node"); err != nil {
		fmt.Printf("Warning: Calico node daemonset may not be fully ready: %v\n", err)
		// Show daemonset status
		ds, err := clientset.AppsV1().DaemonSets("kube-system").Get(ctx, "calico-node", metav1.GetOptions{})
		if err == nil {
			fmt.Printf("Daemonset status: %d/%d pods ready\n", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
		}
	}

	fmt.Println("Calico CNI installed ✓")
	return nil
}


