package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"

	"github.com/spf13/cobra"
)

const (
	cdiOperatorURL = "https://github.com/kubevirt/containerized-data-importer/releases/latest/download/cdi-operator.yaml"
	cdiCRURL       = "https://github.com/kubevirt/containerized-data-importer/releases/latest/download/cdi-cr.yaml"
)

func newInstallCdiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cdi",
		Short: "Install CDI",
		Long:  "Install Containerized Data Importer (CDI) for image uploads",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isCdiInstalled() {
				fmt.Println("CDI is already installed ✓")
				return nil
			}
			return installCdi()
		},
	}

	return cmd
}

func newUninstallCdiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cdi",
		Short: "Uninstall CDI",
		Long:  "Uninstall Containerized Data Importer (CDI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isCdiInstalled() {
				fmt.Println("CDI is not installed")
				return nil
			}
			return uninstallCdi()
		},
	}

	return cmd
}

func newReinstallCdiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cdi",
		Short: "Reinstall CDI",
		Long:  "Reinstall Containerized Data Importer (CDI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isCdiInstalled() {
				fmt.Println("Uninstalling CDI...")
				if err := uninstallCdi(); err != nil {
					return fmt.Errorf("failed to uninstall CDI: %w", err)
				}
				fmt.Println("CDI uninstalled ✓")
				// Wait a bit for cleanup
				time.Sleep(3 * time.Second)
			}
			fmt.Println("Installing CDI...")
			return installCdi()
		},
	}

	return cmd
}

func isCdiInstalled() bool {
	clientset, err := getKubeClient()
	if err != nil {
		return false
	}

	ctx := context.Background()
	deployment, err := clientset.AppsV1().Deployments("cdi").Get(ctx, "cdi-operator", metav1.GetOptions{})
	if err != nil {
		return false
	}

	// Check if deployment is available
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

func installCdi() error {
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	// Check if CDI namespace exists and is terminating
	checkCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ns, err := clientset.CoreV1().Namespaces().Get(checkCtx, "cdi", metav1.GetOptions{})
	if err == nil {
		// Namespace exists - check if it's terminating
		if ns.Status.Phase == corev1.NamespaceTerminating {
			fmt.Println("CDI namespace is terminating, waiting for it to be fully deleted...")
			if err = waitForNamespaceDeleted(checkCtx, clientset, "cdi"); err != nil {
				return fmt.Errorf("failed to wait for CDI namespace deletion: %w", err)
			}
			fmt.Println("CDI namespace deleted ✓")
		}
	}

	fmt.Println("Installing CDI (Containerized Data Importer)...")

	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Apply operator manifest
	if err := applyManifestFromURL(dynamicClient, config, cdiOperatorURL); err != nil {
		return fmt.Errorf("failed to apply CDI operator manifest: %w", err)
	}

	// Apply CR manifest
	if err := applyManifestFromURL(dynamicClient, config, cdiCRURL); err != nil {
		return fmt.Errorf("failed to apply CDI CR manifest: %w", err)
	}

	// Wait for CDI operator deployment
	fmt.Println("Waiting for CDI to be ready...")
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer waitCancel()

	if err := waitForDeployment(waitCtx, clientset, "cdi", "cdi-operator"); err != nil {
		return fmt.Errorf("failed to wait for CDI operator: %w", err)
	}

	// Wait for CDI CR to be ready
	fmt.Println("Waiting for CDI CR to be ready...")
	if err := waitForCdiCRReady(waitCtx, dynamicClient); err != nil {
		return fmt.Errorf("failed to wait for CDI CR: %w", err)
	}

	fmt.Println("CDI installed ✓")
	return nil
}

func waitForCdiCRReady(ctx context.Context, dynamicClient dynamic.Interface) error {
	cdiGVR := schema.GroupVersionResource{
		Group:    "cdi.kubevirt.io",
		Version:  "v1beta1",
		Resource: "cdis",
	}

	fmt.Println("Checking CDI status...")

	// First try to wait for Available condition with a short timeout (like Makefile does with 10s)
	conditionCtx, conditionCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer conditionCancel()

	conditionMet := false
	_ = wait.PollUntilContextCancel(conditionCtx, 1*time.Second, true, func(checkCtx context.Context) (bool, error) {
		// Try cluster-scoped first (CDI CR is typically cluster-scoped)
		var cdi *unstructured.Unstructured
		var err error

		cdi, err = dynamicClient.Resource(cdiGVR).Get(checkCtx, "cdi", metav1.GetOptions{})
		if err != nil {
			// If cluster-scoped fails, try namespaced (some setups might use namespaced)
			cdi, err = dynamicClient.Resource(cdiGVR).Namespace("cdi").Get(checkCtx, "cdi", metav1.GetOptions{})
			if err != nil {
				// CDI CR might not exist yet, keep polling
				return false, nil
			}
		}

		// Check for Available condition
		conditions, found, err := unstructured.NestedSlice(cdi.Object, "status", "conditions")
		if found && err == nil {
			for _, cond := range conditions {
				if condMap, ok := cond.(map[string]interface{}); ok {
					if condType, ok := condMap["type"].(string); ok && condType == "Available" {
						if condStatus, ok := condMap["status"].(string); ok && condStatus == "True" {
							fmt.Println("✓ CDI is ready (Available condition met)")
							conditionMet = true
							return true, nil
						}
					}
				}
			}
		}
		return false, nil
	})

	if conditionMet {
		return nil
	}

	// If condition check didn't succeed, wait for phase to be Deployed
	fmt.Println("Waiting for CDI phase to be Deployed...")
	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(checkCtx context.Context) (bool, error) {
		// Try cluster-scoped first
		var cdi *unstructured.Unstructured
		var err error

		cdi, err = dynamicClient.Resource(cdiGVR).Get(checkCtx, "cdi", metav1.GetOptions{})
		if err != nil {
			// If cluster-scoped fails, try namespaced
			cdi, err = dynamicClient.Resource(cdiGVR).Namespace("cdi").Get(checkCtx, "cdi", metav1.GetOptions{})
			if err != nil {
				fmt.Print(".")
				return false, nil
			}
		}

		phase, found, err := unstructured.NestedString(cdi.Object, "status", "phase")
		if !found || err != nil {
			fmt.Print(".")
			return false, nil
		}

		if phase == "Deployed" {
			fmt.Printf("\n✓ CDI is ready (phase: %s)\n", phase)
			return true, nil
		}

		fmt.Print(".")
		return false, nil
	})
}

func uninstallCdi() error {
	fmt.Println("Uninstalling CDI...")

	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Delete CR first, then operator
	if err := deleteResourcesFromManifestURL(dynamicClient, config, cdiCRURL); err != nil {
		return fmt.Errorf("failed to delete CDI CR: %w", err)
	}

	if err := deleteResourcesFromManifestURL(dynamicClient, config, cdiOperatorURL); err != nil {
		return fmt.Errorf("failed to delete CDI operator: %w", err)
	}

	// Wait for namespace deletion
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	if err := waitForNamespaceDeleted(ctx, clientset, "cdi"); err != nil {
		return fmt.Errorf("failed to wait for CDI namespace deletion: %w", err)
	}

	fmt.Println("CDI uninstalled ✓")
	return nil
}
