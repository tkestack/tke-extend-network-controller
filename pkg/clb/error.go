package clb

import (
	"strings"
)

func IsLbIdNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "InvalidParameter.LBIdNotFound")
}

func IsLoadBalancerNotExistsError(err error) bool {
	return strings.Contains(err.Error(), "LoadBalancer not exist") || strings.Contains(err.Error(), "LB not exist")
}

func IsRequestLimitExceededError(err error) bool {
	return strings.Contains(err.Error(), "RequestLimitExceeded")
}

func IsPortCheckFailedError(err error) bool {
	return strings.Contains(err.Error(), "InvalidParameter.PortCheckFailed")
}

func IsListenerNotFound(err error) bool {
	return strings.Contains(err.Error(), "InvalidParameter") && strings.Contains(err.Error(), "some ListenerId") && strings.Contains(err.Error(), "not found")
}
