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

// IsResourceInOperatingError 判断是否为 CLB desState 冲突错误。
// 当 CLB 正在处理其它任务（desState 非 normal）时，写操作会返回
// FailedOperation.ResourceInOperating，需要退避后重试。
func IsResourceInOperatingError(err error) bool {
	return strings.Contains(err.Error(), "FailedOperation.ResourceInOperating")
}

func IsPortCheckFailedError(err error) bool {
	return strings.Contains(err.Error(), "InvalidParameter.PortCheckFailed")
}

func IsListenerNotFound(err error) bool {
	return strings.Contains(err.Error(), "InvalidParameter") && strings.Contains(err.Error(), "some ListenerId") && strings.Contains(err.Error(), "not found")
}
