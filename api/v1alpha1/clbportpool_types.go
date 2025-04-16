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
	"github.com/imroc/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CLBPortPoolSpec defines the desired state of CLBPortPool.
type CLBPortPoolSpec struct {
	// 端口池的起始端口号
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	StartPort uint16 `json:"startPort"`
	// 端口池的结束端口号
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	EndPort *uint16 `json:"endPort,omitempty"`
	// 端口段的长度
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	SegmentLength *uint16 `json:"segmentLength,omitempty"`
	// 地域代码，如ap-chengdu
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"
	// +optional
	Region *string `json:"region,omitempty"`
	// 已有负载均衡器ID列表
	ExsistedLoadBalancerIDs []string `json:"exsistedLoadBalancerIDs,omitempty"`
	// 自动创建配置
	AutoCreate *AutoCreateConfig `json:"autoCreate,omitempty"`
}

func (pool *CLBPortPool) CanCreateLB() bool {
	// 还未初始化的端口池，不能创建负载均衡器
	if pool.Status.State == "" || pool.Status.State == CLBPortPoolStatePending {
		return false
	}
	// 没有显式启用自动创建的端口池，不能创建负载均衡器
	if pool.Spec.AutoCreate == nil {
		return false
	}
	if !pool.Spec.AutoCreate.Enabled {
		return false
	}
	// 自动创建的 CLB 数量达到配置上限的端口池，不能创建负载均衡器
	if !util.IsZero(pool.Spec.AutoCreate.MaxLoadBalancers) {
		// 检查是否已创建了足够的 CLB
		num := uint16(0)
		for _, lbStatus := range pool.Status.LoadbalancerStatuses {
			if lbStatus.AutoCreated != nil && *lbStatus.AutoCreated {
				num++
			}
		}
		// 如果已创建数量已满，则直接返回
		if num >= *pool.Spec.AutoCreate.MaxLoadBalancers {
			return false
		}
	}
	// 其余情况，允许创建负载均衡器
	return true
}

func (pool *CLBPortPool) GetRegion() string {
	return util.GetRegionFromPtr(pool.Spec.Region)
}

// AutoCreateConfig 定义自动创建CLB的配置
type AutoCreateConfig struct {
	// 是否启用自动创建
	Enabled bool `json:"enabled"`
	// 自动创建的最大负载均衡器数量
	MaxLoadBalancers *uint16 `json:"maxLoadBalancers,omitempty"`
	// 自动创建参数
	Parameters *CreateLBParameters `json:"parameters,omitempty"`
}

