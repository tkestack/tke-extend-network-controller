package util

import (
	"context"

	"github.com/pkg/errors"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PortPool struct {
	*networkingv1alpha1.CLBPortPool
	client.Client
}

func (p *PortPool) TryNotifyCreateLB(ctx context.Context) (int, error) {
	pp := networkingv1alpha1.CLBPortPool{}
	if err := p.Get(ctx, client.ObjectKeyFromObject(p.CLBPortPool), &pp); err != nil {
		return 0, errors.WithStack(err)
	}
	if pp.CanCreateLB() {
		return -1, nil
	}
	if p.CLBPortPool.Status.State == networkingv1alpha1.CLBPortPoolStateScaling { // 已经在扩容了
		return 2, nil
	}
	p.CLBPortPool.Status.State = networkingv1alpha1.CLBPortPoolStateScaling
	if err := p.Client.Status().Update(ctx, p.CLBPortPool); err != nil {
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
