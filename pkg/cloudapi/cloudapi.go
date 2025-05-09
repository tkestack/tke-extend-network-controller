package cloudapi

import "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"

var credential *common.Credential

func Init(secretId, secretKey string) {
	if secretId == "" || secretKey == "" {
		panic("secretId and secretKey are required")
	}
	credential = common.NewCredential(
		secretId,
		secretKey,
	)
}

func GetCredential() *common.Credential {
	return credential
}
