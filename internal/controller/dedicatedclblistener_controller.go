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
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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
	"github.com/imroc/tke-extend-network-controller/pkg/util"
)

const dedicatedCLBListenerFinalizer = "dedicatedclblistener.finalizers.networking.cloud.tencent.com"

// DedicatedCLBListenerReconciler reconciles a DedicatedCLBListener object
type DedicatedCLBListenerReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	APIReader client.Reader
	Recorder  record.EventRecorder
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclblisteners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclblisteners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclblisteners/finalizers,verbs=update
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclblisteners/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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
	lis := &networkingv1alpha1.DedicatedCLBListener{}
	if err := r.Get(ctx, req.NamespacedName, lis); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return util.RequeueIfConflict(r.reconcile(ctx, lis))
}

func (r *DedicatedCLBListenerReconciler) reconcile(ctx context.Context, lis *networkingv1alpha1.DedicatedCLBListener) error {
	if lis.Status.State == "" {
		if err := r.changeState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStatePending, ""); err != nil {
			return err
		}
	}

	log := log.FromContext(ctx)
	if !lis.DeletionTimestamp.IsZero() { // 正在删除
		if err := r.syncDelete(ctx, log, lis); err != nil { // 清理
			return err
		}
		// 监听器删除成功后再删除 finalizer
		if controllerutil.ContainsFinalizer(lis, dedicatedCLBListenerFinalizer) && controllerutil.RemoveFinalizer(lis, dedicatedCLBListenerFinalizer) {
			if err := r.Update(ctx, lis); err != nil {
				return err
			}
		}
		return nil
	}

	// 确保 finalizers 存在
	if !controllerutil.ContainsFinalizer(lis, dedicatedCLBListenerFinalizer) && controllerutil.AddFinalizer(lis, dedicatedCLBListenerFinalizer) {
		if err := r.Update(ctx, lis); err != nil {
			return err
		}
	}

	return r.sync(ctx, log, lis)
}

