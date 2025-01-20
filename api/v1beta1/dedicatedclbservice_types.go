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

type ExistedCLB struct {
	// CLB 实例的 ID。
	ID string `json:"id"`
	// region of the CLB instance.
	// +optional
	Region string `json:"region"`
	// Alias of the CLB instance.
	Alias string `json:"alias"`
}

func (e *ExistedCLB) ToCLBInfo() CLBInfo {
	return CLBInfo{
		LbId:   e.ID,
		Region: e.Region,
		Alias:  e.Alias,
	}
}

type Target struct {
	Pod  *TargetPods  `json:"pod,omitempty"`
	Node *TargetNodes `json:"node,omitempty"`
}

type TargetPods struct {
	PodSelector map[string]string `json:"podSelector"`
}

type TargetNodes struct {
	// +optional
	NodeSelector map[string]string `json:"nodeSelector"`
}

// DedicatedCLBServiceSpec defines the desired state of DedicatedCLBService
type DedicatedCLBServiceSpec struct {
	// CLB 端口范围的最小端口号。
	// +optional
	// +kubebuilder:default:value=500
	MinPort int64 `json:"minPort,omitempty"`
	// CLB 端口范围的最大端口号。
	// +optional
	MaxPort int64 `json:"maxPort,omitempty"`
	// 限制单个 CLB 的 Pod/监听器 的最大数量。
	// +optional
	MaxListener *int64 `json:"maxListener,omitempty"`
	// Number of ports in the port range listener if not null.
	// +optional
	PortSegment *int64 `json:"portSegment,omitempty"`
	Target      Target `json:"target"`
	// Pod 监听的端口。
	Ports []DedicatedCLBServicePort `json:"ports"`
	// 创建监听器的参数，JSON 格式，详细参数请参考 CreateListener 接口: https://cloud.tencent.com/document/api/214/30693
	// +optional
	ListenerExtensiveParameters *string `json:"listenerExtensiveParameters,omitempty"`
	// 复用的已有的 CLB ID，可动态追加。
	// +optional
	ExistedCLBs [][]ExistedCLB `json:"existedCLBs,omitempty"`
	// 启用自动创建 CLB 的功能。
	// +optional
	LbAutoCreate LbAutoCreate `json:"lbAutoCreate,omitempty"`
}

type LBParameter struct {
	// +optional
	Alias *string `json:"alias"`
	// +optional
	Region string `json:"region"`
	// +optional
	VpcId *string `json:"vpcId"`
	// +kubebuilder:validation:Enum=IPV4;IPV6;IPv6FullChain
	// +optional
	AddressIPVersion *string `json:"addressIPVersion"`
	// +optional
	VipIsp *string `json:"vipIsp"`
	// +optional
	LoadBalancerName *string `json:"loadBalancerName"`
	// 创建 CLB 时的参数，JSON 格式，详细参数请参考 CreateLoadBalancer 接口：https://cloud.tencent.com/document/api/214/30692
	// +optional
	ExtensiveParameters *string `json:"extensiveParameters,omitempty"`
}

type LbAutoCreate struct {
	// 是否启用自动创建 CLB 的功能，如果启用，当 CLB 不足时，会自动创建新的 CLB。
	// +optional
	Enable bool `json:"enable,omitempty"`
	// +optional
	Parameters []LBParameter `json:"parameters,omitempty"`
}

type DedicatedCLBServicePort struct {
	// 端口协议，支持 TCP、UDP、TCPUDP。
	Protocol string `json:"protocol"`
	// 目标端口。
	TargetPort int64 `json:"targetPort"`
	// Pod 外部地址的注解，如果设置，Pod 被映射的外部 CLB 地址将会被自动写到 Pod 的该注解中，Pod 内部可通过 Downward API 感知到自身的外部地址。
	// +optional
	AddressPodAnnotation string `json:"addressPodAnnotation"`
}

// DedicatedCLBServiceStatus defines the observed state of DedicatedCLBService
type DedicatedCLBServiceStatus struct {
	// +optional
	CLBs [][]CLBInfo `json:"clbs"`
}

type CLBInfo struct {
	// CLB 实例的 ID。
	LbId   string `json:"lbId"`
	Region string `json:"region"`
	// +optional
	Alias string `json:"alias"`
	// 是否是自动创建的 CLB。如果是，删除 DedicatedCLBService 时，CLB 也会被清理。
	// +optional
	AutoCreate bool `json:"autoCreate"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
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
