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

// DedicatedCLBServiceSpec defines the desired state of DedicatedCLBService
type DedicatedCLBServiceSpec struct {
	// CLB 所在地域，不填则使用 TKE 集群所在的地域。
	// +optional
	LbRegion string `json:"lbRegion,omitempty"`
	// CLB 所在 VPC ID，不填则使用 TKE 集群所在的 VPC 的 ID。
	// +optional
	VpcId *string `json:"vpcId"`
	// CLB 端口范围的最小端口号。
	// +optional
	// +kubebuilder:default:value=500
	MinPort int64 `json:"minPort,omitempty"`
	// CLB 端口范围的最大端口号。
	// +optional
	// +kubebuilder:default:value=50000
	MaxPort int64 `json:"maxPort,omitempty"`
	// 限制单个 CLB 的 Pod/监听器 的最大数量。
	// +optional
	MaxPod *int64 `json:"maxPod,omitempty"`
	// Pod 的标签选择器，被选中的 Pod 会被绑定到 CLB 监听器下。
	Selector map[string]string `json:"selector"`
	// Pod 监听的端口。
	Ports []DedicatedCLBServicePort `json:"ports"`
	// 创建监听器的参数，JSON 格式，详细参数请参考 CreateListener 接口：https://cloud.tencent.com/document/api/214/30693
	// +optional
	ListenerExtensiveParameters string `json:"listenerExtensiveParameters,omitempty"`
	// 复用的已有的 CLB ID，可动态追加。
	// +optional
	ExistedLbIds []string `json:"existedLbIds,omitempty"`
	// 启用自动创建 CLB 的功能。
	// +optional
	LbAutoCreate LbAutoCreate `json:"lbAutoCreate,omitempty"`
}

type LbAutoCreate struct {
	// 是否启用自动创建 CLB 的功能，如果启用，当 CLB 不足时，会自动创建新的 CLB。
	// +optional
	Enable bool `json:"enable,omitempty"`
	// 创建 CLB 时的参数，JSON 格式，详细参数请参考 CreateLoadBalancer 接口：https://cloud.tencent.com/document/api/214/30692
	// +optional
	ExtensiveParameters string `json:"extensiveParameters,omitempty"`
}

type DedicatedCLBServicePort struct {
	// 端口协议，支持 TCP、UDP。
	Protocol string `json:"protocol"`
	// 目标端口。
	TargetPort int64 `json:"targetPort"`
	// Pod 外部地址的注解，如果设置，Pod 被映射的外部 CLB 地址将会被自动写到 Pod 的该注解中，Pod 内部可通过 Downward API 感知到自身的外部地址。
	// +optional
	AddressPodAnnotation string `json:"addressPodAnnotation"`
}

// DedicatedCLBServiceStatus defines the observed state of DedicatedCLBService
type DedicatedCLBServiceStatus struct {
	// 可分配端口的 CLB 列表
	// +optional
	AllocatableLb []AllocatableCLBInfo `json:"allocatableLb"`
	// 已分配完端口的 CLB 列表
	// +optional
	AllocatedLb []AllocatedCLBInfo `json:"allocatedLb"`
}

type AllocatableCLBInfo struct {
	// CLB 实例的 ID。
	LbId string `json:"lbId"`
	// CLB 当前已被分配的端口。
	// +optional
	CurrentPort int64 `json:"currentPort"`
	// 是否是自动创建的 CLB。如果是，删除 DedicatedCLBService 时，CLB 也会被清理。
	AutoCreate bool `json:"autoCreate"`
}

type AllocatedCLBInfo struct {
	// CLB 实例的 ID。
	LbId string `json:"lbId"`
	// 是否是自动创建的 CLB。如果是，删除 DedicatedCLBService 时，CLB 也会被清理。
	AutoCreate bool `json:"autoCreate"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DedicatedCLBService is the Schema for the dedicatedclbservices API
type DedicatedCLBService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DedicatedCLBServiceSpec   `json:"spec,omitempty"`
	Status DedicatedCLBServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DedicatedCLBServiceList contains a list of DedicatedCLBService
type DedicatedCLBServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DedicatedCLBService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DedicatedCLBService{}, &DedicatedCLBServiceList{})
}
