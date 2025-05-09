package clb

import (
	"context"
	"sync"
	"time"

	"github.com/imroc/tke-extend-network-controller/pkg/clusterinfo"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
)

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

func SyncQuota(ctx context.Context, region string) (map[string]int64, error) {
	client := GetClient(region)
	req := clb.NewDescribeQuotaRequest()
	resp, err := client.DescribeQuotaWithContext(ctx, req)
	LogAPI(nil, "DescribeQuota", req, resp, err)
	if err != nil {
		return nil, err
	}
	m := make(map[string]int64)
	for _, q := range resp.Response.QuotaSet {
		m[*q.QuotaId] = *q.QuotaLimit
	}
	quotaLock.Lock()
	_, workerStarted := quota[region]
	quota[region] = m
	quotaLock.Unlock()
	if !workerStarted {
		go func() {
			for {
				time.Sleep(5 * time.Minute)
				m, err := SyncQuota(context.Background(), region)
				if err != nil {
					clbLog.Error(err, "failed to sync clb quota periodically", "region", region)
				} else {
					clbLog.V(2).Info("sync clb quota successfully", "region", region, "quota", m)
				}
			}
		}()
	}
	return m, nil
}

func GetQuota(ctx context.Context, region, id string) (int64, error) {
	if region == "" {
		region = clusterinfo.Region
	}
	quotaLock.Lock()
	m, ok := quota[region]
	quotaLock.Unlock()
	if !ok {
		var err error
		if m, err = SyncQuota(ctx, region); err != nil {
			return 0, err
		}
	}
	return m[id], nil
}
