package clb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"github.com/pkg/errors"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type CreateListenerTask struct {
	Ctx                 context.Context
	Region              string
	LbId                string
	Port                int64
	Protocol            string
	ExtensiveParameters string
	Result              chan *ListenerResult
}

type ListenerResult struct {
	ListenerId string
	Err        error
}

func (t *CreateListenerTask) GetLbId() string {
	return t.LbId
}

func (t *CreateListenerTask) GetRegion() string {
	return t.Region
}

var CreateListenerChan = make(chan *CreateListenerTask, 100)

type listenerKey struct {
	Protocol            string
	ExtensiveParameters string
}

const TkeListenerName = "TKE-LISTENER"

func doBatchCreateListener(apiName, region, lbId, protocol, extensiveParameters string, tasks []*CreateListenerTask) (listenerIds []string, err error) {
	req := clb.NewCreateListenerRequest()
	req.HealthCheck = &clb.HealthCheck{
		HealthSwitch: common.Int64Ptr(0),
		SourceIpType: common.Int64Ptr(1),
	}
	if extensiveParameters != "" {
		err = json.Unmarshal([]byte(extensiveParameters), req)
		if err != nil {
			err = errors.WithStack(err)
			return
		}
	}
	req.LoadBalancerId = &lbId
	req.Protocol = &protocol
	for _, task := range tasks {
		req.Ports = append(req.Ports, common.Int64Ptr(task.Port))
		req.ListenerNames = append(req.ListenerNames, common.StringPtr(TkeListenerName))
	}
	client := GetClient(region)
	resp, err := client.CreateListener(req)
	clbLog.Info(apiName, "request", req, "response", resp, "err", err)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	listenerIds = util.ConvertPtrSlice(resp.Response.ListenerIds)
	_, err = Wait(context.Background(), region, *resp.Response.RequestId, apiName)
	if err != nil {
		err = errors.WithStack(err)
	}
	return
}

func startCreateListenerProccessor() {
	apiName := "CreateListener"
	StartBatchProccessor(apiName, CreateListenerChan, func(region, lbId string, tasks []*CreateListenerTask) {
		startTime := time.Now()
		defer func() {
			clbLog.V(10).Info(fmt.Sprintf("batch proccess %s performance", apiName), "cost", time.Since(startTime).String())
		}()
		groupTask := make(map[listenerKey][]*CreateListenerTask)
		for _, task := range tasks {
			key := listenerKey{
				Protocol:            task.Protocol,
				ExtensiveParameters: task.ExtensiveParameters,
			}
			groupTask[key] = append(groupTask[key], task)
		}
		for lis, tasks := range groupTask {
			listenerIds, err := doBatchCreateListener(apiName, region, lbId, lis.Protocol, lis.ExtensiveParameters, tasks)
			if err != nil {
				clbLog.Error(err, "batch create listener failed")
				for _, task := range tasks {
					task.Result <- &ListenerResult{
						Err: err,
					}
				}
			} else if len(listenerIds) != len(tasks) {
				err := fmt.Errorf("number of listener created is not match, expect %d got %d", len(tasks), len(listenerIds))
				clbLog.Error(err, "batch create listener failed")
				for _, task := range tasks {
					task.Result <- &ListenerResult{
						Err: err,
					}
				}
			} else {
				for i, task := range tasks {
					task.Result <- &ListenerResult{
						ListenerId: listenerIds[i],
					}
				}
			}
		}
	})
}
