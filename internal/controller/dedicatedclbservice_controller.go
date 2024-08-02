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
	"strconv"

	corev1 "k8s.io/api/core/v1"
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
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
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
	log := log.FromContext(ctx)
	ds := &networkingv1alpha1.DedicatedCLBService{}
	if err := r.Get(ctx, req.NamespacedName, ds); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err := r.ensureStatus(ctx, ds); err != nil {
		return ctrl.Result{}, err
	}
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
	toUpdate, toCreate, err := r.diff(ctx, ds, pods.Items, listeners.Items)
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, listener := range toUpdate {
		if err := r.Update(ctx, listener); err != nil {
			return ctrl.Result{}, err
		}
	}
	for protocol, num := range toCreate {
		log.Info("create new listener", "protocol", protocol, "num", num)
		if err := r.allocateNewListener(ctx, log, ds, protocol, num); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("create new listener succeed", "protocol", protocol, "num", num)
	}
	if err := r.ensurePodAnnotation(ctx, ds, pods.Items); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *DedicatedCLBServiceReconciler) getAddr(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService, port int64, protocol string, pod *corev1.Pod) (addr string, err error) {
	list := &networkingv1alpha1.DedicatedCLBListenerList{}
	if err = r.List(
		ctx, list,
		client.InNamespace(ds.Namespace),
		client.MatchingLabels{labelKeyDedicatedCLBServiceName: ds.Name},
		client.MatchingFields{
			"spec.backendPod.podName": pod.Name,
			"spec.backendPod.port":    strconv.Itoa(int(port)),
			"spec.protocol":           protocol,
			"status.state":            networkingv1alpha1.DedicatedCLBListenerStateBound,
		},
	); err != nil {
		return
	}
	if len(list.Items) == 0 { // 没有已绑定的监听器，忽略
		return
	}
	if len(list.Items) > 1 {
		err = fmt.Errorf("found %d DedicatedCLBListener for backend pod", len(list.Items))
		log.FromContext(ctx).Error(err, "pod", pod.Name, "port", port, "protocol", protocol)
		return
	}
	lis := list.Items[0]
	if lis.Status.Address == "" {
		log.FromContext(ctx).Info("bound listener without external address", "listener", lis.Name)
		return
	}
	addr = lis.Status.Address
	return
}

func (r *DedicatedCLBServiceReconciler) ensurePodAnnotation(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService, pods []corev1.Pod) error {
	if len(pods) == 0 {
		return nil
	}
	for _, port := range ds.Spec.Ports {
		if port.AddressAnnotation == "" {
			continue
		}
		for _, pod := range pods {
			addr := pod.Annotations[port.AddressAnnotation]
			realAddr, err := r.getAddr(ctx, ds, port.TargetPort, port.Protocol, &pod)
			if err != nil {
				return err
			}
			if realAddr != "" && realAddr != addr {
				if err := kube.SetPodAnnotation(ctx, &pod, port.AddressAnnotation, realAddr); err != nil {
					return err
				}
			}
		}
	}
	return nil
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
	toUpdate []*networkingv1alpha1.DedicatedCLBListener, toCreate map[string]int,
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
	toCreate = make(map[string]int)
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
			toCreate[bind.Protocol]++
		}
	}
	return
}

func (r *DedicatedCLBServiceReconciler) allocateNewListener(ctx context.Context, log logr.Logger, ds *networkingv1alpha1.DedicatedCLBService, protocol string, num int) error {
	if len(ds.Status.LbList) == 0 {
		return errors.New("no clb found")
	}
	updateStatus := false
	var err error
	lbIndex := 0
	generateName := ds.Name + "-"
OUTER_LOOP:
	for n := num; n > 0; n-- { // 每个n个端口的循环
		for { // 分配单个lb端口的循环
			if lbIndex >= len(ds.Status.LbList) {
				log.Info("lb is not enough, stop trying allocate new listener")
				break OUTER_LOOP
			}
			lb := ds.Status.LbList[lbIndex]
			port := lb.MaxPort + 1
			if port < ds.Spec.MinPort {
				port = ds.Spec.MinPort
			}
			if port > ds.Spec.MaxPort { // 该lb已分配完所有端口，尝试下一个lb
				lbIndex++
				continue
			}
			// 没有同名监听器，新建
			lis := &networkingv1alpha1.DedicatedCLBListener{}
			lis.Spec.LbId = lb.LbId
			lis.Spec.LbPort = port
			lis.Spec.Protocol = protocol
			lis.Spec.ListenerConfig = ds.Spec.ListenerConfig
			lis.Spec.LbRegion = ds.Spec.LbRegion
			lis.Namespace = ds.Namespace
			lis.GenerateName = generateName
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
			updateStatus = true
			break // 创建成功，跳出本次端口分配的循环
		}
	}

	if updateStatus { // 有成功创建过，更新status
		log.V(5).Info("update lbList status", "lbList", ds.Status.LbList)
		if statusErr := r.Status().Update(ctx, ds); statusErr != nil {
			return statusErr // TODO: 两个err可能同时发生，出现err覆盖
		}
		return err
	}
	log.V(5).Info("no listener created")
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
