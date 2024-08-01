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
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
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
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get

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
	// log := log.FromContext(ctx)
	ds := &networkingv1alpha1.DedicatedCLBService{}
	if err := r.Get(ctx, req.NamespacedName, ds); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err := r.ensureStatus(ctx, ds); err != nil {
		return ctrl.Result{}, err
	}
	log := log.FromContext(ctx)
	log.V(7).Info("list related pods", "podNamespace", req.Namespace, "podSelector", ds.Spec.Selector)
	pods := &corev1.PodList{}
	if err := r.List(
		ctx, pods,
		client.InNamespace(req.Namespace),
		client.MatchingLabels(ds.Spec.Selector),
	); err != nil {
		return ctrl.Result{}, err
	}
	if len(pods.Items) == 0 {
		log.Info("no pods matches the selector")
		return ctrl.Result{}, nil
	}
	listeners := &networkingv1alpha1.DedicatedCLBListenerList{}
	if err := r.List(
		ctx, listeners,
		client.InNamespace(req.Namespace),
		client.MatchingLabels{labelKeyDedicatedCLBServiceName: req.Name},
	); err != nil {
		return ctrl.Result{}, err
	}
	log.V(5).Info("find related pods and listeners", "pods", len(pods.Items), "listeners", len(listeners.Items), "podSelector", ds.Spec.Selector)
	toUpdate, err := r.diff(ctx, ds, pods.Items, listeners.Items)
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, listener := range toUpdate {
		if err := r.Update(ctx, listener); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *DedicatedCLBServiceReconciler) ensureStatus(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService) error {
	lbMap := map[string]networkingv1alpha1.DedicatedCLBInfo{}
	for _, lb := range ds.Status.LbList {
		lbMap[lb.LbId] = lb
	}
	needUpdate := false
	for _, lbId := range ds.Spec.ExistedLbIds {
		if _, ok := lbMap[lbId]; !ok {
			ds.Status.LbList = append(ds.Status.LbList, networkingv1alpha1.DedicatedCLBInfo{LbId: lbId})
			needUpdate = true
		}
	}
	if needUpdate {
		return r.Status().Update(ctx, ds)
	}
	return nil
}

func getListenerKey(podName string, port int64, protocol string) string {
	return fmt.Sprintf("%s-%s-%d", podName, protocol, port)
}

func (r *DedicatedCLBServiceReconciler) diff(
	ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService,
	pods []corev1.Pod, listeners []networkingv1alpha1.DedicatedCLBListener,
) (
	toUpdate []*networkingv1alpha1.DedicatedCLBListener,
	err error,
) {
	log := log.FromContext(ctx)
	usedListeners := make(map[string]*networkingv1alpha1.DedicatedCLBListener)          // pod-port-protocol --> listener
	allocatableListeners := make(map[string][]*networkingv1alpha1.DedicatedCLBListener) // protocol --> listeners
	for _, listener := range listeners {
		backendPod := listener.Spec.BackendPod
		if backendPod == nil {
			allocatableListeners[listener.Spec.Protocol] = append(allocatableListeners[listener.Spec.Protocol], &listener)
		} else {
			usedListeners[getListenerKey(backendPod.PodName, backendPod.Port, listener.Spec.Protocol)] = &listener
		}
	}
	type bind struct {
		BackendPod *networkingv1alpha1.BackendPod
		Protocol   string
	}
	binds := []bind{}
	for _, pod := range pods {
		for _, port := range ds.Spec.Ports {
			key := getListenerKey(pod.Name, port.TargetPort, port.Protocol)
			_, ok := usedListeners[key]
			if ok { // pod 已绑定到监听器
				delete(usedListeners, key)
				continue
			}
			// 没绑定到监听器，尝试找一个
			binds = append(binds, bind{Protocol: port.Protocol, BackendPod: &networkingv1alpha1.BackendPod{PodName: pod.Name, Port: port.TargetPort}})
		}
	}
	// 所有pod都绑定了，直接返回
	if len(binds) == 0 {
		log.V(7).Info("all pods are bound, ignore")
		return
	}
	// 还有需要绑定的pod
	// 先将配置了其它未知Pod的监听器合并到可被分配的监听器列表
	for _, listener := range usedListeners {
		allocatableListeners[listener.Spec.Protocol] = append(allocatableListeners[listener.Spec.Protocol], listener)
	}
	if len(allocatableListeners) > 0 {
		for protocol, listeners := range allocatableListeners {
			log.V(7).Info("found allocatable listeners", "protocol", protocol, "listeners", len(listeners))
		}
	}
	listenersNeedCreate := make(map[string]int)
	for _, bind := range binds {
		listeners := allocatableListeners[bind.Protocol]
		if len(listeners) > 0 { // 还有可被分配的监听器
			listener := listeners[0]
			listener.Spec.BackendPod = bind.BackendPod
			toUpdate = append(toUpdate, listener)
			allocatableListeners[bind.Protocol] = allocatableListeners[bind.Protocol][1:]
			log.V(5).Info(
				"bind pod to listener",
				"listener", listener.Name,
				"pod", bind.BackendPod.PodName, "backendPort",
				bind.BackendPod.Port, "lbId", listener.Spec.LbId,
				"lbPort", listener.Spec.LbPort,
			)
		} else { // 没有可被分配的监听器，新建一个
			listenersNeedCreate[bind.Protocol]++
		}
	}
	if len(listenersNeedCreate) > 0 {
		for protocol, num := range listenersNeedCreate {
			log.Info("create new listener", "protocol", protocol, "num", num)
		}
		if err = r.allocateNewListener(ctx, log, ds, listenersNeedCreate); err != nil {
			return
		}
	}
	return
}

