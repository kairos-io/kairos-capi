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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KairosControlPlaneTemplateSpec defines the desired state of KairosControlPlaneTemplate
type KairosControlPlaneTemplateSpec struct {
	// Template is the KairosControlPlane template to be used
	Template KairosControlPlaneTemplateResource `json:"template"`
}

// KairosControlPlaneTemplateResource defines the template for KairosControlPlane
type KairosControlPlaneTemplateResource struct {
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the specification of the KairosControlPlane
	Spec KairosControlPlaneSpec `json:"spec"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=kairoscontrolplanetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// KairosControlPlaneTemplate is the Schema for the kairoscontrolplanetemplates API
type KairosControlPlaneTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KairosControlPlaneTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// KairosControlPlaneTemplateList contains a list of KairosControlPlaneTemplate
type KairosControlPlaneTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KairosControlPlaneTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KairosControlPlaneTemplate{}, &KairosControlPlaneTemplateList{})
}

