package clb

import (
	"context"
	"errors"
	"fmt"

	"github.com/imroc/tke-extend-network-controller/pkg/util"
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

type BatchTarget struct {
	ListenerProtocol string
	ListenerPort     int32
	Target
}

func BatchDeregisterTargets(ctx context.Context, region, lbId string, targets []BatchTarget) (err error) {
	type listenerKey struct {
		protocol string
		port     int32
	}
	var allError []error
	listenerIds := make(map[listenerKey]string)
	var batchTargets []*clb.BatchTarget
	for _, target := range targets {
		k := listenerKey{port: target.ListenerPort, protocol: target.ListenerProtocol}
		id, ok := listenerIds[k]
		if !ok {
			id, err = GetListenerId(ctx, region, lbId, target.ListenerPort, target.ListenerProtocol)
			if err != nil {
				return
			}
			listenerIds[k] = id
		}
		if id == "" {
			allError = append(allError, fmt.Errorf("listener not found: %d/%s", target.ListenerPort, target.ListenerProtocol))
			continue
		}
		batchTargets = append(batchTargets, &clb.BatchTarget{
			ListenerId: &id,
			Port:       common.Int64Ptr(int64(target.TargetPort)),
			EniIp:      &target.TargetIP,
		})
	}
	if len(batchTargets) > 0 {
		req := clb.NewBatchDeregisterTargetsRequest()
		req.LoadBalancerId = &lbId
		req.Targets = batchTargets
		client := GetClient(region)
		resp, err := client.BatchDeregisterTargetsWithContext(ctx, req)
		if err != nil {
			allError = append(allError, err)
		}
		if failedIds := resp.Response.FailListenerIdSet; len(failedIds) > 0 {
			allError = append(allError, fmt.Errorf("batch deregister targets failed: %v", util.ConvertStringPointSlice(failedIds)))
		}
	}
	if len(allError) > 0 {
		err = errors.Join(allError...)
	}
	return
}

type Target struct {
	TargetIP   string
	TargetPort int32
}

func DeregisterTargets(ctx context.Context, region, lbId string, port int32, protocol string, targets []Target) error {
	id, err := GetListenerId(ctx, region, lbId, port, protocol)
	if err != nil {
		return err
	}
	var clbTargets []*clb.Target
	for _, target := range targets {
		clbTargets = append(clbTargets, &clb.Target{
			Port:  common.Int64Ptr(int64(target.TargetPort)),
			EniIp: &target.TargetIP,
		})
	}
	req := clb.NewDeregisterTargetsRequest()
	req.LoadBalancerId = &lbId
	req.ListenerId = &id
	req.Targets = clbTargets
	client := GetClient(region)
	_, err = client.DeregisterTargetsWithContext(ctx, req)
	if err != nil {
		return err
	}
	return nil
}
