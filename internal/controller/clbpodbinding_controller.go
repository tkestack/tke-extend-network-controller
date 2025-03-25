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
	"encoding/json"
	"fmt"
	"time"

	"github.com/imroc/tke-extend-network-controller/internal/constant"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"github.com/pkg/errors"
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

type portKey struct {
	Port     uint16
	Protocol string
	Pool     string
}

func (r *CLBPodBindingReconciler) sync(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding) (result ctrl.Result, err error) {
	// 确保 State 不为空
	if pb.Status.State == "" {
		pb.Status.State = networkingv1alpha1.CLBPodBindingStatePending
		if err = r.Status().Update(ctx, pb); err != nil {
			return result, errors.WithStack(err)
		}
	}
	// 确保所有端口都已分配且绑定 Pod
	if err := r.ensureCLBPodBinding(ctx, pb); err != nil {
		// 如果是等待端口池扩容 CLB，确保状态为 WaitForLB，并重新入队，以便在 CLB 扩容完成后能自动分配端口并绑定 Pod
		if errors.Is(err, portpool.ErrWaitLBScale) {
			if err := r.ensureState(ctx, pb, networkingv1alpha1.CLBPodBindingStateWaitForLB); err != nil {
				return result, errors.WithStack(err)
			}
			result.RequeueAfter = 3 * time.Second
			return result, nil
		}
		// 其它非资源冲突的错误，将错误记录到状态中方便排障
		if !apierrors.IsConflict(err) {
			pb.Status.State = networkingv1alpha1.CLBPodBindingStateFailed
			pb.Status.Message = err.Error()
			if err := r.Status().Update(ctx, pb); err != nil {
				return result, errors.WithStack(err)
			}
			return result, errors.WithStack(err)
		}
	}
	return result, err
}

