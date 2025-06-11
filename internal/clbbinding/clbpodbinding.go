package clbbinding

import (
	"context"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/eventsource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func NewCLBPodBinding() *CLBPodBinding {
	return &CLBPodBinding{
		CLBPodBinding: &networkingv1alpha1.CLBPodBinding{},
	}
}

func WrapCLBPodBinding(pb *networkingv1alpha1.CLBPodBinding) *CLBPodBinding {
	return &CLBPodBinding{
		CLBPodBinding: pb,
	}
}

type CLBPodBinding struct {
	*networkingv1alpha1.CLBPodBinding
}

func (b *CLBPodBinding) GetSpec() *networkingv1alpha1.CLBBindingSpec {
	return &b.Spec
}

func (b *CLBPodBinding) GetStatus() *networkingv1alpha1.CLBBindingStatus {
	return &b.CLBPodBinding.Status
}

func (b *CLBPodBinding) GetObject() client.Object {
	return b.CLBPodBinding
}

func (b *CLBPodBinding) GetType() string {
	return "CLBPodBinding"
}

func (b *CLBPodBinding) FetchObject(ctx context.Context, c client.Client) (client.Object, error) {
	cpb := &networkingv1alpha1.CLBPodBinding{}
	err := c.Get(ctx, client.ObjectKeyFromObject(b.CLBPodBinding), cpb)
	if err == nil {
		b.CLBPodBinding = cpb
		return cpb, nil
	} else {
		return nil, errors.WithStack(err)
	}
}

type podBackend struct {
	*corev1.Pod
	client.Client
}

func (b podBackend) GetIP() string {
	return b.Pod.Status.PodIP
}

func (b podBackend) GetObject() client.Object {
	return b.Pod
}

func (b podBackend) GetAgonesGameServer() *agonesv1.GameServer {
	return nil
}

var ErrNodeNameIsEmpty = errors.New("node name is empty")

func (b podBackend) GetNode(ctx context.Context) (*corev1.Node, error) {
	nodeName := b.Pod.Spec.NodeName
	if nodeName == "" {
		return nil, ErrNodeNameIsEmpty
	}
	node := &corev1.Node{}
	err := b.Client.Get(ctx, client.ObjectKey{Name: nodeName}, node)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (b podBackend) TriggerReconcile() {
	eventsource.Pod <- event.TypedGenericEvent[client.Object]{
		Object: b.Pod,
	}
}

func (b *CLBPodBinding) GetAssociatedObject(ctx context.Context, apiClient client.Client) (Backend, error) {
	pod := &corev1.Pod{}
	if err := apiClient.Get(ctx, client.ObjectKeyFromObject(b), pod); err != nil {
		return nil, errors.WithStack(err)
	}
	return podBackend{pod, apiClient}, nil
}
