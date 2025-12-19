package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/spf13/cobra"
)

const (
	certManagerVersion = "v1.16.2"
	certManagerURL     = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"
)

func newInstallCertManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert-manager",
		Short: "Install cert-manager",
		Long:  "Install cert-manager for webhook certificates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return installCertManager()
		},
	}

	return cmd
}

func newUninstallCertManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert-manager",
		Short: "Uninstall cert-manager",
		Long:  "Uninstall cert-manager from the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallCertManager()
		},
	}

	return cmd
}

func isCertManagerInstalled() bool {
	clientset, err := getKubeClient()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if cert-manager deployment exists and is available
	deployment, err := clientset.AppsV1().Deployments("cert-manager").Get(ctx, "cert-manager", metav1.GetOptions{})
	if err != nil {
		return false
	}

	// Check if deployment is available
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
			// Also check webhook and cainjector
			webhook, err := clientset.AppsV1().Deployments("cert-manager").Get(ctx, "cert-manager-webhook", metav1.GetOptions{})
			if err != nil {
				return false
			}
			cainjector, err := clientset.AppsV1().Deployments("cert-manager").Get(ctx, "cert-manager-cainjector", metav1.GetOptions{})
			if err != nil {
				return false
			}

			webhookReady := false
			for _, cond := range webhook.Status.Conditions {
				if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
					webhookReady = true
					break
				}
			}

			cainjectorReady := false
			for _, cond := range cainjector.Status.Conditions {
				if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
					cainjectorReady = true
					break
				}
			}

			return webhookReady && cainjectorReady
		}
	}

	return false
}

func installCertManager() error {
	// Check if cert-manager is already installed
	if isCertManagerInstalled() {
		fmt.Println("cert-manager is already installed ✓")
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

	fmt.Printf("Installing cert-manager %s...\n", certManagerVersion)
	certManagerManifestURL := fmt.Sprintf(certManagerURL, certManagerVersion)

	// Download and apply manifest using client-go
	if err := applyManifestFromURL(dynamicClient, config, certManagerManifestURL); err != nil {
		fmt.Printf("Warning: Failed to install cert-manager: %v\n", err)
		fmt.Println("It may already be installed.")
	}

	fmt.Println("Waiting for cert-manager to be ready...")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Wait for cert-manager deployment
	fmt.Println("Waiting for cert-manager deployment...")
	if err := waitForDeployment(ctx, clientset, "cert-manager", "cert-manager"); err != nil {
		fmt.Printf("Warning: cert-manager may not be fully ready: %v\n", err)
	}

	// Wait for cert-manager-webhook deployment
	fmt.Println("Waiting for cert-manager-webhook deployment...")
	if err := waitForDeployment(ctx, clientset, "cert-manager", "cert-manager-webhook"); err != nil {
		fmt.Printf("Warning: cert-manager-webhook may not be fully ready: %v\n", err)
	}

	// Wait for cert-manager-cainjector deployment
	fmt.Println("Waiting for cert-manager-cainjector deployment...")
	if err := waitForDeployment(ctx, clientset, "cert-manager", "cert-manager-cainjector"); err != nil {
		fmt.Printf("Warning: cert-manager-cainjector may not be fully ready: %v\n", err)
	}

	fmt.Println("cert-manager installed ✓")
	return nil
}

func uninstallCertManager() error {
	// Check if cert-manager is installed
	if !isCertManagerInstalled() {
		fmt.Println("cert-manager is not installed")
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

	fmt.Println("Uninstalling cert-manager...")
	certManagerManifestURL := fmt.Sprintf(certManagerURL, certManagerVersion)

	// Delete cert-manager manifest
	if err := deleteResourcesFromManifestURL(dynamicClient, config, certManagerManifestURL); err != nil {
		return fmt.Errorf("failed to delete cert-manager manifest: %w", err)
	}

	// Wait for namespace to be fully deleted
	fmt.Println("Waiting for cert-manager namespace to be deleted...")
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := waitForNamespaceDeleted(ctx, clientset, "cert-manager"); err != nil {
		fmt.Printf("Warning: cert-manager namespace may still be terminating: %v\n", err)
	} else {
		fmt.Println("cert-manager namespace deleted ✓")
	}

	fmt.Println("cert-manager uninstalled ✓")
	return nil
}
