package clb

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	before := time.Now()
	resp, err := client.DescribeTargets(req)
	LogAPI(ctx, false, "DescribeTargets", req, resp, time.Since(before), err)
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

type Target struct {
	TargetIP   string
	TargetPort int64
}

func (t Target) String() string {
	return fmt.Sprintf("%s:%d", t.TargetIP, t.TargetPort)
}

func DeregisterAllTargetsTryBatch(ctx context.Context, region, lbId, listenerId string) error {
	targets, err := DescribeTargetsTryBatch(ctx, region, lbId, listenerId)
	if err != nil {
		return errors.WithStack(err)
	}
	if len(targets) > 0 {
		err = DeregisterTargetsForListener(ctx, region, lbId, listenerId, targets...)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func DeregisterTargetsForListener(ctx context.Context, region, lbId, listenerId string, targets ...*Target) error {
	task := &DeregisterTargetsTask{
		Ctx:        ctx,
		Region:     region,
		LbId:       lbId,
		ListenerId: listenerId,
		Targets:    targets,
		Result:     make(chan error),
	}
	DeregisterTargetsChan <- task
	err := <-task.Result
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
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
	mu := getLbLock(lbId)
	mu.Lock()
	defer mu.Unlock()
	before := time.Now()
	resp, err := client.RegisterTargetsWithContext(ctx, req)
	LogAPI(ctx, true, "RegisterTargets", req, resp, time.Since(before), err)
	if err != nil {
		return err
	}
	_, err = Wait(ctx, region, *resp.Response.RequestId, "RegisterTargets", DefaultWaitInterval)
	return err
}

func RegisterTarget(ctx context.Context, region, lbId, listenerId string, target Target) error {
	task := &RegisterTargetTask{
		Ctx:        ctx,
		Region:     region,
		LbId:       lbId,
		ListenerId: listenerId,
		Target:     target,
		Result:     make(chan error),
	}
	log.FromContext(ctx).V(3).Info("RegisterTarget", "lbId", lbId, "listenerId", listenerId, "target", target)
	RegisterTargetChan <- task
	err := <-task.Result
	return err
}

func DescribeTargetsTryBatch(ctx context.Context, region, lbId, listenerId string) (targets []*Target, err error) {
	task := &DescribeTargetsTask{
		Ctx:        ctx,
		Region:     region,
		LbId:       lbId,
		ListenerId: listenerId,
		Result:     make(chan *DescribeTargetsResult),
	}
	DescribeTargetsChan <- task
	result := <-task.Result
	if result.Err != nil {
		err = errors.WithStack(result.Err)
		return
	}
	targets = result.Targets
	return
}

func DescribeTargets(ctx context.Context, region, lbId, listenerId string) (targets []Target, err error) {
	req := clb.NewDescribeTargetsRequest()
	req.LoadBalancerId = &lbId
	req.ListenerIds = []*string{&listenerId}
	client := GetClient(region)
	before := time.Now()
	resp, err := client.DescribeTargetsWithContext(ctx, req)
	LogAPI(ctx, false, "DescribeTargets", req, resp, time.Since(before), err)
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
