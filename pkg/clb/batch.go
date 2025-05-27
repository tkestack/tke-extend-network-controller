package clb

import (
	"time"

	"github.com/imroc/tke-extend-network-controller/pkg/util"
)

func init() {
	concurrency := util.GetWorkerCount("WORKER_CLB_POD_BINDING_CONTROLLER")
	if nodeBindingConcurrency := util.GetWorkerCount("WORKER_CLB_NODE_BINDING_CONTROLLER"); nodeBindingConcurrency > concurrency {
		concurrency = nodeBindingConcurrency
	}
	if concurrency < 1 {
		concurrency = 1
	}
	// RegisterTargets 一次性最大支持同时绑定 20 个 target: https://cloud.tencent.com/document/api/214/30676
	go startRegisterTargetsProccessor(20)
	go startCreateListenerProccessor(concurrency)
	// DescribeListeners 一次性最大支持查 100 个监听器: https://cloud.tencent.com/document/api/214/30686
	go startDescribeListenerProccessor(100)
	// DescribeTargets 一次性最大支持同时查 20 个监听器: https://cloud.tencent.com/document/api/214/30684
	go startDescribeTargetsProccessor(20)
	// DeregisterTargets 一次性最大支持同时解绑定 20 个 target (由于 task chan 中的 target 是数组，如果有数组长度大于 1，总数就可能操过 20，但正常情况下一个监听器只会有一个 target，不会有问题): https://cloud.tencent.com/document/api/214/30687
	go startDeregisterTargetsProccessor(20)
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

func StartBatchProccessor[T Task](maxAccumulatedTask int, apiName string, writeOp bool, taskChan chan T, doBatch func(region, lbId string, tasks []T)) {
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
		// TODO: 能否细化到部分成功的场景？
		for lb, tasks := range groupTasks {
			go func(region, lbId string, tasks []T) {
				if writeOp { // 写操作加实例锁
					mu := getLbLock(lbId)
					mu.Lock()
					defer mu.Unlock()
				}
				doBatch(region, lbId, tasks)
			}(lb.Region, lb.LbId, tasks)
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
