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

// KairosConfigTemplateSpec defines the desired state of KairosConfigTemplate
type KairosConfigTemplateSpec struct {
	// Template is the KairosConfig template to be used for each Machine
	Template KairosConfigTemplateResource `json:"template"`
}

// KairosConfigTemplateResource defines the template for KairosConfig
type KairosConfigTemplateResource struct {
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the specification of the KairosConfig
	Spec KairosConfigSpec `json:"spec"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=kairosconfigtemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// KairosConfigTemplate is the Schema for the kairosconfigtemplates API
type KairosConfigTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KairosConfigTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// KairosConfigTemplateList contains a list of KairosConfigTemplate
type KairosConfigTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KairosConfigTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KairosConfigTemplate{}, &KairosConfigTemplateList{})
}
