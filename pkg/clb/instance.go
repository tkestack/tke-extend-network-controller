package clb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
	vpcpkg "github.com/tkestack/tke-extend-network-controller/pkg/vpc"
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

	before := time.Now()
	resp, err := client.DescribeLoadBalancersWithContext(ctx, req)
	LogAPI(ctx, "DescribeLoadBalancers", req, resp, time.Since(before), err)
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
	before := time.Now()
	resp, err := client.CreateLoadBalancerWithContext(ctx, req)
	LogAPI(ctx, "CreateLoadBalancer", req, resp, time.Since(before), err)
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
		req.ForceDelete = common.BoolPtr(true)
	}
	client := GetClient(region)
	before := time.Now()
	resp, err := client.DeleteLoadBalancerWithContext(ctx, req)
	LogAPI(ctx, "DeleteLoadBalancer", req, resp, time.Since(before), err)
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
	before := time.Now()
	resp, err := client.CreateLoadBalancerWithContext(ctx, req)
	LogAPI(ctx, "CreateLoadBalancer", req, resp, time.Since(before), err)
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

type CLBInfo struct {
	LoadbalancerID   string
	LoadbalancerName string
	Ips              []string
	Hostname         *string
}

func BatchGetClbInfo(ctx context.Context, lbIds []string, region string) (info map[string]*CLBInfo, err error) {
	res, err := ApiCall(context.Background(), "DescribeLoadBalancers", region, func(ctx context.Context, client *clb.Client) (req *clb.DescribeLoadBalancersRequest, res *clb.DescribeLoadBalancersResponse, err error) {
		req = clb.NewDescribeLoadBalancersRequest()
		req.LoadBalancerIds = common.StringPtrs(lbIds)
		res, err = client.DescribeLoadBalancersWithContext(ctx, req)
		return
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if *res.Response.TotalCount == 0 || len(res.Response.LoadBalancerSet) == 0 {
		return
	}
	info = make(map[string]*CLBInfo)
	insIds := []*string{}
	for _, ins := range res.Response.LoadBalancerSet {
		lbInfo := &CLBInfo{
			LoadbalancerID:   *ins.LoadBalancerId,
			LoadbalancerName: *ins.LoadBalancerName,
		}
		if !util.IsZero(ins.Domain) {
			lbInfo.Hostname = ins.Domain
		} else {
			vips := util.ConvertPtrSlice(ins.LoadBalancerVips)
			if len(vips) > 0 {
				lbInfo.Ips = vips
			}
		}
		info[*ins.LoadBalancerId] = lbInfo
		insIds = append(insIds, ins.LoadBalancerId)
	}
	vpcClient := vpcpkg.GetClient(region)
	addrResp, err := ApiCall(context.Background(), "DescribeAddresses", region, func(ctx context.Context, client *clb.Client) (req *vpc.DescribeAddressesRequest, res *vpc.DescribeAddressesResponse, err error) {
		req = vpc.NewDescribeAddressesRequest()
		req.Filters = []*vpc.Filter{
			{
				Name:   common.StringPtr("instance-id"),
				Values: insIds,
			},
		}
		res, err = vpcClient.DescribeAddressesWithContext(ctx, req)
		return
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for _, addr := range addrResp.Response.AddressSet {
		log.FromContext(ctx).V(3).Info("got clb eip addr", "instanceId", addr.InstanceId)
		if addr.InstanceId != nil {
			if lbInfo, ok := info[*addr.InstanceId]; ok {
				log.FromContext(ctx).V(3).Info("set clb eip addr", "instanceId", addr.InstanceId, "eip", *addr.AddressIp)
				lbInfo.Ips = []string{*addr.AddressIp}
			}
		}
	}
	return
}
