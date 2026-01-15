package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

const (
	kairosImageName = "kairos-kubevirt"
	defaultPort     = 18443
)

func newBuildKairosImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build-kairos-image",
		Short: "Build Kairos cloud image",
		Long:  "Build Kairos cloud image using OSArtifact CR (requires osbuilder to be installed)",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate osbuilder is installed
			if !isOsbuilderInstalled() {
				return fmt.Errorf("osbuilder is not installed. Please install it first with: kubevirt-env install osbuilder")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return buildKairosImage()
		},
	}

	return cmd
}

func getKairosImageBuildDir() string {
	return filepath.Join(getWorkDir(), "osbuilder", "build")
}

func newUploadKairosImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload-kairos-image",
		Short: "Upload Kairos image to KubeVirt",
		Long:  "Upload Kairos image to KubeVirt as a DataVolume using virtctl",
		RunE: func(cmd *cobra.Command, args []string) error {
			return uploadKairosImage()
		},
	}

	return cmd
}

func uploadKairosImage() error {
	fmt.Println("=== Uploading Kairos image using virtctl ===")

	// Find image file
	imageFile, err := findKairosImageFile()
	if err != nil {
		return err
	}
	fmt.Printf("Using image file: %s\n", imageFile)

	// Check virtctl
	virtctlPath, err := findVirtctl()
	if err != nil {
		return err
	}
	fmt.Printf("Using virtctl: %s\n", virtctlPath)

	// Check CDI is installed
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, err = clientset.CoreV1().Services("cdi").Get(ctx, "cdi-uploadproxy", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("CDI upload proxy service not found. Make sure CDI is installed: %w", err)
	}

	// Delete existing DataVolume if present
	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	dvGVR := schema.GroupVersionResource{
		Group:    "cdi.kubevirt.io",
		Version:  "v1beta1",
		Resource: "datavolumes",
	}

	_, err = dynamicClient.Resource(dvGVR).Namespace("default").Get(ctx, kairosImageName, metav1.GetOptions{})
	if err == nil {
		fmt.Printf("DataVolume %s already exists. Deleting for fresh upload...\n", kairosImageName)
		err = dynamicClient.Resource(dvGVR).Namespace("default").Delete(ctx, kairosImageName, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("Warning: Failed to delete existing DataVolume: %v\n", err)
		}
		time.Sleep(2 * time.Second)
	}

	// Set up port-forward
	port := defaultPort
	if envPort := os.Getenv("CDI_UPLOAD_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	fmt.Printf("Setting up port-forward on port %d...\n", port)
	kubeconfigPath := getKubeconfigPath()
	kubectlContext := getKubectlContext()

	portForwardCmd := exec.Command("kubectl", "port-forward", "-n", "cdi", "service/cdi-uploadproxy",
		fmt.Sprintf("%d:443", port), "--kubeconfig", kubeconfigPath, "--context", kubectlContext)
	portForwardCmd.Stdout = os.Stdout
	portForwardCmd.Stderr = os.Stderr

	if err := portForwardCmd.Start(); err != nil {
		return fmt.Errorf("failed to start port-forward: %w", err)
	}

	// Set up signal handling to cleanup port-forward
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		portForwardCmd.Process.Kill()
		os.Exit(1)
	}()

	defer func() {
		if portForwardCmd.Process != nil {
			portForwardCmd.Process.Kill()
			portForwardCmd.Wait()
		}
	}()

	// Wait a bit for port-forward to be ready
	time.Sleep(2 * time.Second)

	// Run virtctl upload
	uploadProxyURL := fmt.Sprintf("https://localhost:%d", port)
	fmt.Printf("Upload proxy URL: %s\n", uploadProxyURL)
	fmt.Println("Starting upload with virtctl...")

	virtctlCmd := exec.Command(virtctlPath, "image-upload",
		"dv", kairosImageName,
		"--size=25Gi",
		"--access-mode=ReadWriteOnce",
		"--image-path", imageFile,
		"--uploadproxy-url", uploadProxyURL,
		"--insecure",
		"--force-bind",
		"--wait-secs=300",
		"--kubeconfig", kubeconfigPath,
		"--context", kubectlContext,
	)
	virtctlCmd.Stdout = os.Stdout
	virtctlCmd.Stderr = os.Stderr

	if err := virtctlCmd.Run(); err != nil {
		return fmt.Errorf("virtctl image-upload failed: %w", err)
	}

	fmt.Println("\n✓ Image upload completed successfully!")
	fmt.Printf("DataVolume %s is ready for use.\n", kairosImageName)
	return nil
}

