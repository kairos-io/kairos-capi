package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"

	"github.com/spf13/cobra"
)

const (
	kairosHelmRepo = "https://kairos-io.github.io/helm-charts"
)

func newInstallOsbuilderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osbuilder",
		Short: "Install osbuilder",
		Long:  "Install osbuilder using Helm charts (includes CRDs)",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateHelmInstalled()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return installOsbuilder()
		},
	}

	return cmd
}

func newUninstallOsbuilderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osbuilder",
		Short: "Uninstall osbuilder",
		Long:  "Uninstall osbuilder from the cluster",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateHelmInstalled()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallOsbuilder()
		},
	}

	return cmd
}

func newReinstallOsbuilderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osbuilder",
		Short: "Reinstall osbuilder",
		Long:  "Uninstall and reinstall osbuilder",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := uninstallOsbuilder(); err != nil {
				return fmt.Errorf("failed to uninstall osbuilder: %w", err)
			}
			return installOsbuilder()
		},
	}
	return cmd
}

func validateHelmInstalled() error {
	if _, err := exec.LookPath("helm"); err != nil {
		return fmt.Errorf("required command 'helm' not found in PATH. Please install it first")
	}
	return nil
}

func isOsbuilderInstalled() bool {
	clientset, err := getKubeClient()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if osbuilder deployment exists and is available
	deployment, err := clientset.AppsV1().Deployments("default").Get(ctx, "osbuilder", metav1.GetOptions{})
	if err != nil {
		return false
	}

		// Check if deployment is available
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
			// Also check if OSArtifact CRD exists
			config, err := getKubeConfig()
			if err != nil {
				return false
			}
			apiextensionsClient, err := apiextensionsv1.NewForConfig(config)
			if err != nil {
				return false
			}
			_, err = apiextensionsClient.CustomResourceDefinitions().Get(ctx, "osartifacts.build.kairos.io", metav1.GetOptions{})
			return err == nil
		}
	}

	return false
}

func installOsbuilder() error {
	// Check if osbuilder is already installed
	if isOsbuilderInstalled() {
		fmt.Println("osbuilder is already installed ✓")
		return nil
	}

	// Install osbuilder CRDs first
	if err := installOsbuilderCRDs(); err != nil {
		return fmt.Errorf("failed to install osbuilder CRDs: %w", err)
	}

	// Install osbuilder
	if err := installOsbuilderDeployment(); err != nil {
		return fmt.Errorf("failed to install osbuilder: %w", err)
	}

	fmt.Println("osbuilder installed ✓")
	return nil
}

func installOsbuilderCRDs() error {
	fmt.Println("Installing osbuilder CRDs from Helm chart...")

	// Add kairos helm repo
	repoAddCmd := exec.Command("helm", "repo", "add", "kairos", kairosHelmRepo)
	repoAddCmd.Stderr = os.Stderr
	if err := repoAddCmd.Run(); err != nil {
		// Try with --force-update if it already exists
		repoAddCmd = exec.Command("helm", "repo", "add", "kairos", kairosHelmRepo, "--force-update")
		repoAddCmd.Stderr = os.Stderr
		if err := repoAddCmd.Run(); err != nil {
			return fmt.Errorf("failed to add kairos helm repo: %w", err)
		}
	}

	// Update helm repo
	repoUpdateCmd := exec.Command("helm", "repo", "update", "kairos")
	repoUpdateCmd.Stdout = os.Stdout
	repoUpdateCmd.Stderr = os.Stderr
	if err := repoUpdateCmd.Run(); err != nil {
		return fmt.Errorf("failed to update helm repo: %w", err)
	}

	// Install kairos-crds chart
	installCmd := exec.Command("helm", "upgrade", "--install", "kairos-crds", "kairos/kairos-crds", "--namespace", "default", "--create-namespace", "--wait", "--timeout=60s")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install/upgrade osbuilder CRDs: %w", err)
	}

	// Wait for OSArtifact CRD to be established
	fmt.Println("Waiting for OSArtifact CRD to be ready...")
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := waitForCRDEstablished(ctx, clientset, "osartifacts.build.kairos.io"); err != nil {
		return fmt.Errorf("OSArtifact CRD not established: %w", err)
	}

	fmt.Println("osbuilder CRDs installed ✓")
	return nil
}

func installOsbuilderDeployment() error {
	fmt.Println("Installing osbuilder using Helm charts...")

	// Add kairos helm repo (in case it wasn't added earlier)
	repoAddCmd := exec.Command("helm", "repo", "add", "kairos", kairosHelmRepo)
	repoAddCmd.Stderr = os.Stderr
	repoAddCmd.Run() // Ignore error if repo already exists

	// Update helm repo
	repoUpdateCmd := exec.Command("helm", "repo", "update", "kairos")
	repoUpdateCmd.Stdout = os.Stdout
	repoUpdateCmd.Stderr = os.Stderr
	if err := repoUpdateCmd.Run(); err != nil {
		return fmt.Errorf("failed to update helm repo: %w", err)
	}

	// Install osbuilder chart
	installCmd := exec.Command("helm", "upgrade", "--install", "osbuilder", "kairos/osbuilder", "-n", "default", "--create-namespace")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install/upgrade osbuilder: %w", err)
	}

	// Wait for osbuilder deployment
	fmt.Println("Waiting for osbuilder to be ready...")
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	if err := waitForDeployment(ctx, clientset, "default", "osbuilder"); err != nil {
		fmt.Printf("Warning: osbuilder deployment may still be starting: %v\n", err)
		fmt.Println("Check with: kubectl get pods -n default -l app.kubernetes.io/name=osbuilder")
	}

	return nil
}

func uninstallOsbuilder() error {
	// Check if osbuilder is installed
	if !isOsbuilderInstalled() {
		fmt.Println("osbuilder is not installed")
		return nil
	}

	fmt.Println("Uninstalling osbuilder...")

	// Uninstall osbuilder chart
	uninstallCmd := exec.Command("helm", "uninstall", "osbuilder", "-n", "default")
	uninstallCmd.Stdout = os.Stdout
	uninstallCmd.Stderr = os.Stderr
	if err := uninstallCmd.Run(); err != nil {
		fmt.Printf("Warning: failed to uninstall osbuilder chart: %v\n", err)
	}

	// Uninstall kairos-crds chart
	uninstallCRDsCmd := exec.Command("helm", "uninstall", "kairos-crds", "-n", "default")
	uninstallCRDsCmd.Stdout = os.Stdout
	uninstallCRDsCmd.Stderr = os.Stderr
	if err := uninstallCRDsCmd.Run(); err != nil {
		fmt.Printf("Warning: failed to uninstall kairos-crds chart: %v\n", err)
	}

	fmt.Println("osbuilder uninstalled ✓")
	return nil
}

func waitForCRDEstablished(ctx context.Context, clientset kubernetes.Interface, crdName string) error {
	config, err := getKubeConfig()
	if err != nil {
		return err
	}
	apiextensionsClient, err := apiextensionsv1.NewForConfig(config)
	if err != nil {
		return err
	}

	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		crd, err := apiextensionsClient.CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		for _, condition := range crd.Status.Conditions {
			if condition.Type == apiextensionsv1api.Established && condition.Status == apiextensionsv1api.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}
