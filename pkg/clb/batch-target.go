package clb

import (
	"context"
	"fmt"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
)

type RegisterTargetTask struct {
	Ctx        context.Context
	Region     string
	LbId       string
	ListenerId string
	Target     Target
	Result     chan error
}

func (t *RegisterTargetTask) GetLbId() string {
	return t.LbId
}

func (t *RegisterTargetTask) GetRegion() string {
	return t.Region
}

var RegisterTargetChan = make(chan *RegisterTargetTask, 100)

func startRegisterTargetsProccessor(concurrent int) {
	apiName := "BatchRegisterTargets"
	StartBatchProccessor(concurrent, apiName, true, RegisterTargetChan, func(region, lbId string, tasks []*RegisterTargetTask) {
		res, err := ApiCall(context.Background(), apiName, region, func(ctx context.Context, client *clb.Client) (req *clb.BatchRegisterTargetsRequest, res *clb.BatchRegisterTargetsResponse, err error) {
			req = clb.NewBatchRegisterTargetsRequest()
			req.LoadBalancerId = &lbId
			for _, task := range tasks {
				req.Targets = append(req.Targets, &clb.BatchTarget{
					ListenerId: &task.ListenerId,
					Port:       &task.Target.TargetPort,
					EniIp:      &task.Target.TargetIP,
				})
			}
			res, err = client.BatchRegisterTargets(req)
			return
		})
		if err != nil {
			for _, task := range tasks {
				task.Result <- err
			}
			return
		}
		_, err = Wait(context.Background(), region, *res.Response.RequestId, apiName)
		for _, task := range tasks {
			task.Result <- err
		}
	})
}

type DescribeTargetsResult struct {
	Targets []*Target
	Err     error
}

type DescribeTargetsTask struct {
	Ctx        context.Context
	Region     string
	LbId       string
	ListenerId string
	Result     chan *DescribeTargetsResult
}

func (t *DescribeTargetsTask) GetLbId() string {
	return t.LbId
}

func (t *DescribeTargetsTask) GetRegion() string {
	return t.Region
}

var DescribeTargetsChan = make(chan *DescribeTargetsTask, 100)

func startDescribeTargetsProccessor(concurrent int) {
	apiName := "DescribeTargets"
	StartBatchProccessor(concurrent, apiName, false, DescribeTargetsChan, func(region, lbId string, tasks []*DescribeTargetsTask) {
		res, err := ApiCall(context.Background(), apiName, region, func(ctx context.Context, client *clb.Client) (req *clb.DescribeTargetsRequest, res *clb.DescribeTargetsResponse, err error) {
			req = clb.NewDescribeTargetsRequest()
			req.LoadBalancerId = &lbId
			for _, task := range tasks {
				req.ListenerIds = append(req.ListenerIds, &task.ListenerId)
			}
			res, err = client.DescribeTargets(req)
			return
		})
		if err != nil {
			for _, task := range tasks {
				task.Result <- &DescribeTargetsResult{
					Err: err,
				}
			}
			return
		}
		targetsMap := make(map[string][]*Target)
		for _, backend := range res.Response.Listeners {
			targets := []*Target{}
			for _, target := range backend.Targets {
				for _, ip := range target.PrivateIpAddresses {
					targets = append(targets, &Target{
						TargetIP:   *ip,
						TargetPort: *target.Port,
					})
				}
			}
			targetsMap[*backend.ListenerId] = targets
		}
		for _, task := range tasks {
			task.Result <- &DescribeTargetsResult{
				Targets: targetsMap[task.ListenerId],
			}
		}
	})
}

type DeregisterTargetsTask struct {
	Ctx        context.Context
	Region     string
	LbId       string
	ListenerId string
	Targets    []*Target
	Result     chan error
}

func (t *DeregisterTargetsTask) GetLbId() string {
	return t.LbId
}

func (t *DeregisterTargetsTask) GetRegion() string {
	return t.Region
}

var DeregisterTargetsChan = make(chan *DeregisterTargetsTask, 100)

func startDeregisterTargetsProccessor(concurrent int) {
	apiName := "BatchDeregisterTargets"
	StartBatchProccessor(concurrent, apiName, true, DeregisterTargetsChan, func(region, lbId string, tasks []*DeregisterTargetsTask) {
		res, err := ApiCall(context.Background(), apiName, region, func(ctx context.Context, client *clb.Client) (req *clb.BatchDeregisterTargetsRequest, res *clb.BatchDeregisterTargetsResponse, err error) {
			req = clb.NewBatchDeregisterTargetsRequest()
			req.LoadBalancerId = &lbId
			for _, task := range tasks {
				for _, target := range task.Targets {
					req.Targets = append(req.Targets, &clb.BatchTarget{
						ListenerId: &task.ListenerId,
						Port:       &target.TargetPort,
						EniIp:      &target.TargetIP,
					})
				}
			}
			res, err = client.BatchDeregisterTargets(req)
			return
		})
		// 调用失败
		if err != nil {
			for _, task := range tasks {
				task.Result <- err
			}
			return
		}
		// 解绑失败
		_, err = Wait(context.Background(), region, *res.Response.RequestId, apiName)
		if err != nil {
			for _, task := range tasks {
				task.Result <- err
			}
			return
		}
		// 全部解绑成功
		if len(res.Response.FailListenerIdSet) == 0 {
			for _, task := range tasks {
				task.Result <- nil
			}
			return
		}
		// 部分解绑成功
		failMap := make(map[string]bool)
		for _, listenerId := range res.Response.FailListenerIdSet {
			failMap[*listenerId] = true
		}
		for _, task := range tasks {
			if failMap[task.ListenerId] {
				task.Result <- fmt.Errorf("deregister targets failed for %s/%s", task.LbId, task.ListenerId)
			} else {
				task.Result <- nil
			}
		}
	})
}
