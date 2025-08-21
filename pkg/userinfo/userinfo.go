package userinfo

import (
	"github.com/pkg/errors"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"github.com/tkestack/tke-extend-network-controller/pkg/cloudapi"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
)

var OwnerUin string

func Init() error {
	client, err := cam.NewClient(cloudapi.GetCredential(), "", profile.NewClientProfile())
	if err != nil {
		return errors.WithStack(err)
	}
	req := cam.NewGetUserAppIdRequest()
	resp, err := client.GetUserAppId(req)
	if err != nil {
		return errors.WithStack(err)
	}
	OwnerUin = util.GetValue(resp.Response.OwnerUin)
	if OwnerUin == "" {
		return errors.New("got empty owner uin")
	}
	return nil
}
