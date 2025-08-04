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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/internal/clbbinding"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"
)

// CLBNodeBindingReconciler reconciles a CLBNodeBinding object
type CLBNodeBindingReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbnodebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbnodebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbnodebindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *CLBNodeBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ReconcileWithFinalizer(ctx, req, r.Client, &networkingv1alpha1.CLBNodeBinding{}, r.sync, r.cleanup)
}

func (r *CLBNodeBindingReconciler) cleanup(ctx context.Context, pb *networkingv1alpha1.CLBNodeBinding) (result ctrl.Result, err error) {
	rr := CLBBindingReconciler[*clbbinding.CLBNodeBinding]{
		Client:   r.Client,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
	}
	return rr.cleanup(ctx, clbbinding.WrapCLBNodeBinding(pb))
}

func (r *CLBNodeBindingReconciler) sync(ctx context.Context, pb *networkingv1alpha1.CLBNodeBinding) (result ctrl.Result, err error) {
	rr := CLBBindingReconciler[*clbbinding.CLBNodeBinding]{
		Client:   r.Client,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
	}
	return rr.sync(ctx, clbbinding.WrapCLBNodeBinding(pb))
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLBNodeBindingReconciler) SetupWithManager(mgr ctrl.Manager, workers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBNodeBinding{}).
		Watches(
			&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForNode),
		).
		Watches(
			&networkingv1alpha1.CLBPortPool{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForCLBPortPool),
		).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workers,
		}).
		Named("clbnodebinding").
		Complete(r)
}

func (r *CLBNodeBindingReconciler) findObjectsForNode(_ context.Context, node client.Object) []reconcile.Request {
	if anno := node.GetAnnotations(); anno != nil && anno[constant.EnableCLBPortMappingsKey] != "" {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      node.GetName(),
					Namespace: node.GetNamespace(),
				},
			},
		}
	}
	return nil
}

func (r *CLBNodeBindingReconciler) findObjectsForCLBPortPool(ctx context.Context, portpool client.Object) []reconcile.Request {
	list := &networkingv1alpha1.CLBNodeBindingList{}
	if err := r.List(ctx, list); err != nil {
		log.FromContext(ctx).Error(err, "failed to list CLBNodeBinding")
		return []reconcile.Request{}
	}
	ret := []reconcile.Request{}
	for _, cnb := range list.Items {
		if shouldNotify(portpool, cnb.Spec, cnb.Status) {
			ret = append(ret, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: cnb.GetName(),
				},
			})
		}
	}
	return ret
}
