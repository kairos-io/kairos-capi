package main

import (
	"context"
	"fmt"
	"time"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
)

const (
	localPathManifestURL = "https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.28/deploy/local-path-storage.yaml"
	localPathNamespace   = "local-path-storage"
	localPathClassName   = "local-path"
)

func newInstallLocalPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local-path",
		Short: "Install local-path provisioner",
		Long:  "Install local-path provisioner and ensure a default StorageClass exists",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isLocalPathInstalled() {
				fmt.Println("local-path provisioner is already installed ✓")
				return nil
			}
			return installLocalPath()
		},
	}

	return cmd
}

func newUninstallLocalPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local-path",
		Short: "Uninstall local-path provisioner",
		Long:  "Uninstall local-path provisioner",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isLocalPathInstalled() {
				fmt.Println("local-path provisioner is not installed")
				return nil
			}
			return uninstallLocalPath()
		},
	}

	return cmd
}

func newReinstallLocalPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local-path",
		Short: "Reinstall local-path provisioner",
		Long:  "Uninstall and reinstall local-path provisioner",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isLocalPathInstalled() {
				if err := uninstallLocalPath(); err != nil {
					return fmt.Errorf("failed to uninstall local-path: %w", err)
				}
			}
			return installLocalPath()
		},
	}

	return cmd
}

func isLocalPathInstalled() bool {
	clientset, err := getKubeClient()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = clientset.StorageV1().StorageClasses().Get(ctx, localPathClassName, metav1.GetOptions{})
	return err == nil
}

func installLocalPath() error {
	fmt.Println("Installing local-path provisioner...")

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
		return err
	}

	if err := applyManifestFromURL(dynamicClient, config, localPathManifestURL); err != nil {
		return fmt.Errorf("failed to apply local-path manifest: %w", err)
	}

	// Wait for the local-path provisioner deployment
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := waitForDeployment(ctx, clientset, localPathNamespace, "local-path-provisioner"); err != nil {
		fmt.Printf("Warning: local-path provisioner may not be fully ready: %v\n", err)
	}

	// Ensure a default StorageClass is set
	if err := ensureDefaultStorageClass(ctx, clientset); err != nil {
		fmt.Printf("Warning: failed to ensure default StorageClass: %v\n", err)
	} else {
		fmt.Println("✓ Default StorageClass confirmed")
	}

	fmt.Println("local-path provisioner installed ✓")
	return nil
}

func uninstallLocalPath() error {
	fmt.Println("Uninstalling local-path provisioner...")

	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	if err := deleteResourcesFromManifestURL(dynamicClient, config, localPathManifestURL); err != nil {
		return fmt.Errorf("failed to delete local-path resources: %w", err)
	}

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := waitForNamespaceDeleted(ctx, clientset, localPathNamespace); err != nil {
		fmt.Printf("Warning: local-path namespace may still be terminating: %v\n", err)
	}

	fmt.Println("local-path provisioner uninstalled ✓")
	return nil
}

func ensureDefaultStorageClass(ctx context.Context, clientset kubernetes.Interface) error {
	classes, err := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list StorageClasses: %w", err)
	}

	for _, sc := range classes.Items {
		if isDefaultStorageClass(&sc) {
			return nil
		}
	}

	// No default class found; set local-path as default if present
	localPath, err := clientset.StorageV1().StorageClasses().Get(ctx, localPathClassName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("local-path StorageClass not found: %w", err)
	}

	patch := localPath.DeepCopy()
	if patch.Annotations == nil {
		patch.Annotations = map[string]string{}
	}
	patch.Annotations["storageclass.kubernetes.io/is-default-class"] = "true"
	patch.Annotations["storageclass.beta.kubernetes.io/is-default-class"] = "true"

	_, err = clientset.StorageV1().StorageClasses().Update(ctx, patch, metav1.UpdateOptions{})
	return err
}

func isDefaultStorageClass(sc *storagev1.StorageClass) bool {
	if sc.Annotations == nil {
		return false
	}
	if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
		return true
	}
	if sc.Annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true" {
		return true
	}
	return false
}
