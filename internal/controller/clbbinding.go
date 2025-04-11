package controller

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/imroc/tke-extend-network-controller/internal/constant"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/clbbinding"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type CLBBindingReconciler[T clbbinding.CLBBinding] struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

type portKey struct {
	Port     uint16
	Protocol string
	Pool     string
}

func (r *CLBBindingReconciler[T]) sync(ctx context.Context, bd T) (result ctrl.Result, err error) {
	status := bd.GetStatus()
	// 确保 State 不为空
	if status.State == "" {
		status.State = networkingv1alpha1.CLBBindingStatePending
		if err = r.Status().Update(ctx, bd.GetObject()); err != nil {
			return result, errors.WithStack(err)
		}
	}
	// 确保所有端口都已分配且绑定 obj
	if err := r.ensureCLBBinding(ctx, bd); err != nil {
		// 如果是等待端口池扩容 CLB，确保状态为 WaitForLB，并重新入队，以便在 CLB 扩容完成后能自动分配端口并绑定 obj
		if errors.Is(err, portpool.ErrWaitLBScale) {
			if err := r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateWaitForLB); err != nil {
				return result, errors.WithStack(err)
			}
			result.RequeueAfter = 3 * time.Second
			return result, nil
		}
		// 其它非资源冲突的错误，将错误记录到状态中方便排障
		if !apierrors.IsConflict(err) {
			status.State = networkingv1alpha1.CLBBindingStateFailed
			status.Message = err.Error()
			if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
				return result, errors.WithStack(err)
			}
			// lb 已不存在，没必要重新入队对账，保持 Failed 状态即可。
			if clb.IsLbIdNotFoundError(errors.Cause(err)) {
				log.FromContext(ctx).Error(err, "CLB is not exists")
				return result, nil
			}
			return result, errors.WithStack(err)
		}
	}
	return result, err
}

func (r *CLBBindingReconciler[T]) ensureCLBBinding(ctx context.Context, bd clbbinding.CLBBinding) error {
	// 确保所有端口都被分配
	if err := r.ensurePortAllocated(ctx, bd); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有监听器都已创建
	if err := r.ensureListeners(ctx, bd); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有监听器都已绑定到 obj
	if err := r.ensureBackendBindings(ctx, bd); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CLBBindingReconciler[T]) ensureListeners(ctx context.Context, bd clbbinding.CLBBinding) error {
	log.FromContext(ctx).V(10).Info("ensureListeners")
	status := bd.GetStatus()
	for i := range status.PortBindings {
		binding := &status.PortBindings[i]
		needUpdate, err := r.ensureListener(ctx, binding)
		if err != nil {
			return errors.WithStack(err)
		}
		if needUpdate {
			if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}

func (r *CLBBindingReconciler[T]) ensureBackendBindings(ctx context.Context, bd clbbinding.CLBBinding) error {
	status := bd.GetStatus()
	log.FromContext(ctx).V(10).Info("ensureBackendBindings")
	ensureWaitForBackend := func() error {
		if err := bd.EnsureWaitBackendState(ctx, r.Client); err != nil {
			return errors.WithStack(err)
		}
		needUpdate := false
		for i := range status.PortBindings {
			binding := &status.PortBindings[i]
			if binding.ListenerId != "" {
				if err := clb.DeregisterAllTargets(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId); err != nil {
					return errors.WithStack(err)
				}
				needUpdate = true
			}
		}
		if needUpdate {
			if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	}
	backend, err := bd.GetAssociatedObject(ctx, r.Client)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := ensureWaitForBackend(); err != nil {
				return errors.WithStack(err)
			}
			return nil
		}
	}
	if backend.GetIP() == "" { // 等待 obj 分配 IP
		if err := ensureWaitForBackend(); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	// obj 准备就绪，将 CLB 监听器绑定到 obj
	for i := range status.PortBindings {
		binding := &status.PortBindings[i]
		if err := r.ensurePortBound(ctx, backend, binding); err != nil {
			return errors.WithStack(err)
		}
	}
	// 所有端口都已绑定，更新状态并将绑定信息写入 obj 注解
	if status.State != networkingv1alpha1.CLBBindingStateBound {
		status.State = networkingv1alpha1.CLBBindingStateBound
		status.Message = ""
		if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
			return errors.WithStack(err)
		}
	}
	// 确保 status 注解正确
	if err := r.ensureBackendStatusAnnotation(ctx, bd, backend); err != nil {
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

func (r *CLBBindingReconciler[T]) ensureBackendStatusAnnotation(ctx context.Context, bd clbbinding.CLBBinding, backend clbbinding.Backend) error {
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
	bdStatus := bd.GetStatus()
	for _, binding := range bdStatus.PortBindings {
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

	if annotations := backend.GetAnnotations(); annotations != nil && annotations[constant.CLBPortMappingResultKey] == string(val) {
		// 注解符合预期，无需更新
		return nil
	}
	patchMap := map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				constant.CLBPortMappingResultKey:  string(val),
				constant.CLBPortMappingStatuslKey: "Ready",
			},
		},
	}
	if err := kube.PatchMap(ctx, r.Client, backend.GetObject(), patchMap); err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).V(10).Info("patch clb port mapping status success", "value", string(val))
	return nil
}

