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
	StartBatchProccessor(concurrent, apiName, RegisterTargetChan, func(region, lbId string, tasks []*RegisterTargetTask) {
		req := clb.NewBatchRegisterTargetsRequest()
		req.LoadBalancerId = &lbId
		for _, task := range tasks {
			req.Targets = append(req.Targets, &clb.BatchTarget{
				ListenerId: &task.ListenerId,
				Port:       &task.Target.TargetPort,
				EniIp:      &task.Target.TargetIP,
			})
		}
		client := GetClient(region)
		resp, err := client.BatchRegisterTargets(req)
		LogAPI(nil, apiName, req, resp, err)
		if err != nil {
			for _, task := range tasks {
				task.Result <- err
			}
			return
		}
		_, err = Wait(context.Background(), region, *resp.Response.RequestId, apiName)
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
	StartBatchProccessor(concurrent, apiName, DescribeTargetsChan, func(region, lbId string, tasks []*DescribeTargetsTask) {
		req := clb.NewDescribeTargetsRequest()
		req.LoadBalancerId = &lbId
		for _, task := range tasks {
			req.ListenerIds = append(req.ListenerIds, &task.ListenerId)
		}
		client := GetClient(region)
		resp, err := client.DescribeTargets(req)
		LogAPI(nil, apiName, req, resp, err)
		if err != nil {
			for _, task := range tasks {
				task.Result <- &DescribeTargetsResult{
					Err: err,
				}
			}
			return
		}
		targetsMap := make(map[string][]*Target)
		for _, backend := range resp.Response.Listeners {
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
			targets := targetsMap[task.ListenerId]
			if targets == nil {
				task.Result <- &DescribeTargetsResult{
					Err: fmt.Errorf("no targets result for %s/%s", task.LbId, task.ListenerId),
				}
			} else {
				task.Result <- &DescribeTargetsResult{
					Targets: targetsMap[task.ListenerId],
				}
			}
		}
	})
}
