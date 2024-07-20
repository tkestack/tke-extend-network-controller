package clb

import (
	"context"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

func ContainsRs(ctx context.Context, region, lbId string, port int64, protocol, rsIp string, rsPort int64) (bool, error) {
	req := clb.NewDescribeTargetsRequest()
	req.Protocol = common.StringPtr(protocol)
	req.Port = common.Int64Ptr(port)
	req.Filters = []*clb.Filter{
		{
			Name:   common.StringPtr("private-ip-address"),
			Values: []*string{common.StringPtr(rsIp)},
		},
	}
	client := GetClient(region)
	resp, err := client.DescribeTargets(req)
	if err != nil {
		return false, err
	}
	for _, listener := range resp.Response.Listeners {
		if *listener.Protocol == protocol && *listener.Port == port {
			for _, rs := range listener.Targets {
				if *rs.Port == rsPort {
					for _, ip := range rs.PrivateIpAddresses {
						if *ip == rsIp {
							return true, nil
						}
					}
				}
			}
		}
	}
	return false, nil
}
