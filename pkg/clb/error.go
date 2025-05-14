package clb

import sdkerror "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"

func IsLbIdNotFoundError(err error) bool {
	if serr, ok := err.(*sdkerror.TencentCloudSDKError); ok && serr.Code == "InvalidParameter.LBIdNotFound" {
		return true
	}
	return false
}

func IsLoadBalancerNotExistsError(err error) bool {
	if serr, ok := err.(*sdkerror.TencentCloudSDKError); ok && serr.Code == "InvalidParameter" && serr.Message == "LoadBalancer not exists" {
		return true
	}
	return false
}

func IsRequestLimitExceededError(err error) bool {
	if serr, ok := err.(*sdkerror.TencentCloudSDKError); ok && serr.Code == "RequestLimitExceeded" {
		return true
	}
	return false
}
