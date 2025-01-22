package clb

import (
	"context"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CLBListenerAllocator struct {
	ID                 string
	Region             string
	maxListener        int
	allocatedListeners map[ListenerPort]bool
	quota              int
}

func (c *CLBListenerAllocator) Allocate(port int64, protocol string) {
	lp := ListenerPort{
		Protocol: protocol,
		Port:     port,
	}
	c.allocatedListeners[lp] = true
}

func (c *CLBListenerAllocator) CanAllocate(ctx context.Context, port int64, protocol string) (havePorts bool, canAllocate bool) {
	// 以下三种情况无法继续分配端口，其它 CLB 也应一起停止分配
	if len(c.allocatedListeners) >= c.quota { // 监听器数量超配额
		log.FromContext(ctx).V(9).Info("exceed quota when allocation", "lbId", c.ID, "port", port, "protocol", protocol, "allocatedListeners", len(c.allocatedListeners), "quota", c.quota)
		return
	}
	if c.maxListener > 0 && len(c.allocatedListeners) >= c.maxListener { // 监听器数量超配置数量
		log.FromContext(ctx).V(9).Info("exceed maxListener when allocation", "lbId", c.ID, "port", port, "protocol", protocol, "allocatedListeners", len(c.allocatedListeners), "maxListener", c.maxListener)
		return
	}
	// 还有剩余端口
	havePorts = true

	lp := ListenerPort{Protocol: protocol, Port: port}
	if c.allocatedListeners[lp] { // 端口已被占用
		return
	}
	// 端口没被占用，可被分配
	canAllocate = true
	return
}

func (c *CLBListenerAllocator) Init(ctx context.Context) (err error) {
	req := clb.NewDescribeListenersRequest()
	req.LoadBalancerId = &c.ID
	client := GetClient(c.Region)
	resp, err := client.DescribeListenersWithContext(ctx, req)
	if err != nil {
		return
	}
	c.allocatedListeners = make(map[ListenerPort]bool)
	for _, lis := range resp.Response.Listeners {
		c.allocatedListeners[ListenerPort{Protocol: *lis.Protocol, Port: *lis.Port}] = true
	}
	quota, err := GetListenerQuota(ctx, c.Region)
	if err != nil {
		return
	}
	log.FromContext(ctx).V(9).Info("get listener quota", "region", c.Region, "quota", quota)
	c.quota = int(quota)
	return
}

type ListenerAssignee interface {
	AssignListener(protocol string, port int64, clbs []*CLBListenerAllocator)
}

type ListenerAllocationRequest struct {
	Protocol  string
	Assignees []ListenerAssignee
}

type BatchListenerAllocator struct {
	CLBs                     []*CLBListenerAllocator
	MinPort, MaxPort         int64
	MaxListener, PortSegment *int64
}

func (l *BatchListenerAllocator) Init(ctx context.Context) (err error) {
	for _, clb := range l.CLBs {
		err = clb.Init(ctx)
		if err != nil {
			return
		}
		if l.MaxListener != nil {
			clb.maxListener = int(*l.MaxListener)
		}
	}
	return
}

func (l *BatchListenerAllocator) Allocate(ctx context.Context, reqs []*ListenerAllocationRequest) (err error) {
	log := log.FromContext(ctx)
	port := l.MinPort
	segment := int64(1)
	if l.PortSegment != nil {
		segment = *l.PortSegment
	}
OUT:
	for {
		tried := false
	MIDDLE:
		for _, req := range reqs {
			if len(req.Assignees) == 0 { // 当前 protocol 已分配完毕，尝试下一个 protocol
				log.V(9).Info("all allocated", "protocol", req.Protocol)
				continue
			}
			tried = true
			// IN:
			for _, clb := range l.CLBs {
				havePorts, canAllocate := clb.CanAllocate(ctx, port, req.Protocol)
				if !havePorts { // 有 CLB 无法继续分配端口，不再尝试所有 clb，跳出外层循环
					log.V(9).Info("no more ports", "protocol", req.Protocol, "lbId", clb.ID, "port", port)
					break OUT
				}
				if !canAllocate { // 当前 port + protocol 有 clb 已分配监听器，尝试下一个 protocol
					log.V(9).Info("port have been ocuppied", "protocol", req.Protocol, "lbId", clb.ID, "port", port)
					continue MIDDLE
				}
			}
			// 当前 port + protocol 没有 clb 已分配监听器，尝试分配
			for _, clb := range l.CLBs {
				clb.Allocate(port, req.Protocol)
			}
			log.V(9).Info("assign listener", "protocol", req.Protocol, "port", port, "clbs", l.CLBs)
			req.Assignees[0].AssignListener(req.Protocol, port, l.CLBs)
			req.Assignees = req.Assignees[1:]
		}
		if !tried {
			break
		}
		// 端口递增
		port += segment
		if l.MaxPort > 0 && port > l.MaxPort {
			break
		}
	}
	return
}
