package clb

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tag "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"github.com/tkestack/tke-extend-network-controller/pkg/cloudapi"
	"github.com/tkestack/tke-extend-network-controller/pkg/userinfo"
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
	LogAPI(ctx, false, "DescribeLoadBalancers", req, resp, time.Since(before), err)
	if err != nil {
		return
	}
	if *resp.Response.TotalCount == 0 {
		return nil, ErrLbIdNotFound
	}
	instance = resp.Response.LoadBalancerSet[0]
	return
}

// ListCLBsByTags lists CLBs by tags filter
func ListCLBsByTags(ctx context.Context, region string, tags map[string]string) ([]*clb.LoadBalancer, error) {
	client := GetClient(region)
	req := clb.NewDescribeLoadBalancersRequest()

	filters := []*clb.Filter{}
	for k, v := range tags {
		filters = append(filters, &clb.Filter{
			Name:   common.StringPtr("tag:" + k),
			Values: []*string{common.StringPtr(v)},
		})
	}
	req.Filters = filters

	var lbs []*clb.LoadBalancer
	var offset int64 = 0
	var limit int64 = 100

	for {
		req.Offset = common.Int64Ptr(offset)
		req.Limit = common.Int64Ptr(limit)

		before := time.Now()
		resp, err := client.DescribeLoadBalancersWithContext(ctx, req)
		LogAPI(ctx, false, "DescribeLoadBalancers", req, resp, time.Since(before), err)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		lbs = append(lbs, resp.Response.LoadBalancerSet...)

		if len(resp.Response.LoadBalancerSet) < int(limit) {
			break
		}
		offset += limit
	}

	return lbs, nil
}

// EnsureCLBTags ensures CLB has the specified tags
func EnsureCLBTags(ctx context.Context, region, lbId string, tags map[string]string) error {
	client, err := tag.NewClient(cloudapi.GetCredential(), "", profile.NewClientProfile())
	if err != nil {
		return errors.WithStack(err)
	}
	req := tag.NewTagResourcesRequest()
	req.ResourceList = []*string{common.StringPtr(fmt.Sprintf("qcs::clb:%s:uin/%s:clb/%s", region, userinfo.OwnerUin, lbId))}
	for k, v := range tags {
		req.Tags = append(req.Tags, &tag.Tag{
			TagKey:   common.StringPtr(k),
			TagValue: common.StringPtr(v),
		})
	}
	before := time.Now()
	resp, err := client.TagResources(req)
	LogAPI(ctx, true, "TagResources", req, resp, time.Since(before), err)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
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
	LogAPI(ctx, true, "DeleteLoadBalancer", req, resp, time.Since(before), err)
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
	_, err = Wait(ctx, region, *resp.Response.RequestId, "DeleteLoadBalancer", LongWaitInterval)
	return err
}

// 创建单个 CLB
func CreateCLB(ctx context.Context, region string, req *clb.CreateLoadBalancerRequest) (lbId string, err error) {
	log.FromContext(ctx).V(3).Info("CreateLoadBalancer", "req", *req)
	client := GetClient(region)
	before := time.Now()
	resp, err := client.CreateLoadBalancerWithContext(ctx, req)
	LogAPI(ctx, true, "CreateLoadBalancer", req, resp, time.Since(before), err)
	if err != nil {
		return
	}
	ids := resp.Response.LoadBalancerIds
	if len(ids) == 0 || *ids[0] == "" {
		err = fmt.Errorf("no loadbalancer created")
		return
	}
	lbId = *ids[0]
	if _, err = Wait(ctx, region, *resp.Response.RequestId, "CreateLoadBalancer", LongWaitInterval); err != nil {
		return
	}
	return
}

type CLBInfo struct {
	LoadbalancerID   string
	LoadbalancerName string
	Ips              []string
	Hostname         *string
	Tags             map[string]string
}

func getTagsMap(tags []*clb.TagInfo) map[string]string {
	m := make(map[string]string)
	for _, tag := range tags {
		m[*tag.TagKey] = *tag.TagValue
	}
	return m
}

func BatchGetClbInfo(ctx context.Context, lbIds []string, region string) (info map[string]*CLBInfo, err error) {
	info = make(map[string]*CLBInfo)
	insIds := []*string{}
	for len(lbIds) > 0 {
		lbs := lbIds
		if len(lbIds) > 19 { // 分页查询
			lbs = lbIds[:19]
			lbIds = lbIds[19:]
		} else {
			lbIds = nil
		}
		res, err := ApiCall(context.Background(), false, "DescribeLoadBalancers", region, func(ctx context.Context, client *clb.Client) (req *clb.DescribeLoadBalancersRequest, res *clb.DescribeLoadBalancersResponse, err error) {
			req = clb.NewDescribeLoadBalancersRequest()
			req.LoadBalancerIds = common.StringPtrs(lbs)
			res, err = client.DescribeLoadBalancersWithContext(ctx, req)
			return
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if *res.Response.TotalCount == 0 || len(res.Response.LoadBalancerSet) == 0 {
			return nil, nil
		}
		for _, ins := range res.Response.LoadBalancerSet {
			lbInfo := &CLBInfo{
				LoadbalancerID:   *ins.LoadBalancerId,
				LoadbalancerName: *ins.LoadBalancerName,
				Tags:             getTagsMap(ins.Tags),
			}
			if util.GetValue(ins.Domain) != "" {
				lbInfo.Hostname = ins.Domain
			} else {
				lbInfo.Ips = util.ConvertPtrSlice(ins.LoadBalancerVips)
			}
			info[*ins.LoadBalancerId] = lbInfo
			insIds = append(insIds, ins.LoadBalancerId)
		}
	}
	vpcClient := vpcpkg.GetClient(region)
	for len(insIds) > 0 {
		ids := insIds
		if len(insIds) > 99 { // 分页查询
			ids = insIds[:99]
			insIds = insIds[99:]
		} else {
			insIds = nil
		}
		addrResp, err := ApiCall(context.Background(), false, "DescribeAddresses", region, func(ctx context.Context, client *clb.Client) (req *vpc.DescribeAddressesRequest, res *vpc.DescribeAddressesResponse, err error) {
			req = vpc.NewDescribeAddressesRequest()
			req.Filters = []*vpc.Filter{
				{
					Name:   common.StringPtr("instance-id"),
					Values: ids,
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
	}
	return
}
