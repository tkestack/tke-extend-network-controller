package util

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PortPool struct {
	*networkingv1alpha1.CLBPortPool
	client.Client
	record.EventRecorder
	mu       sync.Mutex
	creating bool
}

func (p *PortPool) TryCreateLB(ctx context.Context) (portpool.CreateLbResult, error) {
	if p.creating {
		return portpool.CreateLbResultCreating, nil
	}
	log.FromContext(ctx).V(10).Info("TryCreateLB")
	pp := &networkingv1alpha1.CLBPortPool{}
	if err := p.Get(ctx, client.ObjectKeyFromObject(p.CLBPortPool), pp); err != nil {
		return portpool.CreateLbResultError, errors.WithStack(err)
	}
	// 还未初始化的端口池，不能创建负载均衡器
	if pp.Status.State == "" || pp.Status.State == networkingv1alpha1.CLBPortPoolStatePending {
		return portpool.CreateLbResultForbidden, nil
	}
	// 没有显式启用自动创建的端口池，不能创建负载均衡器
	if pp.Spec.AutoCreate == nil || !pp.Spec.AutoCreate.Enabled {
		log.FromContext(ctx).V(10).Info("not able to create lb cuz auto create is not enabled")
		return portpool.CreateLbResultForbidden, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(pp.Status.LoadbalancerStatuses) > len(p.CLBPortPool.Status.LoadbalancerStatuses) {
		p.CLBPortPool = pp
	}
	// 自动创建的 CLB 数量达到配置上限的端口池，不能创建负载均衡器
	if !util.IsZero(pp.Spec.AutoCreate.MaxLoadBalancers) { // 要读结构体中的，cache 获取到不一定是实时最新的
		// 检查是否已创建了足够的 CLB
		num := uint16(0)
		for _, lbStatus := range p.CLBPortPool.Status.LoadbalancerStatuses {
			if lbStatus.AutoCreated != nil && *lbStatus.AutoCreated && lbStatus.State != networkingv1alpha1.LoadBalancerStateNotFound {
				num++
			}
		}
		// 如果已创建数量已满，则直接返回
		if num >= *pp.Spec.AutoCreate.MaxLoadBalancers {
			log.FromContext(ctx).V(10).Info("max auto-created loadbalancers is reached", "num", num, "max", *pp.Spec.AutoCreate.MaxLoadBalancers)
			return portpool.CreateLbResultForbidden, nil
		}
	}
	p.creating = true
	defer func() {
		p.creating = false
	}()
	p.EventRecorder.Event(pp, corev1.EventTypeNormal, "CreateLoadBalancer", "try to create clb")
	lbId, err := clb.CreateCLB(ctx, pp.GetRegion(), clb.ConvertCreateLoadBalancerRequest(pp.Spec.AutoCreate.Parameters))
	if err != nil {
		p.EventRecorder.Eventf(pp, corev1.EventTypeWarning, "CreateLoadBalancer", "create clb failed: %s", err.Error())
		return portpool.CreateLbResultError, errors.WithStack(err)
	}
	p.EventRecorder.Eventf(pp, corev1.EventTypeNormal, "CreateLoadBalancer", "create clb success: %s", lbId)
	if err := portpool.Allocator.AddLbId(pp.Name, lbId); err != nil {
		return portpool.CreateLbResultError, errors.WithStack(err)
	}
	addLbIdToStatus := func() error {
		pp := &networkingv1alpha1.CLBPortPool{}
		if err := p.Client.Get(ctx, client.ObjectKeyFromObject(p.CLBPortPool), pp); err != nil {
			return errors.WithStack(err)
		}
		pp.Status.State = networkingv1alpha1.CLBPortPoolStateActive // 创建成功，状态改为 Active，以便再次可分配端口
		pp.Status.LoadbalancerStatuses = append(pp.Status.LoadbalancerStatuses, networkingv1alpha1.LoadBalancerStatus{
			LoadbalancerID: lbId,
			AutoCreated:    util.GetPtr(true),
		})
		if err := p.Client.Status().Update(ctx, pp); err != nil {
			return errors.WithStack(err)
		}
		p.CLBPortPool = pp
		return nil
	}
	if err := util.RetryIfPossible(addLbIdToStatus); err != nil {
		return portpool.CreateLbResultError, errors.WithStack(err)
	}
	return portpool.CreateLbResultSuccess, nil
}

func (p *PortPool) GetStartPort() uint16 {
	return p.Spec.StartPort
}

func (p *PortPool) GetEndPort() uint16 {
	if p.Spec.EndPort == nil {
		return 65535
	}
	return *p.Spec.EndPort
}

func (p *PortPool) GetSegmentLength() uint16 {
	if p.Spec.SegmentLength == nil || *p.Spec.SegmentLength == 0 {
		return 1
	}
	return *p.Spec.SegmentLength
}

func NewPortPool(pp *networkingv1alpha1.CLBPortPool, c client.Client, recorder record.EventRecorder) *PortPool {
	return &PortPool{
		CLBPortPool:   pp,
		Client:        c,
		EventRecorder: recorder,
	}
}

func CanCreateLB(ctx context.Context, pp *networkingv1alpha1.CLBPortPool) bool {
	// 还未初始化的端口池，不能创建负载均衡器
	if pp.Status.State == "" || pp.Status.State == networkingv1alpha1.CLBPortPoolStatePending {
		return false
	}
	// 没有显式启用自动创建的端口池，不能创建负载均衡器
	if pp.Spec.AutoCreate == nil || !pp.Spec.AutoCreate.Enabled {
		log.FromContext(ctx).V(10).Info("not able to create lb cuz auto create is not enabled")
		return false
	}
	// 自动创建的 CLB 数量达到配置上限的端口池，不能创建负载均衡器
	if !util.IsZero(pp.Spec.AutoCreate.MaxLoadBalancers) {
		// 检查是否已创建了足够的 CLB
		num := uint16(0)
		for _, lbStatus := range pp.Status.LoadbalancerStatuses {
			if lbStatus.AutoCreated != nil && *lbStatus.AutoCreated && lbStatus.State != networkingv1alpha1.LoadBalancerStateNotFound {
				num++
			}
		}
		// 如果已创建数量已满，则直接返回
		if num >= *pp.Spec.AutoCreate.MaxLoadBalancers {
			return false
		}
		log.FromContext(ctx).V(10).Info("can create lb cus max loadbalancers is not reached", "num", num, "max", *pp.Spec.AutoCreate.MaxLoadBalancers)
	} else {
		log.FromContext(ctx).V(10).Info("can create lb cus auto create is enabled and not limit the max lb")
	}

	// 其余情况，允许创建负载均衡器
	return true
}
