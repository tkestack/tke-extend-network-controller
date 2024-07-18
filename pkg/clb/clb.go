package clb

import (
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

var credential *common.Credential

func Init(secretId, secretKey string) {
	credential = common.NewCredential(
		secretId,
		secretKey,
	)
}

var clients = make(map[string]*clb.Client)

func GetClient(region string) *clb.Client {
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
