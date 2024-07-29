package clb

import (
	"fmt"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

func GetClb(lbId, region string) (instance *clb.LoadBalancer, err error) {
	client := GetClient(region)
	req := clb.NewDescribeLoadBalancersRequest()
	req.LoadBalancerIds = []*string{&lbId}
	resp, err := client.DescribeLoadBalancers(req)
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

func Create(region string) (err error) {
	req := clb.NewCreateLoadBalancerRequest()
	req.LoadBalancerType = common.StringPtr("OPEN")
	return
}