func (r *CLBPodBindingReconciler) ensureCLBPodBinding(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding) error {
	// 确保所有端口都被分配
	if err := r.ensurePortAllocated(ctx, pb); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有监听器都已创建
	if err := r.ensureListeners(ctx, pb); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有监听器都已绑定到 Pod
	if err := r.ensurePodBindings(ctx, pb); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CLBPodBindingReconciler) ensureListeners(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding) error {
	log.FromContext(ctx).V(10).Info("ensureListeners")
	for i := range pb.Status.PortBindings {
		binding := &pb.Status.PortBindings[i]
		needUpdate, err := r.ensureListener(ctx, binding)
		if err != nil {
			return errors.WithStack(err)
		}
		if needUpdate {
			log.FromContext(ctx).V(10).Info("update status", "status", pb.Status)
			if err := r.Status().Update(ctx, pb); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}

func (r *CLBPodBindingReconciler) ensurePodBindings(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding) error {
	log.FromContext(ctx).V(10).Info("ensurePodBindings")
	ensureWaitForPod := func(msg string) error {
		if pb.Status.State != networkingv1alpha1.CLBPodBindingStateWaitForPod {
			pb.Status.State = networkingv1alpha1.CLBPodBindingStateWaitForPod
			pb.Status.Message = msg
			if err := r.Status().Update(ctx, pb); err != nil {
				return errors.WithStack(err)
			}
		}
		needUpdate := false
		for i := range pb.Status.PortBindings {
			binding := &pb.Status.PortBindings[i]
			if binding.ListenerId != "" {
				if err := clb.DeregisterAllTargets(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId); err != nil {
					return errors.WithStack(err)
				}
				needUpdate = true
			}
		}
		if needUpdate {
			if err := r.Status().Update(ctx, pb); err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	}
	pod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(pb), pod); err != nil {
		if apierrors.IsNotFound(err) { // Pod 不存在，等待 Pod 创建（通常是 networking.cloud.tencent.com/retain 为 true 时）
			if err := ensureWaitForPod("waiting for pod to be created"); err != nil {
				return errors.WithStack(err)
			}
			// 确保监听器为空
			return nil
		}
		return errors.WithStack(err)
	}
	if pod.Status.PodIP == "" { // 等待 Pod 分配 IP
		if err := ensureWaitForPod("waiting for pod to start"); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	// Pod 准备就绪，将 CLB 监听器绑定到 Pod
	for i := range pb.Status.PortBindings {
		binding := &pb.Status.PortBindings[i]
		if err := r.ensurePortBound(ctx, pod, binding); err != nil {
			return errors.WithStack(err)
		}
	}
	// 所有端口都已绑定，更新状态并将绑定信息写入 pod 注解
	if pb.Status.State != networkingv1alpha1.CLBPodBindingStateBound {
		pb.Status.State = networkingv1alpha1.CLBPodBindingStateBound
		pb.Status.Message = ""
		if err := r.Status().Update(ctx, pb); err != nil {
			return errors.WithStack(err)
		}
	}
	// 确保 status 注解正确
	if err := r.ensurePodStatusAnnotation(ctx, pb, pod); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

type PortBindingStatus struct {
	networkingv1alpha1.PortBindingStatus `json:",inline"`
	EndPort                              *uint16  `json:"endPort,omitempty"`
	Hostname                             *string  `json:"hostname,omitempty"`
	Ips                                  []string `json:"ips,omitempty"`
}

func lbStatusKey(poolName, lbId string) string {
	return fmt.Sprintf("%s/%s", poolName, lbId)
}

func (r *CLBPodBindingReconciler) ensurePodStatusAnnotation(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding, pod *corev1.Pod) error {
	lbStatuses := make(map[string]*networkingv1alpha1.LoadBalancerStatus)
	getLbStatus := func(poolName, lbId string) (*networkingv1alpha1.LoadBalancerStatus, error) {
		status, exists := lbStatuses[lbStatusKey(poolName, lbId)]
		if exists {
			return status, nil
		}
		pp := &networkingv1alpha1.CLBPortPool{}
		if err := r.Get(ctx, client.ObjectKey{Name: poolName}, pp); err != nil {
			return nil, errors.WithStack(err)
		}
		for i := range pp.Status.LoadbalancerStatuses {
			status := &pp.Status.LoadbalancerStatuses[i]
			lbStatuses[lbStatusKey(poolName, status.LoadbalancerID)] = status
		}
		status, exists = lbStatuses[lbStatusKey(poolName, lbId)]
		if exists {
			return status, nil
		}
		return nil, errors.Errorf("loadbalancer %s not found in pool %s", lbId, poolName)
	}
	statuses := []PortBindingStatus{}
	for _, binding := range pb.Status.PortBindings {
		status, err := getLbStatus(binding.Pool, binding.LoadbalancerId)
		if err != nil {
			return errors.WithStack(err)
		}
		var endPort *uint16
		if !util.IsZero(binding.LoadbalancerEndPort) {
			val := binding.Port + (*binding.LoadbalancerEndPort - binding.LoadbalancerPort)
			endPort = &val
		}
		statuses = append(statuses, PortBindingStatus{
			PortBindingStatus: binding,
			EndPort:           endPort,
			Hostname:          status.Hostname,
			Ips:               status.Ips,
		})
	}
	val, err := json.Marshal(statuses)
	if err != nil {
		return errors.WithStack(err)
	}
	patchMap := map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				constant.CLBPortMappingResultKey:  string(val),
				constant.CLBPortMappingStatuslKey: "Ready",
			},
		},
	}
	patch, err := json.Marshal(patchMap)
	if err != nil {
		return errors.WithStack(err)
	}
	if pod.Annotations != nil {
		if pod.Annotations[constant.CLBPortMappingResultKey] == string(val) { // 注解符合预期，无需更新
			return nil
		}
	}
	if err := r.Patch(ctx, pod, client.RawPatch(types.StrategicMergePatchType, patch)); err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).V(10).Info("patch clb port mapping status success", "value", string(val))
	return nil
}

