package clb

import (
	"context"
	"fmt"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

func GetListenerId(ctx context.Context, region, lbId string, port int64, protocol string) (id string, err error) {
	req := clb.NewDescribeListenersRequest()
	req.Protocol = common.StringPtr(protocol)
	req.Port = common.Int64Ptr(port)
	req.LoadBalancerId = common.StringPtr(lbId)
	client := GetClient(region)
	resp, err := client.DescribeListenersWithContext(ctx, req)
	if err != nil {
		return
	}
	if len(resp.Response.Listeners) == 0 {
		return
	}
	if len(resp.Response.Listeners) > 1 {
		err = fmt.Errorf("found %d listeners for %d/%s", len(resp.Response.Listeners), port, protocol)
		return
	}
	listener := resp.Response.Listeners[0]
	id = *listener.ListenerId
	return
}
