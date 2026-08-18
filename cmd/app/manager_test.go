package app

import (
	"testing"

	"github.com/tkestack/tke-extend-network-controller/pkg/cloudapi"
)

func TestInitCloudAPIEnvironment(t *testing.T) {
	t.Run("非法环境", func(t *testing.T) {
		t.Setenv("CLOUD_API_ENVIRONMENT", "invalid")
		if err := initCloudAPIEnvironment(); err == nil {
			t.Fatal("expect an error for invalid environment")
		}
	})

	t.Run("测试环境", func(t *testing.T) {
		t.Setenv("CLOUD_API_ENVIRONMENT", "test")
		if err := initCloudAPIEnvironment(); err != nil {
			t.Fatalf("init cloud API environment failed: %v", err)
		}
		if err := cloudapi.SetEnvironment(cloudapi.EnvironmentProd); err != nil {
			t.Fatalf("reset cloud API environment failed: %v", err)
		}
	})
}
