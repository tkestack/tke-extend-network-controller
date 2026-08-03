package clb

import (
	"time"

	"github.com/tkestack/tke-extend-network-controller/pkg/util"
)

func init() {
	concurrency := util.GetWorkerCount("WORKER_CLB_POD_BINDING_CONTROLLER")
	if nodeBindingConcurrency := util.GetWorkerCount("WORKER_CLB_NODE_BINDING_CONTROLLER"); nodeBindingConcurrency > concurrency {
		concurrency = nodeBindingConcurrency
	}
	if concurrency < 1 {
		concurrency = 1
	}
	// BatchRegisterTargets 一次性最大支持同时绑定 500 个 target（CLB 侧确认的真实上限）
	go startRegisterTargetsProccessor(500)
	go startCreateListenerProccessor(800)
	// DescribeListeners 一次性最大支持查 100 个监听器: https://cloud.tencent.com/document/api/214/30686
	go startDescribeListenerProccessor(100)
	// DescribeTargets 一次性最大支持同时查 20 个监听器: https://cloud.tencent.com/document/api/214/30684
	go startDescribeTargetsProccessor(20)
	// BatchDeregisterTargets 一次性最大支持同时解绑定 500 个 target（与 BatchRegisterTargets 对齐，CLB 侧确认）
	go startDeregisterTargetsProccessor(500)
	go startDeleteListenerProccessor(20)
}

const (
	MaxBatchInternal = 2 * time.Second
)

type lbKey struct {
	LbId   string
	Region string
}

type Task interface {
	GetLbId() string
	GetRegion() string
}

const maxAccumulatedTask = 800

func StartBatchProccessor[T Task](maxTaskOneBatch int, apiName string, writeOp bool, taskChan chan T, doBatch func(region, lbId string, tasks []T)) {
	tasks := []T{}
	timer := time.NewTimer(MaxBatchInternal)
	batchRequest := func() {
		timer = time.NewTimer(MaxBatchInternal)
		if len(tasks) == 0 {
			return
		}
		defer func() {
			tasks = []T{}
		}()
		// 按 lb 维度合并 task
		groupTasks := map[lbKey][]T{}
		for _, task := range tasks {
			k := lbKey{LbId: task.GetLbId(), Region: task.GetRegion()}
			groupTasks[k] = append(groupTasks[k], task)
		}
		// 将合并后的 task 通过 clb 的 BatchXXX 接口批量操作
		for lb, tasks := range groupTasks {
			for len(tasks) > 0 {
				num := min(len(tasks), maxTaskOneBatch)
				t := tasks[:num]
				tasks = tasks[num:]
				go func(region, lbId string, tasks []T) {
					if writeOp { // 写操作加实例锁
						mu := getLbLock(lbId)
						mu.Lock()
						defer mu.Unlock()
					}
					doBatch(region, lbId, tasks)
				}(lb.Region, lb.LbId, t)
			}
		}
	}
	for {
		select {
		case task, ok := <-taskChan:
			if !ok { // 优雅终止，通道关闭，执行完批量操作
				batchRequest()
				return
			}
			tasks = append(tasks, task)
			if len(tasks) >= maxAccumulatedTask {
				batchRequest()
			}
		case <-timer.C: // 累计时间后执行批量操作
			batchRequest()
		}
	}
}