// CreateLBParameters 定义创建负载均衡器的参数
type CreateLBParameters struct {
	// 仅适用于公网负载均衡。目前仅广州、上海、南京、济南、杭州、福州、北京、石家庄、武汉、长沙、成都、重庆地域支持静态单线 IP 线路类型，如需体验，请联系商务经理申请。申请通过后，即可选择中国移动（CMCC）、中国联通（CUCC）或中国电信（CTCC）的运营商类型，网络计费模式只能使用按带宽包计费(BANDWIDTH_PACKAGE)。 如果不指定本参数，则默认使用BGP。可通过 DescribeResources 接口查询一个地域所支持的Isp。
	VipIsp *string `json:"vipIsp,omitempty"`
	// 带宽包ID，指定此参数时，网络计费方式（InternetAccessible.InternetChargeType）只支持按带宽包计费（BANDWIDTH_PACKAGE），带宽包的属性即为其结算方式。非上移用户购买的 IPv6 负载均衡实例，且运营商类型非 BGP 时 ，不支持指定具体带宽包id。
	BandwidthPackageId *string `json:"bandwidthPackageId,omitempty"`
	// 仅适用于公网负载均衡。IP版本，可取值：IPV4、IPV6、IPv6FullChain，不区分大小写，默认值 IPV4。说明：取值为IPV6表示为IPV6 NAT64版本；取值为IPv6FullChain，表示为IPv6版本。
	AddressIPVersion *string `json:"addressIPVersion,omitempty"`
	// Target是否放通来自CLB的流量。开启放通（true）：只验证CLB上的安全组；不开启放通（false）：需同时验证CLB和后端实例上的安全组。默认值为 true。
	LoadBalancerPassToTarget *bool `json:"loadBalancerPassToTarget,omitempty"`
	// 是否创建域名化负载均衡。
	DynamicVip *bool `json:"dynamicVip,omitempty"`
	// 负载均衡后端目标设备所属的网络 ID，如vpc-12345678，可以通过 DescribeVpcs 接口获取。 不填此参数则默认为当前集群所在 VPC。创建内网负载均衡实例时，此参数必填。
	VpcId *string `json:"vpcId,omitempty"`
	// 仅适用于公网负载均衡。目前仅广州、上海、南京、济南、杭州、福州、北京、石家庄、武汉、长沙、成都、重庆地域支持静态单线 IP 线路类型，如需体验，请联系商务经理申请。申请通过后，即可选择中国移动（CMCC）、中国联通（CUCC）或中国电信（CTCC）的运营商类型，网络计费模式只能使用按带宽包计费(BANDWIDTH_PACKAGE)。 如果不指定本参数，则默认使用BGP。可通过 DescribeResources 接口查询一个地域所支持的Isp。
	Vip *string `json:"vip,omitempty"`
	// 购买负载均衡的同时，给负载均衡打上标签，最大支持20个标签键值对。
	Tags []TagInfo `json:"tags,omitempty"`
	// 负载均衡实例所属的项目 ID，可以通过 DescribeProject 接口获取。不填此参数则视为默认项目。
	ProjectId *int64 `json:"projectId,omitempty"`
	// 负载均衡实例的名称。规则：1-60 个英文、汉字、数字、连接线“-”或下划线“_”。 注意：如果名称与系统中已有负载均衡实例的名称相同，则系统将会自动生成此次创建的负载均衡实例的名称。
	LoadBalancerName *string `json:"loadBalancerName,omitempty"`
	// 负载均衡实例的网络类型：OPEN：公网属性， INTERNAL：内网属性。默认使用 OPEN（公网负载均衡）。
	LoadBalancerType *string `json:"loadBalancerType,omitempty"`
	// 仅适用于公网且IP版本为IPv4的负载均衡。设置跨可用区容灾时的主可用区ID，例如 100001 或 ap-guangzhou-1
	// 注：主可用区是需要承载流量的可用区，备可用区默认不承载流量，主可用区不可用时才使用备可用区。目前仅广州、上海、南京、北京、成都、深圳金融、中国香港、首尔、法兰克福、新加坡地域的 IPv4 版本的 CLB 支持主备可用区。可通过 DescribeResources 接口查询一个地域的主可用区的列表。【如果您需要体验该功能，请通过 工单申请】
	MasterZoneId *string `json:"masterZoneId,omitempty"`
	// 仅适用于公网且IP版本为IPv4的负载均衡。可用区ID，指定可用区以创建负载均衡实例。
	ZoneId *string `json:"zoneId,omitempty"`
	// 在私有网络内购买内网负载均衡实例的情况下，必须指定子网 ID，内网负载均衡实例的 VIP 将从这个子网中产生。创建内网负载均衡实例时，此参数必填，创建公网IPv4负载均衡实例时，不支持指定该参数。
	SubnetId *string `json:"subnetId,omitempty"`
	// 性能容量型规格。
	// 若需要创建性能容量型实例，则此参数必填，取值范围：
	// clb.c2.medium：标准型规格
	// clb.c3.small：高阶型1规格
	// clb.c3.medium：高阶型2规格
	// clb.c4.small：超强型1规格
	// clb.c4.medium：超强型2规格
	// clb.c4.large：超强型3规格
	// clb.c4.xlarge：超强型4规格
	// 若需要创建共享型实例，则无需填写此参数。
	// +kubebuilder:validation:Enum=clb.c2.medium;clb.c3.small;clb.c3.medium;clb.c4.small;clb.c4.medium;clb.c4.large;clb.c4.xlarge
	SlaType *string `json:"slaType,omitempty"`
	// 负载均衡实例计费类型，取值：POSTPAID_BY_HOUR，PREPAID，默认是POSTPAID_BY_HOUR。
	// +kubebuilder:validation:Enum=POSTPAID_BY_HOUR;PREPAID
	LBChargeType *string `json:"lbChargeType,omitempty"`
	// 仅适用于公网负载均衡。负载均衡的网络计费模式。
	InternetAccessible *InternetAccessible `json:"internetAccessible,omitempty"`
}

