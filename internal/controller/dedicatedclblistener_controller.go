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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
)

// DedicatedCLBListenerReconciler reconciles a DedicatedCLBListener object
type DedicatedCLBListenerReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	APIReader client.Reader
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
	state := lis.Status.State
	if state != networkingv1alpha1.DedicatedCLBListenerStateOccupied && state != networkingv1alpha1.DedicatedCLBListenerStateAvailable {
		return nil
	}
	// 解绑所有后端
	// if err := clb.DeregisterAllTargets(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol); err != nil {
	// 	return err
	// }
	if lis.Status.ListenerId != "" {
		log.V(5).Info("delete listener", "listenerId", lis.Status.ListenerId)
		return clb.DeleteListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId)
	}
	return nil
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
	backendkPod := lis.Spec.BackendPod
	if backendkPod == nil { // 没配置后端 pod
		if lis.Status.State == networkingv1alpha1.DedicatedCLBListenerStateOccupied { // 但监听器状态是已占用，需要解绑
			// 解绑所有后端
			if err := clb.DeregisterAllTargets(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId); err != nil {
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
	pod := &corev1.Pod{}
	err := r.Get(
		ctx,
		client.ObjectKey{
			Namespace: lis.Namespace,
			Name:      backendkPod.PodName,
		},
		pod,
	)
	if err != nil {
		return err
	}
	podFinalizerName := "dedicatedclblistener.networking.cloud.tencent.com/" + lis.Name
	if pod.DeletionTimestamp.IsZero() { // pod 没有在删除
		// 确保 pod finalizer 存在
		if controllerutil.ContainsFinalizer(pod, podFinalizerName) {
			if err := util.UpdatePodFinalizer(
				ctx, pod, podFinalizerName, r.APIReader, r.Client, true,
			); err != nil {
				return err
			}
		}
		if pod.Status.PodIP == "" {
			return fmt.Errorf("no IP found for pod %s", pod.Name)
		}
		// 绑定rs
		targets, err := clb.DescribeTargets(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId)
		if err != nil {
			return err
		}
		toDel := []clb.Target{}
		needAdd := false
		for _, target := range targets {
			if target.TargetIP == pod.Status.PodIP && target.TargetPort == backendkPod.Port {
				needAdd = true
			} else {
				toDel = append(toDel, target)
			}
		}
		if len(toDel) > 0 {
			if err := clb.DeregisterTargetsForListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId, toDel...); err != nil {
				return err
			}
		}
		if !needAdd {
			return nil
		}
		if err := clb.RegisterTargets(
			ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId,
			clb.Target{TargetIP: pod.Status.PodIP, TargetPort: backendkPod.Port},
		); err != nil {
			return err
		}
		// 更新监听器状态
		lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStateOccupied
		if err := r.Status().Update(ctx, lis); err != nil {
			return err
		}
	} else { // pod 正在删除
		// 清理rs
		if err := clb.DeregisterAllTargets(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId); err != nil {
			return err
		}
		// 清理成功，删除 pod finalizer
		if controllerutil.ContainsFinalizer(pod, podFinalizerName) {
			if err := util.UpdatePodFinalizer(
				ctx, pod, podFinalizerName, r.APIReader, r.Client, false,
			); err != nil {
				return err
			}
		}
		// 更新 DedicatedCLBListener
		lis.Spec.BackendPod = nil
		if err := r.Update(ctx, lis); err != nil {
			return err
		}
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) ensureListener(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	switch lis.Status.State {
	case networkingv1alpha1.DedicatedCLBListenerStatePending:
		log.V(5).Info("listener is pending, try to create")
		return r.createListener(ctx, log, lis)
	case networkingv1alpha1.DedicatedCLBListenerStateAvailable, networkingv1alpha1.DedicatedCLBListenerStateOccupied:
		listenerId := lis.Status.ListenerId
		if listenerId == "" { // 不应该没有监听器ID，重建监听器
			lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStatePending
			if err := r.Status().Update(ctx, lis); err != nil {
				return err
			}
			log.Info("listener id not found, try to recreate")
			return r.createListener(ctx, log, lis)
		}
		listener, err := clb.GetListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, listenerId)
		if err != nil {
			return err
		}
		if listener == nil { // 监听器不存在，重建监听器
			log.Info("listener not found, try to recreate", "listenerId", listenerId)
			return r.createListener(ctx, log, lis)
		}
		if listener.Port != lis.Spec.LbPort || listener.Protocol != lis.Spec.Protocol { // 监听器端口和协议不符预期，清理重建
			log.Info("unexpected listener, invalid port or protocol, try to recreate", "want", fmt.Sprintf("%d/%s", lis.Spec.LbPort, lis.Spec.Protocol), "got", fmt.Sprintf("%d/%s", listener.Port, listener.Protocol))
			lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStatePending
			lis.Status.ListenerId = ""
			if err := r.Status().Update(ctx, lis); err != nil {
				return err
			}
			// 清理不需要的监听器
			if err := clb.DeleteListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, listener.ListenerId); err != nil {
				return err
			}
			// 重建监听器
			return r.createListener(ctx, log, lis)
		}
	}
	if lis.Status.ListenerId == "" {
		return errors.New("listener id is empty")
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) createListener(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
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
	// 创建监听器 TODO: 如果端口冲突，考虑强制删除重建监听器
	log.V(5).Info("try to create listener")
	id, err := clb.CreateListener(ctx, lis.Spec.LbRegion, config.Spec.CreateListenerRequest(lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol))
	if err != nil {
		return err
	}

	log.V(5).Info("listener successfully created", "id", id)
	lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStateAvailable
	lis.Status.ListenerId = id
	return r.Status().Update(ctx, lis)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DedicatedCLBListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.DedicatedCLBListener{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
