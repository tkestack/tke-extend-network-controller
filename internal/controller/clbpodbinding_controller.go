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
	"reflect"

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
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbpodbindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbpodbindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbpodbindings/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
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

	finalizerName := "clbpodbinding.networking.cloud.tencent.com/finalizer"

	// handle finalizer and deletion
	if b.DeletionTimestamp.IsZero() { // 没有在删除状态
		if !controllerutil.ContainsFinalizer(b, finalizerName) { // 如果没有 finalizer 就自动加上
			controllerutil.AddFinalizer(b, finalizerName)
			if err = r.Update(ctx, b); err != nil {
				logger.Error(err, "failed to add finalizer to CLBPodBinding", "name", b.Name, "namespace", b.Namespace)
			}
		}
		if err = r.sync(ctx, b); err != nil {
			return ctrl.Result{}, nil
		}
	} else { // 正在删除状态
		if controllerutil.ContainsFinalizer(b, finalizerName) { // 只处理自己关注的 finalizer
			err = r.syncDelete(ctx, b)
			if err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(b, finalizerName) // 清理rs后清理 finalizer
			if err = r.Update(ctx, b); err != nil {
				logger.Error(err, "failed to remove finalizer to CLBPodBinding", "name", b.Name, "namespace", b.Namespace)
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *CLBPodBindingReconciler) getPodIpByClbPodBinding(ctx context.Context, b *networkingv1alpha1.CLBPodBinding) (ip string, err error) {
	pod := &corev1.Pod{}
	err = r.Get(
		ctx,
		client.ObjectKey{
			Namespace: b.Namespace,
			Name:      b.Spec.PodName,
		},
		pod,
	)
	if err != nil {
		return
	}
	ip = pod.Status.PodIP
	return
}

func (r *CLBPodBindingReconciler) sync(ctx context.Context, b *networkingv1alpha1.CLBPodBinding) error {
	logger := log.FromContext(ctx)
	logger.Info("sync CLBPodBinding", "name", b.Name, "namespace", b.Namespace)
	podIP, err := r.getPodIpByClbPodBinding(ctx, b)
	if err != nil {
		return err
	}
	target := clb.Target{
		TargetIP:   podIP,
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
		return err
	}
	if contains { // 已绑定
		logger.Info("target already registered", "lbId", b.Spec.LbId, "lbPort", b.Spec.LbPort, "target", target.String())
		return nil
	}
	return clb.RegisterTargets(ctx, b.Spec.LbRegion, b.Spec.LbId, b.Spec.LbPort, b.Spec.Protocol, target)
}

func (r *CLBPodBindingReconciler) syncDelete(ctx context.Context, b *networkingv1alpha1.CLBPodBinding) error {
	logger := log.FromContext(ctx)
	logger.Info("sync delete CLBPodBinding", "name", b.Name, "namespace", b.Namespace)
	podIP, err := r.getPodIpByClbPodBinding(ctx, b)
	if err != nil {
		return err
	}

	return clb.DeregisterTargets(
		ctx,
		b.Spec.LbRegion,
		b.Spec.LbId,
		b.Spec.LbPort,
		b.Spec.Protocol,
		clb.Target{
			TargetIP:   podIP,
			TargetPort: b.Spec.TargetPort,
		},
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLBPodBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	indexer := mgr.GetFieldIndexer()
	indexer.IndexField(context.TODO(), &networkingv1alpha1.CLBPodBinding{}, "spec.podName", func(o client.Object) []string {
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
	logger.Info("pod update", "name", obj.GetName(), "namespace", obj.GetNamespace())
	list := &networkingv1alpha1.CLBPodBindingList{}
	err := e.List(ctx, list, client.MatchingFields{
		"spec.podName": obj.GetName(),
	})
	if err != nil {
		logger.Error(err, "failed to get CLBPodBinding")
		return
	}
	for _, b := range list.Items {
		logger.Info("trigger CLBPodBinding update", "name", obj.GetName(), "namespace", obj.GetNamespace())
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
	newObj := evt.ObjectNew
	err := e.Get(ctx, client.ObjectKeyFromObject(newObj), &networkingv1alpha1.CLBPodBinding{})
	if err != nil {
		return
	}
	oldObj := evt.ObjectOld
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	e.triggerUpdate(ctx, newObj, q)
}

// Delete implements EventHandler.
func (e *podEventHandler) Delete(ctx context.Context, evt event.TypedDeleteEvent[client.Object], q workqueue.RateLimitingInterface) {
	obj := evt.Object
	err := e.Get(ctx, client.ObjectKeyFromObject(obj), &networkingv1alpha1.CLBPodBinding{})
	if err != nil {
		return
	}
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
