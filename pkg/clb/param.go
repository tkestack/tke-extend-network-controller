package clb

import (
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
)

func ConvertCreateLoadBalancerRequest(p *networkingv1alpha1.CreateLBParameters) *clb.CreateLoadBalancerRequest {
	req := clb.NewCreateLoadBalancerRequest()
	req.LoadBalancerType = util.GetPtr("OPEN") // 默认使用公网 CLB
	req.VpcId = &clusterinfo.VpcId

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
	if p.VpcId != nil {
		req.VpcId = p.VpcId
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
		req.LoadBalancerType = util.GetPtr("INTERNAL") // 指定了子网，必须是内网 CLB
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
	if req.DynamicVip == nil { // 默认不用域名化的 CLB
		req.DynamicVip = common.BoolPtr(false)
	}
	if req.LoadBalancerPassToTarget == nil { // 默认放通后端
		req.LoadBalancerPassToTarget = common.BoolPtr(true)
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
