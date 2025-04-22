package clb

import (
	"context"
	"time"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
)

type CreateListenerTask struct {
	Ctx                 context.Context
	Region              string
	LbId                string
	Port                int64
	EndPort             int64
	Protocol            string
	ExtensiveParameters string
}

type RegisterTargetTask struct {
	Ctx        context.Context
	Region     string
	LbId       string
	ListenerId string
	Target     Target
	Result     chan error
}

var RegisterTargetChan = make(chan *RegisterTargetTask, 100)

func init() {
	go startRegisterTargetProccessor()
}

const MaxBatchInternal = 2 * time.Second

type lbKey struct {
	LbId   string
	Region string
}

type batchReq struct {
	Req   *clb.BatchRegisterTargetsRequest
	Tasks []*RegisterTargetTask
}

func startRegisterTargetProccessor() {
	batch := []*RegisterTargetTask{}
	timer := time.NewTimer(MaxBatchInternal)
	batchRegisterTargets := func() {
		timer = time.NewTimer(MaxBatchInternal)
		if len(batch) == 0 {
			return
		}
		defer func() {
			batch = []*RegisterTargetTask{}
		}()
		// 按 lb 维度合并 task
		reqs := map[lbKey]*batchReq{}
		for _, task := range batch {
			k := lbKey{LbId: task.LbId, Region: task.Region}
			req, ok := reqs[k]
			if !ok {
				req = &batchReq{}
				req.Req = clb.NewBatchRegisterTargetsRequest()
				req.Req.LoadBalancerId = &task.LbId
				req.Tasks = []*RegisterTargetTask{}
				reqs[k] = req
			}
			req.Req.Targets = append(req.Req.Targets, &clb.BatchTarget{
				ListenerId: &task.ListenerId,
				Port:       &task.Target.TargetPort,
				EniIp:      &task.Target.TargetIP,
			})
			req.Tasks = append(req.Tasks, task)
		}
		// 执行批量绑定 rs，并将结果返回给所有关联的 task
		// TODO: 能否细化到部分成功的场景？
		for lb, req := range reqs {
			go func(region string, req *batchReq) {
				mu := getLbLock(*req.Req.LoadBalancerId)
				mu.Lock()
				defer mu.Unlock()
				client := GetClient(region)
				resp, err := client.BatchRegisterTargets(req.Req)
				clbLog.V(10).Info("BatchRegisterTargets", "req", req, "resp", resp, "err", err)
				_, err = Wait(context.Background(), region, *resp.Response.RequestId, "BatchRegisterTargets")
				for _, task := range req.Tasks {
					task.Result <- err
				}
			}(lb.Region, req)
		}
	}
	for {
		select {
		case task, ok := <-RegisterTargetChan:
			if !ok { // 优雅终止，通道关闭，执行完批量操作
				batchRegisterTargets()
				return
			}
			batch = append(batch, task)
			if len(batch) > 200 {
				batchRegisterTargets()
			}
		case <-timer.C: // 累计时间后执行批量操作
			batchRegisterTargets()
		}
	}
}
