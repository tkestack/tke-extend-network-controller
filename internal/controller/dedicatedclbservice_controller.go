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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
)

// DedicatedCLBServiceReconciler reconciles a DedicatedCLBService object
type DedicatedCLBServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclbservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclbservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclbservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DedicatedCLBService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *DedicatedCLBServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconcile DedicatedCLBService start")
	defer log.Info("reconcile DedicatedCLBService end")
	ds := &networkingv1alpha1.DedicatedCLBService{}
	if err := r.Get(ctx, req.NamespacedName, ds); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(req.Namespace), client.MatchingLabels(ds.Spec.Selector)); err != nil {
		return ctrl.Result{}, err
	}
	for _, pod := range podList.Items {
		if err := r.syncPod(ctx, ds, &pod); err != nil { // TODO: 部分错误不影响其它pod
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

const (
	backendPodNameField = "spec.backendPod.podName"
	lbPortField         = "spec.lbPort"
	protocolField       = "spec.protocol"
)

func (r *DedicatedCLBServiceReconciler) syncPodPort(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService, pod *corev1.Pod, port int64, protocol string) error {
	list := &networkingv1alpha1.DedicatedCLBListenerList{}
	if err := r.List(
		ctx, list, client.InNamespace(pod.Namespace),
		client.MatchingFields{
			backendPodNameField: pod.Name,
			protocolField:       protocol,
		},
	); err != nil {
		return err
	}
	if len(list.Items) == 0 { // 没找到DedicatedCLBListener，创建一个
		if err := r.createDedicatedCLBListener(ctx, ds, pod, port, protocol); err != nil {
			return err
		}
	}
	if len(list.Items) > 1 {
		return fmt.Errorf("found %d DedicatedCLBListener for pod %s(%s/%d)", len(list.Items), pod.Name, protocol, port)
	}
	return nil
}

func (r *DedicatedCLBServiceReconciler) syncPod(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService, pod *corev1.Pod) error {
	for _, port := range ds.Spec.Ports {
		if err := r.syncPodPort(ctx, ds, pod, port.Port, port.Protocol); err != nil {
			return err
		}
	}
	return nil
}

func (r *DedicatedCLBServiceReconciler) useDedicatedCLBListener(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService, lis *networkingv1alpha1.DedicatedCLBListener, pod *corev1.Pod, port int64) error {
	lis.Spec.BackendPod = &networkingv1alpha1.BackendPod{
		PodName: pod.Name,
		Port:    port,
	}
	if err := r.Update(ctx, lis); err != nil {
		return err
	}
	return nil
}

func (r *DedicatedCLBServiceReconciler) createDedicatedCLBListener(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService, pod *corev1.Pod, port int64, protocol string) error {
	// 找到有空余端口的clb
	var clbInfo *networkingv1alpha1.DedicatedCLBInfo
	for _, lb := range ds.Status.LbList {
		if lb.MaxPort < ds.Spec.MaxPort {
			clbInfo = &lb
			break
		}
	}
	if clbInfo == nil {
		// TODO: 没有空余端口的clb，创建新clb
		ds.Status.LbList = append(ds.Status.LbList, *clbInfo)
	}
	lis := &networkingv1alpha1.DedicatedCLBListener{}
	lis.Namespace = ds.Namespace
	lis.Spec.ListenerConfig = ds.Spec.ListenerConfig
	lis.Spec.LbRegion = ds.Spec.LbRegion
	lis.Spec.LbId = clbInfo.LbId
	lis.Spec.LbPort = clbInfo.MaxPort + 1
	lis.Spec.BackendPod = &networkingv1alpha1.BackendPod{
		PodName: pod.Name,
		Port:    port,
	}
	if err := r.Create(ctx, lis); err != nil {
		return err
	}
	if err := r.Status().Update(ctx, lis); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DedicatedCLBServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	indexer := mgr.GetFieldIndexer()
	indexer.IndexField(
		context.TODO(), &networkingv1alpha1.DedicatedCLBListener{}, backendPodNameField,
		func(o client.Object) []string {
			backendPod := o.(*networkingv1alpha1.DedicatedCLBListener).Spec.BackendPod
			if backendPod != nil {
				return []string{backendPod.PodName}
			}
			return []string{""}
		},
	)
	indexer.IndexField(
		context.TODO(), &networkingv1alpha1.DedicatedCLBListener{}, lbPortField,
		func(o client.Object) []string {
			lbPort := o.(*networkingv1alpha1.DedicatedCLBListener).Spec.LbPort
			return []string{strconv.Itoa(int(lbPort))}
		},
	)
	indexer.IndexField(
		context.TODO(), &networkingv1alpha1.DedicatedCLBListener{}, protocolField,
		func(o client.Object) []string {
			protocol := o.(*networkingv1alpha1.DedicatedCLBListener).Spec.Protocol
			return []string{protocol}
		},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.DedicatedCLBService{}).
		Complete(r)
}
