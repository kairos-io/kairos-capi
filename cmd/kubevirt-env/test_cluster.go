package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"

	"github.com/spf13/cobra"
)

const (
	sampleClusterFile = "config/samples/capk/kairos_cluster_k0s_single_node.yaml"
	clusterName       = "kairos-cluster"
	clusterNamespace  = "default"
)

func newTestControlPlaneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test-control-plane",
		Short: "Create a test cluster and verify machines are created",
		Long:  "Create a test cluster manifest, apply it, and verify machines are being created",
		RunE: func(cmd *cobra.Command, args []string) error {
			return testControlPlane()
		},
	}

	return cmd
}

func newTestClusterStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test-cluster-status",
		Short: "Show status of the test cluster",
		Long:  "Show status of the test cluster including cluster, machines, VMs, and pods",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showTestClusterStatus()
		},
	}

	return cmd
}

func newDeleteTestClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-test-cluster",
		Short: "Delete the test cluster",
		Long:  "Delete the test cluster and wait for resources to be cleaned up",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteTestCluster()
		},
	}

	return cmd
}

func testControlPlane() error {
	// Create sample cluster manifest
	if err := createSampleCluster(); err != nil {
		return fmt.Errorf("failed to create sample cluster manifest: %w", err)
	}

	// Verify CAPK CRDs are established
	if err := verifyCAPKCRDs(); err != nil {
		return fmt.Errorf("failed to verify CAPK CRDs: %w", err)
	}

	// Delete existing immutable resources
	if err := deleteExistingMachineTemplate(); err != nil {
		return fmt.Errorf("failed to delete existing machine template: %w", err)
	}

	// Apply cluster manifest
	fmt.Println("Creating test cluster...")
	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	if err := applyManifestFromFile(dynamicClient, config, sampleClusterFile); err != nil {
		return fmt.Errorf("failed to apply cluster manifest: %w", err)
	}

	// Wait for cluster to be provisioned
	fmt.Println("Waiting for cluster to be provisioned...")
	if err := waitForClusterProvisioned(); err != nil {
		fmt.Printf("Warning: Cluster provisioning may not be complete: %v\n", err)
	}

	// Show status
	return showTestClusterStatus()
}

func createSampleCluster() error {
	fmt.Println("Creating sample cluster manifest...")
	manifestDir := filepath.Dir(sampleClusterFile)
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	// Read the script and execute it
	scriptPath := filepath.Join("hack", "create-sample-cluster.sh")
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run create-sample-cluster script: %w", err)
	}

	return nil
}

func verifyCAPKCRDs() error {
	fmt.Println("Verifying CAPK CRDs are installed and established...")

	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}

	crds := []string{
		"kubevirtclusters.infrastructure.cluster.x-k8s.io",
		"kubevirtmachinetemplates.infrastructure.cluster.x-k8s.io",
		"kubevirtmachines.infrastructure.cluster.x-k8s.io",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, crdName := range crds {
		crd, err := dynamicClient.Resource(crdGVR).Get(ctx, crdName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("CRD %s not found. Make sure CAPI is installed: %w", crdName, err)
		}

		conditions, found, err := unstructured.NestedSlice(crd.Object, "status", "conditions")
		if found && err == nil {
			established := false
			for _, cond := range conditions {
				if condMap, ok := cond.(map[string]interface{}); ok {
					if condType, ok := condMap["type"].(string); ok && condType == "Established" {
						if condStatus, ok := condMap["status"].(string); ok && condStatus == "True" {
							established = true
							break
						}
					}
				}
			}
			if !established {
				fmt.Printf("Warning: CRD %s is not established yet. Waiting...\n", crdName)
				// Wait for it to be established using the existing function
				clientset, err := getKubeClient()
				if err != nil {
					return fmt.Errorf("failed to get kube client: %w", err)
				}
				if err := waitForCRDEstablished(ctx, clientset, crdName); err != nil {
					return fmt.Errorf("CRD %s failed to become established: %w", crdName, err)
				}
			}
		}
	}

	fmt.Println("✓ All CAPK CRDs are established")
	return nil
}

