package clb

import (
	"context"
	"fmt"
	"time"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
)

func Wait(ctx context.Context, region, reqId string) error {
	client := GetClient(region)
	for i := 0; i < 10; i++ {
		req := clb.NewDescribeTaskStatusRequest()
		req.TaskId = &reqId
		resp, err := client.DescribeTaskStatusWithContext(ctx, req)
		if err != nil {
			return err
		}
		switch *resp.Response.Status {
		case 2:
			time.Sleep(2 * time.Second)
			continue
		case 1:
			return fmt.Errorf("clb task %s failed", reqId)
		}
		break
	}
	return fmt.Errorf("clb task %s wait too long", reqId)
}
