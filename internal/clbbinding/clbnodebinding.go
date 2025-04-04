package clbbinding

import (
	"context"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/eventsource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func NewCLBNodeBinding() *CLBNodeBinding {
	return &CLBNodeBinding{
		CLBNodeBinding: &networkingv1alpha1.CLBNodeBinding{},
	}
}

func WrapCLBNodeBinding(bd *networkingv1alpha1.CLBNodeBinding) *CLBNodeBinding {
	return &CLBNodeBinding{
		CLBNodeBinding: bd,
	}
}

type CLBNodeBinding struct {
	*networkingv1alpha1.CLBNodeBinding
}

func (b *CLBNodeBinding) GetSpec() *networkingv1alpha1.CLBBindingSpec {
	return &b.Spec
}

func (b *CLBNodeBinding) GetStatus() *networkingv1alpha1.CLBBindingStatus {
	return &b.Status
}

func (b *CLBNodeBinding) GetObject() client.Object {
	return b.CLBNodeBinding
}

type nodeBackend struct {
	*corev1.Node
}

func (b nodeBackend) GetIP() string {
	for _, address := range b.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}
	return ""
}

func (b nodeBackend) GetObject() client.Object {
	return b.Node
}

func (b nodeBackend) TriggerReconcile() {
	eventsource.Node <- event.TypedGenericEvent[client.Object]{
		Object: b.Node,
	}
}

func (b *CLBNodeBinding) GetAssociatedObject(ctx context.Context, apiClient client.Client) (Backend, error) {
	node := &corev1.Node{}
	if err := apiClient.Get(ctx, client.ObjectKeyFromObject(b), node); err != nil {
		return nil, errors.WithStack(err)
	}
	return nodeBackend{node}, nil
}

func (b *CLBNodeBinding) EnsureWaitBackendState(ctx context.Context, apiClient client.Client) error {
	if b.Status.State != networkingv1alpha1.CLBBindingStateWaitForNode {
		b.Status.State = networkingv1alpha1.CLBBindingStateWaitForNode
		b.Status.Message = "wait pod network to be ready"
		if err := apiClient.Status().Update(ctx, b.CLBNodeBinding); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
