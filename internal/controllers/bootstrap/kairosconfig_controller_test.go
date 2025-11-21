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

package bootstrap

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	bootstrapv1beta2 "github.com/wrkode/kairos-capi/api/bootstrap/v1beta2"
)

func TestGenerateK0sCloudConfig_ControlPlaneSingleNode(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(bootstrapv1beta2.AddToScheme(scheme)).To(Succeed())
	g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &KairosConfigReconciler{
		Client: client,
		Scheme: scheme,
	}

	kairosConfig := &bootstrapv1beta2.KairosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: bootstrapv1beta2.KairosConfigSpec{
			Role:              "control-plane",
			Distribution:      "k0s",
			KubernetesVersion: "v1.30.0+k0s.0",
			SingleNode:        true,
			UserName:          "kairos",
			UserPassword:      "kairos",
			UserGroups:        []string{"admin"},
		},
	}

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	cloudConfig, err := reconciler.generateK0sCloudConfig(
		context.Background(),
		log.Log,
		kairosConfig,
		machine,
		cluster,
		"control-plane",
		"",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cloudConfig).To(ContainSubstring("#cloud-config"))
	g.Expect(cloudConfig).To(ContainSubstring("k0s:"))
	g.Expect(cloudConfig).To(ContainSubstring("enabled: true"))
	g.Expect(cloudConfig).To(ContainSubstring("--single"))
	g.Expect(cloudConfig).NotTo(ContainSubstring("k0s-worker:"))
}

func TestGenerateK0sCloudConfig_ControlPlaneMultiNode(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(bootstrapv1beta2.AddToScheme(scheme)).To(Succeed())
	g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &KairosConfigReconciler{
		Client: client,
		Scheme: scheme,
	}

	kairosConfig := &bootstrapv1beta2.KairosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: bootstrapv1beta2.KairosConfigSpec{
			Role:              "control-plane",
			Distribution:      "k0s",
			KubernetesVersion: "v1.30.0+k0s.0",
			SingleNode:        false,
			UserName:          "kairos",
			UserPassword:      "kairos",
			UserGroups:        []string{"admin"},
		},
	}

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	cloudConfig, err := reconciler.generateK0sCloudConfig(
		context.Background(),
		log.Log,
		kairosConfig,
		machine,
		cluster,
		"control-plane",
		"",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cloudConfig).To(ContainSubstring("#cloud-config"))
	g.Expect(cloudConfig).To(ContainSubstring("k0s:"))
	g.Expect(cloudConfig).To(ContainSubstring("enabled: true"))
	g.Expect(cloudConfig).NotTo(ContainSubstring("--single"))
	g.Expect(cloudConfig).NotTo(ContainSubstring("k0s-worker:"))
}

func TestGenerateK0sCloudConfig_WorkerWithToken(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(bootstrapv1beta2.AddToScheme(scheme)).To(Succeed())
	g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &KairosConfigReconciler{
		Client: client,
		Scheme: scheme,
	}

	kairosConfig := &bootstrapv1beta2.KairosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: bootstrapv1beta2.KairosConfigSpec{
			Role:              "worker",
			Distribution:      "k0s",
			KubernetesVersion: "v1.30.0+k0s.0",
			WorkerToken:       "test-token-12345",
			UserName:          "kairos",
			UserPassword:      "kairos",
			UserGroups:        []string{"admin"},
		},
	}

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	cloudConfig, err := reconciler.generateK0sCloudConfig(
		context.Background(),
		log.Log,
		kairosConfig,
		machine,
		cluster,
		"worker",
		"https://control-plane:6443",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cloudConfig).To(ContainSubstring("#cloud-config"))
	g.Expect(cloudConfig).To(ContainSubstring("k0s-worker:"))
	g.Expect(cloudConfig).To(ContainSubstring("enabled: true"))
	g.Expect(cloudConfig).To(ContainSubstring("--token-file /etc/k0s/token"))
	g.Expect(cloudConfig).To(ContainSubstring("path: /etc/k0s/token"))
	g.Expect(cloudConfig).To(ContainSubstring("test-token-12345"))
	g.Expect(cloudConfig).NotTo(ContainSubstring("k0s:"))
}

