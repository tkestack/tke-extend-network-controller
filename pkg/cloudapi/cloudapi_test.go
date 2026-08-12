package cloudapi

import "testing"

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
				t.Fatalf("service %s: expect endpoint %q, got %q", svc, want, p.HttpProfile.Endpoint)
			}
		}
	})
}
