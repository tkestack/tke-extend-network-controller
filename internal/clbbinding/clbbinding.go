package clbbinding

import (
	"context"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CLBBinding interface {
	client.Object
	GetSpec() *networkingv1alpha1.CLBBindingSpec
	GetStatus() *networkingv1alpha1.CLBBindingStatus
	GetAssociatedObject(context.Context, client.Client) (Backend, error)
	GetAssociatedObjectByIP(context.Context, client.Client, string) (Backend, error)
	// IsListenerOwnedByBackend 判断 other 对象是否通过自己的 CLBBinding 合法占用了指定的 listenerId。
	// 用于区分"IP 被无关对象复用"（历史残留，可安全清理）与"另一 binding 合法占用同一监听器"（真冲突，需保护）。
	IsListenerOwnedByBackend(ctx context.Context, c client.Client, other Backend, listenerId string) (bool, error)
	GetObject() client.Object
	GetType() string
	FetchObject(context.Context, client.Client) (client.Object, error)
}

type Backend interface {
	client.Object
	GetIP() string
	GetIPv6() string
	GetObject() client.Object
	GetNode(ctx context.Context) (*corev1.Node, error)
	TriggerReconcile()
}
