package clb

import (
	"context"

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
