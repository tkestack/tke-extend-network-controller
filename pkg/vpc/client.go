package vpc

import (
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"github.com/tkestack/tke-extend-network-controller/pkg/cloudapi"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	ctrl "sigs.k8s.io/controller-runtime"
)

var vpcLog = ctrl.Log.WithName("vpc")

var clients = make(map[string]*vpc.Client)

func GetClient(region string) *vpc.Client {
	if region == "" {
		region = clusterinfo.Region
	}
	if client, ok := clients[region]; ok {
		return client
	}
	client, err := vpc.NewClient(cloudapi.GetCredential(), region, cloudapi.NewClientProfile("vpc"))
	if err != nil {
		panic(err)
	}
	client.WithHttpTransport(cloudapi.NewHTTPTransport("vpc"))
	clients[region] = client
	return client
}