func (r *CLBBindingReconciler[T]) ensureListener(ctx context.Context, binding *networkingv1alpha1.PortBindingStatus) (needUpdate bool, err error) {
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

func (r *CLBBindingReconciler[T]) ensurePortBound(ctx context.Context, backend clbbinding.Backend, binding *networkingv1alpha1.PortBindingStatus) error {
	log.FromContext(ctx).V(10).Info("ensurePortBound", "port", binding.Port, "protocol", binding.Protocol)
	targets, err := clb.DescribeTargets(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId)
	if err != nil {
		return errors.WithStack(err)
	}
	backendTarget := clb.Target{
		TargetIP:   backend.GetIP(),
		TargetPort: int64(binding.Port),
	}
	targetToDelete := []clb.Target{}
	alreadyAdded := false
	for _, target := range targets {
		if target == backendTarget {
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
	// 绑定后端
	if !alreadyAdded {
		log.FromContext(ctx).V(10).Info("register target", "target", backendTarget)
		if err := clb.RegisterTargets(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, backendTarget); err != nil {
			return errors.WithStack(err)
		}
	}
	// 到这里，能确保后端 已绑定到所有 lb 监听器
	return nil
}

func (r *CLBBindingReconciler[T]) ensurePortAllocated(ctx context.Context, bd clbbinding.CLBBinding) error {
	status := bd.GetStatus()
	bindings := make(map[portKey]*networkingv1alpha1.PortBindingStatus)
	for i := range status.PortBindings {
		binding := &status.PortBindings[i]
		key := portKey{
			Port:     binding.Port,
			Protocol: binding.Protocol,
			Pool:     binding.Pool,
		}
		bindings[key] = binding
	}
	var allocatedPorts portpool.PortAllocations
	spec := bd.GetSpec()
LOOP_PORT:
	for _, port := range spec.Ports { // 检查 spec 中的端口是否都已分配
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
			status.PortBindings = append(status.PortBindings, binding)
		}
		if len(allocated) > 0 {
			allocatedPorts = append(allocatedPorts, allocated...)
		}
	}

	if len(bindings) > 0 {
		for _, binding := range bindings {
			_, err := clb.DeleteListenerByPort(ctx, binding.Region, binding.LoadbalancerId, int64(binding.LoadbalancerPort), binding.Protocol)
			if err != nil {
				return errors.WithStack(err)
			}
		}
		statuses := []networkingv1alpha1.PortBindingStatus{}
		for _, port := range status.PortBindings {
			key := portKey{
				Port:     port.Port,
				Protocol: port.Protocol,
				Pool:     port.Pool,
			}
			if _, exists := bindings[key]; !exists {
				statuses = append(statuses, port)
			}
		}
		status.PortBindings = statuses
	}

	if len(allocatedPorts) == 0 && len(bindings) == 0 { // 没有新端口分配，也没有多余端口需要删除，直接返回
		return nil
	}
	// 将已分配的端口写入 status
	if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
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

func (r *CLBBindingReconciler[T]) ensureState(ctx context.Context, bd clbbinding.CLBBinding, state networkingv1alpha1.CLBBindingState) error {
	status := bd.GetStatus()
	if status.State == state {
		return nil
	}
	status.State = state
	if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// 清理 CLBBinding
func (r *CLBBindingReconciler[T]) cleanup(ctx context.Context, bd T) (result ctrl.Result, err error) {
	log := log.FromContext(ctx)
	log.Info("cleanup CLBBinding")
	if err = r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateDeleting); err != nil {
		return result, errors.WithStack(err)
	}
	status := bd.GetStatus()
	for _, binding := range status.PortBindings {
		// 解绑 lb
		if _, err := clb.DeleteListenerByPort(ctx, binding.Region, binding.LoadbalancerId, int64(binding.LoadbalancerPort), binding.Protocol); err != nil {
			return result, errors.Wrapf(err, "failed to delete listener (%s/%d/%s)", binding.LoadbalancerId, binding.LoadbalancerPort, binding.Protocol)
		}
		// 释放端口
		portpool.Allocator.Release(binding.Pool, binding.LoadbalancerId, portFromPortBindingStatus(&binding))
	}
	// 清理完成，检查 obj 是否是正常状态，如果是，通常是手动删除 CLBobjBinding 场景，此时触发一次 obj 对账，让被删除的 CLBobjBinding 重新创建出来
	backend, err := bd.GetAssociatedObject(ctx, r.Client)
	if err != nil {
		if apierrors.IsNotFound(err) { // 后端没有重建出来，忽略
			return result, nil
		}
		return result, errors.WithStack(err)
	}
	if !backend.GetDeletionTimestamp().IsZero() { // 忽略正在删除的后端
		return result, nil
	}
	// 新的同名后端已经创建，通知对应 Controller 重新对账，以便让新的 CLBBinding 创建出来
	backend.TriggerReconcile()
	return result, nil
}

func generatePortsFromAnnotation(anno string) (ports []networkingv1alpha1.PortEntry, err error) {
	rd := bufio.NewReader(strings.NewReader(anno))
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		fields := strings.Fields(string(line))
		if len(fields) < 3 {
			return nil, fmt.Errorf("invalid port mapping: %s", string(line))
		}
		portStr := fields[0]
		portUint64, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("bad port number in port mapping: %s", string(line))
		}
		port := uint16(portUint64)
		protocol := fields[1]
		pools := strings.Split(fields[2], ",")
		var useSamePortAcrossPools *bool
		if len(fields) >= 4 && fields[3] == "useSamePortAcrossPools" {
			b := true
			useSamePortAcrossPools = &b
		}
		ports = append(ports, networkingv1alpha1.PortEntry{
			Port:                   port,
			Protocol:               protocol,
			Pools:                  pools,
			UseSamePortAcrossPools: useSamePortAcrossPools,
		})
	}
	return
}

func generateCLBBindingSpec(anno, enablePortMappings string) (*networkingv1alpha1.CLBBindingSpec, error) {
	ports, err := generatePortsFromAnnotation(anno)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	spec := &networkingv1alpha1.CLBBindingSpec{}
	spec.Ports = ports
	if enablePortMappings == "false" {
		spec.Disabled = util.GetPtr(true)
	}
	return spec, nil
}

func (r *CLBBindingReconciler[T]) syncCLBBinding(ctx context.Context, obj client.Object, binding T) (result ctrl.Result, err error) {
	if !obj.GetDeletionTimestamp().IsZero() { // 忽略正在删除的 Object
		return
	}
	anno := obj.GetAnnotations()

	portMappings := anno[constant.CLBPortMappingsKey]
	if portMappings == "" {
		log.FromContext(ctx).V(10).Info("skip without clb-port-mapping annotation")
		return
	}

	// 获取 obj 的注解
	enablePortMappings := anno[constant.EnableCLBPortMappingsKey]
	switch enablePortMappings {
	case "true", "false": // 确保 CLBobjBinding 存在且符合预期
		// 获取 obj 对应的 CLBobjBinding
		bd := binding.GetObject()
		if err := r.Get(ctx, client.ObjectKeyFromObject(obj), bd); err != nil {
			if apierrors.IsNotFound(err) { // 不存在，自动创建
				// 没有 CLBobjBinding，自动创建
				bd.SetName(obj.GetName())
				bd.SetNamespace(obj.GetNamespace())
				// 生成期望的 CLBobjBindingSpec
				spec, err := generateCLBBindingSpec(portMappings, enablePortMappings)
				if err != nil {
					return result, errors.Wrap(err, "failed to generate CLBobjBinding spec")
				}
				*binding.GetSpec() = *spec
				// 给 CLBobjBinding 添加 OwnerReference，让 obj 被删除时，CLBobjBinding 也被清理，保留 IP 场景除外
				if obj.GetAnnotations()[constant.Ratain] != "true" {
					if err := controllerutil.SetOwnerReference(obj, bd, r.Scheme); err != nil {
						return result, errors.WithStack(err)
					}
				}
				log.FromContext(ctx).V(10).Info("create clbbinding", "binding", bd)
				if err := r.Create(ctx, bd); err != nil {
					r.Recorder.Event(obj, corev1.EventTypeWarning, "CreateCLBBinding", fmt.Sprintf("create CLBBinding %s failed: %s", obj.GetName(), err.Error()))
					return result, errors.WithStack(err)
				}
				r.Recorder.Event(obj, corev1.EventTypeNormal, "CreateCLBobjBinding", fmt.Sprintf("create CLBobjBinding %s successfully", obj.GetName()))
			} else { // 其它错误，直接返回错误
				return result, errors.WithStack(err)
			}
		} else { // 存在
			// 正在删除，重新入队（滚动更新场景，旧的解绑完，确保 CLBobjBinding 要重建出来）
			if !binding.GetDeletionTimestamp().IsZero() {
				result.RequeueAfter = time.Second
				return result, nil
			}
			objRecreated := false
			for _, ref := range binding.GetOwnerReferences() {
				gvk := obj.GetObjectKind().GroupVersionKind()
				if ref.Kind == gvk.Kind && ref.Name == obj.GetName() && ref.APIVersion == gvk.GroupVersion().String() {
					if ref.UID != obj.GetUID() {
						objRecreated = true
					}
					break
				}
			}
			if objRecreated { // 检测到 obj 已被重建，CLBobjBinding 在等待被 GC 清理，重新入队，以便被 GC 清理后重新对账让新的 CLBobjBinding 被创建出来
				result.RequeueAfter = 3 * time.Second
				r.Recorder.Event(obj, corev1.EventTypeNormal, "WaitCLBobjBindingGC", "wait old clbobjbinding to be deleted")
				return result, nil
			}
			// CLBobjBinding 存在且没有被删除，对账 spec 是否符合预期
			spec, err := generateCLBBindingSpec(portMappings, enablePortMappings)
			if err != nil {
				return result, errors.Wrap(err, "failed to generate CLBobjBinding spec")
			}
			actualSpec := binding.GetSpec()
			if !reflect.DeepEqual(*actualSpec, *spec) { // spec 不一致，更新
				log.FromContext(ctx).Info("update clbobjbinding", "oldSpec", *actualSpec, "newSpec", *spec)
				*actualSpec = *spec
				if err := r.Update(ctx, bd); err != nil {
					r.Recorder.Eventf(obj, corev1.EventTypeWarning, "CLBobjBindingChanged", "update CLBobjBinding %s failed: %s", obj.GetName(), err.Error())
					return result, errors.WithStack(err)
				}
				r.Recorder.Eventf(obj, corev1.EventTypeNormal, "CLBobjBindingChanged", "update CLBobjBinding %s successfully", obj.GetName())
			}
		}
	default:
		log.FromContext(ctx).Info("skip invalid enable-clb-port-mapping value", "value", enablePortMappings)
	}
	return
}