func findKairosImageFile() (string, error) {
	// Check KAIROS_IMAGE_FILE env var
	if envFile := os.Getenv("KAIROS_IMAGE_FILE"); envFile != "" {
		if _, err := os.Stat(envFile); err == nil {
			return envFile, nil
		}
	}

	// Check default location
	defaultFile := filepath.Join(getKairosImageBuildDir(), fmt.Sprintf("%s.raw", kairosImageName))
	if _, err := os.Stat(defaultFile); err == nil {
		return defaultFile, nil
	}

	// Search build directory
	buildDir := getKairosImageBuildDir()
	if entries, err := os.ReadDir(buildDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				name := entry.Name()
				if strings.HasPrefix(name, kairosImageName) {
					ext := filepath.Ext(name)
					if ext == ".raw" || ext == ".qcow2" {
						return filepath.Join(buildDir, name), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("image file not found. Expected: %s or in %s", defaultFile, buildDir)
}

func findVirtctl() (string, error) {
	// Check PATH
	if path, err := exec.LookPath("virtctl"); err == nil {
		return path, nil
	}

	// Check bin directory
	binPath := filepath.Join(".", "bin", "virtctl")
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	return "", fmt.Errorf("virtctl not found in PATH or ./bin/virtctl. Please install virtctl first")
}

func buildKairosImage() error {
	fmt.Println("Building Kairos cloud image using OSArtifact CR...")
	fmt.Println("Note: osbuilder controller will create a Job to build the image.")
	fmt.Println("The built image will be served via nginx service.")

	workDir := getWorkDir()
	buildDir := getKairosImageBuildDir()

	// Create directories
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
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

	// Create cloud-config Secret
	if err := createCloudConfigSecret(clientset); err != nil {
		return fmt.Errorf("failed to create cloud-config secret: %w", err)
	}

	// Create OSArtifact CR
	if err := createOSArtifactCR(dynamicClient, config); err != nil {
		return fmt.Errorf("failed to create OSArtifact CR: %w", err)
	}

	// Wait for OSArtifact to be ready
	if err := waitForOSArtifactReady(dynamicClient); err != nil {
		return fmt.Errorf("failed to wait for OSArtifact: %w", err)
	}

	// Download built image from nginx
	if err := downloadImageFromNginx(clientset, buildDir); err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	fmt.Println("Kairos image build complete ✓")
	return nil
}

func createCloudConfigSecret(clientset kubernetes.Interface) error {
	fmt.Println("Creating cloud-config Secret with console parameters...")

	cloudConfig := `#cloud-config

# Add console parameters to kernel cmdline for serial console access
# console=ttyS0 enables serial console, console=tty0 enables VGA console
install:
  grub_options:
    extra_cmdline: "console=ttyS0 console=tty0"
`

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-cloud-config", kairosImageName),
			Namespace: "default",
		},
		Data: map[string][]byte{
			"cloud_config.yaml": []byte(cloudConfig),
		},
	}

	ctx := context.Background()
	_, err := clientset.CoreV1().Secrets("default").Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		// Try update if it already exists
		_, err = clientset.CoreV1().Secrets("default").Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create/update secret: %w", err)
		}
	}

	return nil
}

