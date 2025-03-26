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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/clbbinding"
	"github.com/imroc/tke-extend-network-controller/internal/constant"
)

// CLBPodBindingReconciler reconciles a CLBPodBinding object
type CLBPodBindingReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbpodbindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbpodbindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbpodbindings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CLBPodBinding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *CLBPodBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ReconcileWithFinalizer(ctx, req, r.Client, &networkingv1alpha1.CLBPodBinding{}, r.sync, r.cleanup)
}

func (r *CLBPodBindingReconciler) cleanup(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding) (result ctrl.Result, err error) {
	rr := CLBBindingReconciler[*clbbinding.CLBPodBinding]{
		Client:   r.Client,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
	}
	return rr.cleanup(ctx, clbbinding.WrapCLBPodBinding(pb))
}

func (r *CLBPodBindingReconciler) sync(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding) (result ctrl.Result, err error) {
	rr := CLBBindingReconciler[*clbbinding.CLBPodBinding]{
		Client:   r.Client,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
	}
	return rr.sync(ctx, clbbinding.WrapCLBPodBinding(pb))
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLBPodBindingReconciler) SetupWithManager(mgr ctrl.Manager, workers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBPodBinding{}).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPod),
		).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workers,
		}).
		Named("clbpodbinding").
		Complete(r)
}

func (r *CLBPodBindingReconciler) findObjectsForPod(ctx context.Context, pod client.Object) []reconcile.Request {
	if time := pod.GetDeletionTimestamp(); time != nil && time.IsZero() { // 忽略正在删除的 Pod，默认情况下，Pod 删除完后会自动 GC 删除掉关联的 CLBPodBinding
		return []reconcile.Request{}
	}
	if anno := pod.GetAnnotations(); anno == nil || anno[constant.EnableCLBPortMappingsKey] == "" {
		return []reconcile.Request{}
	}
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      pod.GetName(),
				Namespace: pod.GetNamespace(),
			},
		},
	}
}
