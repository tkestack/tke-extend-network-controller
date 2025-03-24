package clb

import (
	"context"
	"fmt"
	"time"

	"github.com/imroc/tke-extend-network-controller/pkg/util"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Wait(ctx context.Context, region, reqId, taskName string) (ids []string, err error) {
	client := GetClient(region)
	for i := 0; i < 100; i++ {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
			req := clb.NewDescribeTaskStatusRequest()
			req.TaskId = &reqId
			resp, err := client.DescribeTaskStatusWithContext(ctx, req)
			if err != nil {
				return nil, err
			}
			switch *resp.Response.Status {
			case 2: // 任务进行中，继续等待
				time.Sleep(2 * time.Second)
				log.FromContext(ctx).V(5).Info("task still waiting", "reqId", reqId, "taskName", taskName)
				continue
			case 1: // 任务失败，返回错误
				return nil, fmt.Errorf("clb task %s failed", reqId)
			case 0: // 任务成功，返回nil
				return util.ConvertPtrSlice(resp.Response.LoadBalancerIds), nil
			default: // 未知状态码，返回错误
				return nil, fmt.Errorf("unknown task status %d", *resp.Response.Status)
			}
		}
	}
	return nil, fmt.Errorf("clb task %s wait too long", reqId)
}
