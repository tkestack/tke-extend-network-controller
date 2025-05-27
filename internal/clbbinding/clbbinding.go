package clbbinding

import (
	"context"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CLBBinding interface {
	client.Object
	GetSpec() *networkingv1alpha1.CLBBindingSpec
	GetStatus() *networkingv1alpha1.CLBBindingStatus
	GetAssociatedObject(context.Context, client.Client) (Backend, error)
	GetObject() client.Object
	GetType() string
	GetNewest(context.Context, client.Client) (CLBBinding, error)
}

type Backend interface {
	client.Object
	GetIP() string
	GetObject() client.Object
	GetNode(ctx context.Context) (*corev1.Node, error)
	TriggerReconcile()
}
