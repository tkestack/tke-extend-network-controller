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
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"k8s.io/apimachinery/pkg/runtime"
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

var (
	// +kubebuilder:webhook:path=/validate-networking-cloud-tencent-com-v1alpha1-clbpodbinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=clbpodbindings,verbs=create;update,versions=v1alpha1,name=vclbpodbinding.kb.io,admissionReviewVersions=v1
	_ webhook.Validator = &CLBPodBinding{}

	// +kubebuilder:webhook:path=/mutate-networking-cloud-tencent-com-v1alpha1-clbpodbinding,mutating=true,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=clbpodbindings,verbs=create;update,versions=v1alpha1,name=mclbpodbinding.kb.io,admissionReviewVersions=v1
	_ webhook.Defaulter = &CLBPodBinding{}
)

// Default implements admission.Defaulter.
func (r *CLBPodBinding) Default() {
	r.Spec.LbRegion = clb.DefaultRegion()
}

// func (r *CLBPodBinding) validateCLBBindings() (admission.Warnings, error) {
// }

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CLBPodBinding) ValidateCreate() (admission.Warnings, error) {
	clbpodbindinglog.Info("validate create", "name", r.Name)

	return nil, nil
	// return r.validateCLBBindings()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CLBPodBinding) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	clbpodbindinglog.Info("validate update", "name", r.Name)

	return nil, nil
	// return r.validateCLBBindings()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CLBPodBinding) ValidateDelete() (admission.Warnings, error) {
	clbpodbindinglog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