// ExportCreateLoadBalancerRequest 转换为腾讯云CLB创建请求
func (p *CreateLBParameters) ExportCreateLoadBalancerRequest() *clb.CreateLoadBalancerRequest {
	req := clb.NewCreateLoadBalancerRequest()
	req.LoadBalancerType = util.GetPtr("OPEN") // 默认使用公网 CLB

	if p == nil { // 没指定参数，直接返回请求
		return req
	}

	// 直接映射相同字段
	if p.VipIsp != nil {
		req.VipIsp = p.VipIsp
	}
	if p.BandwidthPackageId != nil {
		req.BandwidthPackageId = p.BandwidthPackageId
	}
	if p.AddressIPVersion != nil {
		req.AddressIPVersion = p.AddressIPVersion
	}
	if p.LoadBalancerPassToTarget != nil {
		req.LoadBalancerPassToTarget = p.LoadBalancerPassToTarget
	}
	if p.DynamicVip != nil {
		req.DynamicVip = p.DynamicVip
	}
	if p.DynamicVip == nil {
		req.DynamicVip = util.GetPtr(true)
	}
	if p.VpcId != nil {
		req.VpcId = p.VpcId
	}
	if util.IsZero(req.VpcId) {
		req.VpcId = &clusterinfo.VpcId
	}
	if p.Vip != nil {
		req.Vip = p.Vip
	}
	if p.ProjectId != nil {
		req.ProjectId = p.ProjectId
	}
	if p.LoadBalancerName != nil {
		req.LoadBalancerName = p.LoadBalancerName
	}
	if p.LoadBalancerType != nil {
		req.LoadBalancerType = p.LoadBalancerType
	}
	if p.MasterZoneId != nil {
		req.MasterZoneId = p.MasterZoneId
	}
	if p.ZoneId != nil {
		req.ZoneId = p.ZoneId
	}
	if p.SubnetId != nil {
		req.SubnetId = p.SubnetId
	}
	if p.SlaType != nil {
		req.SlaType = p.SlaType
	}
	if p.LBChargeType != nil {
		req.LBChargeType = p.LBChargeType
	}

	// 处理嵌套结构 InternetAccessible
	if p.InternetAccessible != nil {
		req.InternetAccessible = &clb.InternetAccessible{
			InternetChargeType:      p.InternetAccessible.InternetChargeType,
			InternetMaxBandwidthOut: p.InternetAccessible.InternetMaxBandwidthOut,
			BandwidthpkgSubType:     p.InternetAccessible.BandwidthpkgSubType,
		}
	}

	if req.VipIsp != nil { // The parameter InternetAccessible.InternetChargeType must be BANDWIDTH_PACKAGE when specify parameter VipIsp
		if req.InternetAccessible == nil {
			req.InternetAccessible = &clb.InternetAccessible{}
		}
		req.InternetAccessible.InternetChargeType = util.GetPtr("BANDWIDTH_PACKAGE")
	}

	// 转换Tags
	if len(p.Tags) > 0 {
		req.Tags = make([]*clb.TagInfo, 0, len(p.Tags))
		for _, tag := range p.Tags {
			req.Tags = append(req.Tags, &clb.TagInfo{
				TagKey:   &tag.TagKey,
				TagValue: &tag.TagValue,
			})
		}
	}
	return req
}

