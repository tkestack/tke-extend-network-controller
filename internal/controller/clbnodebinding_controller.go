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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/clbbinding"
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CLBNodeBinding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
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
func (r *CLBNodeBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBNodeBinding{}).
		Watches(
			&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(findObjectsForCLBPortMapping),
		).
		Named("clbnodebinding").
		Complete(r)
}
