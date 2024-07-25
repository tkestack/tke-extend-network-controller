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

package controller

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
)

// CLBListenerReconciler reconciles a CLBListener object
type CLBListenerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clblisteners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clblisteners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clblisteners/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CLBListener object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *CLBListenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	lis := &networkingv1alpha1.CLBListener{}
	if err := r.Get(ctx, req.NamespacedName, lis); err != nil {
		return ctrl.Result{}, err
	}
	id, err := clb.GetListenerId(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol)
	if err != nil {
		return ctrl.Result{}, err
	}
	if id != "" { // 监听器已创建，忽略
		return ctrl.Result{}, err
	}
	configName := lis.Spec.ListenerConfig
	if configName == "" {
		log.Info("ignore empty listenerConfig", "CLBListener", req.String())
		return ctrl.Result{}, nil
	}
	config := &networkingv1alpha1.CLBListenerConfig{}
	if err := r.Get(ctx, client.ObjectKey{Name: configName}, config); err != nil {
		if apierrors.IsNotFound(err) {
			config = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if err := clb.CreateListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, config.Spec.CreateListenerRequest(lis.Spec.LbPort, lis.Spec.Protocol)); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLBListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBListener{}).
		Complete(r)
}
