package vpc

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"github.com/tkestack/tke-extend-network-controller/pkg/cloudapi"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var vpcLog = ctrl.Log.WithName("vpc")

var clients = make(map[string]*vpc.Client)

func GetClient(region string) *vpc.Client {
	if region == "" {
		region = clusterinfo.Region
	}
	if client, ok := clients[region]; ok {
		return client
	}
	client, err := vpc.NewClient(cloudapi.GetCredential(), region, profile.NewClientProfile())
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
		logger = vpcLog
	}
	logger.Info("VPC API Call", "api", apiName, "request", req, "response", resp, "error", err)
}
