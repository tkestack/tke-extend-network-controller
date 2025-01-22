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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CLB struct {
	// CLB 实例的 LbId。
	LbId string `json:"lbId"`
	// region of the CLB instance.
	// +optional
	Region string `json:"region"`
}

// DedicatedCLBListenerSpec defines the desired state of DedicatedCLBListener
type DedicatedCLBListenerSpec struct {
	CLBs []CLB `json:"clbs"`
	// CLB port.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	Port int64 `json:"port"`
	// CLB Endport.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	// +optional
	EndPort *int64 `json:"endPort"`
	// CLB 监听器的协议。
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	// +kubebuilder:validation:Enum=TCP;UDP;TCPUDP
	Protocol string `json:"protocol"`
	// 创建监听器的参数，JSON 格式，详细参数请参考 CreateListener 接口：https://cloud.tencent.com/document/api/214/30693
	// +optional
	ExtensiveParameters *string `json:"extensiveParameters,omitempty"`
	// Target pod of the CLB listener.
	// +optional
	TargetPod *TargetPod `json:"targetPod,omitempty"`
	// Target node of the CLB listener.
	// +optional
	TargetNode *TargetNode `json:"targetNode,omitempty"`
}

type TargetNode struct {
	// Node 的名称。
	NodeName string `json:"nodeName"`
	// Node 监听的端口。
	Port int64 `json:"port"`
}

type TargetPod struct {
	// Pod 的名称。
	PodName string `json:"podName"`
	// Pod 监听的端口。
	Port int64 `json:"port"`
}

type ListenerStatus struct {
	CLB CLB `json:"clb,omitempty"`
	// CLB 监听器的 ID。
	ListenerId string `json:"listenerId,omitempty"`
	// CLB 监听器的状态。
	State string `json:"state,omitempty"`
	// 记录 CLB 监听器的失败信息。
	// +optional
	Message string `json:"message,omitempty"`
	// CLB 监听器的外部地址。
	Address string `json:"address,omitempty"`
}

// DedicatedCLBListenerStatus defines the observed state of DedicatedCLBListener
type DedicatedCLBListenerStatus struct {
	ListenerStatuses []ListenerStatus `json:"listenerStatuses,omitempty"`
}

const (
	DedicatedCLBListenerStateBound     = "Bound"
	DedicatedCLBListenerStateAvailable = "Available"
	DedicatedCLBListenerStatePending   = "Pending"
	DedicatedCLBListenerStateFailed    = "Failed"
	DedicatedCLBListenerStateDeleting  = "Deleting"
	DedicatedCLBListenerStateDeleted   = "Deleted"
)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Port",type="integer",JSONPath=".spec.port",description="Port of CLB Listener"
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
