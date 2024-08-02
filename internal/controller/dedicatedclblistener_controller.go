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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
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
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclblisteners/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get

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

func (r *DedicatedCLBListenerReconciler) cleanPodFinalizer(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	backend := lis.Spec.BackendPod
	if backend == nil {
		return nil
	}
	log = log.WithValues("pod", backend.PodName)
	log.V(5).Info("clean pod finalizer before delete DedicatedCLBListener")
	pod := &corev1.Pod{}
	err := r.Get(
		ctx,
		client.ObjectKey{
			Namespace: lis.Namespace,
			Name:      backend.PodName,
		},
		pod,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.V(5).Info("configured pod not found, ignore clean pod finalizer")
			return nil
		}
		return err
	}
	podFinalizerName := getDedicatedCLBListenerPodFinalizerName(lis)
	if controllerutil.ContainsFinalizer(pod, podFinalizerName) {
		if err := kube.RemovePodFinalizer(ctx, pod, podFinalizerName); err != nil {
			log.Error(err, "failed to remove pod finalizer")
			return err
		}
		log.V(5).Info("remove pod finalizer success")
	} else {
		log.V(5).Info("pod finalizer not found, ignore remove pod finalizer")
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) cleanListener(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	if lis.Status.ListenerId == "" {
		return nil
	}
	log = log.WithValues("listenerId", lis.Status.ListenerId, "port", lis.Spec.LbPort, "protocol", lis.Spec.Protocol)
	log.V(5).Info("start cleanListener")
	defer log.V(5).Info("end cleanListener")
	// 删除监听器
	log.V(5).Info("delete listener")
	listenerId, err := clb.DeleteListenerByPort(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol)
	if err != nil {
		return err
	}
	if listenerId != lis.Status.ListenerId {
		log.Info(
			"deleted clb port's listenerId is not equal to listenerId in status",
			"deletedListenerId", listenerId,
		)
	}
	log.V(7).Info("listener deleted, remove listenerId from status")
	lis.Status.ListenerId = ""
	if err := r.Status().Update(ctx, lis); err != nil {
		return err
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) syncDelete(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	state := lis.Status.State

	// 如果监听器还没创建过，直接返回
	if state == networkingv1alpha1.DedicatedCLBListenerStatePending || state == "" {
		return nil
	}

	// 确保处于删除状态，避免重入
	if state != networkingv1alpha1.DedicatedCLBListenerStateDeleting {
		lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStateDeleting
		if err := r.Status().Update(ctx, lis); err != nil {
			return err
		}
	}
	// 确保监听器已删除
	if err := r.cleanListener(ctx, log, lis); err != nil {
		return err
	}
	// 删除监听器后，清理后端 pod 的 finalizer
	return r.cleanPodFinalizer(ctx, log, lis)
}

func (r *DedicatedCLBListenerReconciler) sync(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	if err := r.ensureListener(ctx, log, lis); err != nil {
		return err
	}
	if err := r.ensureBackendPod(ctx, log, lis); err != nil {
		return err
	}
	return nil
}

func getDedicatedCLBListenerPodFinalizerName(lis *networkingv1alpha1.DedicatedCLBListener) string {
	return "dedicatedclblistener.networking.cloud.tencent.com/" + lis.Name
}

func (r *DedicatedCLBListenerReconciler) setState(ctx context.Context, lis *networkingv1alpha1.DedicatedCLBListener, state string) error {
	if lis.Status.State != state {
		log.FromContext(ctx).V(5).Info("set listener state", "state", state)
		lis.Status.State = state
		if err := r.Status().Update(ctx, lis); err != nil {
			return err
		}
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) ensureBackendPod(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	// 到这一步 listenerId 一定不为空
	log = log.WithValues("listenerId", lis.Status.ListenerId)
	// 没配置后端 pod，确保后端rs全被解绑，并且状态为 Available
	backendPod := lis.Spec.BackendPod
	if backendPod == nil {
		if lis.Status.State == networkingv1alpha1.DedicatedCLBListenerStateBound { // 但监听器状态是已占用，需要解绑
			log.V(6).Info("no backend pod configured, try to deregister all targets")
			// 解绑所有后端
			if err := clb.DeregisterAllTargets(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId); err != nil {
				return err
			}
			// 更新监听器状态
			if err := r.setState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStateAvailable); err != nil {
				return err
			}
		}
		return nil
	}
	// 有配置后端 pod，对pod对账
	log = log.WithValues("pod", backendPod.PodName, "port", backendPod.Port)
	log.V(6).Info("ensure backend pod")
	pod := &corev1.Pod{}
	err := r.Get(
		ctx,
		client.ObjectKey{
			Namespace: lis.Namespace,
			Name:      backendPod.PodName,
		},
		pod,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.V(5).Info("configured pod not found, ignore reconcile the pod")
			// 后端pod不存在，确保状态为 Available
			if lis.Status.State != networkingv1alpha1.DedicatedCLBListenerStateAvailable {
				log.V(5).Info("configured pod not found but state is not available, reset state to available", "oldState", lis.Status.State)
				lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStateAvailable
				if err := r.Status().Update(ctx, lis); err != nil {
					return err
				}
				return nil
			}
			return nil
		}
		return err
	}
	podFinalizerName := getDedicatedCLBListenerPodFinalizerName(lis)
	if pod.DeletionTimestamp.IsZero() { // pod 没有在删除
		// 确保 pod finalizer 存在
		if !controllerutil.ContainsFinalizer(pod, podFinalizerName) {
			log.V(6).Info(
				"add pod finalizer",
				"finalizerName", podFinalizerName,
			)
			if err := kube.AddPodFinalizer(ctx, pod, podFinalizerName); err != nil {
				log.Error(err, "failed to add pod finalizer")
				return err
			}
		}
		if pod.Status.PodIP == "" {
			log.V(5).Info("pod ip not ready, ignore")
			return nil
		}
		// 绑定rs
		targets, err := clb.DescribeTargets(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId)
		if err != nil {
			return err
		}
		toDel := []clb.Target{}
		needAdd := true
		for _, target := range targets {
			if target.TargetIP == pod.Status.PodIP && target.TargetPort == backendPod.Port {
				needAdd = false
			} else {
				toDel = append(toDel, target)
			}
		}
		if len(toDel) > 0 {
			log.Info("deregister extra rs", "extraRs", toDel)
			if err := clb.DeregisterTargetsForListener(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId, toDel...); err != nil {
				return err
			}
		}
		if !needAdd {
			log.V(6).Info("backend pod already registered", "podIP", pod.Status.PodIP)
			return nil
		}
		log.V(6).Info("register backend pod", "podIP", pod.Status.PodIP)
		if err := clb.RegisterTargets(
			ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId,
			clb.Target{TargetIP: pod.Status.PodIP, TargetPort: backendPod.Port},
		); err != nil {
			return err
		}
		// 更新监听器状态
		log.V(6).Info("set listener state to bound")
		lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStateBound
		if err := r.Status().Update(ctx, lis); err != nil {
			return err
		}
		// 更新Pod注解
		addr, err := clb.GetClbExternalAddress(ctx, lis.Spec.LbId, lis.Spec.LbRegion)
		if err != nil {
			return err
		}
		addr = fmt.Sprintf("%s:%d", addr, lis.Spec.LbPort)
		log.V(6).Info("set external address to status", "address", addr)
		lis.Status.Address = addr
		return r.Status().Update(ctx, lis)
	}

	// pod 正在删除,清理rs
	log.V(6).Info("pod deleting, try to deregister all targets")
	if err := clb.DeregisterAllTargets(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Status.ListenerId); err != nil {
		return err
	}
	// 清理成功，删除 pod finalizer
	log.V(6).Info(
		"pod deregisterd, remove pod finalizer",
		"finalizerName", podFinalizerName,
	)
	if err := kube.RemovePodFinalizer(ctx, pod, podFinalizerName); err != nil {
		return err
	}
	// 更新 DedicatedCLBListener
	return r.setState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStateAvailable)
}

