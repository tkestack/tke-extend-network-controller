package clb

import (
	"context"
	"errors"
	"fmt"
	"time"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

func GetClbExternalAddress(ctx context.Context, lbId, region string) (address string, err error) {
	lb, err := GetClb(ctx, lbId, region)
	if err != nil {
		return
	}
	if lb.LoadBalancerDomain != nil {
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

func GetClb(ctx context.Context, lbId, region string) (instance *clb.LoadBalancer, err error) {
	client := GetClient(region)
	req := clb.NewDescribeLoadBalancersRequest()
	req.LoadBalancerIds = []*string{&lbId}
	resp, err := client.DescribeLoadBalancersWithContext(ctx, req)
	if err != nil {
		return
	}
	if *resp.Response.TotalCount == 0 {
		err = fmt.Errorf("%s is not exists in %s", lbId, region)
		return
	} else if *resp.Response.TotalCount > 1 {
		err = fmt.Errorf("%s found %d instances in %s", lbId, *resp.Response.TotalCount, region)
		return
	}
	instance = resp.Response.LoadBalancerSet[0]
	return
}

func Create(ctx context.Context, region, vpcId, name string) (lbId string, err error) {
	if vpcId == "" {
		vpcId = defaultVpcId
	}
	req := clb.NewCreateLoadBalancerRequest()
	req.LoadBalancerType = common.StringPtr("OPEN")
	req.VpcId = &vpcId
	client := GetClient(region)
	resp, err := client.CreateLoadBalancer(req)
	if err != nil {
		return
	}
	ids := resp.Response.LoadBalancerIds
	if len(ids) == 0 {
		err = errors.New("no loadbalancer created")
		return
	}
	if len(ids) > 1 {
		err = fmt.Errorf("multiple loadbalancers created: %v", ids)
		return
	}
	lbId = *ids[0]
	for {
		lb, err := GetClb(ctx, lbId, region)
		if err != nil {
			return "", err
		}
		if *lb.Status == 0 { // 创建中，等待一下
			time.Sleep(time.Second * 3)
			continue
		}
		break
	}
	return
}
