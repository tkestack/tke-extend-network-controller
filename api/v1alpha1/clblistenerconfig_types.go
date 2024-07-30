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
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

func (s *CLBListenerConfigSpec) CreateListenerRequest(lbId string, port int64, protocol string) *clb.CreateListenerRequest {
	req := clb.NewCreateListenerRequest()
	req.Ports = []*int64{&port}
	req.Protocol = &protocol
	req.LoadBalancerId = &lbId
	req.HealthCheck = &clb.HealthCheck{
		SourceIpType: common.Int64Ptr(1),
		HealthSwitch: common.Int64Ptr(0),
	}
	if s == nil {
		return req
	}
	if hc := s.Healthcheck; hc != nil {
		if hc.HealthSwitch != nil {
			req.HealthCheck.HealthSwitch = hc.HealthSwitch
		}
		req.HealthCheck.HealthSwitch = hc.HealthSwitch
		req.HealthCheck.TimeOut = hc.TimeOut
		req.HealthCheck.IntervalTime = hc.IntervalTime
		req.HealthCheck.HealthNum = hc.HealthNum
		req.HealthCheck.UnHealthNum = hc.UnHealthNum
		req.HealthCheck.HttpCode = hc.HttpCode
		req.HealthCheck.HttpCheckPath = hc.HttpCheckPath
		req.HealthCheck.HttpCheckDomain = hc.HttpCheckDomain
		req.HealthCheck.HttpCheckMethod = hc.HttpCheckMethod
		req.HealthCheck.CheckPort = hc.CheckPort
		req.HealthCheck.ContextType = hc.ContextType
		req.HealthCheck.SendContext = hc.SendContext
		req.HealthCheck.RecvContext = hc.RecvContext
		req.HealthCheck.CheckType = hc.CheckType
		req.HealthCheck.HttpVersion = hc.HttpVersion
		if hc.SourceIpType != nil {
			req.HealthCheck.SourceIpType = hc.SourceIpType
		}
		req.HealthCheck.ExtendedCode = hc.ExtendedCode
	}
	if cert := s.Certificate; cert != nil {
		req.Certificate = &clb.CertificateInput{
			SSLMode:  cert.SSLMode,
			CertId:   cert.CertId,
			CertCaId: cert.CertCaId,
		}
	}
	req.SessionExpireTime = s.SessionExpireTime
	req.Scheduler = s.Scheduler
	req.SniSwitch = s.SniSwitch
	req.TargetType = s.TargetType
	req.SessionType = s.SessionType
	req.KeepaliveEnable = s.KeepaliveEnable
	req.EndPort = s.EndPort
	req.DeregisterTargetRst = s.DeregisterTargetRst
	if mci := s.MultiCertInfo; mci != nil {
		req.MultiCertInfo = &clb.MultiCertInfo{
			SSLMode: mci.SSLMode,
		}
		for _, cert := range mci.CertList {
			req.MultiCertInfo.CertList = append(req.MultiCertInfo.CertList, &clb.CertInfo{
				CertId: cert.CertId,
			})
		}
	}
	req.MaxConn = s.MaxConn
	req.MaxCps = s.MaxCps
	req.IdleConnectTimeout = s.IdleConnectTimeout
	req.SnatEnable = s.SnatEnable
	return req
}

// CLBListenerConfigSpec defines the desired state of CLBListenerConfig
type CLBListenerConfigSpec struct {
	// +optional
	Healthcheck *CLBHealthcheck `json:"healthcheck,omitempty"`
	// +optional
	Certificate *Certificate `json:"certificate,omitempty"`
	// +optional
	SessionExpireTime *int64 `json:"sessionExpireTime,omitempty"`
	// +optional
	Scheduler *string `json:"scheduler,omitempty"`
	// +optional
	SniSwitch *int64 `json:"sniSwitch,omitempty"`
	// +optional
	TargetType *string `json:"targetType,omitempty"`
	// +optional
	SessionType *string `json:"sessionType,omitempty"`
	// +optional
	KeepaliveEnable *int64 `json:"keepaliveEnable,omitempty"`
	// +optional
	EndPort *uint64 `json:"endPort,omitempty"`
	// +optional
	DeregisterTargetRst *bool `json:"deregisterTargetRst,omitempty"`
	// +optional
	MultiCertInfo *MultiCertInfo `json:"multiCertInfo,omitempty"`
	// +optional
	MaxConn *int64 `json:"maxConn,omitempty"`
	// +optional
	MaxCps *int64 `json:"maxCps,omitempty"`
	// +optional
	IdleConnectTimeout *int64 `json:"idleConnectTimeout,omitempty"`
	// +optional
	SnatEnable *bool `json:"snatEnable,omitempty"`
}

type CLBHealthcheck struct {
	// whether to enable the health check, 1(enable), 0(disable)
	// +optional
	HealthSwitch *int64 `json:"healthSwitch,omitempty"`
	// health check timeout, unit: second, range 2~60, default 2
	// +optional
	TimeOut *int64 `json:"timeOut,omitempty"`
	// health check interval, unit: second
	// +optional
	IntervalTime *int64 `json:"intervalTime,omitempty"`
	// health check threshold, range 2~10, default 3
	// +optional
	HealthNum *int64 `json:"healthNum,omitempty"`
	// unhealthy threshold, range 2~10, default 3
	// +optional
	UnHealthNum *int64 `json:"unHealthNum,omitempty"`
	// health check http code, range 1~31, default 31
	// +optional
	HttpCode *int64 `json:"httpCode,omitempty"`
	// health check http path
	// +optional
	HttpCheckPath *string `json:"httpCheckPath,omitempty"`
	// http check domain
	// +optional
	HttpCheckDomain *string `json:"httpCheckDomain,omitempty"`
	// http check method
	// +optional
	HttpCheckMethod *string `json:"httpCheckMethod,omitempty"`
	// customized check port
	// +optional
	CheckPort *int64 `json:"checkPort,omitempty"`
	// +optional
	ContextType *string `json:"contextType,omitempty"`
	// +optional
	SendContext *string `json:"sendContext,omitempty"`
	// +optional
	RecvContext *string `json:"recvContext,omitempty"`
	// +optional
	CheckType *string `json:"checkType,omitempty"`
	// +optional
	HttpVersion *string `json:"httpVersion,omitempty"`
	// +optional
	SourceIpType *int64 `json:"sourceIpType,omitempty"`
	// +optional
	ExtendedCode *string `json:"extendedCode,omitempty"`
}

type Certificate struct {
	// +optional
	SSLMode *string `json:"sslMode,omitempty"`
	// +optional
	CertId *string `json:"certId,omitempty"`
	// +optional
	CertCaId *string `json:"certCaId,omitempty"`
}

type MultiCertInfo struct {
	SSLMode  *string    `json:"sslMode,omitempty"`
	CertList []CertInfo `json:"certList,omitempty"`
}

type CertInfo struct {
	CertId *string `json:"certId,omitempty"`
}

// CLBListenerConfigStatus defines the observed state of CLBListenerConfig
type CLBListenerConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

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
