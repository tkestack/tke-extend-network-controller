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

// CLBNodeBindingSpec defines the desired state of CLBNodeBinding.
type CLBNodeBindingSpec struct {
	// 网络隔离
	// +optional
	Disabled *bool `json:"disabled,omitempty"`
	// 需要绑定的端口配置列表
	Ports []PortEntry `json:"ports"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=cnb
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State"

// CLBNodeBinding is the Schema for the clbnodebindings API.
type CLBNodeBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CLBBindingSpec   `json:"spec,omitempty"`
	Status CLBBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CLBNodeBindingList contains a list of CLBNodeBinding.
type CLBNodeBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CLBNodeBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CLBNodeBinding{}, &CLBNodeBindingList{})
}
