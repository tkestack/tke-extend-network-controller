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
	"math"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
)

// DedicatedCLBServiceReconciler reconciles a DedicatedCLBService object
type DedicatedCLBServiceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclbservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclbservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=dedicatedclbservices/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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
	ds := &networkingv1alpha1.DedicatedCLBService{}
	if err := r.Get(ctx, req.NamespacedName, ds); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return util.RequeueIfConflict(r.reconcile(ctx, ds))
}

const dedicatedCLBServiceFinalizer = "dedicatedclbservice.finalizers.networking.cloud.tencent.com"

func (r *DedicatedCLBServiceReconciler) syncDelete(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService) error {
	lbToDelete := []string{}
	for _, lb := range ds.Status.LbList {
		if lb.AutoCreate {
			lbToDelete = append(lbToDelete, lb.LbId)
		}
	}
	if len(lbToDelete) > 0 {
		r.Recorder.Event(ds, corev1.EventTypeNormal, "DeleteCLB", fmt.Sprintf("delete auto created clb instances: %v", lbToDelete))
		if err := clb.Delete(ctx, ds.Spec.LbRegion, lbToDelete...); err != nil {
			r.Recorder.Event(ds, corev1.EventTypeWarning, "DeleteCLB", err.Error())
			return err
		}
	}
	return nil
}

func (r *DedicatedCLBServiceReconciler) reconcile(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService) error {
	if !ds.DeletionTimestamp.IsZero() { // 正在删除
		if err := r.syncDelete(ctx, ds); err != nil { // 清理
			return err
		}
		// 确保自动创建的CLB清理后，移除finalizer
		if controllerutil.ContainsFinalizer(ds, dedicatedCLBServiceFinalizer) && controllerutil.RemoveFinalizer(ds, dedicatedCLBServiceFinalizer) {
			if err := r.Update(ctx, ds); err != nil {
				return err
			}
		}
		return nil
	}

	// 确保 finalizers 存在
	if !controllerutil.ContainsFinalizer(ds, dedicatedCLBServiceFinalizer) && controllerutil.AddFinalizer(ds, dedicatedCLBServiceFinalizer) {
		if err := r.Update(ctx, ds); err != nil {
			return err
		}
	}

	return r.sync(ctx, ds)
}

func (r *DedicatedCLBServiceReconciler) sync(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService) error {
	if err := r.ensureStatus(ctx, ds); err != nil {
		return err
	}
	log := log.FromContext(ctx)
	log.V(7).Info("list related pods", "podNamespace", ds.Namespace, "podSelector", ds.Spec.Selector)
	pods := &corev1.PodList{}
	if err := r.List(
		ctx, pods,
		client.InNamespace(ds.Namespace),
		client.MatchingLabels(ds.Spec.Selector),
	); err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		log.V(5).Info("no pods matches the selector")
		return nil
	}
	listeners := &networkingv1alpha1.DedicatedCLBListenerList{}
	if err := r.List(
		ctx, listeners,
		client.InNamespace(ds.Namespace),
		client.MatchingLabels{labelKeyDedicatedCLBServiceName: ds.Name},
	); err != nil {
		return err
	}
	log.V(5).Info(
		"find related pods and listeners",
		"pods", len(pods.Items),
		"listeners", len(listeners.Items),
		"podSelector", ds.Spec.Selector,
	)
	toUpdate, toCreate, boundListeners, err := r.diff(ctx, ds, pods.Items, listeners.Items)
	if err != nil {
		return err
	}
	for _, listener := range toUpdate {
		if err := r.Update(ctx, listener); err != nil {
			return err
		}
	}
	for protocol, num := range toCreate {
		if createdNum, err := r.allocateNewListener(ctx, log, ds, protocol, num); err != nil {
			r.Recorder.Event(ds, corev1.EventTypeWarning, "CreateListener", fmt.Sprintf("failed to create %d listeners with %s protocol: %s", num, protocol, err.Error()))
			return err
		} else {
			if createdNum > 0 {
				r.Recorder.Event(ds, corev1.EventTypeNormal, "CreateListener", fmt.Sprintf("successfully created %d listeners with %s protocol", createdNum, protocol))
			}
			if ds.Spec.LbAutoCreate.Enable {
				if gapNum := num - createdNum; gapNum > 0 {
					lbNum := float64(gapNum) / (float64(ds.Spec.MaxPort) - float64(ds.Spec.MinPort) + 1)
					lbToCreate := int(math.Ceil(lbNum))
					r.Recorder.Event(ds, corev1.EventTypeNormal, "CreateCLB", fmt.Sprintf("clb is not enough, try to create %d clb instance", lbToCreate))
					if err := r.allocateNewCLB(ctx, ds, lbToCreate); err != nil {
						r.Recorder.Event(ds, corev1.EventTypeWarning, "CreateCLB", fmt.Sprintf("failed to create %d clb instance: %s", lbToCreate, err.Error()))
						return err
					}
					break
				}
			}
		}
	}
	if err := r.ensurePodAnnotation(ctx, ds, pods.Items, boundListeners); err != nil {
		return err
	}
	return nil
}

