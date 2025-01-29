package clb

import (
	"context"
	"fmt"
	"strings"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ContainsTarget(ctx context.Context, lb CLB, port ListenerPort, target Target) (bool, error) {
	req := clb.NewDescribeTargetsRequest()
	req.LoadBalancerId = &lb.LbId
	req.Protocol = &port.Protocol
	req.Port = &port.Port
	req.Filters = []*clb.Filter{
		{
			Name:   common.StringPtr("private-ip-address"),
			Values: []*string{&target.TargetIP},
		},
	}
	client := GetClient(lb.Region)
	resp, err := client.DescribeTargets(req)
	if err != nil {
		return false, err
	}
	for _, listener := range resp.Response.Listeners {
		if *listener.Protocol == port.Protocol && *listener.Port == port.Port {
			targets := convertClbBackendToTargets(listener.Targets)
			for _, t := range targets {
				if t == target {
					return true, nil
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
	InstanceId string
	TargetIP   string
	TargetPort int64
}

func (t Target) String() string {
	if t.InstanceId != "" {
		return fmt.Sprintf("%s:%d", t.InstanceId, t.TargetPort)
	} else if t.TargetIP != "" {
		return fmt.Sprintf("%s:%d", t.TargetIP, t.TargetPort)
	}
	return fmt.Sprintf("invalid target, only have target port %d", t.TargetPort)
}

func DeregisterAllTargets(ctx context.Context, lis Listener) error {
	queryReq := clb.NewDescribeTargetsRequest()
	queryReq.LoadBalancerId = &lis.LbId
	queryReq.ListenerIds = []*string{&lis.ListenerId}
	client := GetClient(lis.Region)
	resp, err := client.DescribeTargetsWithContext(ctx, queryReq)
	if err != nil {
		return err
	}
	if len(resp.Response.Listeners) == 1 {
		return fmt.Errorf("wrong listener count: expected 1 got %d", len(resp.Response.Listeners))
	}
	targets := convertClbBackendToTargets(resp.Response.Listeners[0].Targets)
	if len(targets) > 0 {
		return DeregisterTargets(ctx, lis, targets...)
	}
	return nil
}

func DeregisterTargets(ctx context.Context, lis Listener, targets ...Target) error {
	log.FromContext(ctx).V(7).Info("deregister targets", "listenerId", "listenerId", "lbId", lis.LbId, "targets", targets)
	mu := getLbLock(lis.LbId)
	mu.Lock()
	defer mu.Unlock()
	req := clb.NewDeregisterTargetsRequest()
	req.LoadBalancerId = &lis.LbId
	req.ListenerId = &lis.ListenerId
	req.Targets = convertTargetsToClbTargets(targets)
	client := GetClient(lis.Region)
	resp, err := client.DeregisterTargetsWithContext(ctx, req)
	if err != nil {
		return err
	}
	_, err = Wait(ctx, lis.Region, *resp.Response.RequestId, "DeregisterTargets")
	return err
}

func convertClbBackendToTargets(backends []*clb.Backend) (targets []Target) {
	for _, backend := range backends {
		target := Target{
			TargetPort: *backend.Port,
		}
		if backend.InstanceId != nil && strings.HasPrefix(*backend.InstanceId, "ins-") {
			target.InstanceId = *backend.InstanceId
			targets = append(targets, target)
		} else if len(backend.PrivateIpAddresses) > 0 {
			target.TargetIP = *backend.PrivateIpAddresses[0]
			targets = append(targets, target)
		}
	}
	return
}

func convertTargetsToClbTargets(targets []Target) (clbTargets []*clb.Target) {
	for _, target := range targets {
		ct := &clb.Target{
			Port: &target.TargetPort,
		}
		if target.InstanceId != "" {
			ct.InstanceId = &target.InstanceId
		} else if target.TargetIP != "" {
			ct.EniIp = &target.TargetIP
		}
		clbTargets = append(clbTargets, ct)
	}
	return
}

func EnsureSingleTarget(ctx context.Context, lis Listener, target Target) (registered bool, err error) {
	targets, err := DescribeTargets(ctx, lis)
	if err != nil {
		return
	}
	toDel := []Target{}
	shouldAdd := true
	for _, t := range targets {
		if t != target {
			toDel = append(toDel, t)
		} else {
			shouldAdd = false
		}
	}
	if len(toDel) > 0 {
		if err = DeregisterTargets(ctx, lis, toDel...); err != nil {
			return
		}
	}
	if shouldAdd {
		if err = RegisterTargets(ctx, lis, target); err != nil {
			return
		}
		registered = true
	}
	return
}

func RegisterTargets(ctx context.Context, lis Listener, targets ...Target) error {
	log.FromContext(ctx).V(7).Info("register targets", "listenerId", lis.ListenerId, "lbId", lis.LbId, "targets", targets)
	clbTargets := convertTargetsToClbTargets(targets)
	req := clb.NewRegisterTargetsRequest()
	req.LoadBalancerId = &lis.LbId
	req.ListenerId = &lis.ListenerId
	req.Targets = clbTargets
	client := GetClient(lis.Region)
	mu := getLbLock(lis.LbId)
	mu.Lock()
	defer mu.Unlock()
	resp, err := client.RegisterTargetsWithContext(ctx, req)
	if err != nil {
		return err
	}
	_, err = Wait(ctx, lis.Region, *resp.Response.RequestId, "RegisterTargets")
	return err
}

func DescribeTargets(ctx context.Context, lis Listener) (targets []Target, err error) {
	req := clb.NewDescribeTargetsRequest()
	req.LoadBalancerId = &lis.LbId
	req.ListenerIds = []*string{&lis.ListenerId}
	client := GetClient(lis.Region)
	resp, err := client.DescribeTargetsWithContext(ctx, req)
	if err != nil {
		return
	}
	for _, lis := range resp.Response.Listeners {
		for _, backend := range lis.Targets {
			target := Target{
				TargetPort: *backend.Port,
			}
			if backend.InstanceId != nil && strings.HasPrefix(*backend.InstanceId, "ins-") {
				target.InstanceId = *backend.InstanceId
			} else if len(backend.PrivateIpAddresses) > 0 {
				target.TargetIP = *backend.PrivateIpAddresses[0]
			} else {
				err = fmt.Errorf("unknown backend: %+v", backend)
				return
			}
			targets = append(targets, target)
		}
	}
	return
}
