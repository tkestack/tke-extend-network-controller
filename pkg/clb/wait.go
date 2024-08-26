package clb

import (
	"context"
	"fmt"
	"time"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
)

func Wait(ctx context.Context, region, reqId string) error {
	client := GetClient(region)
	for i := 0; i < 20; i++ {
		req := clb.NewDescribeTaskStatusRequest()
		req.TaskId = &reqId
		resp, err := client.DescribeTaskStatusWithContext(ctx, req)
		if err != nil {
			return err
		}
		switch *resp.Response.Status {
		case 2: // 任务进行中，继续等待
			time.Sleep(2 * time.Second)
			continue
		case 1: // 任务失败，返回错误
			return fmt.Errorf("clb task %s failed", reqId)
		case 0: // 任务成功，返回nil
			return nil
		default: // 未知状态码，返回错误
			return fmt.Errorf("unknown task status %d", *resp.Response.Status)
		}
	}
	return fmt.Errorf("clb task %s wait too long", reqId)
}
