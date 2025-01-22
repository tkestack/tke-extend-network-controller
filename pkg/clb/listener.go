package clb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type Listener struct {
	CLB
	ListenerId string
}

type ListenerInfo struct {
	Listener
	Port         int64
	Protocol     string
	ListenerId   string
	ListenerName string
}

type ListenerPort struct {
	Port     int64
	Protocol string
}

// func GetListener(ctx context.Context, region, lbId, listenerId string) (lis *ListenerInfo, err error) {
// 	req := clb.NewDescribeListenersRequest()
// 	req.LoadBalancerId = &lbId
// 	req.ListenerIds = []*string{&listenerId}
// 	client := GetClient(region)
// 	resp, err := client.DescribeListenersWithContext(ctx, req)
// 	if err != nil {
// 		return
// 	}
// 	if len(resp.Response.Listeners) == 0 {
// 		return
// 	}
// 	if len(resp.Response.Listeners) > 1 {
// 		err = fmt.Errorf("found %d listeners for %s", len(resp.Response.Listeners), listenerId)
// 		return
// 	}
// 	lis = convertListener(resp.Response.Listeners[0])
// 	return
// }

func convertListener(lbLis *clb.Listener) *ListenerInfo {
	return &ListenerInfo{
		Listener: Listener{
			ListenerId: *lbLis.ListenerId,
		},
		ListenerName: *lbLis.ListenerName,
		Protocol:     *lbLis.Protocol,
		Port:         *lbLis.Port,
	}
}

func GetListenerInfoByPort(ctx context.Context, lb CLB, port ListenerPort) (lis *ListenerInfo, err error) {
	req := clb.NewDescribeListenersRequest()
	req.Port = &port.Port
	req.LoadBalancerId = &lb.LbId
	req.Protocol = &port.Protocol
	client := GetClient(lb.Region)
	resp, err := client.DescribeListenersWithContext(ctx, req)
	if err != nil {
		return
	}
	if len(resp.Response.Listeners) > 0 { // TODO: 精细化判断数量(超过1个的不可能发生的情况)
		lis = convertListener(resp.Response.Listeners[0])
		lis.CLB = lb
		return
	}
	return
}

const TkePodListenerName = "TKE-DEDICATED-POD"

func CreateListener(ctx context.Context, region, lbId string, port int64, endPort *int64, protocol string, extensiveParameters *string) (id string, err error) {
	req := clb.NewCreateListenerRequest()
	req.HealthCheck = &clb.HealthCheck{
		HealthSwitch: common.Int64Ptr(0),
		SourceIpType: common.Int64Ptr(1),
	}
	if extensiveParameters != nil && len(*extensiveParameters) > 0 {
		err = json.Unmarshal([]byte(*extensiveParameters), req)
		if err != nil {
			return
		}
	}
	req.LoadBalancerId = &lbId
	req.Ports = []*int64{&port}
	if endPort != nil {
		req.EndPort = common.Uint64Ptr(uint64(*endPort))
	}
	req.Protocol = &protocol
	req.ListenerNames = []*string{common.StringPtr(TkePodListenerName)}
	client := GetClient(region)
	mu := getLbLock(lbId)
	mu.Lock()
	defer mu.Unlock()
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
	_, err = Wait(ctx, region, *resp.Response.RequestId, "CreateListener")
	if err != nil {
		return
	}
	id = *resp.Response.ListenerIds[0]
	return
}

func DeleteListenerByPort(ctx context.Context, lb CLB, port ListenerPort) (id string, err error) {
	lis, err := GetListenerInfoByPort(ctx, lb, port)
	if err != nil {
		return
	}
	if lis == nil { // 监听器不存在，忽略
		return
	}
	id = lis.ListenerId
	err = DeleteListener(ctx, lis.Listener)
	return
}

func DeleteListener(ctx context.Context, lis Listener) error {
	req := clb.NewDeleteListenerRequest()
	req.LoadBalancerId = &lis.LbId
	req.ListenerId = &lis.ListenerId
	client := GetClient(lis.Region)
	mu := getLbLock(lis.LbId)
	mu.Lock()
	defer mu.Unlock()
	resp, err := client.DeleteListenerWithContext(ctx, req)
	if err != nil {
		return err
	}
	_, err = Wait(ctx, lis.Region, *resp.Response.RequestId, "DeleteListener")
	return err
}
