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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CLBPodBindingReconciler reconciles a CLBPodBinding object
type CLBPodBindingReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	APIReader client.Reader
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbpodbindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbpodbindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbpodbindings/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;patch;update
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CLBPodBinding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *CLBPodBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile CLBPodBinding start", "namespace", req.Namespace, "name", req.Name)
	defer logger.Info("reconcile CLBPodBinding end", "namespace", req.Namespace, "name", req.Name)

	b := &networkingv1alpha1.CLBPodBinding{}
	err := r.Get(ctx, req.NamespacedName, b)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if b.Spec.PodName == "" {
		return ctrl.Result{}, nil
	}

	pod, err := r.getPodByClbPodBinding(ctx, b)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("pod not found, remove the podName from CLBPodBinding", "name", b.Name, "namespace", b.Namespace, "podName", b.Spec.PodName)
			b.Spec.PodName = ""
			if err := r.Update(ctx, b); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get pod when reconcile the CLBPodBinding", "name", b.Name, "namespace", b.Namespace)
		return ctrl.Result{}, err
	}

	podFinalizerName := "finializer.clbpodbinding.networking.cloud.tencent.com/" + b.Name
	shouldDeregister := false
	if pod.DeletionTimestamp.IsZero() { // Pod 没有在删除
		if !controllerutil.ContainsFinalizer(pod, podFinalizerName) { // 如果没有 finalizer 就自动加上
			if err = r.updatePodFinalizer(ctx, pod, podFinalizerName, true); err != nil {
				logger.Error(err, "failed to add pod finalizer for CLBPodBinding", "name", pod.Name, "namespace", pod.Namespace)
			}
		}
	} else { // Pod正在删除
		shouldDeregister = true
	}

	isClbPodBindingDeleting := false
	finalizerName := "clbpodbinding.networking.cloud.tencent.com/finalizer"
	if b.DeletionTimestamp.IsZero() { // 没有在删除状态
		if !controllerutil.ContainsFinalizer(b, finalizerName) && controllerutil.AddFinalizer(b, finalizerName) { // 如果没有 finalizer 就自动加上
			if err = r.Update(ctx, b); err != nil {
				logger.Error(err, "failed to add finalizer to CLBPodBinding", "name", b.Name, "namespace", b.Namespace)
			}
		}
		if err = r.syncRegister(ctx, b, pod); err != nil {
			return ctrl.Result{}, err
		}
	} else { // 正在删除状态
		isClbPodBindingDeleting = true
		shouldDeregister = true
	}

	if shouldDeregister {
		if err := r.syncDeregister(ctx, b, pod); err != nil {
			logger.Error(err, "failed to deregister target", "lbId", b.Spec.LbId, "lbPort", b.Spec.LbPort, "podName", b.Spec.PodName, "port", b.Spec.TargetPort)
			return ctrl.Result{}, err
		}
		if controllerutil.RemoveFinalizer(pod, podFinalizerName) {
			if err = r.updatePodFinalizer(ctx, pod, podFinalizerName, false); err != nil {
				logger.Error(
					err, "failed to remove pod finalizer for CLBPodBinding",
					"Pod", fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
					"CLBPodBinding", fmt.Sprintf("%s/%s", b.Namespace, b.Name),
				)
			}
		}
		b.Spec.PodName = ""
		if isClbPodBindingDeleting {
			controllerutil.RemoveFinalizer(b, finalizerName)
		}
		if err = r.Update(ctx, b); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *CLBPodBindingReconciler) updatePodFinalizer(ctx context.Context, obj client.Object, finalizerName string, add bool) error {
	pod := &corev1.Pod{}
	if err := r.APIReader.Get(ctx, client.ObjectKeyFromObject(obj), pod); err != nil {
		return err
	}
	if add { // 添加 finalizer
		if controllerutil.AddFinalizer(pod, finalizerName) {
			if err := r.Update(ctx, pod); err != nil {
				return err
			}
		}
	} else { // 删除 finalizer
		if controllerutil.RemoveFinalizer(pod, finalizerName) {
			if err := r.Update(ctx, pod); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *CLBPodBindingReconciler) getPodByClbPodBinding(ctx context.Context, b *networkingv1alpha1.CLBPodBinding) (pod *corev1.Pod, err error) {
	pod = &corev1.Pod{}
	err = r.Get(
		ctx,
		client.ObjectKey{
			Namespace: b.Namespace,
			Name:      b.Spec.PodName,
		},
		pod,
	)
	return
}

func (r *CLBPodBindingReconciler) syncRegister(ctx context.Context, b *networkingv1alpha1.CLBPodBinding, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	logger.Info("sync CLBPodBinding", "name", b.Name, "namespace", b.Namespace)
	target := clb.Target{
		TargetIP:   pod.Status.PodIP,
		TargetPort: b.Spec.TargetPort,
	}
	contains, err := clb.ContainsTarget(
		ctx,
		b.Spec.LbRegion,
		b.Spec.LbId,
		int64(b.Spec.LbPort),
		b.Spec.Protocol,
		target,
	)
	if err != nil {
		logger.Error(err, "failed to check target if exists", "lbId", b.Spec.LbId, "lbPort", b.Spec.LbPort, "target", target.String())
		return err
	}
	if contains { // 已绑定
		logger.Info("target already registered", "lbId", b.Spec.LbId, "lbPort", b.Spec.LbPort, "target", target.String())
		return nil
	}
	logger.Info("register target", "lbId", b.Spec.LbId, "lbPort", b.Spec.LbPort, "target", target.String())
	return clb.RegisterTargets(ctx, b.Spec.LbRegion, b.Spec.LbId, b.Spec.LbPort, b.Spec.Protocol, target)
}

func (r *CLBPodBindingReconciler) syncDeregister(ctx context.Context, b *networkingv1alpha1.CLBPodBinding, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	logger.Info("sync delete CLBPodBinding", "name", b.Name, "namespace", b.Namespace)

	return clb.DeregisterTargets(
		ctx,
		b.Spec.LbRegion,
		b.Spec.LbId,
		b.Spec.LbPort,
		b.Spec.Protocol,
		clb.Target{
			TargetIP:   pod.Status.PodIP,
			TargetPort: b.Spec.TargetPort,
		},
	)
}

const podNameField = "spec.podName"

// SetupWithManager sets up the controller with the Manager.
func (r *CLBPodBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	indexer := mgr.GetFieldIndexer()
	indexer.IndexField(context.TODO(), &networkingv1alpha1.CLBPodBinding{}, podNameField, func(o client.Object) []string {
		podName := o.(*networkingv1alpha1.CLBPodBinding).Spec.PodName
		if podName != "" {
			return []string{podName}
		}
		return nil
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBPodBinding{}).
		Watches(&corev1.Pod{}, &podEventHandler{mgr.GetClient()}).
		Complete(r)
}

type podEventHandler struct {
	client.Client
}

func (e *podEventHandler) triggerUpdate(ctx context.Context, obj client.Object, q workqueue.RateLimitingInterface) {
	logger := log.FromContext(ctx)
	list := &networkingv1alpha1.CLBPodBindingList{}
	err := e.List(
		ctx, list,
		client.MatchingFields{
			podNameField: obj.GetName(),
		},
		client.InNamespace(obj.GetNamespace()),
	)
	if err != nil {
		logger.Error(err, "failed to get CLBPodBinding")
		return
	}

	for _, b := range list.Items {
		q.Add(reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&b),
		})
	}
}

// Create implements EventHandler.
func (e *podEventHandler) Create(ctx context.Context, evt event.TypedCreateEvent[client.Object], q workqueue.RateLimitingInterface) {
	obj := evt.Object
	e.triggerUpdate(ctx, obj, q)
}

// Update implements EventHandler.
func (e *podEventHandler) Update(ctx context.Context, evt event.TypedUpdateEvent[client.Object], q workqueue.RateLimitingInterface) {
	newPod, ok := evt.ObjectNew.(*corev1.Pod)
	if !ok {
		return
	}
	oldPod, ok := evt.ObjectOld.(*corev1.Pod)
	if !ok {
		return
	}
	// 只在删除状态、Pod IP、Ready 状态发生变化时触发更新
	if !newPod.DeletionTimestamp.Equal(oldPod.DeletionTimestamp) ||
		newPod.Status.PodIP != oldPod.Status.PodIP ||
		isPodReady(oldPod) != isPodReady(newPod) {
		e.triggerUpdate(ctx, newPod, q)
	}
}

func isPodReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// Delete implements EventHandler.
func (e *podEventHandler) Delete(ctx context.Context, evt event.TypedDeleteEvent[client.Object], q workqueue.RateLimitingInterface) {
	obj := evt.Object
	logger := log.FromContext(ctx)
	logger.Info("delete event", "name", obj.GetName(), "namespace", obj.GetNamespace())
	e.triggerUpdate(ctx, obj, q)
}

// Generic implements EventHandler.
func (e *podEventHandler) Generic(ctx context.Context, evt event.TypedGenericEvent[client.Object], q workqueue.RateLimitingInterface) {
	obj := evt.Object
	logger := log.FromContext(ctx)
	logger.Info("generic event", "name", obj.GetName(), "namespace", obj.GetNamespace())
	e.triggerUpdate(ctx, obj, q)
}
