package clb

import (
	"context"
	"errors"
	"fmt"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type Listener struct {
	Port       int64
	Protocol   string
	ListenerId string
}

func GetListener(ctx context.Context, region, lbId, listenerId string) (lis *Listener, err error) {
	req := clb.NewDescribeListenersRequest()
	req.LoadBalancerId = &lbId
	req.ListenerIds = []*string{&listenerId}
	client := GetClient(region)
	resp, err := client.DescribeListenersWithContext(ctx, req)
	if err != nil {
		return
	}
	if len(resp.Response.Listeners) == 0 {
		return
	}
	if len(resp.Response.Listeners) > 1 {
		err = fmt.Errorf("found %d listeners for %s", len(resp.Response.Listeners), listenerId)
		return
	}
	listener := resp.Response.Listeners[0]
	lis = &Listener{
		ListenerId: *listener.ListenerId,
		Protocol:   *listener.Protocol,
		Port:       *listener.Port,
	}
	return
}

func GetListenersByPort(ctx context.Context, region, lbId string, port int64) (listeners []*Listener, err error) {
	req := clb.NewDescribeListenersRequest()
	req.Port = &port
	req.LoadBalancerId = &lbId
	client := GetClient(region)
	resp, err := client.DescribeListenersWithContext(ctx, req)
	if err != nil {
		return
	}
	for _, lis := range resp.Response.Listeners {
		listeners = append(listeners, &Listener{
			ListenerId: *lis.ListenerId,
			Protocol:   *lis.Protocol,
			Port:       *lis.Port,
		})
	}
	return
}

const TkePodListenerName = "TKE-DEDICATED-POD"

func CreateListener(ctx context.Context, region string, req *clb.CreateListenerRequest) (id string, err error) {
	req.ListenerNames = []*string{common.StringPtr(TkePodListenerName)}
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
	err = Wait(ctx, region, *resp.Response.RequestId)
	if err != nil {
		return
	}
	id = *resp.Response.ListenerIds[0]
	return
}

func DeleteListenerByPort(ctx context.Context, region, lbId string, port int64, protocol string) error {
	lis, err := GetListenersByPort(ctx, region, lbId, port)
	if err != nil {
		return err
	}
	id := ""
	for _, l := range lis {
		if l.Protocol == protocol && port == l.Port {
			id = l.ListenerId
			break
		}
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
	resp, err := client.DeleteListenerWithContext(ctx, req)
	if err != nil {
		return err
	}
	if err := Wait(ctx, region, *resp.Response.RequestId); err != nil {
		return err
	}
	return err
}
