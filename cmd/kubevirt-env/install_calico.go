package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yamlserializer "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	"github.com/spf13/cobra"
)

const (
	calicoVersion    = "v3.29.1"
	calicoManifestURL = "https://raw.githubusercontent.com/projectcalico/calico/%s/manifests/calico.yaml"
)

func newInstallCalicoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-calico",
		Short: "Install Calico CNI",
		Long:  "Install Calico CNI (required for LoadBalancer support in KubeVirt)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return installCalico()
		},
	}

	return cmd
}

func installCalico() error {
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

	fmt.Printf("Installing Calico CNI %s...\n", calicoVersion)
	calicoURL := fmt.Sprintf(calicoManifestURL, calicoVersion)

	// Download and apply manifest using client-go
	if err := applyManifestFromURL(dynamicClient, config, calicoURL); err != nil {
		return fmt.Errorf("failed to apply Calico manifest: %w", err)
	}

	fmt.Println("Waiting for Calico to be ready...")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Wait for calico-kube-controllers deployment using client-go
	fmt.Println("Waiting for Calico kube-controllers deployment...")
	if err := waitForDeployment(ctx, clientset, "kube-system", "calico-kube-controllers"); err != nil {
		fmt.Printf("Warning: Calico kube-controllers may not be fully ready: %v\n", err)
	}

	// Wait for calico-node daemonset using client-go
	fmt.Println("Waiting for Calico node daemonset pods to be ready...")
	if err := waitForDaemonset(ctx, clientset, "kube-system", "calico-node"); err != nil {
		fmt.Printf("Warning: Calico node daemonset may not be fully ready: %v\n", err)
		// Show daemonset status
		ds, err := clientset.AppsV1().DaemonSets("kube-system").Get(ctx, "calico-node", metav1.GetOptions{})
		if err == nil {
			fmt.Printf("Daemonset status: %d/%d pods ready\n", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
		}
	}

	fmt.Println("Calico CNI installed ✓")
	return nil
}

func waitForDeployment(ctx context.Context, clientset kubernetes.Interface, namespace, name string) error {
	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}

func waitForDaemonset(ctx context.Context, clientset kubernetes.Interface, namespace, name string) error {
	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled && ds.Status.DesiredNumberScheduled > 0 {
			fmt.Printf("\n✓ Calico node daemonset is ready (%d/%d pods)\n", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
			return true, nil
		}

		fmt.Print(".")
		return false, nil
	})
}

func applyManifestFromURL(dynamicClient dynamic.Interface, config *rest.Config, url string) error {
	// Download manifest
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download manifest: HTTP %d", resp.StatusCode)
	}

	// Create discovery client for REST mapper
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Get REST mapper
	gr, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return fmt.Errorf("failed to get API group resources: %w", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(gr)

	// Parse YAML and apply each resource
	decoder := yaml.NewYAMLOrJSONDecoder(resp.Body, 4096)
	dec := yamlserializer.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if len(rawObj.Raw) == 0 {
			continue
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := dec.Decode(rawObj.Raw, nil, obj)
		if err != nil {
			fmt.Printf("Warning: failed to decode resource: %v\n", err)
			continue
		}

		// Get REST mapping
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			fmt.Printf("Warning: failed to get REST mapping for %s: %v\n", gvk, err)
			continue
		}

		// Apply the resource
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == "namespace" && obj.GetNamespace() != "" {
			dr = dynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			dr = dynamicClient.Resource(mapping.Resource)
		}

		// Convert to apply configuration
		obj.SetManagedFields(nil)
		_, err = dr.Apply(context.Background(), obj.GetName(), obj, metav1.ApplyOptions{
			FieldManager: "kubevirt-env",
		})
		if err != nil {
			// Try create if apply fails (for resources that don't support apply)
			_, createErr := dr.Create(context.Background(), obj, metav1.CreateOptions{})
			if createErr != nil {
				// Ignore already exists errors
				if !strings.Contains(createErr.Error(), "already exists") {
					fmt.Printf("Warning: failed to apply %s/%s: %v\n", gvk.Kind, obj.GetName(), err)
				}
			}
		}
	}

	return nil
}
