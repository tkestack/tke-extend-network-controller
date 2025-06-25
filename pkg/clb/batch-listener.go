package clb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"github.com/pkg/errors"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type CreateListenerTask struct {
	Ctx                 context.Context
	Region              string
	LbId                string
	CertId              string
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
	CertId              string
	ExtensiveParameters string
}

const TkeListenerName = "TKE-LISTENER"

func doBatchCreateListener(apiName, region, lbId, protocol, certId, extensiveParameters string, tasks []*CreateListenerTask) (listenerIds []string, err error) {
	res, err := ApiCall(context.Background(), apiName, region, func(ctx context.Context, client *clb.Client) (req *clb.CreateListenerRequest, res *clb.CreateListenerResponse, err error) {
		req = clb.NewCreateListenerRequest()
		req.LoadBalancerId = &lbId
		req.HealthCheck = &clb.HealthCheck{
			HealthSwitch: common.Int64Ptr(0),
			SourceIpType: common.Int64Ptr(1),
		}
		if certId != "" {
			req.Certificate = &clb.CertificateInput{
				SSLMode: common.StringPtr("UNIDIRECTIONAL"),
				CertId:  &certId,
			}
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
		res, err = client.CreateListener(req)
		if err == nil && len(res.Response.ListenerIds) != len(tasks) {
			err = fmt.Errorf("number of listener created is not match, expect %d got %d", len(tasks), len(res.Response.ListenerIds))
		}
		return
	})
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	listenerIds = util.ConvertPtrSlice(res.Response.ListenerIds)
	_, err = Wait(context.Background(), region, *res.Response.RequestId, apiName)
	if err != nil {
		err = errors.WithStack(err)
	}
	return
}

func startCreateListenerProccessor(concurrent int) {
	apiName := "CreateListener"
	StartBatchProccessor(concurrent, apiName, true, CreateListenerChan, func(region, lbId string, tasks []*CreateListenerTask) {
		groupTask := make(map[listenerKey][]*CreateListenerTask)
		for _, task := range tasks {
			key := listenerKey{
				Protocol:            task.Protocol,
				CertId:              task.CertId,
				ExtensiveParameters: task.ExtensiveParameters,
			}
			groupTask[key] = append(groupTask[key], task)
		}
		for lis, tasks := range groupTask {
			listenerIds, err := doBatchCreateListener(apiName, region, lbId, lis.Protocol, lis.CertId, lis.ExtensiveParameters, tasks)
			if err != nil {
				clbLog.Error(
					err, "batch create listener failed",
					"lbId", lbId,
					"protocol", lis.Protocol,
					"tasks", len(tasks),
				)
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

type DescribeListenerResult struct {
	Listener *Listener
	Err      error
}

type DescribeListenerTask struct {
	Ctx        context.Context
	Region     string
	LbId       string
	ListenerId string
	Result     chan *DescribeListenerResult
}

func (t *DescribeListenerTask) GetLbId() string {
	return t.LbId
}

func (t *DescribeListenerTask) GetRegion() string {
	return t.Region
}

var DescribeListenerChan = make(chan *DescribeListenerTask, 100)

func startDescribeListenerProccessor(concurrent int) {
	apiName := "DescribeListeners"
	StartBatchProccessor(concurrent, apiName, false, DescribeListenerChan, func(region, lbId string, tasks []*DescribeListenerTask) {
		res, err := ApiCall(context.Background(), apiName, region, func(ctx context.Context, client *clb.Client) (req *clb.DescribeListenersRequest, res *clb.DescribeListenersResponse, err error) {
			req = clb.NewDescribeListenersRequest()
			req.LoadBalancerId = &lbId
			for _, task := range tasks {
				req.ListenerIds = append(req.ListenerIds, &task.ListenerId)
			}
			res, err = client.DescribeListenersWithContext(ctx, req)
			return
		})
		// 查询失败，所有 task 都失败，全部返回错误
		if err != nil {
			for _, task := range tasks {
				task.Result <- &DescribeListenerResult{
					Err: err,
				}
			}
			return
		}
		// 查询成功
		taskMap := make(map[string]*DescribeListenerTask)
		for _, task := range tasks {
			taskMap[task.ListenerId] = task
		}
		// 给查到结果的 task 返回 listener 信息
		for _, lis := range res.Response.Listeners {
			task := taskMap[*lis.ListenerId]
			result := &DescribeListenerResult{
				Listener: &Listener{
					ListenerId:   *lis.ListenerId,
					Protocol:     *lis.Protocol,
					Port:         *lis.Port,
					ListenerName: *lis.ListenerName,
				},
			}
			if lis.EndPort != nil {
				result.Listener.EndPort = *lis.EndPort
			}
			task.Result <- result
			delete(taskMap, *lis.ListenerId)
		}
		// 不存在的 listener 返回空结果
		for _, task := range taskMap { // 没找到监听器，返回空结果
			task.Result <- &DescribeListenerResult{}
		}
	})
}

type DeleteListenerTask struct {
	Ctx        context.Context
	Region     string
	LbId       string
	ListenerId string
	Result     chan error
}

func (t *DeleteListenerTask) GetLbId() string {
	return t.LbId
}

func (t *DeleteListenerTask) GetRegion() string {
	return t.Region
}

var (
	DeleteListenerChan       = make(chan *DeleteListenerTask, 100)
	ErrListenerNotFound      = errors.New("listener not found")
	ErrOtherListenerNotFound = errors.New("other listener not found")
)

func startDeleteListenerProccessor(concurrent int) {
	apiName := "DeleteLoadBalancerListeners"
	StartBatchProccessor(concurrent, apiName, true, DeleteListenerChan, func(region, lbId string, tasks []*DeleteListenerTask) {
		res, err := ApiCall(context.Background(), apiName, region, func(ctx context.Context, client *clb.Client) (req *clb.DeleteLoadBalancerListenersRequest, res *clb.DeleteLoadBalancerListenersResponse, err error) {
			req = clb.NewDeleteLoadBalancerListenersRequest()
			req.LoadBalancerId = &lbId
			for _, task := range tasks {
				req.ListenerIds = append(req.ListenerIds, &task.ListenerId)
			}
			res, err = client.DeleteLoadBalancerListeners(req)
			return
		})
		if err != nil {
			// 部分要删除的监听器不存在
			if strings.Contains(err.Error(), "Code=InvalidParameter") && strings.Contains(err.Error(), "some ListenerId") && strings.Contains(err.Error(), "not found") {
				for _, task := range tasks {
					if strings.Contains(err.Error(), task.ListenerId) { // 返回不存在的错误，让上层不要重试
						task.Result <- ErrListenerNotFound
					} else { // 因其它监听器不存在导致本批次监听器没有删除，需要重试
						task.Result <- ErrOtherListenerNotFound
					}
				}
			} else {
				// 其它错误，全部重试
				clbLog.V(10).Info("other bad error", "rawErr", err.Error())
				for _, task := range tasks {
					task.Result <- err
				}
			}
			return
		}
		_, err = Wait(context.Background(), region, *res.Response.RequestId, apiName)
		if err != nil {
			for _, task := range tasks {
				task.Result <- err
			}
			return
		}
		for _, task := range tasks {
			task.Result <- nil
		}
	})
}
