package clb

import (
	"context"
	"time"

	"github.com/pkg/errors"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ApiCall[Req, Res any](ctx context.Context, apiName, region string, doReq func(ctx context.Context, client *clb.Client) (req Req, res Res, err error)) (res Res, reqCount int, err error) {
	client := GetClient(region)
	for {
		if l, ok := limiter[apiName]; ok {
			if err = l.Wait(ctx); err != nil {
				return
			}
		}
		before := time.Now()
		req, res, err := doReq(ctx, client)
		LogAPI(ctx, apiName, req, res, time.Since(before), err)
		reqCount++
		if err != nil {
			if IsRequestLimitExceededError(err) { // 云 API 限频，重试
				log.FromContext(ctx).V(3).Info("clb api request limit exceeded")
				select { // context 撤销，不继续重试
				case <-ctx.Done():
					return res, reqCount, err
				default:
				}
				time.Sleep(time.Second)
				continue
			} else { // 其它错误，抛给调用者
				return res, reqCount, errors.WithStack(err)
			}
		} else { // 请求成功，返回 response
			return res, reqCount, nil
		}
	}
}