func (r *DedicatedCLBListenerReconciler) ensureListener(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	// 如果监听器状态是 Pending，尝试创建
	if lis.Status.State == networkingv1alpha1.DedicatedCLBListenerStatePending {
		log.V(5).Info("listener is pending, try to create")
		return r.createListener(ctx, log, lis)
	}
	// 监听器已创建，检查监听器
	listenerId := lis.Status.ListenerId
	if listenerId == "" { // 不应该没有监听器ID，重建监听器
		log.Info("listener id not found from status, try to recreate", "state", lis.Status.State)
		lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStatePending
		if err := r.Status().Update(ctx, lis); err != nil {
			return err
		}
		return r.createListener(ctx, log, lis)
	}
	// 监听器存在，进行对账
	log.V(5).Info("ensure listener", "listenerId", listenerId)
	listener, err := clb.GetListenerByPort(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol)
	if err != nil {
		return err
	}
	if listener == nil { // 监听器不存在，重建监听器
		log.Info("listener not found, try to recreate", "listenerId", listenerId)
		return r.createListener(ctx, log, lis)
	}
	if listener.ListenerId != listenerId { // 监听器ID不匹配，更新监听器ID
		log.Info("listener id not match, update listenerId in status", "realListenerId", listener.ListenerId)
		lis.Status.ListenerId = listener.ListenerId
		if err := r.Status().Update(ctx, lis); err != nil {
			return err
		}
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
	existedLis, err := clb.GetListenerByPort(ctx, lis.Spec.LbRegion, lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol)
	if err != nil {
		return err
	}
	var listenerId string
	if existedLis != nil { // 端口冲突，如果是控制器创建的，则直接复用，如果不是，则报错引导用户人工确认手动清理冲突的监听器（避免直接重建误删用户有用的监听器）
		log = log.WithValues("listenerId", existedLis.ListenerId, "port", lis.Spec.LbPort, "protocol", lis.Spec.Protocol)
		log.Info("lb port already existed", "listenerName", existedLis.ListenerName)
		if existedLis.ListenerName == clb.TkePodListenerName { // 已经创建了，直接复用
			log.Info("reuse already existed listener")
			listenerId = existedLis.ListenerId
		} else {
			err = errors.New("lb port already existed, but not created by tke, please confirm and delete the conficted listener manually")
			log.Error(err, "listenerName", existedLis.ListenerName)
			return err
		}
	} else { // 没有端口冲突，创建监听器
		log.V(5).Info("try to create listener")
		id, err := clb.CreateListener(ctx, lis.Spec.LbRegion, config.Spec.CreateListenerRequest(lis.Spec.LbId, lis.Spec.LbPort, lis.Spec.Protocol))
		if err != nil {
			return err
		}
		log.V(5).Info("listener successfully created", "listenerId", id)
		listenerId = id
	}
	log.V(5).Info("listener ready, set state to available")
	lis.Status.State = networkingv1alpha1.DedicatedCLBListenerStateAvailable
	lis.Status.ListenerId = listenerId
	return r.Status().Update(ctx, lis)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DedicatedCLBListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("dedicatedclblistener").
		// For(&networkingv1alpha1.DedicatedCLBListener{}).
		// WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(
			&networkingv1alpha1.DedicatedCLBListener{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPod),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *DedicatedCLBListenerReconciler) findObjectsForPod(ctx context.Context, pod client.Object) []reconcile.Request {
	list := &networkingv1alpha1.DedicatedCLBListenerList{}
	log := log.FromContext(ctx)
	err := r.List(
		ctx,
		list,
		client.InNamespace(pod.GetNamespace()),
		client.MatchingFields{
			"spec.backendPod.podName": pod.GetName(),
		},
	)
	if err != nil {
		log.Error(err, "failed to list dedicatedclblisteners", "podName", pod.GetName())
		return []reconcile.Request{}
	}
	if len(list.Items) == 0 {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(list.Items))
	for i, item := range list.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

type podChangedPredicate struct {
	predicate.Funcs
}

func (podChangedPredicate) Update(e event.UpdateEvent) bool {
	oldPod := e.ObjectOld.(*corev1.Pod)
	newPod := e.ObjectNew.(*corev1.Pod)
	if oldPod == nil || newPod == nil {
		panic("unexpected nil pod")
	}
	//  只有正在删除或 IP 变化时才触发 DedicatedCLBListener 的对账
	if (oldPod.DeletionTimestamp == nil && newPod.DeletionTimestamp != nil) ||
		(oldPod.Status.PodIP != newPod.Status.PodIP) {
		return true
	}
	return false
}

// Pod 已彻底删除的事件不触发对账，避免解绑 Pod 后删除 pod finalizer 后触发删除事件又立即重新对账导致CLB接口报错:
// [TencentCloudSDKError] Code=FailedOperation.ResourceInOperating, Message=Your task is working (DelQLBL4ListenerDevice). Pls wait for the task to complete and try again.
func (podChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}
