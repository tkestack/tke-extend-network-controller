package clbbinding

import (
	"context"
	"strings"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/pkg/errors"
	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/pkg/eventsource"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// GetIPv6 返回 Pod 的 IPv6 地址，如果 Pod 没有分配 IPv6 地址则返回空字符串
func (b podBackend) GetIPv6() string {
	for _, podIP := range b.Pod.Status.PodIPs {
		ip := podIP.IP
		if strings.Contains(ip, ":") {
			return ip
		}
	}
	return ""
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

func (b *CLBPodBinding) GetAssociatedObjectByIP(ctx context.Context, apiClient client.Client, ip string) (Backend, error) {
	podList := &corev1.PodList{}
	if err := apiClient.List(ctx, podList, client.MatchingFields{
		"status.podIP": ip,
	}); err != nil {
		return nil, errors.WithStack(err)
	}
	if len(podList.Items) > 0 {
		return podBackend{&podList.Items[0], apiClient}, nil
	}
	return nil, nil
}

func (b *CLBPodBinding) GetAssociatedObject(ctx context.Context, apiClient client.Client) (Backend, error) {
	pod := &corev1.Pod{}
	if err := apiClient.Get(ctx, client.ObjectKeyFromObject(b), pod); err != nil {
		return nil, errors.WithStack(err)
	}
	return podBackend{pod, apiClient}, nil
}

func (b *CLBPodBinding) IsListenerOwnedByBackend(ctx context.Context, c client.Client, other Backend, listenerId string) (bool, error) {
	obj := other.GetObject()
	cpb := &networkingv1alpha1.CLBPodBinding{}
	err := c.Get(ctx, client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}, cpb)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil // 该 Pod 没有 CLBPodBinding（如 fluent-bit）→ 非合法占用
		}
		return false, errors.WithStack(err)
	}
	for _, pb := range cpb.Status.PortBindings {
		if pb.ListenerId == listenerId {
			return true, nil // 该 Pod 确实经本组件占用了同一监听器 → 真冲突
		}
	}
	return false, nil
}
