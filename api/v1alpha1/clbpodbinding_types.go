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

// CLBPodBindingSpec defines the desired state of CLBPodBinding.
type CLBPodBindingSpec struct {
	// 网络隔离
	// +optional
	Disabled *bool `json:"disabled,omitempty"`
	// 需要绑定的端口配置列表
	Ports []PortEntry `json:"ports"`
}

// PortEntry 定义单个端口的绑定配置
type PortEntry struct {
	// 应用监听的端口号
	Port uint16 `json:"port"`
	// 端口使用的协议
	// +kubebuilder:validation:Enum=TCP;UDP;TCPUDP
	Protocol string `json:"protocol"`
	// 使用的端口池列表
	Pools []string `json:"pools"`
	// 是否跨端口池分配相同端口号
	// +optional
	UseSamePortAcrossPools *bool `json:"useSamePortAcrossPools,omitempty"`
}

type CLBPodBindingState string

const (
	CLBPodBindingStatePending    CLBPodBindingState = "Pending"
	CLBPodBindingStateBound      CLBPodBindingState = "Bound"
	CLBPodBindingStateWaitForPod CLBPodBindingState = "WaitForPod"
	CLBPodBindingStateDisabled   CLBPodBindingState = "Disabled"
	CLBPodBindingStateFailed     CLBPodBindingState = "Failed"
	CLBPodBindingStateDeleting   CLBPodBindingState = "Deleting"
)

// CLBPodBindingStatus defines the observed state of CLBPodBinding.
type CLBPodBindingStatus struct {
	// 绑定状态
	State CLBPodBindingState `json:"state"`
	// 状态信息
	Message string `json:"message,omitempty"`
	// 端口绑定详情
	PortBindings []PortBindingStatus `json:"portBindings,omitempty"`
}

// PortBindingStatus 描述单个端口的实际绑定情况
type PortBindingStatus struct {
	// 应用端口
	Port uint16 `json:"port"`
	// 协议类型
	Protocol string `json:"protocol"`
	// 使用的端口池
	Pool string `json:"pool"`
	// 地域信息
	Region string `json:"region"`
	// 负载均衡器ID
	LoadbalancerId string `json:"loadbalancerId"`
	// 负载均衡器端口
	LoadbalancerPort uint16 `json:"loadbalancerPort"`
	// 负载均衡器端口段结束端口（当使用端口段时）
	// +optional
	LoadbalancerEndPort *uint16 `json:"loadbalancerEndPort,omitempty"`
	// 监听器ID
	ListenerId string `json:"listenerId"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cpb
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State"

// CLBPodBinding is the Schema for the clbpodbindings API.
type CLBPodBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CLBPodBindingSpec   `json:"spec,omitempty"`
	Status CLBPodBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CLBPodBindingList contains a list of CLBPodBinding.
type CLBPodBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CLBPodBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CLBPodBinding{}, &CLBPodBindingList{})
}
