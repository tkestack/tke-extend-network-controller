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

	"github.com/tkestack/tke-extend-network-controller/internal/clbbinding"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"
	"github.com/tkestack/tke-extend-network-controller/pkg/eventsource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	CLBBindingReconciler[*clbbinding.CLBNodeBinding]
}

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return Reconcile(ctx, req, r.Client, &corev1.Node{}, r.sync)
}

func (r *NodeReconciler) sync(ctx context.Context, node *corev1.Node) (result ctrl.Result, err error) {
	result, err = r.syncCLBBinding(ctx, node, clbbinding.NewCLBNodeBinding())
	if err != nil {
		return result, errors.WithStack(err)
	}
	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager, workers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForNode),
		).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workers,
		}).
		WatchesRawSource(source.Channel(eventsource.Node, &handler.EnqueueRequestForObject{})).
		Named("node").
		Complete(r)
}

// 过滤带有 networking.cloud.tencent.com/enable-clb-port-mapping 注解的 Node
func (r *NodeReconciler) findObjectsForNode(_ context.Context, obj client.Object) []reconcile.Request {
	if anno := obj.GetAnnotations(); anno != nil && anno[constant.EnableCLBPortMappingsKey] != "" {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      obj.GetName(),
					Namespace: obj.GetNamespace(),
				},
			},
		}
	}
	return []reconcile.Request{}
}
