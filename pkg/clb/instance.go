package clb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/imroc/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	vpcpkg "github.com/imroc/tke-extend-network-controller/pkg/vpc"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func GetClbExternalAddress(ctx context.Context, lbId, region string) (address string, err error) {
	lb, err := GetClb(ctx, lbId, region)
	if err != nil {
		return
	}
	if lb.LoadBalancerDomain != nil && *lb.LoadBalancerDomain != "" {
		address = *lb.LoadBalancerDomain
		return
	}
	if len(lb.LoadBalancerVips) > 0 {
		address = *lb.LoadBalancerVips[0]
		return
	}
	err = fmt.Errorf("no external address found for clb %s", lbId)
	return
}

var ErrLbIdNotFound = errors.New("lb id not found")

func GetClb(ctx context.Context, lbId, region string) (instance *clb.LoadBalancer, err error) {
	client := GetClient(region)
	req := clb.NewDescribeLoadBalancersRequest()
	req.LoadBalancerIds = []*string{&lbId}

	resp, err := client.DescribeLoadBalancersWithContext(ctx, req)
	LogAPI(ctx, "DescribeLoadBalancers", req, resp, err)
	if err != nil {
		return
	}
	if *resp.Response.TotalCount == 0 {
		return nil, ErrLbIdNotFound
	}
	instance = resp.Response.LoadBalancerSet[0]
	return
}

