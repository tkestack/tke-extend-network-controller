package clb

import (
	"strings"
)

func IsLbIdNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "InvalidParameter.LBIdNotFound")
}

func IsLoadBalancerNotExistsError(err error) bool {
	return strings.Contains(err.Error(), "LoadBalancer not exist")
}

func IsRequestLimitExceededError(err error) bool {
	return strings.Contains(err.Error(), "RequestLimitExceeded")
}

func IsPortCheckFailedError(err error) bool {
	return strings.Contains(err.Error(), "InvalidParameter.PortCheckFailed")
}
