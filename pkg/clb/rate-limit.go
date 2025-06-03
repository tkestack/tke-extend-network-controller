package clb

import (
	"os"
	"strconv"

	"golang.org/x/time/rate"
)

var limiter map[string]*rate.Limiter = make(map[string]*rate.Limiter)

var envMap = map[string]string{
	"API_RATELIMIT_DESCRIBE_LOAD_BALANCERS":        "DescribeLoadBalancers",
	"API_RATELIMIT_CREATE_LISTENER":                "CreateListener",
	"API_RATELIMIT_DESCRIBE_LISTENERS":             "DescribeListeners",
	"API_RATELIMIT_DELETE_LOAD_BALANCER_LISTENERS": "DeleteLoadBalancerListeners",
	"API_RATELIMIT_BATCH_REGISTER_TARGETS":         "BatchRegisterTargets",
	"API_RATELIMIT_DESCRIBE_TARGETS":               "DescribeTargets",
	"API_RATELIMIT_BATCH_DEREGISTER_TARGETS":       "BatchDeregisterTargets",
	"API_RATELIMIT_DESCRIBE_TASK_STATUS":           "DescribeTaskStatus",
}

func init() {
	for envName, apiName := range envMap {
		v := os.Getenv(envName)
		if v != "" {
			if vv, err := strconv.Atoi(v); err == nil {
				limiter[apiName] = rate.NewLimiter(rate.Limit(vv), 1)
				clbLog.Info("clb api rate limit", "api", apiName, "limit", vv)
			}
		}
	}
}
