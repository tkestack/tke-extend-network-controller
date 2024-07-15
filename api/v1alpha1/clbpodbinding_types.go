/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Binding struct {
	LbId       string `json:"lbId"`
	Port       int32  `json:"port"`
	Protocol   string `json:"protocol"`
	TargetPort int32  `json:"targetPort"`
}

// CLBPodBindingSpec defines the desired state of CLBPodBinding
type CLBPodBindingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Bindings []Binding `json:"bindings"`
}

// CLBPodBindingStatus defines the observed state of CLBPodBinding
type CLBPodBindingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State string `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CLBPodBinding is the Schema for the clbpodbindings API
type CLBPodBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CLBPodBindingSpec   `json:"spec,omitempty"`
	Status CLBPodBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CLBPodBindingList contains a list of CLBPodBinding
type CLBPodBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CLBPodBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CLBPodBinding{}, &CLBPodBindingList{})
}