func (r *DedicatedCLBServiceReconciler) allocateNewCLB(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService, num int) error {
	ids, err := clb.Create(ctx, ds.Spec.LbRegion, ds.Spec.VpcId, ds.Spec.LbAutoCreate.ConfigJson, num)
	if err != nil {
		return err
	}
	for _, lbId := range ids {
		ds.Status.LbList = append(ds.Status.LbList, networkingv1alpha1.DedicatedCLBInfo{
			LbId:       lbId,
			AutoCreate: true,
		})
	}
	return r.Status().Update(ctx, ds)
}

func (r *DedicatedCLBServiceReconciler) ensurePodAnnotation(ctx context.Context, ds *networkingv1alpha1.DedicatedCLBService, pods []corev1.Pod, boundListeners map[string]*networkingv1alpha1.DedicatedCLBListener) error {
	if len(pods) == 0 {
		return nil
	}
	for _, port := range ds.Spec.Ports {
		if port.AddressPodAnnotation == "" {
			continue
		}
		for _, pod := range pods {
			key := getListenerKey(pod.Name, port.TargetPort, port.Protocol)
			lis, ok := boundListeners[key]
			if !ok || lis.Status.Address == "" {
				continue
			}
			realAddr := lis.Status.Address
			currentAddr := pod.Annotations[port.AddressPodAnnotation]
			if realAddr == currentAddr {
				continue
			}
			log.FromContext(ctx).V(5).Info(
				"set external address to pod annotation",
				"pod", pod.Name,
				"port", port.TargetPort,
				"protocol", port.Protocol,
				"oldAddr", currentAddr, "realAddr", realAddr,
			)
			if err := kube.SetPodAnnotation(ctx, &pod, port.AddressPodAnnotation, realAddr); err != nil {
				r.Recorder.Event(ds, corev1.EventTypeWarning, "UpdatePodAnnotation", fmt.Sprintf("set pod %s's annotation (%s: %s): %s", pod.Name, port.AddressPodAnnotation, realAddr, err.Error()))
				return err
			} else {
				r.Recorder.Event(ds, corev1.EventTypeNormal, "UpdatePodAnnotation", fmt.Sprintf("set pod %s's annotation (%s: %s)", pod.Name, port.AddressPodAnnotation, realAddr))
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
	if needUpdate { // 有新增已有CLB，加进待分配LB列表
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
	boundListeners map[string]*networkingv1alpha1.DedicatedCLBListener, err error,
) {
	log := log.FromContext(ctx)
	usedListeners := make(map[string]*networkingv1alpha1.DedicatedCLBListener)          // pod-port-protocol --> listener
	allocatableListeners := make(map[string][]*networkingv1alpha1.DedicatedCLBListener) // protocol --> listeners
	boundListeners = make(map[string]*networkingv1alpha1.DedicatedCLBListener)          // pod-port-protocol --> listener

	for _, listener := range listeners {
		targetPod := listener.Spec.TargetPod
		if targetPod == nil {
			allocatableListeners[listener.Spec.Protocol] = append(allocatableListeners[listener.Spec.Protocol], &listener)
		} else {
			usedListeners[getListenerKey(targetPod.PodName, targetPod.TargetPort, listener.Spec.Protocol)] = &listener
		}
	}
	type bind struct {
		*networkingv1alpha1.TargetPod
		Protocol string
	}
	binds := []bind{}
	for _, pod := range pods {
		for _, port := range ds.Spec.Ports {
			key := getListenerKey(pod.Name, port.TargetPort, port.Protocol)
			listener, ok := usedListeners[key]
			if ok { // pod 已绑定到监听器
				delete(usedListeners, key)
				boundListeners[key] = listener
				continue
			}
			// 没绑定到监听器，尝试找一个
			binds = append(binds, bind{Protocol: port.Protocol, TargetPod: &networkingv1alpha1.TargetPod{PodName: pod.Name, TargetPort: port.TargetPort}})
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
			listener.Spec.TargetPod = bind.TargetPod
			toUpdate = append(toUpdate, listener)
			allocatableListeners[bind.Protocol] = allocatableListeners[bind.Protocol][1:]
			r.Recorder.Event(ds, corev1.EventTypeNormal, "BindPod", "bind pod "+bind.PodName+" to listener "+listener.Name)
			log.V(5).Info(
				"bind pod to listener",
				"listener", listener.Name,
				"pod", bind.PodName,
				"targetPort", bind.TargetPort,
				"lbId", listener.Spec.LbId,
				"lbPort", listener.Spec.LbPort,
			)
		} else { // 没有可被分配的监听器，新建一个
			toCreate[bind.Protocol]++
		}
	}
	return
}

func (r *DedicatedCLBServiceReconciler) allocateNewListener(ctx context.Context, log logr.Logger, ds *networkingv1alpha1.DedicatedCLBService, protocol string, num int) (int, error) {
	if len(ds.Status.LbList) == 0 {
		return 0, errors.New("no clb found")
	}
	updateStatus := false
	var err error
	lbIndex := 0
	generateName := ds.Name + "-"
	createdNum := 0
OUTER_LOOP:
	for ; createdNum < num; createdNum++ { // 每个n个端口的循环
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
			if err = r.Create(ctx, lis); err != nil {
				break OUTER_LOOP
			}
			lb.MaxPort = port
			ds.Status.LbList[lbIndex] = lb
			updateStatus = true
			break // 创建成功，跳出本次端口分配的循环
		}
	}

	// if createdNum > 0 { // 空闲监听器不够
	// 	log.V(5).Info("idle listener is not enough", "gapNum", num)
	// }

	if updateStatus { // 有成功创建过，更新status
		log.V(5).Info("update lbList status", "lbList", ds.Status.LbList)
		if statusErr := r.Status().Update(ctx, ds); statusErr != nil {
			return createdNum, statusErr // TODO: 两个err可能同时发生，出现err覆盖
		}
		return createdNum, err
	}
	log.V(5).Info("no listener created")
	return createdNum, err
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
			log.V(5).Info(
				"pod matched dedicatedclbservice's selector, trigger reconcile",
				"pod", pod.GetName(),
				"dedicatedclbservice", ds.Name,
				"namespace", ds.Namespace,
			)
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
func (r *DedicatedCLBServiceReconciler) SetupWithManager(mgr ctrl.Manager, workers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.DedicatedCLBService{}).
		Owns(&networkingv1alpha1.DedicatedCLBListener{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workers,
		}).
		Watches( // TODO: 只关注创建，删除待考虑
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPod),
		).
		Complete(r)
}