func deleteExistingMachineTemplate() error {
	fmt.Println("Deleting existing immutable resources if they exist...")
	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	templateGVR := schema.GroupVersionResource{
		Group:    "infrastructure.cluster.x-k8s.io",
		Version:  "v1alpha1",
		Resource: "kubevirtmachinetemplates",
	}

	ctx := context.Background()
	err = dynamicClient.Resource(templateGVR).Namespace(clusterNamespace).Delete(ctx, "kairos-control-plane-template", metav1.DeleteOptions{})
	if err != nil {
		// Ignore not found errors
		return nil
	}

	return nil
}

func waitForClusterProvisioned() error {
	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	clusterGVR := schema.GroupVersionResource{
		Group:    "cluster.x-k8s.io",
		Version:  "v1beta2",
		Resource: "clusters",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	return wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		cluster, err := dynamicClient.Resource(clusterGVR).Namespace(clusterNamespace).Get(ctx, clusterName, metav1.GetOptions{})
		if err != nil {
			fmt.Print(".")
			return false, nil
		}

		phase, found, err := unstructured.NestedString(cluster.Object, "status", "phase")
		if !found || err != nil {
			fmt.Print(".")
			return false, nil
		}

		if phase == "Provisioned" {
			fmt.Println("\n✓ Cluster is provisioned")
			return true, nil
		}

		fmt.Print(".")
		return false, nil
	})
}

func showTestClusterStatus() error {
	kubeconfigPath := getKubeconfigPath()
	kubectlContext := getKubectlContext()

	fmt.Println("\n=== Cluster Status ===")
	cmd := exec.Command("kubectl", "get", "cluster", clusterName, "-n", clusterNamespace,
		"--kubeconfig", kubeconfigPath, "--context", kubectlContext)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	fmt.Println("\n=== Control Plane Status ===")
	cmd = exec.Command("kubectl", "get", "kairoscontrolplane", "kairos-control-plane", "-n", clusterNamespace,
		"--kubeconfig", kubeconfigPath, "--context", kubectlContext)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	fmt.Println("\n=== Machine Status ===")
	cmd = exec.Command("kubectl", "get", "machines", "-n", clusterNamespace,
		"-l", fmt.Sprintf("cluster.x-k8s.io/cluster-name=%s", clusterName),
		"--kubeconfig", kubeconfigPath, "--context", kubectlContext)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	fmt.Println("\n=== KubeVirt VM Status ===")
	cmd = exec.Command("kubectl", "get", "vms", "-n", clusterNamespace,
		"-l", fmt.Sprintf("cluster.x-k8s.io/cluster-name=%s", clusterName),
		"--kubeconfig", kubeconfigPath, "--context", kubectlContext)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	fmt.Println("\n=== Pods Status ===")
	cmd = exec.Command("kubectl", "get", "pods", "-n", clusterNamespace,
		"-l", fmt.Sprintf("cluster.x-k8s.io/cluster-name=%s", clusterName),
		"--kubeconfig", kubeconfigPath, "--context", kubectlContext)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	fmt.Println("\nTest cluster created. Check status above to verify machines are being created.")
	return nil
}

func deleteTestCluster() error {
	fmt.Println("Deleting test cluster...")
	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	if err := deleteResourcesFromManifestFile(dynamicClient, config, sampleClusterFile); err != nil {
		return fmt.Errorf("failed to delete cluster manifest: %w", err)
	}

	fmt.Println("Waiting for resources to be deleted...")
	deleteConfig, err := getKubeConfig()
	if err != nil {
		return err
	}

	deleteDynamicClient, err := dynamic.NewForConfig(deleteConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	clusterGVR := schema.GroupVersionResource{
		Group:    "cluster.x-k8s.io",
		Version:  "v1beta2",
		Resource: "clusters",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(checkCtx context.Context) (bool, error) {
		_, err := deleteDynamicClient.Resource(clusterGVR).Namespace(clusterNamespace).Get(checkCtx, clusterName, metav1.GetOptions{})
		if err != nil {
			fmt.Println("\n✓ Cluster deleted")
			return true, nil
		}
		fmt.Print(".")
		return false, nil
	})
}
