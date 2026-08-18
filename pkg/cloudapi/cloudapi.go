package cloudapi

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

const (
	// cloudAPIRootDomain 云 API 根域名
	cloudAPIRootDomain = "tencentcloudapi.com"
	// cloudAPIInternalSubdomain 云 API 内网域名子域
	cloudAPIInternalSubdomain = "internal"
)

var (
	credential *common.Credential
	// environment 云 API 环境，支持 prod / test，默认 prod
	environment = EnvironmentProd
)

const (
	// EnvironmentProd 现网环境
	EnvironmentProd = "prod"
	// EnvironmentTest 测试环境
	EnvironmentTest = "test"
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

// SetEnvironment 设置云 API 环境，仅支持 prod / test。
func SetEnvironment(value string) error {
	switch value {
	case EnvironmentProd, EnvironmentTest:
		environment = value
		return nil
	default:
		return fmt.Errorf("unsupported cloud API environment %q, must be %q or %q", value, EnvironmentProd, EnvironmentTest)
	}
}

// NewClientProfile 创建带正确云 API 域名的 ClientProfile。
// service 为产品名（如 clb/vpc/cam/tag）。默认使用 <service>.internal.tencentcloudapi.com，
// 使 CAM 的专有网络访问限制策略可以识别请求来源；测试环境则为 <service>.internal.test.tencentcloudapi.com。
func NewClientProfile(service string) *profile.ClientProfile {
	p := profile.NewClientProfile()
	if environment == EnvironmentTest {
		p.HttpProfile.Endpoint = fmt.Sprintf("%s.%s.%s.%s", service, cloudAPIInternalSubdomain, EnvironmentTest, cloudAPIRootDomain)
	} else {
		p.HttpProfile.Endpoint = fmt.Sprintf("%s.%s.%s", service, cloudAPIInternalSubdomain, cloudAPIRootDomain)
	}
	return p
}

// NewHTTPTransport 为 SDK client 创建 HTTP Transport。
// 测试环境保留 HTTP Host 为内网测试域名，同时使用测试域名完成 TLS 握手与证书校验。
func NewHTTPTransport(service string) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if environment == EnvironmentTest {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.ServerName = fmt.Sprintf("%s.%s.%s", service, EnvironmentTest, cloudAPIRootDomain)
	}
	return transport
}
