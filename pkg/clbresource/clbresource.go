package clbresource

import (
	"context"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClbResource(ctx context.Context, r client.Reader, name string) (clb *networkingv1alpha1.CLB, err error) {
	ret := &networkingv1alpha1.CLB{}
	err = r.Get(
		ctx,
		client.ObjectKey{Name: name},
		ret,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = nil
		}
	} else {
		clb = ret
	}
	return
}
