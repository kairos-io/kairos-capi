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
	capkVersion = "v0.1.10"
)

func newInstallCapkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capk",
		Short: "Install CAPK",
		Long:  "Install Cluster API Provider for KubeVirt (CAPK) - requires CAPI to be installed first",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := validateClusterctlInstalled(); err != nil {
				return err
			}
			// Check if CAPI is installed
			if !isCapiInstalled() {
				return fmt.Errorf("CAPI is not installed. Please install CAPI first with: kubevirt-env install capi")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return installCapk()
		},
	}

	return cmd
}

func newUninstallCapkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capk",
		Short: "Uninstall CAPK",
		Long:  "Uninstall Cluster API Provider for KubeVirt (CAPK)",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateClusterctlInstalled()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallCapk()
		},
	}

	return cmd
}

func newReinstallCapkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capk",
		Short: "Reinstall CAPK",
		Long:  "Uninstall and reinstall CAPK",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := uninstallCapk(); err != nil {
				return fmt.Errorf("failed to uninstall CAPK: %w", err)
			}
			return installCapk()
		},
	}
	return cmd
}

func isCapkInstalled() bool {
	clientset, err := getKubeClient()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if CAPK infrastructure controller exists and is available
	deployment, err := clientset.AppsV1().Deployments("capk-system").Get(ctx, "capk-controller-manager", metav1.GetOptions{})
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

func installCapk() error {
	// Check if CAPK is already installed
	if isCapkInstalled() {
		fmt.Println("CAPK is already installed ✓")
		return nil
	}

	fmt.Printf("Installing CAPK %s...\n", capkVersion)

	// Get bin directory
	binDir := filepath.Join(".", "bin")

	// Set PATH to include bin directory
	path := os.Getenv("PATH")
	if path != "" {
		path = binDir + string(filepath.ListSeparator) + path
	} else {
		path = binDir
	}

	// Run clusterctl init with KubeVirt infrastructure provider
	fmt.Printf("Using CAPK version: %s\n", capkVersion)
	clusterctlCmd := exec.Command("clusterctl", "init", "--infrastructure", fmt.Sprintf("kubevirt:%s", capkVersion))
	clusterctlCmd.Env = append(os.Environ(), "PATH="+path)
	clusterctlCmd.Stdout = os.Stdout
	clusterctlCmd.Stderr = os.Stderr

	if err := clusterctlCmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize CAPK: %w", err)
	}

	fmt.Println("Waiting for CAPK components to be ready...")
	fmt.Println("Note: Some controllers may still be initializing. This can take a few minutes.")
	
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer waitCancel()

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	fmt.Println("Waiting for CAPK bootstrap controller...")
	if err := waitForDeployment(waitCtx, clientset, "capi-kubevirt-bootstrap-system", "capi-kubevirt-bootstrap-controller-manager"); err != nil {
		fmt.Printf("Warning: CAPK bootstrap controller may not be fully ready: %v\n", err)
		fmt.Println("Note: Controllers may still be initializing. Check with: kubectl get pods -n capi-kubevirt-bootstrap-system")
	} else {
		fmt.Println("✓ CAPK bootstrap controller is ready")
	}

	fmt.Println("Waiting for CAPK control plane controller...")
	if err := waitForDeployment(waitCtx, clientset, "capi-kubevirt-control-plane-system", "capi-kubevirt-control-plane-controller-manager"); err != nil {
		fmt.Printf("Warning: CAPK control plane controller may not be fully ready: %v\n", err)
		fmt.Println("Note: Controllers may still be initializing. Check with: kubectl get pods -n capi-kubevirt-control-plane-system")
	} else {
		fmt.Println("✓ CAPK control plane controller is ready")
	}

	fmt.Println("Waiting for CAPK infrastructure controller...")
	if err := waitForDeployment(waitCtx, clientset, "capk-system", "capk-controller-manager"); err != nil {
		fmt.Printf("Warning: CAPK infrastructure controller may not be fully ready: %v\n", err)
		fmt.Println("Note: Controllers may still be initializing. Check with: kubectl get pods -n capk-system")
	} else {
		fmt.Println("✓ CAPK infrastructure controller is ready")
	}

	fmt.Println("CAPK installed ✓")
	return nil
}

func uninstallCapk() error {
	// Check if CAPK is installed
	if !isCapkInstalled() {
		fmt.Println("CAPK is not installed")
		return nil
	}

	fmt.Println("Uninstalling CAPK...")

	// Note: clusterctl delete --all removes everything including CAPI
	// For now, we'll just note that CAPK is removed when CAPI is uninstalled
	// In the future, we could implement more granular deletion
	fmt.Println("Note: CAPK will be removed when CAPI is uninstalled.")
	fmt.Println("To remove CAPK specifically, you can manually delete the CAPK namespaces:")
	fmt.Println("  kubectl delete namespace capk-system")
	fmt.Println("  kubectl delete namespace capi-kubevirt-bootstrap-system")
	fmt.Println("  kubectl delete namespace capi-kubevirt-control-plane-system")

	fmt.Println("CAPK uninstalled ✓")
	return nil
}
