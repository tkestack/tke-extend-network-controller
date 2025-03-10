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

	networkingv1beta1 "github.com/imroc/tke-extend-network-controller/api/v1beta1"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DedicatedCLBServiceReconciler reconciles a DedicatedCLBService object
type DedicatedCLBServiceReconciler struct {
	client.Client
	APIReader client.Reader
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
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
	ds := &networkingv1beta1.DedicatedCLBService{}
	if err := r.APIReader.Get(ctx, req.NamespacedName, ds); err != nil { // 避免从缓存中读取（status可能更新不及时导致状态不一致，造成clb多创等问题）
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return util.RequeueIfConflict(r.reconcile(ctx, ds))
}

const dedicatedCLBServiceFinalizer = "dedicatedclbservice.finalizers.networking.cloud.tencent.com"

func (r *DedicatedCLBServiceReconciler) syncDelete(ctx context.Context, ds *networkingv1beta1.DedicatedCLBService) error {
	lbToDelete := make(map[string][]string)
	for _, clbInfos := range ds.Status.CLBs {
		for _, clbInfo := range clbInfos {
			if clbInfo.AutoCreate {
				lbToDelete[clbInfo.Region] = append(lbToDelete[clbInfo.Region], clbInfo.LbId)
			}
		}
	}
	for region, lbIds := range lbToDelete {
		r.Recorder.Event(ds, corev1.EventTypeNormal, "DeleteCLB", fmt.Sprintf("delete auto created clb instances: %v (region: %s)", lbIds, region))
		if err := clb.Delete(ctx, region, lbIds...); err != nil {
			r.Recorder.Event(ds, corev1.EventTypeWarning, "DeleteCLB", err.Error())
			return err
		}
	}
	return nil
}

func (r *DedicatedCLBServiceReconciler) reconcile(ctx context.Context, ds *networkingv1beta1.DedicatedCLBService) error {
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

func (r *DedicatedCLBServiceReconciler) diffNodes(ctx context.Context, ds *networkingv1beta1.DedicatedCLBService, nodes []corev1.Node, listeners []networkingv1beta1.DedicatedCLBListener) (toDel, toAdd []*networkingv1beta1.DedicatedCLBListener, err error) {
	return
}

type AllocateListenerJob struct {
	Service  *networkingv1beta1.DedicatedCLBService
	Port     *networkingv1beta1.DedicatedCLBServicePort
	Pod      *corev1.Pod
	Node     *corev1.Node
	Listener *networkingv1beta1.DedicatedCLBListener
}

func (j *AllocateListenerJob) AssignListener(protocol string, port int64, clbs []*clb.CLBListenerAllocator) {
	j.Listener = &networkingv1beta1.DedicatedCLBListener{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", j.Service.Name),
			Namespace:    j.Service.Namespace,
		},
		Spec: networkingv1beta1.DedicatedCLBListenerSpec{
			Protocol:            protocol,
			Port:                port,
			ExtensiveParameters: j.Service.Spec.ListenerExtensiveParameters,
		},
	}
	if j.Service.Spec.PortSegment != nil {
		endPort := port + *j.Service.Spec.PortSegment - 1
		j.Listener.Spec.EndPort = &endPort
	}
	if j.Pod != nil {
		j.Listener.Spec.TargetPod = &networkingv1beta1.TargetPod{
			PodName: j.Pod.Name,
			Port:    j.Port.TargetPort,
		}
	}
	for _, clb := range clbs {
		j.Listener.Spec.CLBs = append(j.Listener.Spec.CLBs, networkingv1beta1.CLB{
			LbId:   clb.ID,
			Region: clb.Region,
		})
	}
}

func (r *DedicatedCLBServiceReconciler) diffPods(ctx context.Context, ds *networkingv1beta1.DedicatedCLBService, pods []corev1.Pod, listeners []networkingv1beta1.DedicatedCLBListener) (toDel, toAdd []*networkingv1beta1.DedicatedCLBListener, err error) {
	allocatedListeners := make(map[string]*networkingv1beta1.DedicatedCLBListener)     // pod-port-protocol --> listener
	allocatableListeners := make(map[string][]*networkingv1beta1.DedicatedCLBListener) // protocol --> listeners
	for _, lis := range listeners {
		targetPod := lis.Spec.TargetPod
		if targetPod == nil {
			allocatableListeners[lis.Spec.Protocol] = append(allocatableListeners[lis.Spec.Protocol], &lis)
			continue
		}
		key := getListenerKey(targetPod.PodName, targetPod.Port, lis.Spec.Protocol)
		allocatedListeners[key] = &lis
	}
	log := log.FromContext(ctx)
	log.V(7).Info("diff pods", "pods", len(pods), "allocatedListeners", len(allocatedListeners), "allocatableListeners", len(allocatableListeners))
	toAllocate := []*AllocateListenerJob{}
	for _, pod := range pods {
		for _, port := range ds.Spec.Ports {
			key := getListenerKey(pod.Name, port.TargetPort, port.Protocol)
			if _, ok := allocatedListeners[key]; ok { // 已为 Pod 分配监听器，忽略
				continue
			}
			// pod 还没有分配监听器，标记分配
			toAllocate = append(toAllocate, &AllocateListenerJob{Port: &port, Pod: &pod, Service: ds})
		}
	}
	if len(toAllocate) > 0 {
		toAdd, err = r.allocatedListeners(ctx, ds, toAllocate)
		log.V(7).Info("allocate listeners", "expect", len(toAllocate), "got", len(toAdd))
	}
	return
}

func (r *DedicatedCLBServiceReconciler) allocatedListeners(ctx context.Context, ds *networkingv1beta1.DedicatedCLBService, toAllocate []*AllocateListenerJob) (toAdd []*networkingv1beta1.DedicatedCLBListener, err error) {
	toAllocateMap := make(map[string][]clb.ListenerAssignee) // protocol --> jobs
	for _, job := range toAllocate {
		toAllocateMap[job.Port.Protocol] = append(toAllocateMap[job.Port.Protocol], job)
	}
	var reqs []*clb.ListenerAllocationRequest
	for protocol, jobs := range toAllocateMap {
		reqs = append(reqs, &clb.ListenerAllocationRequest{
			Protocol:  protocol,
			Assignees: jobs,
		})
	}

	for _, clbInfos := range ds.Status.CLBs {
		needAllocate := false
		for _, req := range reqs {
			if len(req.Assignees) > 0 {
				needAllocate = true
				break
			}
		}
		if !needAllocate {
			break
		}
		var clbs []*clb.CLBListenerAllocator
		for _, clbInfo := range clbInfos {
			clbs = append(clbs, &clb.CLBListenerAllocator{
				ID:     clbInfo.LbId,
				Region: clbInfo.Region,
			})
		}
		allocator := &clb.BatchListenerAllocator{
			CLBs:        clbs,
			MinPort:     ds.Spec.MinPort,
			MaxPort:     ds.Spec.MaxPort,
			MaxListener: ds.Spec.MaxListener,
			PortSegment: ds.Spec.PortSegment,
		}
		err = allocator.Init(ctx)
		if err != nil {
			return
		}
		err = allocator.Allocate(ctx, reqs)
		if err != nil {
			return
		}
	}

	for _, job := range toAllocate {
		if job.Listener != nil {
			toAdd = append(toAdd, job.Listener)
		}
	}
	return
}

func (r *DedicatedCLBServiceReconciler) sync(ctx context.Context, ds *networkingv1beta1.DedicatedCLBService) error {
	ctx = clb.InitClbCache(ctx)
	if err := r.ensureStatus(ctx, ds); err != nil {
		return err
	}
	log := log.FromContext(ctx)
	listeners := &networkingv1beta1.DedicatedCLBListenerList{}
	if err := r.List(
		ctx, listeners,
		client.InNamespace(ds.Namespace),
		client.MatchingLabels{labelKeyDedicatedCLBServiceName: ds.Name},
	); err != nil {
		return err
	}
	var toDel, toAdd []*networkingv1beta1.DedicatedCLBListener
	var err error
	if targetPod := ds.Spec.Target.Pod; targetPod != nil {
		pods := &corev1.PodList{}
		log.V(7).Info("list related pods", "podNamespace", ds.Namespace, "podSelector", targetPod.PodSelector)
		if err := r.List(
			ctx, pods,
			client.InNamespace(ds.Namespace),
			client.MatchingLabels(targetPod.PodSelector),
		); err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			log.V(5).Info("no pods matches the selector")
			return nil
		}
		log.V(7).Info("found related pods", "num", len(pods.Items))
		toDel, toAdd, err = r.diffPods(ctx, ds, pods.Items, listeners.Items)
	} else if targetNode := ds.Spec.Target.Node; targetNode != nil {
		nodes := &corev1.NodeList{}
		log.V(7).Info("list related nodes", "nodeSelector", targetNode.NodeSelector)
		if err := r.List(
			ctx, nodes,
			client.MatchingLabels(targetNode.NodeSelector),
		); err != nil {
			return err
		}
		if len(nodes.Items) == 0 {
			log.V(5).Info("no nodes matches the selector")
			return nil
		}
		toDel, toAdd, err = r.diffNodes(ctx, ds, nodes.Items, listeners.Items)
	} else {
		err = errors.New("invalid target, must have pod or node specified")
	}

	if err != nil {
		return err
	}

	for _, lis := range toDel {
		log.V(5).Info("delete listener", "listener", lis.Name)
		if err := r.Delete(ctx, lis); err != nil {
			return err
		}
	}
	for _, lis := range toAdd {
		log.V(5).Info("create listener", "listener", lis.Name)
		if err := r.Create(ctx, lis); err != nil {
			return err
		}
	}

	// log.V(5).Info(
	// 	"find related pods and listeners",
	// 	"pods", len(pods.Items),
	// 	"listeners", len(listeners.Items),
	// 	"podSelector", ds.Spec.Selector,
	// )
	//
	// usedListeners := make(map[string]*networkingv1beta1.DedicatedCLBListener)          // pod-port-protocol --> listener
	// allocatableListeners := make(map[string][]*networkingv1beta1.DedicatedCLBListener) // protocol --> listeners
	//
	// for _, listener := range listeners.Items {
	// 	targetPod := listener.Spec.TargetPod
	// 	if targetPod == nil {
	// 		allocatableListeners[listener.Spec.Protocol] = append(allocatableListeners[listener.Spec.Protocol], &listener)
	// 	} else {
	// 		usedListeners[getListenerKey(targetPod.PodName, targetPod.TargetPort, listener.Spec.Protocol)] = &listener
	// 	}
	// }
	//
	// type bind struct {
	// 	Pod  *corev1.Pod
	// 	Port *networkingv1beta1.DedicatedCLBServicePort
	// }
	//
	// binds := []bind{}
	// for _, pod := range pods.Items {
	// 	for _, port := range ds.Spec.Ports {
	// 		key := getListenerKey(pod.Name, port.TargetPort, port.Protocol)
	// 		lis, ok := usedListeners[key]
	// 		if ok { // pod 已绑定到监听器
	// 			delete(usedListeners, key)
	// 			if port.AddressPodAnnotation != "" && lis.Status.Address != "" && pod.Annotations[port.AddressPodAnnotation] != lis.Status.Address { // 确保外部地址写到对应注解中
	// 				r.Recorder.Event(ds, corev1.EventTypeNormal, "UpdatePodAnnotation", fmt.Sprintf("set pod %s's annotation (%s: %s)", pod.Name, port.AddressPodAnnotation, lis.Status.Address))
	// 				if err := kube.SetPodAnnotation(ctx, &pod, port.AddressPodAnnotation, lis.Status.Address); err != nil {
	// 					r.Recorder.Event(ds, corev1.EventTypeWarning, "UpdatePodAnnotation", fmt.Sprintf("set pod %s's annotation (%s: %s): %s", pod.Name, port.AddressPodAnnotation, lis.Status.Address, err.Error()))
	// 					return err
	// 				}
	// 			}
	// 			continue
	// 		}
	// 		binds = append(binds, bind{Pod: &pod, Port: &port})
	// 	}
	// }
	// for _, lis := range usedListeners { // 绑定了其它非预期 Pod 的监听器认为是可分配的
	// 	allocatableListeners[lis.Spec.Protocol] = append(allocatableListeners[lis.Spec.Protocol], lis)
	// }
	//
	// listenerQuota, err := clb.GetQuota(ctx, ds.Spec.LbRegion, clb.TOTAL_LISTENER_QUOTA)
	// if err != nil {
	// 	r.Recorder.Event(ds, corev1.EventTypeWarning, "GetQuota", fmt.Sprintf("failed to get clb listener quota: %s", err.Error()))
	// 	return err
	// }
	// listenerGap := 0
	// for _, bind := range binds {
	// 	targetPod := &networkingv1beta1.TargetPod{PodName: bind.Pod.Name, TargetPort: bind.Port.TargetPort}
	// 	// 没绑定到监听器，尝试找一个
	// 	if liss, ok := allocatableListeners[bind.Port.Protocol]; ok && len(liss) > 0 { // 有现成的监听器可绑定
	// 		lis := liss[0]
	// 		r.Recorder.Event(ds, corev1.EventTypeNormal, "BindPod", "bind pod "+bind.Pod.Name+" to listener "+lis.Name)
	// 		if err := kube.Update(ctx, lis, func() {
	// 			lis.Spec.TargetPod = targetPod
	// 		}); err != nil {
	// 			return err
	// 		}
	// 		allocatableListeners[bind.Port.Protocol] = liss[1:]
	// 	} else { // 尝试创建新监听器
	// 		ok, err := r.allocateListener(ctx, ds, bind.Port.Protocol, targetPod, listenerQuota)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		if !ok { // 没有可分配的监听器，计算缺失的监听器数量
	// 			listenerGap++
	// 		}
	// 	}
	// }
	// if listenerGap > 0 {
	// 	podLimit := listenerQuota
	// 	if maxPorts := ds.Spec.MaxPort - ds.Spec.MinPort + 1; maxPorts < podLimit {
	// 		podLimit = maxPorts
	// 	}
	// 	if ds.Spec.MaxPod != nil && *ds.Spec.MaxPod < podLimit {
	// 		podLimit = *ds.Spec.MaxPod
	// 	}
	// 	lbNum := float64(listenerGap) / float64(podLimit)
	// 	lbToCreate := int(math.Ceil(lbNum))
	// 	r.Recorder.Event(ds, corev1.EventTypeNormal, "CreateCLB", fmt.Sprintf("clb is not enough, try to create clb instance (num: %d)", lbToCreate))
	// 	if err := r.allocateNewCLB(ctx, ds, lbToCreate); err != nil {
	// 		r.Recorder.Event(ds, corev1.EventTypeWarning, "CreateCLB", fmt.Sprintf("failed to create %d clb instance: %s", lbToCreate, err.Error()))
	// 		return err
	// 	}
	// }
	return nil
}