func (r *CLBPodBindingReconciler) ensureListener(ctx context.Context, binding *networkingv1alpha1.PortBindingStatus) (needUpdate bool, err error) {
	log.FromContext(ctx).V(10).Info("ensureListener", "port", binding.Port, "protocol", binding.Protocol)
	createListener := func() {
		log.FromContext(ctx).V(10).Info("create listener")
		var lisId string
		if lisId, err = clb.CreateListener(
			ctx,
			binding.Region,
			binding.LoadbalancerId,
			int64(binding.LoadbalancerPort),
			int64(util.GetValue(binding.LoadbalancerEndPort)),
			binding.Protocol,
			"",
		); err != nil {
			err = errors.Wrapf(err, "failed to create listener %d/%s", binding.Port, binding.Protocol)
			return
		} else { // 创建监听器成功，更新状态
			binding.ListenerId = lisId
			needUpdate = true
			log.FromContext(ctx).V(10).Info("create listener success", "listenerId", lisId)
		}
	}
	var lis *clb.Listener
	if lis, err = clb.GetListenerByPort(ctx, binding.Region, binding.LoadbalancerId, int64(binding.LoadbalancerPort), binding.Protocol); err != nil {
		err = errors.Wrapf(err, "failed to get listener by port %d/%s", binding.Port, binding.Protocol)
		return
	} else {
		if lis == nil { // 还未创建监听器，执行创建
			log.FromContext(ctx).V(10).Info("listener not create yet")
			createListener()
		} else { // 已创建监听器，检查是否符合预期
			if lis.ListenerId != binding.ListenerId {
				log.FromContext(ctx).V(10).Info("listenerId not match, need update", "expect", binding.ListenerId, "actual", lis.ListenerId)
				binding.ListenerId = lis.ListenerId
				needUpdate = true
			}
		}
	}
	return
}

func (r *CLBPodBindingReconciler) ensurePortBound(ctx context.Context, pod *corev1.Pod, binding *networkingv1alpha1.PortBindingStatus) error {
	log.FromContext(ctx).V(10).Info("ensurePortBound", "port", binding.Port, "protocol", binding.Protocol)
	targets, err := clb.DescribeTargets(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId)
	if err != nil {
		return errors.WithStack(err)
	}
	podTarget := clb.Target{
		TargetIP:   pod.Status.PodIP,
		TargetPort: int64(binding.Port),
	}
	targetToDelete := []clb.Target{}
	alreadyAdded := false
	for _, target := range targets {
		if target == podTarget {
			alreadyAdded = true
		} else {
			targetToDelete = append(targetToDelete, target)
		}
	}
	// 清理多余的 rs
	if len(targetToDelete) > 0 {
		log.FromContext(ctx).V(10).Info("deregister targets", "targets", targetToDelete)
		if err := clb.DeregisterTargetsForListener(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, targetToDelete...); err != nil {
			return errors.WithStack(err)
		}
	}
	// 绑定 pod
	if !alreadyAdded {
		log.FromContext(ctx).V(10).Info("register target", "target", podTarget)
		if err := clb.RegisterTargets(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, podTarget); err != nil {
			return errors.WithStack(err)
		}
	}
	// 到这里，能确保 pod 已绑定到所有 lb 监听器
	return nil
}

func (r *CLBPodBindingReconciler) ensurePortAllocated(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding) error {
	bindings := make(map[portKey]*networkingv1alpha1.PortBindingStatus)
	for i := range pb.Status.PortBindings {
		binding := &pb.Status.PortBindings[i]
		key := portKey{
			Port:     binding.Port,
			Protocol: binding.Protocol,
			Pool:     binding.Pool,
		}
		bindings[key] = binding
	}
	var allocatedPorts portpool.PortAllocations
LOOP_PORT:
	for _, port := range pb.Spec.Ports { // 检查 spec 中的端口是否都已分配
		keys := []portKey{}
		for _, pool := range port.Pools {
			if port.Protocol == constant.ProtocolTCPUDP {
				keys = append(keys, portKey{
					Port:     port.Port,
					Protocol: constant.ProtocolTCP,
					Pool:     pool,
				})
				keys = append(keys, portKey{
					Port:     port.Port,
					Protocol: constant.ProtocolUDP,
					Pool:     pool,
				})
			} else {
				keys = append(keys, portKey{
					Port:     port.Port,
					Protocol: port.Protocol,
					Pool:     pool,
				})
			}
		}
		alreadyAllocated := false
		for _, key := range keys {
			if _, exists := bindings[key]; exists { // 已分配端口，跳过
				delete(bindings, key)
				alreadyAllocated = true
			}
		}
		if alreadyAllocated {
			continue LOOP_PORT
		}
		// 未分配端口，执行分配
		allocated, err := portpool.Allocator.Allocate(ctx, port.Pools, port.Protocol, util.GetValue(port.UseSamePortAcrossPools))
		if err != nil {
			return errors.WithStack(err)
		}
		for _, allocatedPort := range allocated {
			binding := networkingv1alpha1.PortBindingStatus{
				Port:             port.Port,
				Protocol:         allocatedPort.Protocol,
				Pool:             allocatedPort.GetName(),
				LoadbalancerId:   allocatedPort.LbId,
				LoadbalancerPort: allocatedPort.Port,
				Region:           allocatedPort.GetRegion(),
			}
			if allocatedPort.EndPort > 0 {
				binding.LoadbalancerEndPort = &allocatedPort.EndPort
			}
			pb.Status.PortBindings = append(pb.Status.PortBindings, binding)
		}
		allocatedPorts = append(allocatedPorts, allocated...)
	}

	if len(bindings) > 0 {
		for _, binding := range bindings {
			_, err := clb.DeleteListenerByPort(ctx, binding.Region, binding.LoadbalancerId, int64(binding.LoadbalancerPort), binding.Protocol)
			if err != nil {
				return errors.WithStack(err)
			}
		}
		statuses := []networkingv1alpha1.PortBindingStatus{}
		for _, port := range pb.Status.PortBindings {
			key := portKey{
				Port:     port.Port,
				Protocol: port.Protocol,
				Pool:     port.Pool,
			}
			if _, exists := bindings[key]; !exists {
				statuses = append(statuses, port)
			}
		}
		pb.Status.PortBindings = statuses
	}

	if len(allocatedPorts) == 0 && len(bindings) == 0 { // 没有新端口分配，也没有多余端口需要删除，直接返回
		return nil
	}
	// 将已分配的端口写入 status
	if err := r.Status().Update(ctx, pb); err != nil {
		// 更新状态失败，释放已分配端口
		allocatedPorts.Release()
		return errors.WithStack(err)
	}
	return nil
}

