package clb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Listener struct {
	Port         int64
	EndPort      int64
	Protocol     string
	ListenerId   string
	ListenerName string
}

func convertListener(lbLis *clb.Listener) *Listener {
	lis := &Listener{
		ListenerId:   *lbLis.ListenerId,
		ListenerName: *lbLis.ListenerName,
		Protocol:     *lbLis.Protocol,
		Port:         *lbLis.Port,
	}
	if lbLis.EndPort != nil {
		lis.EndPort = *lbLis.EndPort
	}
	return lis
}

func GetListenerById(ctx context.Context, region, lbId string, listenerId string) (lis *Listener, err error) {
	// 有监听器 ID，尝试合并请求查询
	task := &DescribeListenerTask{
		Ctx:        ctx,
		Region:     region,
		LbId:       lbId,
		ListenerId: listenerId,
		Result:     make(chan *DescribeListenerResult),
	}
	DescribeListenerChan <- task
	result := <-task.Result
	if result.Err != nil {
		err = errors.WithStack(result.Err)
		return
	}
	lis = result.Listener
	return
}

// 如果有监听器 ID，尝试通过合并请求方式批量查询；如果没有监听器 ID，再尝试直接用端口查询
func GetListenerByIdOrPort(ctx context.Context, region, lbId string, listenerId string, port int64, protocol string) (lis *Listener, err error) {
	// 没有监听器 ID，尝试用端口+协议查询
	if listenerId == "" {
		lis, err = GetListenerByPort(ctx, region, lbId, port, protocol)
		if err != nil {
			err = errors.WithStack(err)
		}
		return
	}
	lis, err = GetListenerById(ctx, region, lbId, listenerId)
	if err != nil {
		err = errors.WithStack(err)
	}
	return
}

func GetListenerByPort(ctx context.Context, region, lbId string, port int64, protocol string) (lis *Listener, err error) {
	res, err := ApiCall(ctx, "DescribeListeners", region, func(ctx context.Context, client *clb.Client) (req *clb.DescribeListenersRequest, res *clb.DescribeListenersResponse, err error) {
		req = clb.NewDescribeListenersRequest()
		req.Port = &port
		req.LoadBalancerId = &lbId
		req.Protocol = &protocol
		res, err = client.DescribeListenersWithContext(ctx, req)
		return
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(res.Response.Listeners) > 0 {
		lis = convertListener(res.Response.Listeners[0])
		return
	}
	return
}

const TkePodListenerName = "TKE-DEDICATED-POD"

func CreateListenerTryBatch(ctx context.Context, region, lbId string, port, endPort int64, protocol string, certId *string, extensiveParameters string) (id string, err error) {
	cid := ""
	if certId != nil {
		cid = *certId
	}
	if endPort > 0 {
		id, err = CreateListener(ctx, region, lbId, port, endPort, protocol, cid, extensiveParameters, TkeListenerName)
		if err != nil {
			err = errors.WithStack(err)
		}
		return
	}
	task := &CreateListenerTask{
		Ctx:                 ctx,
		Region:              region,
		LbId:                lbId,
		CertId:              cid,
		Port:                port,
		Protocol:            protocol,
		ExtensiveParameters: extensiveParameters,
		Result:              make(chan *ListenerResult),
	}
	startTime := time.Now()
	CreateListenerChan <- task
	result := <-task.Result
	log.FromContext(ctx).V(10).Info("CreateListenerTryBatch performance", "cost", time.Since(startTime).String())
	id = result.ListenerId
	err = result.Err
	return
}

func CreateListener(ctx context.Context, region, lbId string, port, endPort int64, protocol, certId, extensiveParameters, listenerName string) (id string, err error) {
	req := clb.NewCreateListenerRequest()
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
	req.Ports = []*int64{&port}
	if endPort > 0 {
		req.EndPort = common.Uint64Ptr(uint64(endPort))
	}
	req.Protocol = &protocol
	req.ListenerNames = []*string{common.StringPtr(listenerName)}
	client := GetClient(region)
	mu := getLbLock(lbId)
	mu.Lock()
	defer mu.Unlock()
	before := time.Now()
	resp, err := client.CreateListenerWithContext(ctx, req)
	LogAPI(ctx, "CreateListener", req, resp, time.Since(before), err)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	if len(resp.Response.ListenerIds) == 0 {
		err = errors.New("no listener created")
		return
	}
	if len(resp.Response.ListenerIds) > 1 {
		err = fmt.Errorf("found %d listeners created", len(resp.Response.ListenerIds))
		return
	}
	_, err = Wait(ctx, region, *resp.Response.RequestId, "CreateListener")
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	id = *resp.Response.ListenerIds[0]
	return
}

func DeleteListenerByIdOrPort(ctx context.Context, region, lbId, listenerId string, port int64, protocol string) error {
	if listenerId == "" { // 没有监听器 ID，走慢路径
		if _, err := DeleteListenerByPort(ctx, region, lbId, port, protocol); err != nil {
			return errors.WithStack(err)
		}
		// 删除成功
		return nil
	} else { // 有监听器 ID，走快路径 (批量删除)
		if err := DeleteListenerById(ctx, region, lbId, listenerId); err != nil {
			return errors.WithStack(err)
		}
		// 删除成功
		return nil
	}
}

func DeleteListenerById(ctx context.Context, region, lbId, listenerId string) error {
	task := &DeleteListenerTask{
		Ctx:        ctx,
		Region:     region,
		LbId:       lbId,
		ListenerId: listenerId,
		Result:     make(chan error),
	}
	DeleteListenerChan <- task
	err := <-task.Result
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func DeleteListenerByPort(ctx context.Context, region, lbId string, port int64, protocol string) (id string, err error) {
	lis, err := GetListenerByPort(ctx, region, lbId, port, protocol)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	if lis == nil { // 监听器不存在，忽略
		return
	}
	id = lis.ListenerId
	err = DeleteListener(ctx, region, lbId, id)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	return
}

func DeleteListener(ctx context.Context, region, lbId, listenerId string) error {
	req := clb.NewDeleteListenerRequest()
	req.LoadBalancerId = &lbId
	req.ListenerId = &listenerId
	client := GetClient(region)
	mu := getLbLock(lbId)
	mu.Lock()
	defer mu.Unlock()
	resp, err := client.DeleteListenerWithContext(ctx, req)
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = Wait(ctx, region, *resp.Response.RequestId, "DeleteListener")
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
