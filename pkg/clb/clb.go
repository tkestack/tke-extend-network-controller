package clb

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/imroc/tke-extend-network-controller/pkg/clusterinfo"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var credential *common.Credential

var clbLog = ctrl.Log.WithName("clb")

func Init(secretId, secretKey string) {
	if secretId == "" || secretKey == "" {
		panic("secretId and secretKey are required")
	}
	credential = common.NewCredential(
		secretId,
		secretKey,
	)
}

var clients = make(map[string]*clb.Client)

func GetClient(region string) *clb.Client {
	if region == "" {
		region = clusterinfo.Region
	}
	if client, ok := clients[region]; ok {
		return client
	}
	client, err := clb.NewClient(credential, region, profile.NewClientProfile())
	if err != nil {
		panic(err)
	}
	clients[region] = client
	return client
}

func LogAPI(ctx context.Context, apiName string, req, resp any, err error) {
	var logger logr.Logger
	if ctx != nil {
		logger = log.FromContext(ctx)
	} else {
		logger = clbLog
	}
	logger.V(10).Info("CLB API Call", "api", apiName, "request", req, "response", resp, "error", err)
}
