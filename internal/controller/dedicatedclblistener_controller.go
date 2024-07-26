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

// DedicatedCLBListenerReconciler reconciles a DedicatedCLBListener object
type DedicatedCLBListenerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclblisteners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclblisteners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclblisteners/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DedicatedCLBListener object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *DedicatedCLBListenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	lis := &networkingv1alpha1.DedicatedCLBListener{}
	if err := r.Get(ctx, req.NamespacedName, lis); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if lis.Status.State == "" {
		lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStatePending
		if err := r.Status().Update(ctx, lis); err != nil {
			log.Error(err, "failed to update status to Pending")
			return ctrl.Result{}, err
		}
	}

	finalizerName := "dedicatedclblistener.networking.cloud.tencent.com/finalizer"
	if lis.DeletionTimestamp.IsZero() { // 没有在删除状态
		// 确保 finalizers 存在
		if !controllerutil.ContainsFinalizer(lis, finalizerName) && controllerutil.AddFinalizer(lis, finalizerName) {
			if err := r.Update(ctx, lis); err != nil {
				log.Error(err, "failed to add finalizer")
			}
		}
		if err := r.sync(ctx, log, lis); err != nil {
			return ctrl.Result{}, err
		}
	} else { // 删除状态
		if err := r.syncDelete(ctx, log, lis); err != nil {
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

func (r *DedicatedCLBListenerReconciler) syncDelete(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	if lis.Status.State != networkingv1alpha1.DedicatedCLBListenerStateOccupied {
		return nil
	}
	// 解绑所有后端
	// if err := clb.DeregisterAllTargets(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol); err != nil {
	// 	return err
	// }
	if lis.Status.ListenerId != "" {
		return clb.DeleteListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId)
	} else {
		return clb.DeleteListenerByPort(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol)
	}
}

func (r *DedicatedCLBListenerReconciler) sync(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	if err := r.ensureListener(ctx, log, lis); err != nil {
		return err
	}
	if err := r.ensureDedicatedTarget(ctx, log, lis); err != nil {
		return err
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) ensureDedicatedTarget(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	// TODO: 从label取与pod的关联，如果有且pod正在删除，解绑rs并更新状态
	if lis.Spec.DedicatedTarget == nil { // 没配置DedicatedTarget或已删除，确保后端没绑定rs
		if lis.Status.State == networkingv1alpha1.DedicatedCLBListenerStateOccupied { // 但监听器状态是已占用，需要解绑
			// 解绑所有后端
			if err := clb.DeregisterAllTargets(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol); err != nil {
				return err
			}
			// 更新监听器状态
			lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStateAvailable
			if err := r.Status().Update(ctx, lis); err != nil {
				return err
			}
		}
		return nil
	}
	if lis.Status.State != networkingv1alpha1.DedicatedCLBListenerStateAvailable { // 如果已绑定，忽略
		return nil
	}
	// 绑定rs
	target := lis.Spec.DedicatedTarget
	if err := clb.RegisterTargets(
		ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol,
		clb.Target{TargetIP: target.IP, TargetPort: target.Port},
	); err != nil {
		return err
	}
	// 更新监听器状态
	lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStateOccupied
	if err := r.Status().Update(ctx, lis); err != nil {
		return err
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) ensureListener(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	state := lis.Status.State
	if state == networkingv1alpha1.DedicatedCLBListenerStateAvailable || state == networkingv1alpha1.DedicatedCLBListenerStateOccupied {
		id, err := clb.GetListenerId(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol)
		if err != nil {
			return err
		}
		log.Info("find listener", "region", lis.Spec.LbRegion, "lbId", lis.Spec.LbId, "port", lis.Spec.LbPort, "protocol", lis.Spec.Protocol, "id", id, "statusID", lis.Status.ListenerId)
		if id != "" {
			log.Info("found listener exsits", "port", lis.Spec.LbPort, "id", id)
			if id != lis.Status.ListenerId { // 监听器ID变化，需要重新创建
				log.Info("listener id changed, try to recreate")
				log.Info("delete old listener", "id", id)
				if err := clb.DeleteListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, id); err != nil {
					return err
				}
				lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStatePending
				lis.Status.ListenerId = ""
				if err := r.Status().Update(ctx, lis); err != nil {
					return err
				}
			} else { // 监听器ID符合预期，不需要重新创建，直接返回
				log.Info("listener id is expected, no need to recreate")
				return nil
			}
		} else {
			log.Info("listener not found, try to create")
		}
	}
	config := &networkingv1alpha1.CLBListenerConfig{}
	configName := lis.Spec.ListenerConfig
	if configName != "" {
		if err := r.Get(ctx, client.ObjectKey{Name: configName}, config); err != nil {
			if apierrors.IsNotFound(err) {
				config = nil
			} else {
				return err
			}
		}
	}

	// 监听器不存在或ID不符预期，创建监听器
	log.Info("try to create listener")
	id, err := clb.CreateListener(ctx, lis.Spec.LbRegion, config.Spec.CreateListenerRequest(lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol))
	if err != nil {
		return err
	}
	log.Info("listener successfully created", "id", id)
	lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStateAvailable
	lis.Status.ListenerId = id
	return r.Status().Update(ctx, lis)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DedicatedCLBListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.DedicatedCLBListener{}).
		Complete(r)
}