func portFromPortBindingStatus(status *networkingv1alpha1.PortBindingStatus) portpool.ProtocolPort {
	port := portpool.ProtocolPort{
		Port:     status.LoadbalancerPort,
		Protocol: status.Protocol,
	}
	if status.LoadbalancerEndPort != nil {
		port.EndPort = *status.LoadbalancerEndPort
	}
	return port
}

func (r *CLBPodBindingReconciler) ensureState(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding, state networkingv1alpha1.CLBPodBindingState) error {
	if pb.Status.State == state {
		return nil
	}
	pb.Status.State = state
	if err := r.Status().Update(ctx, pb); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// 清理 CLBPodBinding
func (r *CLBPodBindingReconciler) cleanup(ctx context.Context, pb *networkingv1alpha1.CLBPodBinding) (result ctrl.Result, err error) {
	log := log.FromContext(ctx)
	log.Info("cleanup CLBPodBinding")
	if err = r.ensureState(ctx, pb, networkingv1alpha1.CLBPodBindingStateDeleting); err != nil {
		return result, errors.WithStack(err)
	}
	for _, binding := range pb.Status.PortBindings {
		// 解绑 lb
		if _, err := clb.DeleteListenerByPort(ctx, binding.Region, binding.LoadbalancerId, int64(binding.LoadbalancerPort), binding.Protocol); err != nil {
			return result, errors.Wrapf(err, "failed to delete listener (%s/%d/%s)", binding.LoadbalancerId, binding.LoadbalancerPort, binding.Protocol)
		}
		// 释放端口
		portpool.Allocator.Release(binding.Pool, binding.LoadbalancerId, portFromPortBindingStatus(&binding))
	}
	// 清理完成，检查 Pod 是否是正常状态，如果是，通常是手动删除 CLBPodBinding 场景，此时触发一次 Pod 对账，让被删除的 CLBPodBinding 重新创建出来
	pod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(pb), pod); err != nil {
		if apierrors.IsNotFound(err) { // Pod 没有重建出来，忽略
			return result, nil
		}
		return result, errors.WithStack(err)
	}
	if !pod.DeletionTimestamp.IsZero() { // 忽略正在删除的 Pod
		return result, nil
	}
	// 新的同名 Pod 已经创建，通知 PodController 重新对账 Pod，以便让新的 CLBPodBinding 创建出来
	podEventSource <- event.TypedGenericEvent[client.Object]{
		Object: pod,
	}
	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLBPodBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBPodBinding{}).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPod),
		).
		Named("clbpodbinding").
		Complete(r)
}

func (r *CLBPodBindingReconciler) findObjectsForPod(ctx context.Context, pod client.Object) []reconcile.Request {
	if time := pod.GetDeletionTimestamp(); time != nil && time.IsZero() { // 忽略正在删除的 Pod，默认情况下，Pod 删除完后会自动 GC 删除掉关联的 CLBPodBinding
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