// func (r *dedicatedclbservicereconciler) allocatenewclb(ctx context.context, ds *networkingv1beta1.dedicatedclbservice, num int) error {
// 	var vpcId string
// 	if ds.Spec.VpcId != nil {
// 		vpcId = *ds.Spec.VpcId
// 	}
// 	ids, err := clb.Create(ctx, ds.Spec.LbRegion, vpcId, ds.Spec.LbAutoCreate.ExtensiveParameters, num)
// 	if err != nil {
// 		return err
// 	}
// 	r.Recorder.Event(ds, corev1.EventTypeNormal, "CreateCLB", fmt.Sprintf("clb successfully created: %v", ids))
// 	return kube.UpdateStatus(ctx, ds, func() {
// 		for _, lbId := range ids {
// 			ds.Status.AllocatableLb = append(ds.Status.AllocatableLb, networkingv1beta1.CLBInfo{
// 				LbId:       lbId,
// 				AutoCreate: true,
// 			})
// 		}
// 	})
// }

// 保证 spec 中定义的 clb 同步到 status 里
func (r *DedicatedCLBServiceReconciler) ensureStatus(ctx context.Context, ds *networkingv1beta1.DedicatedCLBService) error {
	lbIdsInStatus := make(map[string]bool)
	for _, clbInfos := range ds.Status.CLBs {
		for _, clbInfo := range clbInfos {
			lbIdsInStatus[clbInfo.LbId] = true
		}
	}
	needUpdate := false
	for _, clbs := range ds.Spec.ExistedCLBs {
		var clbInfos []networkingv1beta1.CLBInfo
	IN:
		for _, clb := range clbs {
			if lbIdsInStatus[clb.ID] {
				clbInfos = nil
				break IN
			}
			clbInfos = append(clbInfos, clb.ToCLBInfo())
		}
		if len(clbInfos) > 0 {
			needUpdate = true
			ds.Status.CLBs = append(ds.Status.CLBs, clbInfos)
		}
	}
	if needUpdate { // 有新增已有CLB，加进待分配LB列表
		status := ds.Status
		return kube.UpdateStatus(ctx, ds, func() {
			ds.Status = status
		})
	}
	return nil
}

