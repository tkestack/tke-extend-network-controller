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
	"strconv"

	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var dedicatedclblistenerlog = logf.Log.WithName("dedicatedclblistener-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *DedicatedCLBListener) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-networking-cloud-tencent-com-v1alpha1-dedicatedclblistener,mutating=true,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=dedicatedclblisteners,verbs=create;update,versions=v1alpha1,name=mdedicatedclblistener.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &DedicatedCLBListener{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *DedicatedCLBListener) Default() {
	dedicatedclblistenerlog.Info("default", "name", r.Name)
	r.Spec.LbRegion = clb.DefaultRegion()
}

func validateLbPort(lis *DedicatedCLBListener) error {
	list := &DedicatedCLBListenerList{}
	err := apiClient.List(
		context.Background(), list,
		client.MatchingFields{
			lbIdField:   lis.Spec.LbId,
			lbPortField: strconv.Itoa(int(lis.Spec.LbPort)),
		},
	)
	if err != nil {
		return err
	}
	if len(list.Items) > 0 {
		lbPortPath := field.NewPath("spec").Child("lbPort")
		var allErrs field.ErrorList
		allErrs = append(
			allErrs,
			field.Invalid(
				lbPortPath, lis.Spec.LbPort,
				fmt.Sprintf("lbPort is already used by other DedicatedCLBListener (%s/%s)", list.Items[0].Namespace, list.Items[0].Name),
			),
		)
		return apierrors.NewInvalid(
			schema.GroupKind{Group: "networking.cloud.tencent.com", Kind: "DedicatedCLBListener"},
			lis.Name,
			allErrs,
		)
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-networking-cloud-tencent-com-v1alpha1-dedicatedclblistener,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=dedicatedclblisteners,verbs=create;update,versions=v1alpha1,name=vdedicatedclblistener.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &DedicatedCLBListener{}

func (r *DedicatedCLBListener) validate() (admission.Warnings, error) {
	if err := validateLbPort(r); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DedicatedCLBListener) ValidateCreate() (admission.Warnings, error) {
	dedicatedclblistenerlog.Info("validate create", "name", r.Name)

	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DedicatedCLBListener) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	dedicatedclblistenerlog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DedicatedCLBListener) ValidateDelete() (admission.Warnings, error) {
	dedicatedclblistenerlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
