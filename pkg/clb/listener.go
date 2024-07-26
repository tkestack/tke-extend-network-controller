package clb

import (
	"context"
	"errors"
	"fmt"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

func GetListenerId(ctx context.Context, region, lbId string, port int64, protocol string) (id string, err error) {
	req := clb.NewDescribeListenersRequest()
	req.Protocol = &protocol
	req.Port = &port
	req.LoadBalancerId = &lbId
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

func CreateListener(ctx context.Context, region string, req *clb.CreateListenerRequest) (id string, err error) {
	req.ListenerNames = []*string{common.StringPtr("TKE-DEDICATED-LISTENER")}
	client := GetClient(region)
	resp, err := client.CreateListenerWithContext(ctx, req)
	if err != nil {
		return
	}
	if len(resp.Response.ListenerIds) == 0 {
		err = errors.New("no listener created")
		return
	}
	if len(resp.Response.ListenerIds) > 1 {
		err = fmt.Errorf("found %d listeners created", len(resp.Response.ListenerIds))
		return
	}
	id = *resp.Response.ListenerIds[0]
	return
}

func DeleteListenerByPort(ctx context.Context, region, lbId string, port int64, protocol string) error {
	id, err := GetListenerId(ctx, region, lbId, port, protocol)
	if err != nil {
		return err
	}
	if id == "" { // 监听器不存在，忽略
		return nil
	}
	return DeleteListener(ctx, region, lbId, id)
}

func DeleteListener(ctx context.Context, region, lbId, listenerId string) error {
	req := clb.NewDeleteListenerRequest()
	req.LoadBalancerId = &lbId
	req.ListenerId = &listenerId
	client := GetClient(region)
	_, err := client.DeleteListenerWithContext(ctx, req)
	return err
}
