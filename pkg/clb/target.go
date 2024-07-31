package clb

import (
	"context"
	"fmt"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

func ContainsTarget(ctx context.Context, region, lbId string, port int64, protocol string, target Target) (bool, error) {
	req := clb.NewDescribeTargetsRequest()
	req.LoadBalancerId = &lbId
	req.Protocol = &protocol
	req.Port = &port
	req.Filters = []*clb.Filter{
		{
			Name:   common.StringPtr("private-ip-address"),
			Values: []*string{&target.TargetIP},
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
				if *rs.Port == int64(target.TargetPort) {
					for _, ip := range rs.PrivateIpAddresses {
						if *ip == target.TargetIP {
							return true, nil
						}
					}
				}
			}
		}
	}
	return false, nil
}

type BatchTarget struct {
	ListenerProtocol string
	ListenerPort     int64
	Target
}

// func BatchDeregisterTargets(ctx context.Context, region, lbId string, targets ...BatchTarget) (err error) {
// 	type listenerKey struct {
// 		protocol string
// 		port     int64
// 	}
// 	var allError []error
// 	listenerIds := make(map[listenerKey]string)
// 	var batchTargets []*clb.BatchTarget
// 	for _, target := range targets {
// 		k := listenerKey{port: target.ListenerPort, protocol: target.ListenerProtocol}
// 		id, ok := listenerIds[k]
// 		if !ok {
// 			id, err = GetListenerId(ctx, region, lbId, target.ListenerPort, target.ListenerProtocol)
// 			if err != nil {
// 				return
// 			}
// 			listenerIds[k] = id
// 		}
// 		if id == "" {
// 			allError = append(allError, fmt.Errorf("listener not found: %d/%s", target.ListenerPort, target.ListenerProtocol))
// 			continue
// 		}
// 		batchTargets = append(batchTargets, &clb.BatchTarget{
// 			ListenerId: &id,
// 			Port:       &target.ListenerPort,
// 			EniIp:      &target.TargetIP,
// 		})
// 	}
// 	if len(batchTargets) > 0 {
// 		req := clb.NewBatchDeregisterTargetsRequest()
// 		req.LoadBalancerId = &lbId
// 		req.Targets = batchTargets
// 		client := GetClient(region)
// 		resp, err := client.BatchDeregisterTargetsWithContext(ctx, req)
// 		if err != nil {
// 			allError = append(allError, err)
// 		}
// 		if failedIds := resp.Response.FailListenerIdSet; len(failedIds) > 0 {
// 			allError = append(allError, fmt.Errorf("batch deregister targets failed: %v", util.ConvertStringPointSlice(failedIds)))
// 		}
// 	}
// 	if len(allError) > 0 {
// 		err = errors.Join(allError...)
// 	}
// 	return
// }

type Target struct {
	TargetIP   string
	TargetPort int64
}

func (t Target) String() string {
	return fmt.Sprintf("%s:%d", t.TargetIP, t.TargetPort)
}

func DeregisterAllTargets(ctx context.Context, region, lbId, listenerId string) error {
	queryReq := clb.NewDescribeTargetsRequest()
	queryReq.LoadBalancerId = &lbId
	queryReq.ListenerIds = []*string{&listenerId}
	client := GetClient(region)
	resp, err := client.DescribeTargetsWithContext(ctx, queryReq)
	if err != nil {
		return err
	}
	var targets []Target
	for _, lis := range resp.Response.Listeners {
		if listenerId != *lis.ListenerId {
			return fmt.Errorf("found targets not belong to listener %s/%s targets when deregister", lbId, listenerId)
		}
		for _, target := range lis.Targets {
			for _, ip := range target.PrivateIpAddresses {
				targets = append(targets, Target{TargetIP: *ip, TargetPort: int64(*target.Port)})
			}
		}
	}
	if len(targets) > 0 {
		return DeregisterTargetsForListener(ctx, region, lbId, listenerId, targets...)
	}
	return nil
}

func DeregisterTargetsForListener(ctx context.Context, region, lbId, listenerId string, targets ...Target) error {
	clbTargets := getClbTargets(targets)
	req := clb.NewDeregisterTargetsRequest()
	req.LoadBalancerId = &lbId
	req.ListenerId = &listenerId
	req.Targets = clbTargets
	client := GetClient(region)
	resp, err := client.DeregisterTargetsWithContext(ctx, req)
	if err != nil {
		return err
	}
	return Wait(ctx, region, *resp.Response.RequestId)
}

func getClbTargets(targets []Target) (clbTargets []*clb.Target) {
	for _, target := range targets {
		clbTargets = append(clbTargets, &clb.Target{
			Port:  &target.TargetPort,
			EniIp: &target.TargetIP,
		})
	}
	return
}

func RegisterTargets(ctx context.Context, region, lbId, listenerId string, targets ...Target) error {
	clbTargets := getClbTargets(targets)
	req := clb.NewRegisterTargetsRequest()
	req.LoadBalancerId = &lbId
	req.ListenerId = &listenerId
	req.Targets = clbTargets
	client := GetClient(region)
	_, err := client.RegisterTargetsWithContext(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func DescribeTargets(ctx context.Context, region, lbId, listenerId string) (targets []Target, err error) {
	req := clb.NewDescribeTargetsRequest()
	req.LoadBalancerId = &lbId
	req.ListenerIds = []*string{&listenerId}
	client := GetClient(region)
	resp, err := client.DescribeTargetsWithContext(ctx, req)
	if err != nil {
		return
	}
	for _, lis := range resp.Response.Listeners {
		for _, backend := range lis.Targets {
			for _, ip := range backend.PrivateIpAddresses {
				targets = append(targets, Target{TargetIP: *ip, TargetPort: *backend.Port})
			}
		}
	}
	return
}
