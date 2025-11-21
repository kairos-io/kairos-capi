/*
Copyright 2024 The Kairos CAPI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing
permissions and limitations under the License.
*/

package infrastructure

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CloneInfrastructureMachine clones an infrastructure machine template into a new machine resource
func CloneInfrastructureMachine(ctx context.Context, c client.Client, scheme *runtime.Scheme, templateRef corev1.ObjectReference, machineName, namespace string, labels, annotations map[string]string) (client.Object, error) {
	// Get the template object
	templateObj, err := getTemplateObject(ctx, c, templateRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get infrastructure template: %w", err)
	}

	// Clone based on infrastructure provider type
	switch templateRef.Kind {
	case "DockerMachineTemplate":
		return cloneDockerMachineTemplate(ctx, c, scheme, templateObj, machineName, namespace, labels, annotations)
	case "VSphereMachineTemplate":
		return cloneVSphereMachineTemplate(ctx, c, scheme, templateObj, machineName, namespace, labels, annotations)
	default:
		return nil, fmt.Errorf("unsupported infrastructure provider: %s", templateRef.Kind)
	}
}

func getTemplateObject(ctx context.Context, c client.Client, ref corev1.ObjectReference) (*unstructured.Unstructured, error) {
	// Get the full template object as unstructured
	fullObj := &unstructured.Unstructured{}
	fullObj.SetGroupVersionKind(ref.GroupVersionKind())

	key := client.ObjectKey{
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}

	if err := c.Get(ctx, key, fullObj); err != nil {
		return nil, err
	}

	return fullObj, nil
}

func cloneDockerMachineTemplate(ctx context.Context, c client.Client, scheme *runtime.Scheme, template *unstructured.Unstructured, machineName, namespace string, labels, annotations map[string]string) (client.Object, error) {
	// For CAPD, we create a DockerMachine from DockerMachineTemplate
	// This is a simplified version - in production, you'd use the actual CAPD types

	// Create unstructured DockerMachine
	dockerMachine := &unstructured.Unstructured{}
	dockerMachine.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "DockerMachine",
	})

	dockerMachine.SetName(machineName)
	dockerMachine.SetNamespace(namespace)
	dockerMachine.SetLabels(labels)
	dockerMachine.SetAnnotations(annotations)

	// Copy spec from template
	if spec, ok, _ := unstructured.NestedMap(template.UnstructuredContent(), "spec", "template", "spec"); ok {
		if err := unstructured.SetNestedMap(dockerMachine.UnstructuredContent(), spec, "spec"); err != nil {
			return nil, fmt.Errorf("failed to set spec: %w", err)
		}
	}

	return dockerMachine, nil
}

func cloneVSphereMachineTemplate(ctx context.Context, c client.Client, scheme *runtime.Scheme, template *unstructured.Unstructured, machineName, namespace string, labels, annotations map[string]string) (client.Object, error) {
	// For CAPV, we create a VSphereMachine from VSphereMachineTemplate
	// This is a simplified version - in production, you'd use the actual CAPV types

	vsphereMachine := &unstructured.Unstructured{}
	vsphereMachine.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "VSphereMachine",
	})

	vsphereMachine.SetName(machineName)
	vsphereMachine.SetNamespace(namespace)
	vsphereMachine.SetLabels(labels)
	vsphereMachine.SetAnnotations(annotations)

	// Copy spec from template
	if spec, ok, _ := unstructured.NestedMap(template.UnstructuredContent(), "spec", "template", "spec"); ok {
		if err := unstructured.SetNestedMap(vsphereMachine.UnstructuredContent(), spec, "spec"); err != nil {
			return nil, fmt.Errorf("failed to set spec: %w", err)
		}
	}

	return vsphereMachine, nil
}
