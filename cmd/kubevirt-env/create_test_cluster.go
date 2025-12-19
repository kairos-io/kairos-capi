package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newCreateTestClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-test-cluster",
		Short: "Create a kind cluster for testing",
		Long:  "Create a kind cluster with CNI disabled (for Calico installation)",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateKindInstalled()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			clusterName := getClusterName()
			return createTestCluster(clusterName)
		},
	}

	return cmd
}

func validateKindInstalled() error {
	if _, err := exec.LookPath("kind"); err != nil {
		return fmt.Errorf("required command 'kind' not found in PATH. Please install it first")
	}
	return nil
}

func createTestCluster(clusterName string) error {
	// Check if cluster already exists
	kindCmd := exec.Command("kind", "get", "clusters")
	output, err := kindCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check existing clusters: %w", err)
	}

	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range clusters {
		if strings.TrimSpace(line) == clusterName {
			return fmt.Errorf("cluster '%s' already exists. Delete it first with: kind delete cluster --name %s", clusterName, clusterName)
		}
	}

	// Get work directory
	workDir := getWorkDir()
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	// Get Docker config path
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	dockerConfigPath := filepath.Join(usr.HomeDir, ".docker", "config.json")

	// Create empty Docker config if it doesn't exist
	if _, err := os.Stat(dockerConfigPath); os.IsNotExist(err) {
		fmt.Printf("Warning: Docker config file not found at %s\n", dockerConfigPath)
		fmt.Println("Creating empty Docker config to avoid rate limits...")
		if err := os.MkdirAll(filepath.Dir(dockerConfigPath), 0755); err != nil {
			return fmt.Errorf("failed to create .docker directory: %w", err)
		}
		if err := os.WriteFile(dockerConfigPath, []byte("{}"), 0644); err != nil {
			return fmt.Errorf("failed to create Docker config: %w", err)
		}
	}

	// Create kind config file
	kindConfigPath := filepath.Join(workDir, "kind-config.yaml")
	kindConfig := fmt.Sprintf(`kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: %s
networking:
  disableDefaultCNI: true
nodes:
- role: control-plane
  extraMounts:
  - containerPath: /var/lib/kubelet/config.json
    hostPath: %s
`, clusterName, dockerConfigPath)

	if err := os.WriteFile(kindConfigPath, []byte(kindConfig), 0644); err != nil {
		return fmt.Errorf("failed to create kind config: %w", err)
	}

	fmt.Printf("Kind config created with Docker config mount: %s\n", dockerConfigPath)

	// Create cluster
	fmt.Printf("Creating kind cluster '%s'...\n", clusterName)
	kindCmd = exec.Command("kind", "create", "cluster", "--name", clusterName, "--config", kindConfigPath)
	kindCmd.Stdout = os.Stdout
	kindCmd.Stderr = os.Stderr
	if err := kindCmd.Run(); err != nil {
		return fmt.Errorf("failed to create kind cluster: %w", err)
	}

	// Show cluster info
	fmt.Printf("Showing cluster info for context kind-%s...\n", clusterName)
	kubectlCmd := exec.Command("kubectl", "cluster-info", "--context", fmt.Sprintf("kind-%s", clusterName))
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr
	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("failed to show cluster info: %w", err)
	}

	fmt.Println("Kind cluster created ✓")
	fmt.Println("Note: Default CNI is disabled. Install Calico with: make install-calico")

	return nil
}
