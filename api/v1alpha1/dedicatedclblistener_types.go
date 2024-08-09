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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DedicatedCLBListenerSpec defines the desired state of DedicatedCLBListener
type DedicatedCLBListenerSpec struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	LbId string `json:"lbId"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	// +optional
	LbRegion string `json:"lbRegion,omitempty"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	LbPort int64 `json:"lbPort"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	// +kubebuilder:validation:Enum=TCP;UDP
	Protocol string `json:"protocol"`
	// +optional
	ListenerConfig string `json:"listenerConfig,omitempty"`
	// +optional
	BackendPod *BackendPod `json:"backendPod,omitempty"`
}

type BackendPod struct {
	PodName string `json:"podName"`
	Port    int64  `json:"port"`
}

// DedicatedCLBListenerStatus defines the observed state of DedicatedCLBListener
type DedicatedCLBListenerStatus struct {
	ListenerId string `json:"listenerId,omitempty"`
	State      string `json:"state,omitempty"`
	Address    string `json:"address,omitempty"`
}

const (
	DedicatedCLBListenerStateBound     = "Bound"
	DedicatedCLBListenerStateAvailable = "Available"
	DedicatedCLBListenerStatePending   = "Pending"
	DedicatedCLBListenerStateDeleting  = "Deleting"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="LbId",type="string",JSONPath=".spec.lbId",description="CLB ID"
// +kubebuilder:printcolumn:name="LbPort",type="integer",JSONPath=".spec.lbPort",description="Port of CLB Listener"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State of the dedicated clb listener"

// DedicatedCLBListener is the Schema for the dedicatedclblisteners API
type DedicatedCLBListener struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DedicatedCLBListenerSpec   `json:"spec,omitempty"`
	Status DedicatedCLBListenerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DedicatedCLBListenerList contains a list of DedicatedCLBListener
type DedicatedCLBListenerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DedicatedCLBListener `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DedicatedCLBListener{}, &DedicatedCLBListenerList{})
}