func getListenerKey(podName string, port int64, protocol string) string {
	return fmt.Sprintf("%s-%s-%d", podName, protocol, port)
}

// func (r *DedicatedCLBServiceReconciler) diff(
// 	ctx context.Context, ds *networkingv1beta1.DedicatedCLBService,
// 	pods []corev1.Pod, listeners []networkingv1beta1.DedicatedCLBListener,
// ) (
// 	toUpdate []*networkingv1beta1.DedicatedCLBListener, toCreate map[string]int,
// 	boundListeners map[string]*networkingv1beta1.DedicatedCLBListener, err error,
// ) {
// 	log := log.FromContext(ctx)
// 	usedListeners := make(map[string]*networkingv1beta1.DedicatedCLBListener)          // pod-port-protocol --> listener
// 	allocatableListeners := make(map[string][]*networkingv1beta1.DedicatedCLBListener) // protocol --> listeners
// 	boundListeners = make(map[string]*networkingv1beta1.DedicatedCLBListener)          // pod-port-protocol --> listener
//
// 	for _, listener := range listeners {
// 		targetPod := listener.Spec.TargetPod
// 		if targetPod == nil {
// 			allocatableListeners[listener.Spec.Protocol] = append(allocatableListeners[listener.Spec.Protocol], &listener)
// 		} else {
// 			usedListeners[getListenerKey(targetPod.PodName, targetPod.TargetPort, listener.Spec.Protocol)] = &listener
// 		}
// 	}
// 	type bind struct {
// 		*networkingv1beta1.TargetPod
// 		Protocol string
// 	}
// 	binds := []bind{}
// 	for _, pod := range pods {
// 		for _, port := range ds.Spec.Ports {
// 			key := getListenerKey(pod.Name, port.TargetPort, port.Protocol)
// 			listener, ok := usedListeners[key]
// 			if ok { // pod 已绑定到监听器
// 				delete(usedListeners, key)
// 				boundListeners[key] = listener
// 				continue
// 			}
// 			// 没绑定到监听器，尝试找一个
// 			binds = append(binds, bind{Protocol: port.Protocol, TargetPod: &networkingv1beta1.TargetPod{PodName: pod.Name, TargetPort: port.TargetPort}})
// 		}
// 	}
// 	// 所有pod都绑定了，直接返回
// 	if len(binds) == 0 {
// 		log.V(7).Info("all pods are bound, ignore")
// 		return
// 	}
// 	// 还有需要绑定的pod
// 	// 先将配置了其它未知Pod的监听器合并到可被分配的监听器列表
// 	for _, listener := range usedListeners {
// 		allocatableListeners[listener.Spec.Protocol] = append(allocatableListeners[listener.Spec.Protocol], listener)
// 	}
// 	if len(allocatableListeners) > 0 {
// 		for protocol, listeners := range allocatableListeners {
// 			log.V(7).Info("found allocatable listeners", "protocol", protocol, "listeners", len(listeners))
// 		}
// 	}
// 	toCreate = make(map[string]int)
// 	for _, bind := range binds {
// 		listeners := allocatableListeners[bind.Protocol]
// 		if len(listeners) > 0 { // 还有可被分配的监听器
// 			listener := listeners[0]
// 			listener.Spec.TargetPod = bind.TargetPod
// 			toUpdate = append(toUpdate, listener)
// 			allocatableListeners[bind.Protocol] = allocatableListeners[bind.Protocol][1:]
// 			r.Recorder.Event(ds, corev1.EventTypeNormal, "BindPod", "bind pod "+bind.PodName+" to listener "+listener.Name)
// 			log.V(5).Info(
// 				"bind pod to listener",
// 				"listener", listener.Name,
// 				"pod", bind.PodName,
// 				"targetPort", bind.TargetPort,
// 				"lbId", listener.Spec.LbId,
// 				"lbPort", listener.Spec.LbPort,
// 			)
// 		} else { // 没有可被分配的监听器，新建一个
// 			toCreate[bind.Protocol]++
// 		}
// 	}
// 	return
// }

