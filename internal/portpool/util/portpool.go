package util

import (
	"context"

	"github.com/pkg/errors"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PortPool struct {
	*networkingv1alpha1.CLBPortPool
	client.Client
}

func (p *PortPool) TryNotifyCreateLB(ctx context.Context) (int, error) {
	pp := &networkingv1alpha1.CLBPortPool{}
	if err := p.Get(ctx, client.ObjectKeyFromObject(p.CLBPortPool), pp); err != nil {
		return 0, errors.WithStack(err)
	}
	if !CanCreateLB(ctx, pp) {
		return -1, nil
	}
	if pp.Status.State == networkingv1alpha1.CLBPortPoolStateScaling { // 已经在扩容了
		return 2, nil
	}
	pp.Status.State = networkingv1alpha1.CLBPortPoolStateScaling
	if err := p.Client.Status().Update(ctx, pp); err != nil {
		return 0, errors.WithStack(err)
	}
	return 1, nil // 成功通知扩容
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

func NewPortPool(pp *networkingv1alpha1.CLBPortPool, c client.Client) *PortPool {
	return &PortPool{
		CLBPortPool: pp,
		Client:      c,
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