func (r *DedicatedCLBListenerReconciler) changeState(ctx context.Context, lis *networkingv1alpha1.DedicatedCLBListener, state, msg string) error {
	oldState := lis.Status.State
	newState := state
	if newState == oldState {
		return errors.New("state not changed")
	}
	lis.Status.State = newState
	lis.Status.Message = msg
	if err := r.Status().Update(ctx, lis); err != nil {
		return err
	}
	if oldState != "" {
		r.Recorder.Event(lis, corev1.EventTypeNormal, "UpdateStatus", fmt.Sprintf("listener %s -> %s", oldState, newState))
	} else {
		r.Recorder.Event(lis, corev1.EventTypeNormal, "UpdateStatus", fmt.Sprintf("listener %s", newState))
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) cleanPodFinalizer(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	target := lis.Spec.TargetPod
	if target == nil {
		return nil
	}
	log = log.WithValues("pod", target.PodName)
	log.V(5).Info("clean pod finalizer before delete DedicatedCLBListener")
	pod := &corev1.Pod{}
	err := r.Get(
		ctx,
		client.ObjectKey{
			Namespace: lis.Namespace,
			Name:      target.PodName,
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
		log.Info("pod finalizer not found, ignore remove pod finalizer")
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) cleanListener(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	if lis.Status.ListenerId == "" {
		return nil
	}
	log = log.WithValues("listenerId", lis.Status.ListenerId, "port", lis.Spec.LbPort, "protocol", lis.Spec.Protocol)
	// 删除监听器
	log.V(5).Info("delete listener")
	for _, lbId := range getLbIds(lis.Spec.LbId) {
		_, err := clb.DeleteListenerByPort(ctx, lis.Spec.LbRegion, lbId, lis.Spec.LbPort, lis.Spec.Protocol)
		if err != nil && !clb.IsLbIdNotFoundError(err) {
			r.Recorder.Event(lis, corev1.EventTypeWarning, "DeleteListener", err.Error())
			return err
		}
	}
	log.V(7).Info("listener deleted, remove listenerId from status")
	lis.Status.ListenerId = ""
	return r.Status().Update(ctx, lis)
}

func (r *DedicatedCLBListenerReconciler) syncDelete(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	state := lis.Status.State

	// 如果监听器还没创建过，直接返回
	if state == networkingv1alpha1.DedicatedCLBListenerStatePending || state == "" {
		return nil
	}

	// 确保处于删除状态，避免重入
	if state != networkingv1alpha1.DedicatedCLBListenerStateDeleting {
		if err := r.changeState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStateDeleting, ""); err != nil {
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

const podNameAnnotation = "dedicatedclblistener.networking.cloud.tencent.com/pod-name"

func (r *DedicatedCLBListenerReconciler) ensureBackendPod(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	if lis.Status.ListenerId == "" {
		return nil
	}
	log = log.WithValues("listenerId", lis.Status.ListenerId)
	// 没配置后端 pod，确保后端rs全被解绑，并且状态为 Available
	targetPod := lis.Spec.TargetPod
	podFinalizerName := getDedicatedCLBListenerPodFinalizerName(lis)
	lbInfos := getLbInfos(lis.Spec.LbId)
	ls := getListenerInfos(lis.Status.ListenerId)
	if len(lbInfos) != len(ls) {
		err := fmt.Errorf("exptected ListenerId length %d, got %d", len(lbInfos), len(ls))
		r.Recorder.Event(lis, corev1.EventTypeWarning, "EnsureBackendPod", err.Error())
		return err
	}

	if targetPod == nil {
		if lis.Status.State == networkingv1alpha1.DedicatedCLBListenerStateBound { // 但监听器状态是已占用，需要解绑
			if podName := lis.GetAnnotations()[podNameAnnotation]; podName != "" {
				pod := &corev1.Pod{}
				err := r.Get(
					ctx,
					client.ObjectKey{
						Namespace: lis.Namespace,
						Name:      podName,
					},
					pod,
				)
				if err == nil {
					if err = kube.RemovePodFinalizer(ctx, pod, podFinalizerName); err != nil {
						return err
					}
				} else {
					r.Recorder.Event(lis, corev1.EventTypeWarning, "Deregister", fmt.Sprintf("faild to find pod before deregister: %s", err.Error()))
				}
			}
			r.Recorder.Event(lis, corev1.EventTypeNormal, "PodChangeToNil", "no backend pod configured, try to deregister all targets")
			// 解绑所有后端
			for i, lbInfo := range lbInfos {
				if err := clb.DeregisterAllTargets(ctx, lis.Spec.LbRegion, lbInfo.LbId, ls[i].ListenerId); err != nil {
					r.Recorder.Event(lis, corev1.EventTypeWarning, "Deregister", err.Error())
					return err
				}
			}
			// 更新监听器状态
			if lis.Status.State != networkingv1alpha1.DedicatedCLBListenerStateAvailable {
				if err := r.changeState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStateAvailable, ""); err != nil {
					return err
				}
			}
		}
		return nil
	}
	// 有配置后端 pod，对pod对账
	log = log.WithValues("pod", targetPod.PodName, "port", targetPod.TargetPort)
	log.V(6).Info("ensure backend pod")
	// 确保记录当前 target pod 到注解以便 targetPod 置为 nil 后（不删除listener）可找到pod以删除pod finalizer
	if lis.Annotations == nil {
		lis.Annotations = make(map[string]string)
	}
	if lis.Annotations[podNameAnnotation] != targetPod.PodName {
		log.V(6).Info("set pod name annotation")
		lis.Annotations[podNameAnnotation] = targetPod.PodName
		if err := r.Update(ctx, lis); err != nil {
			return err
		}
	}
	pod := &corev1.Pod{}
	err := r.Get(
		ctx,
		client.ObjectKey{
			Namespace: lis.Namespace,
			Name:      targetPod.PodName,
		},
		pod,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.V(5).Info("configured pod not found, ignore reconcile the pod")
			// 后端pod不存在，确保状态为 Available
			if lis.Status.State != networkingv1alpha1.DedicatedCLBListenerStateAvailable {
				msg := fmt.Sprintf("configured pod not found but state is %q, reset state to available", lis.Status.State)
				r.Recorder.Event(lis, corev1.EventTypeNormal, "UpdateStatus", msg)
				if err := r.changeState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStateAvailable, ""); err != nil {
					return err
				}
				return nil
			}
			return nil
		}
		return err
	}

	if !pod.DeletionTimestamp.IsZero() { // pod 正在删除
		return r.syncPodDelete(ctx, log, lis, podFinalizerName, pod)
	}

	// 检测节点类型
	nodeName := pod.Spec.NodeName
	if nodeName == "" { // Pod 还没调度成功，忽略
		return nil
	}
	node := &corev1.Node{}
	err = r.Get(ctx, client.ObjectKey{Name: nodeName}, node)
	if err != nil {
		return err
	}
	if !util.IsNodeTypeSupported(node) {
		r.Recorder.Event(lis, corev1.EventTypeWarning, "NodeNotSupported", "node type not supported, please make sure pod scheduled to super node or native node")
		return nil
	}

	// 确保 pod finalizer 存在
	if !controllerutil.ContainsFinalizer(pod, podFinalizerName) {
		log.V(6).Info(
			"add pod finalizer",
			"finalizerName", podFinalizerName,
		)
		r.Recorder.Event(lis, corev1.EventTypeNormal, "AddPodFinalizer", "add finalizer to pod "+pod.Name)
		if err := kube.AddPodFinalizer(ctx, pod, podFinalizerName); err != nil {
			r.Recorder.Event(lis, corev1.EventTypeWarning, "AddPodFinalizer", fmt.Sprintf("failed to add finalizer to pod: %s", err.Error()))
			return err
		}
	}
	if pod.Status.PodIP == "" {
		log.V(5).Info("pod ip not ready, ignore")
		return nil
	}
	// 绑定rs
	for i, lbInfo := range lbInfos {
		listenerId := ls[i].ListenerId
		targets, err := clb.DescribeTargets(ctx, lis.Spec.LbRegion, lbInfo.LbId, listenerId)
		if err != nil {
			r.Recorder.Event(lis, corev1.EventTypeWarning, "DescribeTargets", err.Error())
			return err
		}
		toDel := []clb.Target{}
		needAdd := true
		for _, target := range targets {
			if target.TargetIP == pod.Status.PodIP && target.TargetPort == targetPod.TargetPort {
				needAdd = false
			} else {
				toDel = append(toDel, target)
			}
		}
		if len(toDel) > 0 {
			r.Recorder.Event(lis, corev1.EventTypeNormal, "DeregisterExtra", fmt.Sprintf("deregister extra rs %v", toDel))
			log.Info("deregister extra rs", "extraRs", toDel)
			if err := clb.DeregisterTargetsForListener(ctx, lis.Spec.LbRegion, lbInfo.LbId, listenerId, toDel...); err != nil {
				r.Recorder.Event(lis, corev1.EventTypeWarning, "DeregisterExtra", err.Error())
				return err
			}
		}
		if !needAdd {
			continue
		}
		r.Recorder.Event(
			lis, corev1.EventTypeNormal, "RegisterPod",
			fmt.Sprintf("register pod %s/%s:%d to %s", pod.Name, pod.Status.PodIP, targetPod.TargetPort, lbInfo.LbId),
		)
		if err := clb.RegisterTargets(
			ctx, lis.Spec.LbRegion, lbInfo.LbId, listenerId,
			clb.Target{TargetIP: pod.Status.PodIP, TargetPort: targetPod.TargetPort},
		); err != nil {
			r.Recorder.Event(lis, corev1.EventTypeWarning, "RegisterPod", err.Error())
			return err
		}
	}
	// 更新监听器状态
	if lis.Status.State != networkingv1alpha1.DedicatedCLBListenerStateBound {
		if err := r.changeState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStateBound, ""); err != nil {
			return err
		}
	}
	// 更新Pod注解
	var externalAddress string
	if len(lbInfos) == 1 {
		addr, err := clb.GetClbExternalAddress(ctx, lbInfos[0].LbId, lis.Spec.LbRegion)
		if err != nil {
			r.Recorder.Event(lis, corev1.EventTypeWarning, "GetClbExternalAddress", err.Error())
			return err
		}
		externalAddress = fmt.Sprintf("%s:%d", addr, lis.Spec.LbPort)
	} else {
		addresses := make(map[string]any)
		for _, lbInfo := range lbInfos {
			addr, err := clb.GetClbExternalAddress(ctx, lbInfo.LbId, lis.Spec.LbRegion)
			if err != nil {
				r.Recorder.Event(lis, corev1.EventTypeWarning, "GetClbExternalAddress", err.Error())
				return err
			}
			name := lbInfo.Name()
			addresses[name] = addr
		}
		addresses["port"] = lis.Spec.LbPort
		data, err := json.Marshal(addresses)
		if err != nil {
			return err
		}
		externalAddress = string(data)
	}
	lis.Status.Address = externalAddress
	if err := r.Status().Update(ctx, lis); err != nil {
		return err
	}
	r.Recorder.Event(lis, corev1.EventTypeNormal, "UpdateStatus", "got external address "+externalAddress)
	return nil
}

func (r *DedicatedCLBListenerReconciler) syncPodDelete(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener, podFinalizerName string, pod *corev1.Pod) error {
	r.Recorder.Event(lis, corev1.EventTypeNormal, "PodDeleting", "deregister all targets")
	lbIds := getLbIds(lis.Spec.LbId)
	ls := getListenerInfos(lis.Status.ListenerId)
	for i, lbId := range lbIds {
		if err := clb.DeregisterAllTargets(ctx, lis.Spec.LbRegion, lbId, ls[i].ListenerId); err != nil {
			r.Recorder.Event(lis, corev1.EventTypeWarning, "DeregisterFailed", err.Error())
			return err
		}
	}
	// 清理成功，删除 pod finalizer
	log.V(6).Info(
		"pod deregisterd, remove pod finalizer",
		"finalizerName", podFinalizerName,
	)

	r.Recorder.Event(lis, corev1.EventTypeNormal, "RemovePodFinalizer", "remove finalizer from pod "+pod.Name)
	if err := kube.RemovePodFinalizer(ctx, pod, podFinalizerName); err != nil {
		return err
	}
	// 更新 DedicatedCLBListener
	if lis.Status.State != networkingv1alpha1.DedicatedCLBListenerStateAvailable {
		if err := r.changeState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStateAvailable, ""); err != nil {
			return err
		}
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) createOneListener(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener, lbId string) (listenerId string, err error) {
	log.V(5).Info("try to create listener")
	existedLis, err := clb.GetListenerByPort(ctx, lis.Spec.LbRegion, lbId, lis.Spec.LbPort, lis.Spec.Protocol)
	if err != nil {
		r.Recorder.Event(lis, corev1.EventTypeWarning, "CreateListener", err.Error())
		return
	}
	if existedLis != nil { // 端口冲突，如果是控制器创建的，则直接复用，如果不是，则报错引导用户人工确认手动清理冲突的监听器（避免直接重建误删用户有用的监听器）
		log = log.WithValues("listenerId", existedLis.ListenerId, "port", lis.Spec.LbPort, "protocol", lis.Spec.Protocol)
		log.V(5).Info("lb port already existed", "listenerName", existedLis.ListenerName)
		if existedLis.ListenerName == clb.TkePodListenerName { // 已经创建了，直接复用
			log.V(5).Info("reuse already existed listener")
			listenerId = existedLis.ListenerId
		} else {
			msg := "lb port already existed, but not created by tke, please confirm and delete the conficted listener manually"
			r.Recorder.Event(
				lis, corev1.EventTypeWarning, "CreateListener",
				msg,
			)
			err = errors.New(msg)
			if lis.Status.State != networkingv1alpha1.DedicatedCLBListenerStateFailed {
				lis.Status.ListenerId = "" // 确保 status 里没有监听器ID，避免 ensureBackendPod 时误认为监听器已创建
				r.changeState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStateFailed, msg)
			}
			return
		}
	} else { // 没有端口冲突，创建监听器
		var id string
		id, err = clb.CreateListener(ctx, lis.Spec.LbRegion, lbId, lis.Spec.LbPort, lis.Spec.LbEndPort, lis.Spec.Protocol, lis.Spec.ExtensiveParameters)
		if err != nil {
			r.Recorder.Event(lis, corev1.EventTypeWarning, "CreateListener", err.Error())
			return
		}
		r.Recorder.Event(lis, corev1.EventTypeNormal, "CreateListener", fmt.Sprintf("clb listener successfully created (%s)", id))
		listenerId = id
	}
	return
}