func createOSArtifactCR(dynamicClient dynamic.Interface, config *rest.Config) error {
	fmt.Println("Creating OSArtifact CustomResource...")

	osartifactYAML := fmt.Sprintf(`apiVersion: build.kairos.io/v1alpha2
kind: OSArtifact
metadata:
  name: %s
  namespace: default
spec:
  imageName: "quay.io/kairos/fedora:40-core-amd64-generic-v3.6.1-beta2"
  cloudImage: true
  diskSize: "32000"
  cloudConfigRef:
    name: %s-cloud-config
    key: cloud_config.yaml
  exporters:
  - template:
      spec:
        restartPolicy: Never
        containers:
        - name: upload
          image: quay.io/curl/curl
          command:
          - /bin/sh
          args:
          - -c
          - |
              for f in $(ls /artifacts)
              do
              curl -T /artifacts/$f http://osartifactbuilder-operator-osbuilder-nginx/upload/$f
              done
          volumeMounts:
          - name: artifacts
            mountPath: /artifacts
`, kairosImageName, kairosImageName)

	// Apply YAML content directly using dynamic client
	if err := applyManifestContent(dynamicClient, config, []byte(osartifactYAML)); err != nil {
		return fmt.Errorf("failed to apply OSArtifact: %w", err)
	}

	return nil
}

func waitForOSArtifactReady(dynamicClient dynamic.Interface) error {
	fmt.Println("Waiting for OSArtifact to be ready...")
	ctx, cancel := context.WithTimeout(context.Background(), 1800*time.Second)
	defer cancel()

	osartifactGVR := schema.GroupVersionResource{
		Group:    "build.kairos.io",
		Version:  "v1alpha2",
		Resource: "osartifacts",
	}

	return wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		osartifact, err := dynamicClient.Resource(osartifactGVR).Namespace("default").Get(ctx, kairosImageName, metav1.GetOptions{})
		if err != nil {
			fmt.Print(".")
			return false, nil
		}

		phase, found, err := unstructured.NestedString(osartifact.Object, "status", "phase")
		if !found || err != nil {
			fmt.Print(".")
			return false, nil
		}

		if phase == "Ready" {
			fmt.Printf("\n✓ OSArtifact is ready (phase: %s)\n", phase)
			return true, nil
		}

		if phase == "Error" {
			fmt.Println("\n✗ OSArtifact build failed. Check logs:")
			// Print the full object for debugging
			if objBytes, err := osartifact.MarshalJSON(); err == nil {
				fmt.Println(string(objBytes))
			}
			return false, fmt.Errorf("OSArtifact build failed with phase: %s", phase)
		}

		fmt.Print(".")
		return false, nil
	})
}

func downloadImageFromNginx(clientset kubernetes.Interface, buildDir string) error {
	fmt.Println("Downloading built image from nginx...")

	ctx := context.Background()

	// Find nginx service
	services, err := clientset.CoreV1().Services("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	var nginxService *corev1.Service
	for _, svc := range services.Items {
		if svc.Spec.Type == corev1.ServiceTypeNodePort {
			// Check if it's an nginx service
			if svc.Name == "osartifactbuilder-operator-osbuilder-nginx" || strings.Contains(strings.ToLower(svc.Name), "nginx") {
				nginxService = &svc
				break
			}
		}
	}

	if nginxService == nil {
		return fmt.Errorf("could not find nginx service")
	}

	if len(nginxService.Spec.Ports) == 0 {
		return fmt.Errorf("nginx service has no ports")
	}

	nodePort := nginxService.Spec.Ports[0].NodePort
	if nodePort == 0 {
		return fmt.Errorf("nginx service nodePort is not set")
	}

	// Get node IP
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil || len(nodes.Items) == 0 {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	var nodeIP string
	for _, addr := range nodes.Items[0].Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			nodeIP = addr.Address
			break
		}
	}

	if nodeIP == "" {
		return fmt.Errorf("could not determine node IP")
	}

	// Download image
	imageFilename := fmt.Sprintf("%s.raw", kairosImageName)
	nginxURL := fmt.Sprintf("http://%s:%d/%s", nodeIP, nodePort, imageFilename)
	outputFile := filepath.Join(buildDir, imageFilename)

	fmt.Printf("Downloading %s from %s\n", imageFilename, nginxURL)

	resp, err := http.Get(nginxURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	fmt.Printf("Downloaded to: %s\n", outputFile)

	// Check for built image
	fmt.Println("Checking for built image...")
	matches, err := filepath.Glob(filepath.Join(buildDir, fmt.Sprintf("%s*", kairosImageName)))
	if err == nil && len(matches) > 0 {
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && !info.IsDir() {
				fmt.Printf("Found: %s\n", match)
			}
		}
	} else {
		fmt.Println("No image files found.")
	}

	return nil
}