// TagInfo 定义标签结构
type TagInfo struct {
	// 标签的键
	TagKey string `json:"tagKey"`
	// 标签的值
	TagValue string `json:"tagValue"`
}

// InternetAccessible 定义网络计费相关参数
type InternetAccessible struct {
	// TRAFFIC_POSTPAID_BY_HOUR 按流量按小时后计费 ; BANDWIDTH_POSTPAID_BY_HOUR 按带宽按小时后计费; BANDWIDTH_PACKAGE 按带宽包计费;BANDWIDTH_PREPAID按带宽预付费。注意：此字段可能返回 null，表示取不到有效值。
	// +kubebuilder:validation:Enum=TRAFFIC_POSTPAID_BY_HOUR;BANDWIDTH_POSTPAID_BY_HOUR;BANDWIDTH_PACKAGE;BANDWIDTH_PREPAID
	InternetChargeType *string `json:"internetChargeType,omitempty"`
	// 最大出带宽，单位Mbps，仅对公网属性的共享型、性能容量型和独占型 CLB 实例、以及内网属性的性能容量型 CLB 实例生效。
	// - 对于公网属性的共享型和独占型 CLB 实例，最大出带宽的范围为1Mbps-2048Mbps。
	// - 对于公网属性和内网属性的性能容量型 CLB实例，最大出带宽的范围为1Mbps-61440Mbps。
	// （调用CreateLoadBalancer创建LB时不指定此参数则设置为默认值10Mbps。此上限可调整）
	InternetMaxBandwidthOut *int64 `json:"internetMaxBandwidthOut,omitempty"`
	// 带宽包的类型，如SINGLEISP（单线）、BGP（多线）。
	// +kubebuilder:validation:Enum=SINGLEISP;BGP
	BandwidthpkgSubType *string `json:"bandwidthpkgSubType,omitempty"`
}

// CLBPortPoolStatus defines the observed state of CLBPortPool.
type CLBPortPoolStatus struct {
	// 状态: Pending/Active/Scaling
	State CLBPortPoolState `json:"state"`
	// 状态信息
	Message *string `json:"message,omitempty"`
	// 负载均衡器状态列表
	LoadbalancerStatuses []LoadBalancerStatus `json:"loadbalancerStatuses,omitempty"`
}

type CLBPortPoolState string

const (
	CLBPortPoolStatePending  CLBPortPoolState = "Pending"
	CLBPortPoolStateActive   CLBPortPoolState = "Active"
	CLBPortPoolStateScaling  CLBPortPoolState = "Scaling"
	CLBPortPoolStateDeleting CLBPortPoolState = "Deleting"
)

type LoadBalancerState string

const (
	LoadBalancerStateRunning  LoadBalancerState = "Running"
	LoadBalancerStateNotFound LoadBalancerState = "NotFound"
)

// LoadBalancerStatus 定义负载均衡器状态
type LoadBalancerStatus struct {
	AutoCreated      *bool             `json:"autoCreated,omitempty"`
	State            LoadBalancerState `json:"state"`
	LoadbalancerID   string            `json:"loadbalancerID"`
	LoadbalancerName string            `json:"loadbalancerName"`
	Ips              []string          `json:"ips,omitempty"`
	Hostname         *string           `json:"hostname,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=cpp
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State"

// CLBPortPool is the Schema for the clbportpools API.
type CLBPortPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CLBPortPoolSpec   `json:"spec,omitempty"`
	Status CLBPortPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CLBPortPoolList contains a list of CLBPortPool.
type CLBPortPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CLBPortPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CLBPortPool{}, &CLBPortPoolList{})
}