func (r *DedicatedCLBListenerReconciler) ensureListener(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	// 如果监听器状态是 Pending，尝试创建
	switch lis.Status.State {
	case "", networkingv1alpha1.DedicatedCLBListenerStatePending, networkingv1alpha1.DedicatedCLBListenerStateFailed:
		return r.createListener(ctx, log, lis)
	}
	// 监听器已创建，检查监听器
	listenerId := lis.Status.ListenerId
	if listenerId == "" { // 不应该没有监听器ID，重建监听器
		msg := fmt.Sprintf("listener is %s state but no id not found in status, try to recreate", lis.Status.State)
		log.Info(msg)
		r.Recorder.Event(lis, corev1.EventTypeWarning, "ListenerIDNotFound", msg)
		if err := r.changeState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStatePending, ""); err != nil {
			return err
		}
		return r.createListener(ctx, log, lis)
	}
	// 监听器存在，进行对账
	log.V(5).Info("ensure listener", "listenerId", listenerId)
	var ls ListenerInfos
	lbInfos := getLbInfos(lis.Spec.LbId)
	for _, lbInfo := range lbInfos {
		listener, err := clb.GetListenerByPort(ctx, lis.Spec.LbRegion, lbInfo.LbId, lis.Spec.LbPort, lis.Spec.Protocol)
		if err != nil {
			r.Recorder.Event(lis, corev1.EventTypeWarning, "GetListenerByPort", err.Error())
			return err
		}
		if listener == nil { // 监听器不存在，重建监听器
			msg := "listener not found, try to recreate"
			log.Info(msg, "listenerId", listenerId)
			r.Recorder.Event(lis, corev1.EventTypeWarning, "ListenerNotFound", msg)
			return r.createListener(ctx, log, lis)
		}
		ls = append(ls, ListenerInfo{LbName: lbInfo.Name(), ListenerId: listener.ListenerId})
	}
	listenerIdResult := ls.String()
	if listenerIdResult != listenerId { // 监听器ID不匹配，更新监听器ID
		msg := fmt.Sprintf("listener id from status (%s) is not equal with the real listener id (%s), will override to the real listener id", listenerId, listenerIdResult)
		r.Recorder.Event(lis, corev1.EventTypeWarning, "ListenerIdNotMatch", msg)
		lis.Status.ListenerId = listenerIdResult
		if err := r.Status().Update(ctx, lis); err != nil {
			return err
		}
	}
	return nil
}