// func (r *DedicatedCLBServiceReconciler) allocateListener(ctx context.Context, ds *networkingv1beta1.DedicatedCLBService, protocol string, pod *networkingv1beta1.TargetPod, listenerQuota int64) (ok bool, err error) {
// 	if len(ds.Status.AllocatableLb) == 0 {
// 		return false, nil
// 	}
// 	lb := ds.Status.AllocatableLb[0]
// 	var segment int64 = 1
// 	if ds.Spec.PortSegment != nil {
// 		segment = *ds.Spec.PortSegment
// 	}
// 	if lb.CurrentPort <= 0 {
// 		lb.CurrentPort = ds.Spec.MinPort - segment
// 	}
// 	havePort := func() bool {
// 		if lb.CurrentPort >= ds.Spec.MaxPort {
// 			return true
// 		}
// 		allocatedPorts := (lb.CurrentPort - ds.Spec.MinPort + segment) / segment
// 		return allocatedPorts >= listenerQuota || (ds.Spec.MaxPod != nil && allocatedPorts >= *ds.Spec.MaxPod)
// 	}
// 	if havePort() { // 该lb已分配完所有端口，尝试下一个lb
// 		ds.Status.AllocatableLb = ds.Status.AllocatableLb[1:]
// 		ds.Status.AllocatedLb = append(ds.Status.AllocatedLb, networkingv1beta1.AllocatedCLBInfo{LbId: lb.LbId, AutoCreate: lb.AutoCreate})
// 		status := ds.Status
// 		if err := kube.UpdateStatus(ctx, ds, func() {
// 			ds.Status = status
// 		}); err != nil {
// 			return false, err
// 		}
// 		return false, fmt.Errorf("%s's port is exhausted but still in allocatableLb", lb.LbId)
// 	}
// 	lb.CurrentPort += segment
// 	lis := &networkingv1beta1.DedicatedCLBListener{}
// 	lis.Spec.LbId = lb.LbId
// 	lis.Spec.LbPort = lb.CurrentPort
// 	if ds.Spec.PortSegment != nil {
// 		lis.Spec.LbEndPort = common.Int64Ptr(lb.CurrentPort + segment - 1)
// 	}
// 	lis.Spec.Protocol = protocol
// 	lis.Spec.ExtensiveParameters = ds.Spec.ListenerExtensiveParameters
// 	lis.Spec.LbRegion = ds.Spec.LbRegion
// 	lis.Spec.TargetPod = pod
// 	lis.Namespace = ds.Namespace
// 	lis.GenerateName = ds.Name + "-"
// 	lis.Labels = map[string]string{
// 		labelKeyDedicatedCLBServiceName: ds.Name,
// 	}
// 	if err := controllerutil.SetControllerReference(ds, lis, r.Scheme); err != nil {
// 		return false, err
// 	}
// 	if err := r.Create(ctx, lis); err != nil {
// 		return false, err
// 	}
//
// 	r.Recorder.Event(ds, corev1.EventTypeNormal, "AllocateListener", "allocate listener "+lis.Name+" for pod "+pod.PodName)
//
// 	// listener 创建成功，更新 status
// 	if havePort() { // 该lb已分配完所有端口，或者到达配额上限，将其移到已分配的lb列表
// 		ds.Status.AllocatableLb = ds.Status.AllocatableLb[1:]
// 		ds.Status.AllocatedLb = append(ds.Status.AllocatedLb, networkingv1beta1.AllocatedCLBInfo{LbId: lb.LbId, AutoCreate: lb.AutoCreate})
// 		status := ds.Status
// 		return true, kube.UpdateStatus(ctx, ds, func() {
// 			ds.Status = status
// 		})
// 	}
// 	// lb端口还没分配完，更新currentPort
// 	status := ds.Status
// 	status.AllocatableLb[0].CurrentPort = lb.CurrentPort
// 	return true, kube.UpdateStatus(ctx, ds, func() {
// 		ds.Status = status
// 	})
// }

func (r *DedicatedCLBServiceReconciler) findObjectsForPod(ctx context.Context, pod client.Object) []reconcile.Request {
	list := &networkingv1beta1.DedicatedCLBServiceList{}
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
		targetPod := ds.Spec.Target.Pod
		if targetPod == nil {
			continue
		}
		podSelector := labels.Set(targetPod.PodSelector).AsSelector()
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
		For(&networkingv1beta1.DedicatedCLBService{}).
		Owns(&networkingv1beta1.DedicatedCLBListener{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workers,
		}).
		Watches( // TODO: 只关注创建，删除待考虑
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPod),
		).
		Complete(r)
}
