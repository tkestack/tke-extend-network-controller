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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
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
	finalizerName := "clblistener.networking.cloud.tencent.com/finalizer"
	// TODO: 一次reconcile可能多次update，合并成一次
	if lis.DeletionTimestamp.IsZero() { // 没有在删除状态
		// 确保 finalizers 存在
		if !controllerutil.ContainsFinalizer(lis, finalizerName) && controllerutil.AddFinalizer(lis, finalizerName) {
			if err := r.Update(ctx, lis); err != nil {
				log.Error(err, "failed to add finalizer")
			}
		}
		if err := r.syncAdd(ctx, log, lis); err != nil {
			return ctrl.Result{}, err
		}
	} else { // 删除状态
		// 清理CLB监听器
		if err := r.syncDelete(ctx, lis); err != nil {
			return ctrl.Result{}, err
		}
		// 监听器删除成功后再删除 finalizer
		if controllerutil.RemoveFinalizer(lis, finalizerName) {
			if err := r.Update(ctx, lis); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *CLBListenerReconciler) syncDelete(ctx context.Context, lis *networkingv1alpha1.CLBListener) error {
	id, err := getListenerId(ctx, lis)
	if err != nil {
		return err
	}
	// TODO: 如果监听器被手动删除，这里会返回错误，需要忽略
	return clb.DeleteListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, id)
}

func (r *CLBListenerReconciler) syncAdd(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.CLBListener) error {
	id, err := getListenerId(ctx, lis)
	if err != nil {
		return err
	}
	if id != "" { // 监听器已创建
		if id != lis.Status.ListenerId { // 更新监听器ID
			lis.Status.ListenerId = id
			if err := r.Update(ctx, lis); err != nil {
				return err
			}
		}
		return nil
	}
	// 创建监听器
	configName := lis.Spec.ListenerConfig
	if configName == "" {
		log.Info("ignore empty listenerConfig")
		return nil
	}
	config := &networkingv1alpha1.CLBListenerConfig{}
	if err := r.Get(ctx, client.ObjectKey{Name: configName}, config); err != nil {
		if apierrors.IsNotFound(err) {
			config = nil
		} else {
			return err
		}
	}
	id, err = clb.CreateListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, config.Spec.CreateListenerRequest(lis.Spec.LbPort, lis.Spec.Protocol))
	if err != nil {
		return err
	}
	lis.Status.ListenerId = id
	return r.Update(ctx, lis)
}

func getListenerId(ctx context.Context, lis *networkingv1alpha1.CLBListener) (string, error) {
	return clb.GetListenerId(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLBListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBListener{}).
		Complete(r)
}
