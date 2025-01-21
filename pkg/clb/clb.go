package clb

import (
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ctrl "sigs.k8s.io/controller-runtime"
)

var credential *common.Credential

var defaultRegion, defaultVpcId, clusterId string

var clbLog = ctrl.Log.WithName("clb")

func Init(secretId, secretKey, region, vpcId, clusterID string) {
	if secretId == "" || secretKey == "" {
		panic("secretId and secretKey are required")
	}
	credential = common.NewCredential(
		secretId,
		secretKey,
	)
	if region == "" || vpcId == "" {
		panic("default region and vpcId is required")
	}
	defaultRegion = region
	defaultVpcId = vpcId
	if clusterID == "" {
		panic("clusterId is required")
	}
	clusterId = clusterID
}

func DefaultRegion() string {
	return defaultRegion
}

func DefaultVpcId() string {
	return defaultVpcId
}

var clients = make(map[string]*clb.Client)

func GetClient(region string) *clb.Client {
	if region == "" {
		region = defaultRegion
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

func GetRegion(region *string) string {
  if region == nil || *region == "" {
    return defaultRegion
  }
  return *region
}