func TestGenerateK0sCloudConfig_WorkerWithTokenSecretRef(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(bootstrapv1beta2.AddToScheme(scheme)).To(Succeed())
	g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "worker-token",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("secret-token-67890"),
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tokenSecret).Build()
	reconciler := &KairosConfigReconciler{
		Client: client,
		Scheme: scheme,
	}

	kairosConfig := &bootstrapv1beta2.KairosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: bootstrapv1beta2.KairosConfigSpec{
			Role:              "worker",
			Distribution:      "k0s",
			KubernetesVersion: "v1.30.0+k0s.0",
			WorkerTokenSecretRef: &bootstrapv1beta2.WorkerTokenSecretReference{
				Name: "worker-token",
				Key:  "token",
			},
			UserName:     "kairos",
			UserPassword: "kairos",
			UserGroups:   []string{"admin"},
		},
	}

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	cloudConfig, err := reconciler.generateK0sCloudConfig(
		context.Background(),
		log.Log,
		kairosConfig,
		machine,
		cluster,
		"worker",
		"https://control-plane:6443",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cloudConfig).To(ContainSubstring("k0s-worker:"))
	g.Expect(cloudConfig).To(ContainSubstring("secret-token-67890"))
}

func TestGenerateK0sCloudConfig_WorkerTokenPrecedence(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(bootstrapv1beta2.AddToScheme(scheme)).To(Succeed())
	g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "worker-token",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("secret-token-takes-precedence"),
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tokenSecret).Build()
	reconciler := &KairosConfigReconciler{
		Client: client,
		Scheme: scheme,
	}

	kairosConfig := &bootstrapv1beta2.KairosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: bootstrapv1beta2.KairosConfigSpec{
			Role:              "worker",
			Distribution:      "k0s",
			KubernetesVersion: "v1.30.0+k0s.0",
			WorkerToken:       "inline-token-should-be-ignored",
			WorkerTokenSecretRef: &bootstrapv1beta2.WorkerTokenSecretReference{
				Name: "worker-token",
				Key:  "token",
			},
			UserName:     "kairos",
			UserPassword: "kairos",
			UserGroups:   []string{"admin"},
		},
	}

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	cloudConfig, err := reconciler.generateK0sCloudConfig(
		context.Background(),
		log.Log,
		kairosConfig,
		machine,
		cluster,
		"worker",
		"https://control-plane:6443",
	)

	g.Expect(err).NotTo(HaveOccurred())
	// WorkerTokenSecretRef should take precedence over WorkerToken
	g.Expect(cloudConfig).To(ContainSubstring("secret-token-takes-precedence"))
	g.Expect(cloudConfig).NotTo(ContainSubstring("inline-token-should-be-ignored"))
}

func TestGenerateK0sCloudConfig_WorkerMissingToken(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(bootstrapv1beta2.AddToScheme(scheme)).To(Succeed())
	g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &KairosConfigReconciler{
		Client: client,
		Scheme: scheme,
	}

	kairosConfig := &bootstrapv1beta2.KairosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: bootstrapv1beta2.KairosConfigSpec{
			Role:              "worker",
			Distribution:      "k0s",
			KubernetesVersion: "v1.30.0+k0s.0",
			// No token provided
			UserName:     "kairos",
			UserPassword: "kairos",
			UserGroups:   []string{"admin"},
		},
	}

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	_, err := reconciler.generateK0sCloudConfig(
		context.Background(),
		log.Log,
		kairosConfig,
		machine,
		cluster,
		"worker",
		"https://control-plane:6443",
	)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("worker token is required"))
}

func TestGenerateK0sCloudConfig_HostnameTemplating(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(bootstrapv1beta2.AddToScheme(scheme)).To(Succeed())
	g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &KairosConfigReconciler{
		Client: client,
		Scheme: scheme,
	}

	kairosConfig := &bootstrapv1beta2.KairosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Spec: bootstrapv1beta2.KairosConfigSpec{
			Role:              "control-plane",
			Distribution:      "k0s",
			KubernetesVersion: "v1.30.0+k0s.0",
			SingleNode:        true,
			UserName:          "kairos",
			UserPassword:      "kairos",
			UserGroups:        []string{"admin"},
		},
	}

	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
	}

	cloudConfig, err := reconciler.generateK0sCloudConfig(
		context.Background(),
		log.Log,
		kairosConfig,
		machine,
		cluster,
		"control-plane",
		"",
	)

	g.Expect(err).NotTo(HaveOccurred())
	// Verify Kairos templating is preserved (not Go templating)
	g.Expect(cloudConfig).To(ContainSubstring("hostname: metal-{{ trunc 4 .MachineID }}"))
	// Should NOT contain Go template syntax
	g.Expect(cloudConfig).NotTo(ContainSubstring("{{.MachineID}}"))
}
