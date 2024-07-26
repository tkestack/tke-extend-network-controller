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
	LbId string `json:"lbId"`
	// +optional
	LbRegion string `json:"lbRegion,omitempty"`
	LbPort   int64  `json:"lbPort"`
	Protocol string `json:"protocol"`
	// +optional
	ListenerConfig string `json:"listenerConfig,omitempty"`
	// +optional
	DedicatedTarget *DedicatedTarget `json:"dedicatedTarget,omitempty"`
}

type DedicatedTarget struct {
	IP   string `json:"ip"`
	Port int64  `json:"port"`
}

// DedicatedCLBListenerStatus defines the observed state of DedicatedCLBListener
type DedicatedCLBListenerStatus struct {
	ListenerId string `json:"listenerId,omitempty"`
	State      string `json:"state,omitempty"`
}

const (
	DedicatedCLBListenerStateOccupied  = "Occupied"
	DedicatedCLBListenerStateAvailable = "Available"
	DedicatedCLBListenerStatePending   = "Pending"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