// TODO: 支持部分成功
func Create(ctx context.Context, region, vpcId, extensiveParameters string, num int) (ids []string, err error) {
	if vpcId == "" {
		vpcId = clusterinfo.VpcId
	}
	req := clb.NewCreateLoadBalancerRequest()
	req.LoadBalancerType = common.StringPtr("OPEN")
	req.VpcId = &vpcId
	req.Number = common.Uint64Ptr(uint64(num))
	req.Tags = append(req.Tags,
		&clb.TagInfo{
			TagKey:   common.StringPtr("tke-clusterId"), // 与集群关联
			TagValue: common.StringPtr(clusterinfo.ClusterId),
		},
		&clb.TagInfo{
			TagKey:   common.StringPtr("tke-createdBy-flag"), // 用于删除集群时自动清理集群关联的自动创建的 CLB
			TagValue: common.StringPtr("yes"),
		},
	)
	if extensiveParameters != "" {
		err = json.Unmarshal([]byte(extensiveParameters), req)
		if err != nil {
			return
		}
	}
	client := GetClient(region)
	resp, err := client.CreateLoadBalancerWithContext(ctx, req)
	LogAPI(ctx, "CreateLoadBalancer", req, resp, err)
	if err != nil {
		return
	}
	ids = util.ConvertPtrSlice(resp.Response.LoadBalancerIds)
	if len(ids) == 0 {
		ids, err = Wait(ctx, region, *resp.Response.RequestId, "CreateLoadBalancer")
		if err != nil {
			return nil, err
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no loadbalancer created")
	}
	for _, lbId := range ids {
		for {
			lb, err := GetClb(ctx, lbId, region)
			if err != nil {
				return nil, err
			}
			if *lb.Status == 0 { // 创建中，等待一下
				log.FromContext(ctx).V(5).Info("lb is still creating", "lbId", lbId)
				time.Sleep(time.Second * 3)
				continue
			}
			break
		}
	}
	return
}

func Delete(ctx context.Context, region string, lbIds ...string) error {
	req := clb.NewDeleteLoadBalancerRequest()
	for _, lbId := range lbIds {
		req.LoadBalancerIds = append(req.LoadBalancerIds, &lbId)
	}
	client := GetClient(region)
	resp, err := client.DeleteLoadBalancerWithContext(ctx, req)
	LogAPI(ctx, "DeleteLoadBalancer", req, resp, err)
	if err != nil {
		if IsLbIdNotFoundError(err) {
			if len(lbIds) == 1 { // lb 已全部删除，忽略
				return nil
			} else { // lb 可能全部删除，也可能部分删除，挨个尝试一下
				for _, lbId := range lbIds {
					if err := Delete(ctx, region, lbId); err != nil {
						return err
					}
				}
				return nil
			}
		}
		return err
	}
	_, err = Wait(ctx, region, *resp.Response.RequestId, "DeleteLoadBalancer")
	return err
}

// 创建单个 CLB
func CreateCLB(ctx context.Context, region string, req *clb.CreateLoadBalancerRequest) (lbId string, err error) {
	log.FromContext(ctx).V(10).Info("CreateLoadBalancer", "req", *req)
	client := GetClient(region)
	resp, err := client.CreateLoadBalancerWithContext(ctx, req)
	if err != nil {
		return
	}
	ids := resp.Response.LoadBalancerIds
	if len(ids) == 0 || *ids[0] == "" {
		err = fmt.Errorf("no loadbalancer created")
		return
	}
	lbId = *ids[0]
	if _, err = Wait(ctx, region, *resp.Response.RequestId, "CreateLoadBalancer"); err != nil {
		return
	}
	return
}

func CreateCLBAsync(ctx context.Context, region string, req *clb.CreateLoadBalancerRequest) (lbId string, result <-chan error, err error) {
	client := GetClient(region)
	resp, err := client.CreateLoadBalancerWithContext(ctx, req)
	if err != nil {
		return
	}
	ids := resp.Response.LoadBalancerIds
	if len(ids) == 0 || *ids[0] == "" {
		err = fmt.Errorf("no loadbalancer created")
		return
	}
	lbId = *ids[0]
	ret := make(chan error)
	result = ret
	go func() {
		if _, err := Wait(ctx, region, *resp.Response.RequestId, "CreateLoadBalancer"); err != nil {
			log.FromContext(ctx).Error(err, "clb create task failed", "lbId", lbId)
			ret <- err
			return
		}
		ret <- nil
	}()
	return
}

type CLBInfo struct {
	LoadbalancerID   string
	LoadbalancerName string
	Ips              []string
	Hostname         *string
}

func BatchGetClbInfo(ctx context.Context, lbIds []string, region string) (info map[string]*CLBInfo, err error) {
	client := GetClient(region)
	req := clb.NewDescribeLoadBalancersRequest()
	req.LoadBalancerIds = common.StringPtrs(lbIds)
	resp, err := client.DescribeLoadBalancersWithContext(ctx, req)
	LogAPI(ctx, "DescribeLoadBalancers", req, resp, err)
	if err != nil {
		return
	}
	if *resp.Response.TotalCount == 0 || len(resp.Response.LoadBalancerSet) == 0 {
		return
	}
	info = make(map[string]*CLBInfo)
	insIds := []*string{}
	for _, ins := range resp.Response.LoadBalancerSet {
		info[*ins.LoadBalancerId] = &CLBInfo{
			LoadbalancerID:   *ins.LoadBalancerId,
			LoadbalancerName: *ins.LoadBalancerName,
			Ips:              util.ConvertPtrSlice(ins.LoadBalancerVips),
			Hostname:         ins.Domain,
		}
		insIds = append(insIds, ins.LoadBalancerId)
	}
	vpcClient := vpcpkg.GetClient(region)
	addrReq := vpc.NewDescribeAddressesRequest()
	addrReq.Filters = []*vpc.Filter{
		{
			Name:   common.StringPtr("instance-id"),
			Values: insIds,
		},
	}
	addrResp, err := vpcClient.DescribeAddressesWithContext(ctx, addrReq)
	if err != nil {
		return
	}
	vpcpkg.LogAPI(ctx, "DescribeAddresses", addrReq, addrResp, err)
	for _, addr := range addrResp.Response.AddressSet {
		if !util.IsZero(addr.InstanceId) {
			if lbInfo, ok := info[*addr.InstanceId]; ok {
				lbInfo.Ips = []string{*addr.AddressIp}
			}
		}
	}
	return
}
