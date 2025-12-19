package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
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
	"k8s.io/client-go/tools/clientcmd"
)

func getKubeClient() (kubernetes.Interface, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, err
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

func getKubeConfig() (*rest.Config, error) {
	kubeconfigPath := getKubeconfigPath()
	
	// Check if kubeconfig exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig not found at %s. Please create the cluster first with: kubevirt-env create-test-cluster", kubeconfigPath)
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Set the context if specified
	kubectlContext := getKubectlContext()
	if kubectlContext != "" {
		// Load kubeconfig
		configLoadingRules := &clientcmd.ClientConfigLoadingRules{
			ExplicitPath: kubeconfigPath,
		}
		configOverrides := &clientcmd.ConfigOverrides{
			CurrentContext: kubectlContext,
		}
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, configOverrides).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig with context: %w", err)
		}
	}

	return config, nil
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

func deleteResourcesFromManifestURL(dynamicClient dynamic.Interface, config *rest.Config, url string) error {
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

	// Parse YAML and delete each resource
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
			continue
		}

		// Get REST mapping
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			continue
		}

		// Delete the resource
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == "namespace" && obj.GetNamespace() != "" {
			dr = dynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			dr = dynamicClient.Resource(mapping.Resource)
		}

		err = dr.Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		if err != nil {
			// Ignore not found errors
			if !strings.Contains(err.Error(), "not found") {
				fmt.Printf("Warning: failed to delete %s/%s: %v\n", gvk.Kind, obj.GetName(), err)
			}
		}
	}

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
			fmt.Printf("\n✓ %s daemonset is ready (%d/%d pods)\n", name, ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
			return true, nil
		}

		fmt.Print(".")
		return false, nil
	})
}
