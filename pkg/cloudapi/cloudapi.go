package cloudapi

import (
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// cloudAPIRootDomain 云 API 根域名
const cloudAPIRootDomain = "tencentcloudapi.com"

var (
	credential *common.Credential
	// endpointSuffix 云 API 域名后缀，测试环境为 "test"（如 clb.test.tencentcloudapi.com），为空表示现网
	endpointSuffix string
)

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

// SetEndpointSuffix 设置云 API 域名后缀。
// 测试环境传 "test"，SDK 调云 API 时会走 clb.test.tencentcloudapi.com 等测试域名；不设置则走现网域名。
func SetEndpointSuffix(suffix string) {
	endpointSuffix = suffix
}

// NewClientProfile 创建带正确云 API 域名的 ClientProfile。
// service 为产品名（如 clb/vpc/cam/tag），SDK 默认域名为 <service>.tencentcloudapi.com，
// 设置测试环境后缀后为 <service>.<suffix>.tencentcloudapi.com。
func NewClientProfile(service string) *profile.ClientProfile {
	p := profile.NewClientProfile()
	if endpointSuffix != "" {
		p.HttpProfile.Endpoint = fmt.Sprintf("%s.%s.%s", service, endpointSuffix, cloudAPIRootDomain)
	}
	return p
}
