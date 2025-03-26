package clbbinding

import (
	"context"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CLBBinding interface {
	client.Object
	GetSpec() *networkingv1alpha1.CLBBindingSpec
	GetStatus() *networkingv1alpha1.CLBBindingStatus
	GetAssociatedObject(context.Context, client.Client) (Backend, error)
	EnsureWaitBackendState(context.Context, client.Client) error
	GetObject() client.Object
}

type Backend interface {
	client.Object
	GetIP() string
	GetObject() client.Object
	TriggerReconcile()
}
