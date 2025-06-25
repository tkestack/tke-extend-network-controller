package kube

import (
	"context"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
)

func GetCLBPortPool(ctx context.Context, name string) (*networkingv1alpha1.CLBPortPool, error) {
	pp := &networkingv1alpha1.CLBPortPool{}
	if err := apiClient.Get(ctx, types.NamespacedName{Name: name}, pp); err != nil {
		return nil, errors.WithStack(err)
	}
	return pp, nil
}
