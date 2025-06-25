package clb

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/pkg/errors"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
)

type quotaManager struct {
	cache map[string]map[string]int64
	mu    sync.Mutex
}

var Quota = &quotaManager{
	cache: make(map[string]map[string]int64),
}

func (q *quotaManager) GetQuota(ctx context.Context, region, id string) (int64, error) {
	quota, err := q.Get(ctx, region)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return quota[id], nil
}

func (q *quotaManager) Get(ctx context.Context, region string) (map[string]int64, error) {
	if region == "" {
		region = clusterinfo.Region
	}
	// 尝试从缓存拿 quota
	q.mu.Lock()
	cache, ok := q.cache[region]
	q.mu.Unlock()
	if ok {
		return cache, nil
	}
	// 没有获取到过 quota，调 API 获取并存入缓存
	quotaMap, err := DescribeQuota(ctx, region)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	q.mu.Lock()
	q.cache[region] = quotaMap
	q.mu.Unlock()
	clbLog.Info("clb quota", "region", region, "quota", quotaMap)
	// 首次获取成功，后续定时同步
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			quotaMap, err := DescribeQuota(ctx, region)
			if err != nil {
				clbLog.Error(err, "failed to sync clb quota periodically", "region", region)
			} else {
				q.mu.Lock()
				oldCache := q.cache[region]
				if !reflect.DeepEqual(oldCache, quotaMap) {
					q.cache[region] = quotaMap
					clbLog.Info("sync clb quota successfully", "region", region, "quota", quotaMap)
				}
				q.mu.Unlock()
			}
		}
	}()
	return quotaMap, nil
}

/** Response of DescribeQuota:
* {
*   "Response": {
*     "QuotaSet": [
*       {
*         "QuotaCurrent": 1,
*         "QuotaId": "TOTAL_INTERNAL_CLB_QUOTA",
*         "QuotaLimit": 200
*       },
*       {
*         "QuotaCurrent": null,
*         "QuotaId": "TOTAL_TARGET_BIND_QUOTA",
*         "QuotaLimit": 200
*       },
*       {
*         "QuotaCurrent": 16,
*         "QuotaId": "TOTAL_OPEN_CLB_QUOTA",
*         "QuotaLimit": 100
*       },
*       {
*         "QuotaCurrent": null,
*         "QuotaId": "TOTAL_LISTENER_QUOTA",
*         "QuotaLimit": 50
*       },
*       {
*         "QuotaCurrent": null,
*         "QuotaId": "TOTAL_SNAT_IP_QUOTA",
*         "QuotaLimit": 10
*       },
*       {
*         "QuotaCurrent": null,
*         "QuotaId": "TOTAL_LISTENER_RULE_QUOTA",
*         "QuotaLimit": 50
*       },
*       {
*         "QuotaCurrent": null,
*         "QuotaId": "TOTAL_FULL_PORT_RANGE_LISTENER_QUOTA",
*         "QuotaLimit": 1
*       },
*       {
*         "QuotaCurrent": 0,
*         "QuotaId": "TOTAL_ISP_CLB_QUOTA",
*         "QuotaLimit": 15
*       }
*     ],
*     "RequestId": "f1e9d2d8-bcfe-4ded-bead-b3241bfb274a"
*   }
* }
**/
func DescribeQuota(ctx context.Context, region string) (quotaMap map[string]int64, err error) {
	client := GetClient(region)
	req := clb.NewDescribeQuotaRequest()
	before := time.Now()
	resp, err := client.DescribeQuotaWithContext(ctx, req)
	LogAPI(ctx, "DescribeQuota", req, resp, time.Since(before), err)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	m := make(map[string]int64)
	for _, q := range resp.Response.QuotaSet {
		m[*q.QuotaId] = *q.QuotaLimit
	}
	return m, nil
}

var (
	quota     = make(map[string]map[string]int64)
	quotaLock sync.Mutex
)

const (
	TOTAL_INTERNAL_CLB_QUOTA             = "TOTAL_INTERNAL_CLB_QUOTA"
	TOTAL_TARGET_BIND_QUOTA              = "TOTAL_TARGET_BIND_QUOTA"
	TOTAL_OPEN_CLB_QUOTA                 = "TOTAL_OPEN_CLB_QUOTA"
	TOTAL_LISTENER_QUOTA                 = "TOTAL_LISTENER_QUOTA"
	TOTAL_SNAT_IP_QUOTA                  = "TOTAL_SNAT_IP_QUOTA"
	TOTAL_LISTENER_RULE_QUOTA            = "TOTAL_LISTENER_RULE_QUOTA"
	TOTAL_FULL_PORT_RANGE_LISTENER_QUOTA = "TOTAL_FULL_PORT_RANGE_LISTENER_QUOTA"
	TOTAL_ISP_CLB_QUOTA                  = "TOTAL_ISP_CLB_QUOTA"
)
