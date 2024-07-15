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
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var clbpodbindinglog = logf.Log.WithName("clbpodbinding-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *CLBPodBinding) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-networking-cloud-tencent-com-v1alpha1-clbpodbinding,mutating=true,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=clbpodbindings,verbs=create;update,versions=v1alpha1,name=mclbpodbinding.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &CLBPodBinding{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CLBPodBinding) Default() {
	clbpodbindinglog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-networking-cloud-tencent-com-v1alpha1-clbpodbinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=clbpodbindings,verbs=create;update,versions=v1alpha1,name=vclbpodbinding.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CLBPodBinding{}

func (r *CLBPodBinding) validateCLBBinding() (admission.Warnings, error) {
	var allErrs field.ErrorList
	for i, bind := range r.Spec.Bindings {
		if bind.LbId == "" {
			allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("bindings").Index(i).Child("lbId"), "lbId is required"))
		}
		if bind.Port == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("bindings").Index(i).Child("port"), "port is required"))
		}
		if bind.Protocol == "" {
			allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("bindings").Index(i).Child("protocol"), "protocol is required"))
		}
		if bind.TargetPort == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("bindings").Index(i).Child("targetPort"), "targetPort is required"))
		}
	}
	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(
		schema.GroupKind{Group: "networking.cloud.tencent.com", Kind: "CLBPodBinding"},
		r.Name,
		allErrs,
	)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CLBPodBinding) ValidateCreate() (admission.Warnings, error) {
	clbpodbindinglog.Info("validate create", "name", r.Name)

	return r.validateCLBBinding()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CLBPodBinding) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	clbpodbindinglog.Info("validate update", "name", r.Name)

	return r.validateCLBBinding()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CLBPodBinding) ValidateDelete() (admission.Warnings, error) {
	clbpodbindinglog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
