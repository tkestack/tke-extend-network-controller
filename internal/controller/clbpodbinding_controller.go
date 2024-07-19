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

	clbPodBinding := &networkingv1alpha1.CLBPodBinding{}
	err := r.Get(ctx, req.NamespacedName, clbPodBinding)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	finalizerName := "clbpodbinding.networking.cloud.tencent.com/finalizer"

	// handle finalizer and deletion
	if clbPodBinding.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(clbPodBinding, finalizerName) {
			controllerutil.AddFinalizer(clbPodBinding, finalizerName)
			if err = r.Update(ctx, clbPodBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
		if clbPodBinding.Status.State != "Success" {
			if err = r.syncCreate(ctx, clbPodBinding); err != nil {
				return ctrl.Result{}, nil
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(clbPodBinding, finalizerName) {
			err = r.syncDelete(ctx, clbPodBinding)
			if err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(clbPodBinding, finalizerName)
			if err = r.Update(ctx, clbPodBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *CLBPodBindingReconciler) syncCreate(ctx context.Context, clbPodBinding *networkingv1alpha1.CLBPodBinding) error {
	logger := log.FromContext(ctx)
	logger.Info("sync create CLBPodBinding", "name", clbPodBinding.Name, "namespace", clbPodBinding.Namespace)
	clbPodBinding.Status.State = "Success"
	return r.Status().Update(ctx, clbPodBinding)
}

func (r *CLBPodBindingReconciler) syncDelete(ctx context.Context, clbPodBinding *networkingv1alpha1.CLBPodBinding) error {
	logger := log.FromContext(ctx)
	logger.Info("sync delete CLBPodBinding", "name", clbPodBinding.Name, "namespace", clbPodBinding.Namespace)
	// bindings := clbPodBinding.Spec.Bindings
	// for _, binding := range bindings {
	// lbId := binding.LbId
	// }
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLBPodBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBPodBinding{}).
		Watches(&corev1.Pod{}, &podEventHandler{mgr.GetClient()}).
		Complete(r)
}

type podEventHandler struct {
	client.Client
}

// Create implements EventHandler.
func (e *podEventHandler) Create(ctx context.Context, evt event.TypedCreateEvent[client.Object], q workqueue.RateLimitingInterface) {
	obj := evt.Object
	logger := log.FromContext(context.TODO())
	err := e.Get(ctx, client.ObjectKeyFromObject(obj), &networkingv1alpha1.CLBPodBinding{})
	if err != nil {
		return
	}
	logger.Info("create event", "name", obj.GetName(), "namespace", obj.GetNamespace())
	q.Add(reconcile.Request{
		NamespacedName: client.ObjectKey{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
	})
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
	logger := log.FromContext(context.TODO())
	logger.Info("update event", "name", evt.ObjectNew.GetName(), "namespace", evt.ObjectNew.GetNamespace())
	q.Add(reconcile.Request{
		NamespacedName: client.ObjectKey{
			Namespace: newObj.GetNamespace(),
			Name:      newObj.GetName(),
		},
	})
}

// Delete implements EventHandler.
func (e *podEventHandler) Delete(ctx context.Context, evt event.TypedDeleteEvent[client.Object], q workqueue.RateLimitingInterface) {
	obj := evt.Object
	err := e.Get(ctx, client.ObjectKeyFromObject(obj), &networkingv1alpha1.CLBPodBinding{})
	if err != nil {
		return
	}
	logger := log.FromContext(context.TODO())
	logger.Info("delete event", "name", obj.GetName(), "namespace", obj.GetNamespace())
	q.Add(reconcile.Request{
		NamespacedName: client.ObjectKey{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
	})
}

// Generic implements EventHandler.
func (e *podEventHandler) Generic(ctx context.Context, evt event.TypedGenericEvent[client.Object], q workqueue.RateLimitingInterface) {
	obj := evt.Object
	logger := log.FromContext(context.TODO())
	logger.Info("generic event", "name", obj.GetName(), "namespace", obj.GetNamespace())
}
