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

package v1alpha1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
)

// nolint:unused
// log is for logging in this package.
var clbpodbindinglog = logf.Log.WithName("clbpodbinding-resource")

// SetupCLBPodBindingWebhookWithManager registers the webhook for CLBPodBinding in the manager.
func SetupCLBPodBindingWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&networkingv1alpha1.CLBPodBinding{}).
		WithValidator(&CLBPodBindingCustomValidator{}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-networking-cloud-tencent-com-v1alpha1-clbpodbinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=clbpodbindings,verbs=create;update,versions=v1alpha1,name=vclbpodbinding-v1alpha1.kb.io,admissionReviewVersions=v1

// CLBPodBindingCustomValidator struct is responsible for validating the CLBPodBinding resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type CLBPodBindingCustomValidator struct{}

var _ webhook.CustomValidator = &CLBPodBindingCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type CLBPodBinding.
func (v *CLBPodBindingCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	clbpodbinding, ok := obj.(*networkingv1alpha1.CLBPodBinding)
	if !ok {
		return nil, fmt.Errorf("expected a CLBPodBinding object but got %T", obj)
	}
	clbpodbindinglog.Info("Validation for CLBPodBinding upon creation", "name", clbpodbinding.GetName())

	return nil, v.validate(clbpodbinding)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type CLBPodBinding.
func (v *CLBPodBindingCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	clbpodbinding, ok := newObj.(*networkingv1alpha1.CLBPodBinding)
	if !ok {
		return nil, fmt.Errorf("expected a CLBPodBinding object for the newObj but got %T", newObj)
	}
	clbpodbindinglog.Info("Validation for CLBPodBinding upon update", "name", clbpodbinding.GetName())

	return nil, v.validate(clbpodbinding)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type CLBPodBinding.
func (v *CLBPodBindingCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	clbpodbinding, ok := obj.(*networkingv1alpha1.CLBPodBinding)
	if !ok {
		return nil, fmt.Errorf("expected a CLBPodBinding object but got %T", obj)
	}
	clbpodbindinglog.Info("Validation for CLBPodBinding upon deletion", "name", clbpodbinding.GetName())
	return nil, nil
}

func (v *CLBPodBindingCustomValidator) validate(pb *networkingv1alpha1.CLBPodBinding) error {
	var allErrs field.ErrorList
	for portIndex, port := range pb.Spec.Ports {
		for poolIndex, poolName := range port.Pools {
			if !portpool.Allocator.IsPoolExists(poolName) {
				allErrs = append(
					allErrs,
					field.Invalid(
						field.NewPath("spec").Child("ports").Index(portIndex).Child("pools").Index(poolIndex), nil,
						fmt.Sprintf("port pool %q does not exist", poolName),
					),
				)
			}
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "networking.cloud.tencent.com", Kind: "CLBPodBinding"},
		pb.Name,
		allErrs,
	)
}
