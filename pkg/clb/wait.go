package clb

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Wait(ctx context.Context, region, reqId, taskName string) (ids []string, err error) {
	for range 100 {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
			res, err := ApiCall(context.Background(), "DescribeTaskStatus", region, func(ctx context.Context, client *clb.Client) (req *clb.DescribeTaskStatusRequest, res *clb.DescribeTaskStatusResponse, err error) {
				req = clb.NewDescribeTaskStatusRequest()
				req.TaskId = &reqId
				res, err = client.DescribeTaskStatusWithContext(ctx, req)
				return
			})
			if err != nil {
				return nil, errors.WithStack(err)
			}
			switch *res.Response.Status {
			case 2: // 任务进行中，继续等待
				time.Sleep(1 * time.Second)
				log.FromContext(ctx).V(5).Info("task still waiting", "reqId", reqId, "taskName", taskName)
				continue
			case 1: // 任务失败，返回错误
				msg := fmt.Sprintf("clb task %s failed", reqId)
				if res.Response.Message != nil {
					msg += fmt.Sprintf(": %s", *res.Response.Message)
				}
				return nil, errors.New(msg)
			case 0: // 任务成功，返回nil
				return util.ConvertPtrSlice(res.Response.LoadBalancerIds), nil
			default: // 未知状态码，返回错误
				return nil, fmt.Errorf("unknown task status %d", *res.Response.Status)
			}
		}
	}
	return nil, fmt.Errorf("clb task %s wait too long", reqId)
}
