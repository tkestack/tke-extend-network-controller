package clb

import (
	"fmt"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

var credential *common.Credential

var defaultRegion string

func Init(secretId, secretKey, region string) {
	if secretId == "" || secretKey == "" {
		panic("secretId and secretKey are required")
	}
	credential = common.NewCredential(
		secretId,
		secretKey,
	)
	if region == "" {
		panic("default region is required")
	}
	defaultRegion = region
}

func DefaultRegion() string {
	return defaultRegion
}

var clients = make(map[string]*clb.Client)

func GetClient(region string) *clb.Client {
	if region == "" {
		region = defaultRegion
	}
	fmt.Println("get clb client for region", region)
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
