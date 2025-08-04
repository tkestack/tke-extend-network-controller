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

	"github.com/tkestack/tke-extend-network-controller/pkg/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// nolint:unused
// log is for logging in this package.
var clbportpoollog = logf.Log.WithName("clbportpool-resource")

// SetupCLBPortPoolWebhookWithManager registers the webhook for CLBPortPool in the manager.
func SetupCLBPortPoolWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&networkingv1alpha1.CLBPortPool{}).
		WithValidator(&CLBPortPoolCustomValidator{}).
		WithDefaulter(&CLBPortPoolCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-networking-cloud-tencent-com-v1alpha1-clbportpool,mutating=true,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=clbportpools,verbs=create;update,versions=v1alpha1,name=mclbportpool-v1alpha1.kb.io,admissionReviewVersions=v1

// CLBPortPoolCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind CLBPortPool when those are created or updated.
type CLBPortPoolCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &CLBPortPoolCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind CLBPortPool.
func (d *CLBPortPoolCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	clbportpool, ok := obj.(*networkingv1alpha1.CLBPortPool)

	if !ok {
		return fmt.Errorf("expected an CLBPortPool object but got %T", obj)
	}
	clbportpoollog.Info("Defaulting for CLBPortPool", "name", clbportpool.GetName())

	util.SetIfEmpty(clbportpool.Spec.Region, clusterinfo.Region)
	return nil
}

// +kubebuilder:webhook:path=/validate-networking-cloud-tencent-com-v1alpha1-clbportpool,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.cloud.tencent.com,resources=clbportpools,verbs=create;update,versions=v1alpha1,name=vclbportpool-v1alpha1.kb.io,admissionReviewVersions=v1

// CLBPortPoolCustomValidator struct is responsible for validating the CLBPortPool resource
// when it is created, updated, or deleted.
type CLBPortPoolCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &CLBPortPoolCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type CLBPortPool.
func (v *CLBPortPoolCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	clbportpool, ok := obj.(*networkingv1alpha1.CLBPortPool)
	if !ok {
		return nil, fmt.Errorf("expected a CLBPortPool object but got %T", obj)
	}
	clbportpoollog.Info("Validation for CLBPortPool upon creation", "name", clbportpool.GetName())
	return nil, v.validate(clbportpool)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type CLBPortPool.
func (v *CLBPortPoolCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	clbportpool, ok := newObj.(*networkingv1alpha1.CLBPortPool)
	if !ok {
		return nil, fmt.Errorf("expected a CLBPortPool object for the newObj but got %T", newObj)
	}
	clbportpoollog.Info("Validation for CLBPortPool upon update", "name", clbportpool.GetName())
	return nil, v.validate(clbportpool)
}

func (v *CLBPortPoolCustomValidator) validate(pool *networkingv1alpha1.CLBPortPool) error {
	var allErrs field.ErrorList
	// 确保要有CLB，自动创建或使用已有 CLB，至少有一个指定
	if len(pool.Spec.ExsistedLoadBalancerIDs) == 0 {
		if pool.Spec.AutoCreate == nil || !pool.Spec.AutoCreate.Enabled {
			allErrs = append(
				allErrs,
				field.Invalid(
					field.NewPath("spec").Child("autoCreate").Child("enabled"), nil,
					"autoCreate should be enabled if there is no exsistedLoadBalancerIDs",
				),
			)
		}
	}

	// startPort 必须大于 0
	if pool.Spec.StartPort == 0 {
		allErrs = append(
			allErrs,
			field.Invalid(
				field.NewPath("spec").Child("startPort"), pool.Spec.StartPort,
				"startPort should be greater than 0",
			),
		)
	}

	// 如果定义了 endPort，要确保 endPort 大于 startPort，且不能与 listenerQuota 同时定义
	if pool.Spec.EndPort != nil {
		if *pool.Spec.EndPort <= pool.Spec.StartPort {
			allErrs = append(
				allErrs,
				field.Invalid(
					field.NewPath("spec").Child("endPort"), *pool.Spec.EndPort,
					"endPort should be greater than startPort",
				),
			)
		}
		if pool.Spec.ListenerQuota != nil {
			allErrs = append(
				allErrs,
				field.Invalid(
					field.NewPath("spec").Child("endPort"), *pool.Spec.EndPort,
					"endPort and listenerQuota cannot be specifed at the same time",
				),
			)
		}
	}

	// 如果手动配置了 listenerQuota (提工单对指定 clb 实例调整了 quota)，不能启用自动创建，
	// 因为自动创建粗的 clb 的 quota 只有账号维度的默认 quota 大小，导致同一个端口池中不同
	// clb quota 不一样，端口分配器无法正常工作。
	if pool.Spec.ListenerQuota != nil && pool.Spec.AutoCreate != nil && pool.Spec.AutoCreate.Enabled {
		allErrs = append(
			allErrs,
			field.Invalid(
				field.NewPath("spec").Child("listenerQuota"), *pool.Spec.ListenerQuota,
				"autoCreate should not be enabled if listenerQuota is specified",
			),
		)
	}

	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "networking.cloud.tencent.com", Kind: "CLBPortPool"},
		pool.Name,
		allErrs,
	)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type CLBPortPool.
func (v *CLBPortPoolCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	clbportpool, ok := obj.(*networkingv1alpha1.CLBPortPool)
	if !ok {
		return nil, fmt.Errorf("expected a CLBPortPool object but got %T", obj)
	}
	clbportpoollog.Info("Validation for CLBPortPool upon deletion", "name", clbportpool.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
