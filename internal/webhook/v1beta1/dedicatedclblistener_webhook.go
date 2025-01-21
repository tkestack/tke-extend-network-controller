/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"context"
	"fmt"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	networkingv1beta1 "github.com/imroc/tke-extend-network-controller/api/v1beta1"
)

// nolint:unused
// log is for logging in this package.
var dedicatedclblistenerlog = logf.Log.WithName("dedicatedclblistener-resource")

// SetupDedicatedCLBListenerWebhookWithManager registers the webhook for DedicatedCLBListener in the manager.
func SetupDedicatedCLBListenerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&networkingv1beta1.DedicatedCLBListener{}).
		WithValidator(&DedicatedCLBListenerCustomValidator{mgr.GetClient()}).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-networking-cloud-tencent-com-v1beta1-dedicatedclblistener,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=dedicatedclblisteners,verbs=create;update,versions=v1beta1,name=vdedicatedclblistener-v1beta1.kb.io,admissionReviewVersions=v1

// DedicatedCLBListenerCustomValidator struct is responsible for validating the DedicatedCLBListener resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type DedicatedCLBListenerCustomValidator struct {
	client.Client
}

var _ webhook.CustomValidator = &DedicatedCLBListenerCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type DedicatedCLBListener.
func (v *DedicatedCLBListenerCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	lis, ok := obj.(*networkingv1beta1.DedicatedCLBListener)
	if !ok {
		return nil, fmt.Errorf("expected a DedicatedCLBListener object but got %T", obj)
	}
	dedicatedclblistenerlog.Info("Validation for DedicatedCLBListener upon creation", "name", lis.GetName())

	return v.validate(lis)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type DedicatedCLBListener.
func (v *DedicatedCLBListenerCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	lis, ok := newObj.(*networkingv1beta1.DedicatedCLBListener)
	if !ok {
		return nil, fmt.Errorf("expected a DedicatedCLBListener object for the newObj but got %T", newObj)
	}
	dedicatedclblistenerlog.Info("Validation for DedicatedCLBListener upon update", "name", lis.GetName())
	return v.validate(lis)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type DedicatedCLBListener.
func (v *DedicatedCLBListenerCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *DedicatedCLBListenerCustomValidator) validate(lis *networkingv1beta1.DedicatedCLBListener) (admission.Warnings, error) {
	if err := v.validateLbPort(lis); err != nil {
		return nil, err
	}
	return nil, nil
}

func (d *DedicatedCLBListenerCustomValidator) validateLbPort(lis *networkingv1beta1.DedicatedCLBListener) error {
	var allErrs field.ErrorList
	for _, clb := range lis.Spec.CLBs {
		list := &networkingv1beta1.DedicatedCLBListenerList{}
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
		var dup *networkingv1beta1.DedicatedCLBListener
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
					fmt.Sprintf("lbPort is already used by other DedicatedCLBListener(%s/%s)", dup.Namespace, dup.Name),
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
