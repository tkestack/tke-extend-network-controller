package cloudapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
)

func TestNewClientProfile(t *testing.T) {
	t.Run("默认现网域名", func(t *testing.T) {
		SetEndpointSuffix("")
		p := NewClientProfile("clb")
		if p.HttpProfile.Endpoint != "" {
			t.Fatalf("expect empty endpoint, got %q", p.HttpProfile.Endpoint)
		}
	})

	t.Run("测试环境域名", func(t *testing.T) {
		SetEndpointSuffix("test")
		defer SetEndpointSuffix("")
		cases := map[string]string{
			"clb": "clb.test.tencentcloudapi.com",
			"vpc": "vpc.test.tencentcloudapi.com",
			"cam": "cam.test.tencentcloudapi.com",
			"tag": "tag.test.tencentcloudapi.com",
		}
		for svc, want := range cases {
			p := NewClientProfile(svc)
			if p.HttpProfile.Endpoint != want {
				t.Errorf("service %s: expect endpoint %q, got %q", svc, want, p.HttpProfile.Endpoint)
			}
		}
	})
}

// TestNewClientProfileEndpointReached 验证设置 Endpoint 后 SDK 请求确实打到该域名（集成行为）
func TestNewClientProfileEndpointReached(t *testing.T) {
	gotHost := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost <- r.Host
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Response":{"TotalCount":0,"RequestId":"test-request-id"}}`))
	}))
	defer srv.Close()

	SetEndpointSuffix("test")
	defer SetEndpointSuffix("")
	p := NewClientProfile("clb")
	p.HttpProfile.Scheme = "HTTP"
	p.HttpProfile.Endpoint = strings.TrimPrefix(srv.URL, "http://")

	client, err := clb.NewClient(common.NewCredential("testSecretId", "testSecretKey"), "ap-guangzhou", p)
	if err != nil {
		t.Fatalf("new clb client failed: %v", err)
	}
	if _, err := client.DescribeLoadBalancers(clb.NewDescribeLoadBalancersRequest()); err != nil {
		t.Fatalf("describe load balancers failed: %v", err)
	}
	if host := <-gotHost; host != strings.TrimPrefix(srv.URL, "http://") {
		t.Errorf("expect request sent to %q, got %q", strings.TrimPrefix(srv.URL, "http://"), host)
	}
}
