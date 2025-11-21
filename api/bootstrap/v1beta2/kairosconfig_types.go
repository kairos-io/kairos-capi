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

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// KairosConfigFinalizer allows the reconciler to clean up resources associated with KairosConfig before
	// removing it from the API server.
	KairosConfigFinalizer = "kairosconfig.bootstrap.cluster.x-k8s.io"
)

// KairosConfigSpec defines the desired state of KairosConfig
type KairosConfigSpec struct {
	// Role indicates whether this is a control-plane or worker node
	// +kubebuilder:validation:Enum=control-plane;worker
	// +kubebuilder:default=worker
	Role string `json:"role,omitempty"`

	// Distribution specifies the Kubernetes distribution to install
	// +kubebuilder:validation:Enum=k0s;k3s
	// +kubebuilder:default=k0s
	Distribution string `json:"distribution,omitempty"`

	// KubernetesVersion specifies the Kubernetes version to install
	// +kubebuilder:validation:Required
	KubernetesVersion string `json:"kubernetesVersion"`

	// ServerAddress is the address of the Kubernetes API server (for worker nodes)
	// +optional
	ServerAddress string `json:"serverAddress,omitempty"`

	// Token is the join token for worker nodes (if required by distribution)
	// +optional
	Token string `json:"token,omitempty"`

	// TokenSecretRef is a reference to a Secret containing the join token
	// +optional
	TokenSecretRef *corev1.ObjectReference `json:"tokenSecretRef,omitempty"`

	// CACertHashes are the CA certificate hashes for secure join
	// +optional
	CACertHashes []string `json:"caCertHashes,omitempty"`

	// CACertSecretRef is a reference to a Secret containing the CA certificate
	// +optional
	CACertSecretRef *corev1.ObjectReference `json:"caCertSecretRef,omitempty"`

	// Files specifies additional files to include in the cloud-config
	// +optional
	Files []File `json:"files,omitempty"`

	// PreCommands are commands to run before k0s/k3s installation
	// +optional
	PreCommands []string `json:"preCommands,omitempty"`

	// PostCommands are commands to run after k0s/k3s installation
	// +optional
	PostCommands []string `json:"postCommands,omitempty"`

	// Pause indicates that reconciliation should be paused
	// +optional
	Pause bool `json:"pause,omitempty"`

	// SingleNode indicates this is a single-node control plane cluster
	// When true, k0s will be configured with --single flag
	// +optional
	SingleNode bool `json:"singleNode,omitempty"`

	// UserName is the username for the default user
	// +kubebuilder:default=kairos
	// +optional
	UserName string `json:"userName,omitempty"`

	// UserPassword is the password for the default user
	// +kubebuilder:default=kairos
	// +optional
	UserPassword string `json:"userPassword,omitempty"`

	// UserGroups are the groups for the default user
	// +kubebuilder:default={admin}
	// +optional
	UserGroups []string `json:"userGroups,omitempty"`

	// GitHubUser is the GitHub username for SSH key access (e.g., "octocat")
	// If set, SSH keys will be fetched from GitHub
	// +optional
	GitHubUser string `json:"githubUser,omitempty"`

	// SSHPublicKey is a raw SSH public key (alternative to GitHubUser)
	// +optional
	SSHPublicKey string `json:"sshPublicKey,omitempty"`

	// WorkerToken is the join token for worker nodes
	// Alternative to TokenSecretRef for inline token specification
	// +optional
	WorkerToken string `json:"workerToken,omitempty"`

	// Manifests are Kubernetes manifests to be placed in /var/lib/k0s/manifests/
	// These will be automatically applied by k0s
	// +optional
	Manifests []Manifest `json:"manifests,omitempty"`
}

// Manifest represents a Kubernetes manifest file
type Manifest struct {
	// Name is the directory name under /var/lib/k0s/manifests/
	Name string `json:"name"`

	// File is the filename
	File string `json:"file"`

	// Content is the manifest YAML content
	Content string `json:"content"`
}

// File represents a file to be written in the cloud-config
type File struct {
	// Path is the absolute path where the file should be written
	Path string `json:"path"`

	// Content is the file content
	Content string `json:"content"`

	// Permissions are the file permissions (octal format, e.g., "0644")
	// +optional
	Permissions string `json:"permissions,omitempty"`

	// Owner is the file owner (user:group format, e.g., "root:root")
	// +optional
	Owner string `json:"owner,omitempty"`
}

// KairosConfigStatus defines the observed state of KairosConfig
// Contract: BootstrapConfig v1beta2 MUST expose a dataSecretName and ready status
type KairosConfigStatus struct {
	// Ready indicates the bootstrap data has been generated and is ready
	// Contract: BootstrapConfig MUST indicate bootstrap completion
	// This field MUST be set to true when bootstrap data is available and ready to use.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// DataSecretName is the name of the Secret containing the bootstrap data
	// Contract: BootstrapConfig MUST expose a dataSecretName
	// The Secret must be in the same namespace as the KairosConfig.
	// +optional
	DataSecretName *string `json:"dataSecretName,omitempty"`

	// Conditions defines current service state of the KairosConfig
	// Contract: BootstrapConfig SHOULD expose Conditions
	// Standard CAPI conditions: Ready, BootstrapReady, DataSecretAvailable
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// FailureReason indicates the reason for bootstrap failure
	// This field is set only when bootstrap fails permanently.
	// +optional
	FailureReason string `json:"failureReason,omitempty"`

	// FailureMessage indicates the message for bootstrap failure
	// This field is set only when bootstrap fails permanently.
	// +optional
	FailureMessage string `json:"failureMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=kairosconfigs,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Bootstrap ready"
// +kubebuilder:printcolumn:name="DataSecretName",type="string",JSONPath=".status.dataSecretName",description="Secret containing bootstrap data"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// KairosConfig is the Schema for the kairosconfigs API
type KairosConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KairosConfigSpec   `json:"spec,omitempty"`
	Status KairosConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KairosConfigList contains a list of KairosConfig
type KairosConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KairosConfig `json:"items"`
}

// GetConditions returns the set of conditions for this object.
func (c *KairosConfig) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (c *KairosConfig) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&KairosConfig{}, &KairosConfigList{})
}