func (r *DedicatedCLBServiceReconciler) allocateNewListener(ctx context.Context, log logr.Logger, ds *networkingv1alpha1.DedicatedCLBService, num map[string]int) error {
	if len(ds.Status.LbList) == 0 {
		return errors.New("no clb found")
	}
	needNewListener := func() (protocol string, need bool) {
		for protocol, n := range num {
			n--
			if n <= 0 { // 后续已不再需要该协议的监听器，删除对应的计数器
				delete(num, protocol)
			}
			// 本次需要该协议的监听器
			return protocol, true
		}
		return "", false // 计数器已全部清空，不再需要任何监听器
	}
	created := false
	var err error
	lbIndex := 0
OUTER_LOOP:
	for {
		protocol, need := needNewListener()
		if !need {
			break
		}
		for lbIndex < len(ds.Status.LbList) {
			lb := ds.Status.LbList[lbIndex]
			if lb.MaxPort == 0 {
				lb.MaxPort = ds.Spec.MinPort - 1
			}
			if lb.MaxPort >= ds.Spec.MaxPort { // 该lb已分配完所有端口，尝试下一个lb
				lbIndex++
				continue
			}
			port := lb.MaxPort + 1
			name := strings.ToLower(fmt.Sprintf("%s-%d-%s", ds.Name, port, protocol))
			lis := &networkingv1alpha1.DedicatedCLBListener{}

			getErr := r.Get(ctx, client.ObjectKey{Namespace: ds.Namespace, Name: name}, lis)
			if getErr == nil { // 存在同名监听器，跳过
				continue
			}
			if !apierrors.IsNotFound(getErr) { // 其它错误，直接返回
				err = getErr
				break OUTER_LOOP
			}
			// 没有同名监听器，新建
			lis.Spec.LbId = lb.LbId
			lis.Spec.LbPort = port
			lis.Spec.Protocol = protocol
			lis.Spec.ListenerConfig = ds.Spec.ListenerConfig
			lis.Spec.LbRegion = ds.Spec.LbRegion
			lis.Namespace = ds.Namespace
			lis.Name = name
			lis.Labels = map[string]string{
				labelKeyDedicatedCLBServiceName: ds.Name,
			}
			if err = controllerutil.SetControllerReference(ds, lis, r.Scheme); err != nil {
				break OUTER_LOOP
			}
			log.Info("create new DedicatedCLBListener", "name", lis.Name, "lbId", lis.Spec.LbId)
			if err = r.Create(ctx, lis); err != nil {
				break OUTER_LOOP
			}
			lb.MaxPort = port
			ds.Status.LbList[lbIndex] = lb
			created = true
			if lb.MaxPort >= ds.Spec.MaxPort { // 该lb已分配完所有端口，尝试下一个lb
				lbIndex++
			}
		}
	}
	if created { // 有成功创建过，更新status
		log.V(5).Info("update status", "lbList", ds.Status.LbList)
		if statusErr := r.Status().Update(ctx, ds); statusErr != nil {
			return statusErr // TODO: 两个err可能同时发生，出现err覆盖
		}
		return err
	}
	return err
}

func (r *DedicatedCLBServiceReconciler) findObjectsForPod(ctx context.Context, pod client.Object) []reconcile.Request {
	list := &networkingv1alpha1.DedicatedCLBServiceList{}
	log := log.FromContext(ctx)
	err := r.List(
		ctx,
		list,
		client.InNamespace(pod.GetNamespace()),
	)
	if err != nil {
		log.Error(err, "failed to list dedicatedclbservices", "podName", pod.GetName())
		return []reconcile.Request{}
	}
	if len(list.Items) == 0 {
		return []reconcile.Request{}
	}
	podLabels := labels.Set(pod.GetLabels())
	requests := []reconcile.Request{}
	for _, ds := range list.Items {
		podSelector := labels.Set(ds.Spec.Selector).AsSelector()
		if podSelector.Matches(podLabels) {
			log.V(5).Info("pod matched dedicatedclbservice's selector, trigger reconcile", "pod", pod.GetName(), "dedicatedclbservice", ds.Name, "namespace", ds.Namespace)
			requests = append(
				requests,
				reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(&ds),
				},
			)
		}
	}
	return requests
}

const labelKeyDedicatedCLBServiceName = "networking.cloud.tencent.com/dedicatedclbservice-name"

// SetupWithManager sets up the controller with the Manager.
func (r *DedicatedCLBServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.DedicatedCLBService{}).
		Owns(&networkingv1alpha1.DedicatedCLBListener{}).
		Watches( // TODO: 只关注创建，删除待考虑
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPod),
		).
		Complete(r)
}
