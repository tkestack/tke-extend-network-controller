package clbresource

import (
	"context"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClbResource(ctx context.Context, c client.Client, lbId, region string) (clbresource *networkingv1alpha1.CLB, err error) {
	ret := &networkingv1alpha1.CLB{}
	err = c.Get(
		ctx,
		client.ObjectKey{Name: lbId},
		ret,
	)
	if err != nil {
		if apierrors.IsNotFound(err) { // cr 不能存在，检查clb是否存在
			_, err := clb.GetClb(ctx, lbId, region) // 如果clb存在，自动创建对应cr
			if err != nil {
				return nil, err
			}
			clbresource = &networkingv1alpha1.CLB{}
			clbresource.Name = lbId
			clbresource.Spec.Region = region
			err = c.Create(ctx, clbresource)
			if err != nil {
				return nil, err
			}
		}
	} else {
		clbresource = ret
	}
	return
}
