package v1beta1

import (
	"context"
	"fmt"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/imroc/tke-extend-network-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type DedicatedCLBListenerValidator struct {
	client.Client
}

// ValidateCreate implements admission.CustomValidator.
func (d *DedicatedCLBListenerValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	lis, ok := obj.(*v1beta1.DedicatedCLBListener)
	if !ok {
		return nil, fmt.Errorf("expected a DedicatedCLBListener object but got %T", obj)
	}
	return d.validate(lis)
}

// ValidateDelete implements admission.CustomValidator.
func (d *DedicatedCLBListenerValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

// ValidateUpdate implements admission.CustomValidator.
func (d *DedicatedCLBListenerValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (warnings admission.Warnings, err error) {
	lis, ok := newObj.(*v1beta1.DedicatedCLBListener)
	if !ok {
		return nil, fmt.Errorf("expected a DedicatedCLBListener object but got %T", newObj)
	}
	return d.validate(lis)
}

var _ webhook.CustomValidator = &DedicatedCLBListenerValidator{}

func (r *DedicatedCLBListenerValidator) validate(lis *v1beta1.DedicatedCLBListener) (admission.Warnings, error) {
	if err := r.validateLbPort(lis); err != nil {
		return nil, err
	}
	return nil, nil
}

func (d *DedicatedCLBListenerValidator) validateLbPort(lis *v1beta1.DedicatedCLBListener) error {
	var allErrs field.ErrorList
	for _, clb := range lis.Spec.CLBs {
		list := &v1beta1.DedicatedCLBListenerList{}
		err := d.List(
			context.Background(), list,
			client.MatchingFields{
				"spec.lbId": clb.ID,
				"spec.port": strconv.Itoa(int(lis.Spec.Port)),
			},
		)
		if err != nil {
			return err
		}
		var dup *v1beta1.DedicatedCLBListener
		for _, lis := range list.Items {
			if lis.DeletionTimestamp == nil {
				continue
			}
			dup = &lis
		}
		if dup != nil {
			lbPortPath := field.NewPath("spec").Child("port")
			allErrs = append(
				allErrs,
				field.Invalid(
					lbPortPath, lis.Spec.Port,
					fmt.Sprintf("lbPort is already used by othe(lis *DedicatedCLBListener) (%s/%s)", dup.Namespace, dup.Name),
				),
			)
		}
		if lis.Spec.EndPort != nil && *lis.Spec.EndPort <= lis.Spec.Port {
			lbEndPortPath := field.NewPath("spec").Child("endPort")
			allErrs = append(
				allErrs,
				field.Invalid(
					lbEndPortPath, lis.Spec.EndPort,
					fmt.Sprintf("lbEndPort(%d) should bigger than lbPort(%d)", *lis.Spec.EndPort, lis.Spec.Port),
				),
			)
		}
	}
	if len(allErrs) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{Group: "networking.cloud.tencent.com", Kind: "DedicatedCLBListener"},
			lis.Name,
			allErrs,
		)
	}
	return nil
}
