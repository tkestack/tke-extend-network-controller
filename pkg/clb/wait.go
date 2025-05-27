package clb

import (
	"context"
	"fmt"
	"time"

	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"github.com/pkg/errors"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Wait(ctx context.Context, region, reqId, taskName string) (ids []string, err error) {
	client := GetClient(region)
	for range 100 {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
			req := clb.NewDescribeTaskStatusRequest()
			req.TaskId = &reqId
			before := time.Now()
			resp, err := client.DescribeTaskStatusWithContext(ctx, req)
			LogAPI(ctx, "DescribeTaskStatus", req, resp, time.Since(before), err)
			if err != nil {
				if IsRequestLimitExceededError(err) {
					clbLog.Info("request limit exceeded when wait for task, retry", "reqId", reqId, "taskName", taskName)
					time.Sleep(1 * time.Second)
					continue
				}
				return nil, errors.WithStack(err)
			}
			switch *resp.Response.Status {
			case 2: // 任务进行中，继续等待
				time.Sleep(1 * time.Second)
				log.FromContext(ctx).V(10).Info("task still waiting", "reqId", reqId, "taskName", taskName)
				continue
			case 1: // 任务失败，返回错误
				msg := fmt.Sprintf("clb task %s failed", reqId)
				if resp.Response.Message != nil {
					msg += fmt.Sprintf(": %s", *resp.Response.Message)
				}
				return nil, errors.New(msg)
			case 0: // 任务成功，返回nil
				return util.ConvertPtrSlice(resp.Response.LoadBalancerIds), nil
			default: // 未知状态码，返回错误
				return nil, fmt.Errorf("unknown task status %d", *resp.Response.Status)
			}
		}
	}
	return nil, fmt.Errorf("clb task %s wait too long", reqId)
}
