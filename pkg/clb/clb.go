package clb

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"github.com/tkestack/tke-extend-network-controller/pkg/cloudapi"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var clbLog = ctrl.Log.WithName("clb")

var clients = make(map[string]*clb.Client)

func GetClient(region string) *clb.Client {
	if region == "" {
		region = clusterinfo.Region
	}
	if client, ok := clients[region]; ok {
		return client
	}
	client, err := clb.NewClient(cloudapi.GetCredential(), region, profile.NewClientProfile())
	if err != nil {
		panic(err)
	}
	clients[region] = client
	return client
}

func LogAPI(ctx context.Context, apiName string, req, resp any, cost time.Duration, err error) {
	var logger logr.Logger
	if ctx != nil {
		logger = log.FromContext(ctx)
	} else {
		logger = clbLog
	}
	logger.V(10).Info("CLB API Call", "api", apiName, "request", req, "response", resp, "cost", cost.String(), "error", err)
}
