package clb

import sdkerror "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"

func IsLbIdNotFoundError(err error) bool {
	if serr, ok := err.(*sdkerror.TencentCloudSDKError); ok && serr.Code == "InvalidParameter.LBIdNotFound" {
		return true
	}
	return false
}