func (r *DedicatedCLBListenerReconciler) createListener(ctx context.Context, log logr.Logger, lis *networkingv1alpha1.DedicatedCLBListener) error {
	lbInfos := getLbInfos(lis.Spec.LbId)
	var listeners []string
	for _, lbInfo := range lbInfos {
		listenerId, err := r.createOneListener(ctx, log, lis, lbInfo.LbId)
		if err != nil {
			return err
		}
		listeners = append(listeners, listenerId)
	}
	if len(listeners) != len(lbInfos) {
		return fmt.Errorf("exptected %d listeners, got %d", len(lbInfos), len(listeners))
	}
	if len(lbInfos) == 1 {
		lis.Status.ListenerId = listeners[0]
	} else {
		for i, lbInfo := range lbInfos {
			listeners[i] = listeners[i] + "/" + lbInfo.Name()
		}
	}
	lis.Status.ListenerId = strings.Join(listeners, ",")
	return r.changeState(ctx, lis, networkingv1alpha1.DedicatedCLBListenerStateAvailable, "")
}

// SetupWithManager sets up the controller with the Manager.
func (r *DedicatedCLBListenerReconciler) SetupWithManager(mgr ctrl.Manager, workers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.DedicatedCLBListener{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workers,
		}).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPod),
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
			"spec.targetPod.podName": pod.GetName(),
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
