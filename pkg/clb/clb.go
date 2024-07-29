package clb

import (
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

var credential *common.Credential

var defaultRegion, defaultVpcId string

func Init(secretId, secretKey, region, vpcId string) {
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
