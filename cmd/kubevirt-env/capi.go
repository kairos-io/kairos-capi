package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"
)

const (
	capiVersion = "v1.8.0"
)

func newInstallCapiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capi",
		Short: "Install Cluster API (CAPI)",
		Long:  "Install Cluster API core components on the kind cluster",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateClusterctlInstalled()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return installCapi()
		},
	}

	return cmd
}

func newUninstallCapiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capi",
		Short: "Uninstall Cluster API",
		Long:  "Uninstall Cluster API from the cluster",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateClusterctlInstalled()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallCapi()
		},
	}

	return cmd
}

func newReinstallCapiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capi",
		Short: "Reinstall Cluster API",
		Long:  "Uninstall and reinstall Cluster API",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := uninstallCapi(); err != nil {
				return fmt.Errorf("failed to uninstall CAPI: %w", err)
			}
			return installCapi()
		},
	}
	return cmd
}

func validateClusterctlInstalled() error {
	if _, err := exec.LookPath("clusterctl"); err != nil {
		return fmt.Errorf("required command 'clusterctl' not found in PATH. Please install it first")
	}
	return nil
}

func isCapiInstalled() bool {
	clientset, err := getKubeClient()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if CAPI core controller exists and is available
	deployment, err := clientset.AppsV1().Deployments("capi-system").Get(ctx, "capi-controller-manager", metav1.GetOptions{})
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

func installCapi() error {
	// Check if CAPI is already installed
	if isCapiInstalled() {
		fmt.Println("Cluster API (CAPI) is already installed ✓")
		return nil
	}

	fmt.Printf("Installing Cluster API %s...\n", capiVersion)

	// Get bin directory
	binDir := filepath.Join(".", "bin")

	// Set PATH to include bin directory
	path := os.Getenv("PATH")
	if path != "" {
		path = binDir + string(filepath.ListSeparator) + path
	} else {
		path = binDir
	}

	// Run clusterctl init without infrastructure provider (just CAPI core)
	// clusterctl init by itself installs CAPI core components
	clusterctlCmd := exec.Command("clusterctl", "init")
	clusterctlCmd.Env = append(os.Environ(), "PATH="+path)
	clusterctlCmd.Stdout = os.Stdout
	clusterctlCmd.Stderr = os.Stderr

	if err := clusterctlCmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize CAPI: %w", err)
	}

	fmt.Println("Waiting for CAPI components to be ready...")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	// Wait for CAPI core controller
	fmt.Println("Waiting for CAPI core controller...")
	if err := waitForDeployment(ctx, clientset, "capi-system", "capi-controller-manager"); err != nil {
		fmt.Printf("Warning: CAPI core controller may not be fully ready: %v\n", err)
	}

	fmt.Println("Cluster API (CAPI) installed ✓")
	return nil
}

func uninstallCapi() error {
	// Check if CAPI is installed
	if !isCapiInstalled() {
		fmt.Println("Cluster API (CAPI) is not installed")
		return nil
	}

	fmt.Println("Uninstalling Cluster API...")

	// Get bin directory
	binDir := filepath.Join(".", "bin")

	// Set PATH to include bin directory
	path := os.Getenv("PATH")
	if path != "" {
		path = binDir + string(filepath.ListSeparator) + path
	} else {
		path = binDir
	}

	// Run clusterctl delete --all
	clusterctlCmd := exec.Command("clusterctl", "delete", "--all")
	clusterctlCmd.Env = append(os.Environ(), "PATH="+path)
	clusterctlCmd.Stdout = os.Stdout
	clusterctlCmd.Stderr = os.Stderr

	if err := clusterctlCmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall CAPI: %w", err)
	}

	fmt.Println("Cluster API (CAPI) uninstalled ✓")
	return nil
}
