package clb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type Listener struct {
	Port         int64
	EndPort      int64
	Protocol     string
	ListenerId   string
	ListenerName string
}

func GetListenerNum(ctx context.Context, region, lbId string) (num int64, err error) {
	req := clb.NewDescribeListenersRequest()
	req.LoadBalancerId = &lbId
	client := GetClient(region)
	resp, err := client.DescribeListenersWithContext(ctx, req)
	if err != nil {
		return
	}
	num = int64(len(resp.Response.Listeners))
	return
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
	lis = convertListener(resp.Response.Listeners[0])
	return
}

func convertListener(lbLis *clb.Listener) *Listener {
	lis := &Listener{
		ListenerId:   *lbLis.ListenerId,
		ListenerName: *lbLis.ListenerName,
		Protocol:     *lbLis.Protocol,
		Port:         *lbLis.Port,
	}
	if lbLis.EndPort != nil {
		lis.EndPort = *lbLis.EndPort
	}
	return lis
}

func GetListenerByPort(ctx context.Context, region, lbId string, port int64, protocol string) (lis *Listener, err error) {
	req := clb.NewDescribeListenersRequest()
	req.Port = &port
	req.LoadBalancerId = &lbId
	req.Protocol = &protocol
	client := GetClient(region)
	resp, err := client.DescribeListenersWithContext(ctx, req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(resp.Response.Listeners) > 0 { // TODO: 精细化判断数量(超过1个的不可能发生的情况)
		lis = convertListener(resp.Response.Listeners[0])
		return
	}
	return
}

const TkePodListenerName = "TKE-DEDICATED-POD"

func CreateListener(ctx context.Context, region, lbId string, port, endPort int64, protocol, extensiveParameters string) (id string, err error) {
	req := clb.NewCreateListenerRequest()
	req.HealthCheck = &clb.HealthCheck{
		HealthSwitch: common.Int64Ptr(0),
		SourceIpType: common.Int64Ptr(1),
	}
	if extensiveParameters != "" {
		err = json.Unmarshal([]byte(extensiveParameters), req)
		if err != nil {
			err = errors.WithStack(err)
			return
		}
	}
	req.LoadBalancerId = &lbId
	req.Ports = []*int64{&port}
	if endPort > 0 {
		req.EndPort = common.Uint64Ptr(uint64(endPort))
	}
	req.Protocol = &protocol
	req.ListenerNames = []*string{common.StringPtr(TkePodListenerName)}
	client := GetClient(region)
	mu := getLbLock(lbId)
	mu.Lock()
	defer mu.Unlock()
	resp, err := client.CreateListenerWithContext(ctx, req)
	if err != nil {
		err = errors.WithStack(err)
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
	_, err = Wait(ctx, region, *resp.Response.RequestId, "CreateListener")
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	id = *resp.Response.ListenerIds[0]
	return
}

func DeleteListenerByPort(ctx context.Context, region, lbId string, port int64, protocol string) (id string, err error) {
	lis, err := GetListenerByPort(ctx, region, lbId, port, protocol)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	if lis == nil { // 监听器不存在，忽略
		return
	}
	id = lis.ListenerId
	err = DeleteListener(ctx, region, lbId, id)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	return
}

func DeleteListener(ctx context.Context, region, lbId, listenerId string) error {
	req := clb.NewDeleteListenerRequest()
	req.LoadBalancerId = &lbId
	req.ListenerId = &listenerId
	client := GetClient(region)
	mu := getLbLock(lbId)
	mu.Lock()
	defer mu.Unlock()
	resp, err := client.DeleteListenerWithContext(ctx, req)
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = Wait(ctx, region, *resp.Response.RequestId, "DeleteListener")
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
