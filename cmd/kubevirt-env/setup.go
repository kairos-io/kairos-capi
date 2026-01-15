package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Complete setup: create cluster and install all components",
		Long:  "Create a kind cluster and install all required components (local-path, Calico, CDI, KubeVirt, CAPI, CAPK, osbuilder, cert-manager, Kairos provider) and build/upload the Kairos image",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup()
		},
	}

	return cmd
}

func newCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up everything including the kind cluster",
		Long:  "Delete the kind cluster and clean up work directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCleanup()
		},
	}

	return cmd
}

func runSetup() error {
	clusterName := getClusterName()

	fmt.Println("=== Starting complete setup ===")
	fmt.Printf("Cluster name: %s\n", clusterName)
	fmt.Println()

	// 1. Create test cluster
	fmt.Println("[1/12] Creating kind cluster...")
	if err := createTestCluster(clusterName); err != nil {
		return fmt.Errorf("failed to create test cluster: %w", err)
	}
	fmt.Println()

	// 2. Install Calico
	fmt.Println("[2/12] Installing local-path provisioner...")
	if err := installLocalPath(); err != nil {
		return fmt.Errorf("failed to install local-path provisioner: %w", err)
	}
	fmt.Println()

	// 3. Install Calico
	fmt.Println("[3/12] Installing Calico CNI...")
	if err := installCalico(); err != nil {
		return fmt.Errorf("failed to install Calico: %w", err)
	}
	fmt.Println()

	// 4. Install CDI (required for KubeVirt)
	fmt.Println("[4/12] Installing CDI...")
	if err := installCdi(); err != nil {
		return fmt.Errorf("failed to install CDI: %w", err)
	}
	fmt.Println()

	// 5. Install KubeVirt
	fmt.Println("[5/12] Installing KubeVirt...")
	if err := installKubevirt(); err != nil {
		return fmt.Errorf("failed to install KubeVirt: %w", err)
	}
	fmt.Println()

	// 6. Install CAPI
	fmt.Println("[6/12] Installing Cluster API (CAPI)...")
	if err := installCapi(); err != nil {
		return fmt.Errorf("failed to install CAPI: %w", err)
	}
	fmt.Println()

	// 7. Install CAPK
	fmt.Println("[7/12] Installing CAPK...")
	if err := installCapk(); err != nil {
		return fmt.Errorf("failed to install CAPK: %w", err)
	}
	fmt.Println()

	// 8. Install osbuilder (includes CRDs)
	fmt.Println("[8/12] Installing osbuilder...")
	if err := installOsbuilder(); err != nil {
		return fmt.Errorf("failed to install osbuilder: %w", err)
	}
	fmt.Println()

	// 9. Build Kairos image
	fmt.Println("[9/12] Building Kairos image...")
	if err := buildKairosImage(); err != nil {
		return fmt.Errorf("failed to build Kairos image: %w", err)
	}
	fmt.Println()

	// 10. Upload Kairos image
	fmt.Println("[10/12] Uploading Kairos image...")
	if err := uploadKairosImage(); err != nil {
		return fmt.Errorf("failed to upload Kairos image: %w", err)
	}
	fmt.Println()

	// 11. Install cert-manager (required for Kairos provider)
	fmt.Println("[11/12] Installing cert-manager...")
	if err := installCertManager(); err != nil {
		return fmt.Errorf("failed to install cert-manager: %w", err)
	}
	fmt.Println()

	// 12. Install Kairos provider
	fmt.Println("[12/12] Installing Kairos CAPI Provider...")
	if err := installKairosProvider(); err != nil {
		return fmt.Errorf("failed to install Kairos provider: %w", err)
	}
	fmt.Println()

	fmt.Println("=== Setup complete ===")
	fmt.Println("You can now create a test cluster with: kubevirt-env test-control-plane")
	return nil
}

func runCleanup() error {
	clusterName := getClusterName()

	fmt.Println("=== Cleaning up ===")
	fmt.Printf("Cluster name: %s\n", clusterName)
	fmt.Println()

	// Delete kind cluster
	fmt.Println("Deleting kind cluster...")
	kindCmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
	kindCmd.Stdout = os.Stdout
	kindCmd.Stderr = os.Stderr
	if err := kindCmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to delete kind cluster: %v\n", err)
	} else {
		fmt.Println("Kind cluster deleted ✓")
	}
	fmt.Println()

	// Clean up work directory
	fmt.Println("Cleaning up work directories...")
	workDir := getWorkDir()
	if err := os.RemoveAll(workDir); err != nil {
		fmt.Printf("Warning: Failed to remove work directory %s: %v\n", workDir, err)
	} else {
		fmt.Printf("Work directory %s removed ✓\n", workDir)
	}

	fmt.Println("Cleanup complete ✓")
	return nil
}
