package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"

	"github.com/spf13/cobra"
)

const (
	kubevirtVersion     = "v1.3.0"
	kubevirtOperatorURL = "https://github.com/kubevirt/kubevirt/releases/download/%s/kubevirt-operator.yaml"
	kubevirtCRURL       = "https://github.com/kubevirt/kubevirt/releases/download/%s/kubevirt-cr.yaml"
)

func newInstallKubevirtCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubevirt",
		Short: "Install KubeVirt",
		Long:  "Install KubeVirt on the kind cluster (requires CDI to be installed first)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return installKubevirt()
		},
	}

	return cmd
}

func newUninstallKubevirtCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubevirt",
		Short: "Uninstall KubeVirt",
		Long:  "Uninstall KubeVirt from the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallKubevirt()
		},
	}

	return cmd
}

func newReinstallKubevirtCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubevirt",
		Short: "Reinstall KubeVirt",
		Long:  "Uninstall and reinstall KubeVirt",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := uninstallKubevirt(); err != nil {
				return fmt.Errorf("failed to uninstall KubeVirt: %w", err)
			}
			return installKubevirt()
		},
	}
	return cmd
}

func isKubeVirtInstalled() bool {
	config, err := getKubeConfig()
	if err != nil {
		return false
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	kubevirt, err := getKubeVirtCR(ctx, dynamicClient)
	if err != nil {
		return false
	}

	// Check for Available condition
	conditions, found, err := unstructured.NestedSlice(kubevirt.Object, "status", "conditions")
	if found && err == nil {
		for _, cond := range conditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				if condType, _ := condMap["type"].(string); condType == "Available" {
					if status, _ := condMap["status"].(string); status == "True" {
						return true
					}
				}
			}
		}
	}

	// Check for Deployed phase
	phase, found, err := unstructured.NestedString(kubevirt.Object, "status", "phase")
	if found && err == nil && phase == "Deployed" {
		return true
	}

	return false
}

func installKubevirt() error {
	// Check if KubeVirt is already installed
	if isKubeVirtInstalled() {
		fmt.Println("KubeVirt is already installed ✓")
		return nil
	}

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

	fmt.Printf("Installing KubeVirt %s...\n", kubevirtVersion)

	// Apply KubeVirt operator
	operatorURL := fmt.Sprintf(kubevirtOperatorURL, kubevirtVersion)
	if err := applyManifestFromURL(dynamicClient, config, operatorURL); err != nil {
		return fmt.Errorf("failed to apply KubeVirt operator: %w", err)
	}

	// Apply KubeVirt CR
	crURL := fmt.Sprintf(kubevirtCRURL, kubevirtVersion)
	if err := applyManifestFromURL(dynamicClient, config, crURL); err != nil {
		return fmt.Errorf("failed to apply KubeVirt CR: %w", err)
	}

	fmt.Println("Waiting for KubeVirt to be ready...")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Wait for virt-operator deployment
	fmt.Println("Waiting for virt-operator deployment...")
	if err := waitForDeployment(ctx, clientset, "kubevirt", "virt-operator"); err != nil {
		fmt.Printf("Warning: virt-operator may not be fully ready: %v\n", err)
	}

	// Wait for KubeVirt CR to be ready
	fmt.Println("Waiting for KubeVirt CR to be ready...")
	if err := waitForKubeVirtCR(ctx, dynamicClient); err != nil {
		fmt.Printf("Warning: KubeVirt CR may not be fully ready: %v\n", err)
		// Show KubeVirt status
		kubevirt, err := getKubeVirtCR(ctx, dynamicClient)
		if err == nil && kubevirt != nil {
			phase, _, _ := unstructured.NestedString(kubevirt.Object, "status", "phase")
			fmt.Printf("KubeVirt phase: %s\n", phase)
		}
	}

	fmt.Println("KubeVirt installed ✓")
	return nil
}

func uninstallKubevirt() error {
	// Check if KubeVirt is installed
	if !isKubeVirtInstalled() {
		fmt.Println("KubeVirt is not installed")
		return nil
	}

	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	fmt.Println("Uninstalling KubeVirt...")

	// Delete KubeVirt CR first
	crURL := fmt.Sprintf(kubevirtCRURL, kubevirtVersion)
	if err := deleteResourcesFromManifestURL(dynamicClient, config, crURL); err != nil {
		fmt.Printf("Warning: failed to delete KubeVirt CR: %v\n", err)
	}

	// Delete KubeVirt operator
	operatorURL := fmt.Sprintf(kubevirtOperatorURL, kubevirtVersion)
	if err := deleteResourcesFromManifestURL(dynamicClient, config, operatorURL); err != nil {
		return fmt.Errorf("failed to delete KubeVirt operator: %w", err)
	}

	fmt.Println("KubeVirt uninstalled ✓")
	return nil
}

func waitForKubeVirtCR(ctx context.Context, dynamicClient dynamic.Interface) error {
	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		kubevirt, err := getKubeVirtCR(ctx, dynamicClient)
		if err != nil {
			return false, nil
		}

		// Check for Available condition
		conditions, found, err := unstructured.NestedSlice(kubevirt.Object, "status", "conditions")
		if found && err == nil {
			for _, cond := range conditions {
				if condMap, ok := cond.(map[string]interface{}); ok {
					if condType, _ := condMap["type"].(string); condType == "Available" {
						if status, _ := condMap["status"].(string); status == "True" {
							fmt.Println("✓ KubeVirt is ready (Available condition met)")
							return true, nil
						}
					}
				}
			}
		}

		// Check for Deployed phase
		phase, found, err := unstructured.NestedString(kubevirt.Object, "status", "phase")
		if found && err == nil && phase == "Deployed" {
			fmt.Printf("✓ KubeVirt is ready (phase: %s)\n", phase)
			return true, nil
		}

		fmt.Print(".")
		return false, nil
	})
}

func getKubeVirtCR(ctx context.Context, dynamicClient dynamic.Interface) (*unstructured.Unstructured, error) {
	// Get the KubeVirt CRD resource
	kubevirtGVR := schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "kubevirts",
	}

	kubevirt, err := dynamicClient.Resource(kubevirtGVR).Namespace("kubevirt").Get(ctx, "kubevirt", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return kubevirt, nil
}
