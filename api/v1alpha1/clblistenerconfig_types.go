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

// CLBListenerConfigSpec defines the desired state of CLBListenerConfig
type CLBListenerConfigSpec struct {
	// clb listener protocol, TCP | UDP | HTTP | HTTPS | TCP_SSL | QUIC
	Protocol string `json:"protocol"`
	// +optional
	Healthcheck *CLBHealthcheck `json:"healthcheck,omitempty"`
	// +optional
	Certificate *Certificate `json:"certificate,omitempty"`
	// +optional
	SessionExpireTime int32 `json:"sessionExpireTime,omitempty"`
	// +optional
	Scheduler string `json:"scheduler,omitempty"`
	// +optional
	SniSwitch int32 `json:"sniSwitch,omitempty"`
	// +optional
	TargetType string `json:"targetType,omitempty"`
	// +optional
	SessionType string `json:"sessionType,omitempty"`
	// +optional
	KeepaliveEnable int32 `json:"keepaliveEnable,omitempty"`
	// +optional
	EndPort int32 `json:"endPort,omitempty"`
	// +optional
	DeregisterTargetRst bool `json:"deregisterTargetRst,omitempty"`
	// +optional
	MultiCertInfo *MultiCertInfo `json:"multiCertInfo,omitempty"`
	// +optional
	MaxConn int32 `json:"maxConn,omitempty"`
	// +optional
	MaxCps int32 `json:"maxCps,omitempty"`
	// +optional
	IdleConnectTimeout int32 `json:"idleConnectTimeout,omitempty"`
	// +optional
	SnatEnable bool `json:"snatEnable,omitempty"`
}

type CLBHealthcheck struct {
	// whether to enable the health check, 1(enable), 0(disable)
	// +optional
	HealthSwitch int32 `json:"healthSwitch,omitempty"`
	// health check timeout, unit: second, range 2~60, default 2
	// +optional
	TimeOut int32 `json:"timeOut,omitempty"`
	// health check interval, unit: second
	// +optional
	IntervalTime int32 `json:"intervalTime,omitempty"`
	// health check threshold, range 2~10, default 3
	// +optional
	HealthNum int32 `json:"healthNum,omitempty"`
	// unhealthy threshold, range 2~10, default 3
	// +optional
	UnHealthNum int32 `json:"unHealthNum,omitempty"`
	// health check http code, range 1~31, default 31
	// +optional
	HttpCode int32 `json:"httpCode,omitempty"`
	// health check http path
	// +optional
	HttpCheckPath string `json:"httpCheckPath,omitempty"`
	// http check domain
	// +optional
	HttpCheckDomain string `json:"httpCheckDomain,omitempty"`
	// http check method
	// +optional
	HttpCheckMethod string `json:"httpCheckMethod,omitempty"`
	// customized check port
	// +optional
	CheckPort int32 `json:"checkPort,omitempty"`
	// +optional
	ContextType string `json:"contextType,omitempty"`
	// +optional
	SendContext string `json:"sendContext,omitempty"`
	// +optional
	RecvContext string `json:"recvContext,omitempty"`
	// +optional
	CheckType string `json:"checkType,omitempty"`
	// +optional
	HttpVersion string `json:"httpVersion,omitempty"`
	// +optional
	SourceIpType int32 `json:"sourceIpType,omitempty"`
	// +optional
	ExtendedCode int32 `json:"extendedCode,omitempty"`
}

type Certificate struct {
	// +optional
	SslMode string `json:"sslMode,omitempty"`
	// +optional
	CertId string `json:"certId,omitempty"`
	// +optional
	CertCaId string `json:"certCaId,omitempty"`
}

type MultiCertInfo struct {
	SslMode  string     `json:"sslMode,omitempty"`
	CertList []CertInfo `json:"certList,omitempty"`
}

type CertInfo struct {
	CertId string `json:"certId,omitempty"`
}

// CLBListenerConfigStatus defines the observed state of CLBListenerConfig
type CLBListenerConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CLBListenerConfig is the Schema for the clblistenerconfigs API
type CLBListenerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CLBListenerConfigSpec   `json:"spec,omitempty"`
	Status CLBListenerConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CLBListenerConfigList contains a list of CLBListenerConfig
type CLBListenerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CLBListenerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CLBListenerConfig{}, &CLBListenerConfigList{})
}
