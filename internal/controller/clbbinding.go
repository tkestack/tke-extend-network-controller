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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/clbbinding"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/eventsource"
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
	spec := bd.GetSpec()
	if spec.Disabled != nil && *spec.Disabled {
		if err := r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateDisabled); err != nil {
			return result, errors.WithStack(err)
		}
		return
	}
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
		errCause := errors.Cause(err)
		// 1. 扩容了 lb、或者正在扩容，忽略，因为会自动触发对账。
		// 2. 端口不足无法分配、端口池不存在，忽略，因为如果端口池不改正或扩容 lb，无法重试成功。
		// 3. lb 被删除或监听器被删除，自动移除了 status 中的记录，需重新入队对账。
		switch errCause {
		case portpool.ErrNewLBCreated, portpool.ErrNewLBCreating:
			return result, nil
		case ErrNeedRetry:
			result.Requeue = true
			return result, nil
		case portpool.ErrNoPortAvailable:
			r.Recorder.Event(bd.GetObject(), corev1.EventTypeWarning, "NoPortAvailable", "no port available in port pool, please add clb to port pool")
			if err := r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateNoPortAvailable); err != nil {
				return result, errors.WithStack(err)
			}
			return result, nil
		case portpool.ErrPoolNotFound:
			r.Recorder.Event(bd.GetObject(), corev1.EventTypeWarning, "PoolNotFound", "port pool not found, please check the port pool name")
			if err := r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStatePortPoolNotFound); err != nil {
				return result, errors.WithStack(err)
			}
			return result, nil
		}
		// 如果是被云 API 限流（默认每秒 20 qps 限制），1s 后重新入队
		if clb.IsRequestLimitExceededError(errCause) {
			result.RequeueAfter = time.Second
			return result, nil
		}

		if apierrors.IsConflict(errCause) { // 资源冲突错误，直接重新入队触发重试
			result.Requeue = true
			return result, nil
		} else { // 其它非资源冲突的错误，将错误记录到 event 和状态中方便排障
			r.Recorder.Event(bd.GetObject(), corev1.EventTypeWarning, "SyncFailed", errCause.Error())
			if status.State != networkingv1alpha1.CLBBindingStateFailed {
				status.State = networkingv1alpha1.CLBBindingStateFailed
				status.Message = errCause.Error()
				if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
					return result, errors.WithStack(err)
				}
			}
			// lb 已不存在，没必要重新入队对账，不返回错误，保持 Failed 状态即可。
			if clb.IsLbIdNotFoundError(errCause) {
				return result, nil
			}
			// 其它错误，返回错误触发重试
			return result, errors.WithStack(err)
		}
	}
	return result, err
}

func (r *CLBBindingReconciler[T]) ensureCLBBinding(ctx context.Context, bd clbbinding.CLBBinding) error {
	// 确保依赖的端口池和 CLB 都存在，如果已删除则释放端口并更新状态
	if err := r.ensurePoolAndCLB(ctx, bd); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有端口都被分配
	if err := r.ensurePortAllocated(ctx, bd); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有监听器都已创建
	if err := r.ensureListeners(ctx, bd); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有监听器都已绑定到 backend
	if err := r.ensureBackendBindings(ctx, bd); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CLBBindingReconciler[T]) ensurePoolAndCLB(ctx context.Context, bd clbbinding.CLBBinding) error {
	status := bd.GetStatus()
	newBindings := []networkingv1alpha1.PortBindingStatus{}
	needUpdateStatus := false
	for i := range status.PortBindings {
		binding := &status.PortBindings[i]
		pool := portpool.Allocator.GetPool(binding.Pool)
		if pool == nil { // 端口池不存在，将端口绑定状态置为 Failed，并记录事件
			r.Recorder.Event(bd.GetObject(), corev1.EventTypeWarning, "PortPoolDeleted", fmt.Sprintf("port pool has been deleted (%s/%s/%d)", binding.Pool, binding.LoadbalancerId, binding.LoadbalancerPort))
			needUpdateStatus = true
		} else {
			if !pool.IsLbExists(binding.LoadbalancerId) {
				r.Recorder.Event(bd.GetObject(), corev1.EventTypeWarning, "CLBDeleted", fmt.Sprintf("clb has been deleted (%s/%s/%d)", binding.Pool, binding.LoadbalancerId, binding.LoadbalancerPort))
				needUpdateStatus = true
			} else {
				newBindings = append(newBindings, *binding)
			}
		}
	}
	if needUpdateStatus {
		log.FromContext(ctx).V(10).Info("update status newBindings", "newBindings", newBindings)
		status.PortBindings = newBindings
		if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).V(10).Info("after update status newBindings", "obj", bd.GetObject())
	}
	return nil
}

var ErrNeedRetry = errors.New("need retry")

// TODO: 优化性能：一次性查询所有监听器信息
func (r *CLBBindingReconciler[T]) ensureListeners(ctx context.Context, bd clbbinding.CLBBinding) error {
	newBindings := []networkingv1alpha1.PortBindingStatus{}
	needUpdate := false
	needRetry := false
	status := bd.GetStatus()
	for i := range status.PortBindings {
		binding := &status.PortBindings[i]
		op, err := r.ensureListener(ctx, bd, binding)
		if err != nil {
			return errors.WithStack(err)
		}
		switch op {
		case util.StatusOpNone:
			newBindings = append(newBindings, *binding)
		case util.StatusOpUpdate:
			needUpdate = true
			newBindings = append(newBindings, *binding)
		case util.StatusOpDelete:
			needUpdate = true
			needRetry = true
		}
	}
	if needUpdate {
		status.PortBindings = newBindings
		if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
			return errors.WithStack(err)
		}
	}
	if needRetry {
		return ErrNeedRetry
	}
	return nil
}

func (r *CLBBindingReconciler[T]) ensureBackendBindings(ctx context.Context, bd clbbinding.CLBBinding) error {
	status := bd.GetStatus()
	backend, err := bd.GetAssociatedObject(ctx, r.Client)
	if err != nil {
		if apierrors.IsNotFound(errors.Cause(err)) { // 后端不存在，一般是网络隔离场景，保持待绑定状态（正常情况 clbbinding 的 OwnerReference 是 pod/node，它们被清理后 clbbinding 也会被 gc 自动清理）
			if err = r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateNoBackend); err != nil {
				return errors.WithStack(err)
			}
			return nil
		}
		// 其它错误，直接返回
		return errors.WithStack(err)
	}
	node, err := backend.GetNode(ctx)
	if err != nil {
		if err == clbbinding.ErrNodeNameIsEmpty { // pod 还未调度
			r.Recorder.Event(bd.GetObject(), corev1.EventTypeNormal, "WaitBackend", "wait pod to be scheduled")
			if err = r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateWaitBackend); err != nil {
				return errors.WithStack(err)
			}
			return nil
		}
		return errors.WithStack(err)
	}
	if !util.IsNodeTypeSupported(node) {
		if status.State != networkingv1alpha1.CLBBindingStateNodeTypeNotSupported {
			msg := "current node type is not supported, please use super node or native node"
			r.Recorder.Event(bd.GetObject(), corev1.EventTypeWarning, "NodeNotSupported", msg)
			r.Recorder.Event(backend.GetObject(), corev1.EventTypeWarning, "NodeNotSupported", msg)
			status.State = networkingv1alpha1.CLBBindingStateNodeTypeNotSupported
			status.Message = msg
			if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	}
	if backend.GetIP() == "" { // 等待 backend 分配 IP
		r.Recorder.Event(bd.GetObject(), corev1.EventTypeNormal, "WaitBackend", "wait backend network to be ready")
		if err = r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateWaitBackend); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	// backend 准备就绪，将 CLB 监听器绑定到 bacekend
	for i := range status.PortBindings {
		binding := &status.PortBindings[i]
		if err := r.ensurePortBound(ctx, bd, backend, binding); err != nil {
			return errors.WithStack(err)
		}
	}
	// 所有端口都已绑定，更新状态并将绑定信息写入 backend 注解
	if status.State != networkingv1alpha1.CLBBindingStateBound {
		r.Recorder.Event(bd.GetObject(), corev1.EventTypeNormal, "AllBound", "all targets bound to listener")
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
	log.FromContext(ctx).V(3).Info("patch clb port mapping status success", "value", string(val))
	return nil
}

func (r *CLBBindingReconciler[T]) ensureListener(ctx context.Context, bd clbbinding.CLBBinding, binding *networkingv1alpha1.PortBindingStatus) (op util.StatusOp, err error) {
	createListener := func() {
		var lisId string
		lisId, err = clb.CreateListenerTryBatch(
			ctx,
			binding.Region,
			binding.LoadbalancerId,
			int64(binding.LoadbalancerPort),
			int64(util.GetValue(binding.LoadbalancerEndPort)),
			binding.Protocol,
			binding.CertId,
			"",
		)
		if err != nil {
			err = errors.Wrapf(err, "failed to create clb listener %d/%s", binding.Port, binding.Protocol)
			return
		} else { // 创建监听器成功，更新状态
			binding.ListenerId = lisId
			op = util.StatusOpUpdate
			log.FromContext(ctx).V(3).Info("create clb listener success", "port", binding.Port, "protocl", binding.Protocol, "listenerId", lisId, "lbPort", binding.LoadbalancerPort, "lbId", binding.LoadbalancerId)
		}
	}
	var lis *clb.Listener
	if lis, err = clb.GetListenerByIdOrPort(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, int64(binding.LoadbalancerPort), binding.Protocol); err != nil {
		if clb.IsLbIdNotFoundError(errors.Cause(err)) { // lb 已删除，通知关联的端口池重新对账
			pp := &networkingv1alpha1.CLBPortPool{}
			pp.Name = binding.Pool
			eventsource.PortPool <- event.TypedGenericEvent[client.Object]{
				Object: pp,
			}
			op = util.StatusOpDelete
			err = errors.WithStack(err)
		} else {
			err = errors.Wrapf(err, "failed to get listener by port %d/%s", binding.Port, binding.Protocol)
		}
		return
	} else {
		if lis == nil { // 还未创建监听器，执行创建
			createListener()
		} else { // 已创建监听器，检查是否符合预期
			if lis.ListenerId != binding.ListenerId { // id 不匹配，包括还未写入 id 的情况，更新下 id
				log.FromContext(ctx).V(10).Info("listenerId not match, need update", "expect", binding.ListenerId, "actual", lis.ListenerId)
				binding.ListenerId = lis.ListenerId
				op = util.StatusOpUpdate
			}
		}
	}
	return
}

func (r *CLBBindingReconciler[T]) ensurePortBound(ctx context.Context, bd clbbinding.CLBBinding, backend clbbinding.Backend, binding *networkingv1alpha1.PortBindingStatus) error {
	targets, err := clb.DescribeTargetsTryBatch(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId)
	if err != nil {
		return errors.WithStack(err)
	}
	backendTarget := clb.Target{
		TargetIP:   backend.GetIP(),
		TargetPort: int64(binding.Port),
	}
	targetToDelete := []*clb.Target{}
	alreadyAdded := false
	for _, target := range targets {
		if *target == backendTarget {
			alreadyAdded = true
		} else {
			targetToDelete = append(targetToDelete, target)
		}
	}
	// 清理多余的 rs
	if len(targetToDelete) > 0 {
		r.Recorder.Eventf(bd.GetObject(), corev1.EventTypeNormal, "DeregisterTarget", "remove unexpected target: %v", targetToDelete)
		if err := clb.DeregisterTargetsForListenerTryBatch(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, targetToDelete...); err != nil {
			return errors.WithStack(err)
		}
	}
	// 绑定后端
	if !alreadyAdded {
		startTime := time.Now()
		if err := clb.RegisterTarget(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, backendTarget); err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).V(10).Info("RegisterTarget performance", "cost", time.Since(startTime).String(), "target", backendTarget.String(), "listenerId", binding.LoadbalancerId, "lbPort", binding.LoadbalancerPort, "protocol", binding.Protocol)
	}
	// 到这里，能确保后端 已绑定到所有 lb 监听器
	return nil
}

var ErrCertIdNotFound = errors.New("no cert id found from secret")

func (r *CLBBindingReconciler[T]) ensurePortAllocated(ctx context.Context, bd clbbinding.CLBBinding) error {
	status := bd.GetStatus()
	bindings := make(map[portKey]*networkingv1alpha1.PortBindingStatus)
	bds := []networkingv1alpha1.PortBindingStatus{}
	haveLbRemoved := false
	for i := range status.PortBindings {
		binding := &status.PortBindings[i]
		if portpool.Allocator.IsLbExists(binding.Pool, binding.LoadbalancerId) {
			key := portKey{
				Port:     binding.Port,
				Protocol: binding.Protocol,
				Pool:     binding.Pool,
			}
			bindings[key] = binding
			bds = append(bds, *binding)
		} else { // lb 已被删除
			haveLbRemoved = true
			r.Recorder.Eventf(bd.GetObject(), corev1.EventTypeWarning, "LbNotFound", "lb %q not found for allocated port %d, remove it to re-allocate", binding.LoadbalancerId, binding.LoadbalancerPort)
		}
	}
	if haveLbRemoved {
		status.PortBindings = bds
		if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
			return errors.WithStack(err)
		}
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
		// 未分配端口，先检查证书配置
		var certId string
		if secretName := port.CertSecretName; secretName != nil && *secretName != "" {
			id, err := kube.GetCertIdFromSecret(ctx, r.Client, client.ObjectKey{
				Namespace: bd.GetNamespace(),
				Name:      *secretName,
			})
			if err != nil {
				allocatedPorts.Release()
				if apierrors.IsNotFound(errors.Cause(err)) {
					r.Recorder.Eventf(bd.GetObject(), corev1.EventTypeWarning, "CertNotFound", "cert secret %q not found", *secretName)
					return errors.Wrapf(ErrCertIdNotFound, "cert secret %q not found", *secretName)
				}
				return errors.WithStack(err)
			}
			certId = id
		}
		// 配置无误，执行分配
		allocated, err := portpool.Allocator.Allocate(ctx, port.Pools, port.Protocol, util.GetValue(port.UseSamePortAcrossPools))
		if err != nil {
			return errors.WithStack(err)
		}
		if len(allocated) > 0 { // 预期应该是每次 allocated 长度大于 0，否则应该有 error 返回，不会走到这后面的 else 语句
			for _, allocatedPort := range allocated {
				binding := networkingv1alpha1.PortBindingStatus{
					Port:             port.Port,
					Protocol:         allocatedPort.Protocol,
					CertId:           certId,
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
			allocatedPorts = append(allocatedPorts, allocated...)
		} else { // 兜底：为简化逻辑，保证事务性，只要有一个端口分配失败就认为失败
			allocatedPorts.Release()
			return portpool.ErrNoPortAvailable
		}
	}

	if len(bindings) > 0 { // 删除多余的端口绑定
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
	status.Message = ""
	if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// 清理 CLBBinding
func (r *CLBBindingReconciler[T]) cleanup(ctx context.Context, bd T) (result ctrl.Result, err error) {
	log := log.FromContext(ctx)
	log.Info("cleanup " + bd.GetType())
	if err = r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateDeleting); err != nil {
		return result, errors.WithStack(err)
	}
	status := bd.GetStatus()
	for _, binding := range status.PortBindings {
		// 确保端口从端口池被释放
		allocated := portpool.Allocator.IsAllocated(binding.Pool, binding.LoadbalancerId, portFromPortBindingStatus(&binding))
		if !allocated { // 已经清理过，忽略
			continue
		}
		releasePort := func() {
			portpool.Allocator.Release(binding.Pool, binding.LoadbalancerId, portFromPortBindingStatus(&binding))
		}
		// 解绑 lb
		if err := clb.DeleteListenerByIdOrPort(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, int64(binding.LoadbalancerPort), binding.Protocol); err != nil {
			e := errors.Cause(err)
			switch e {
			case clb.ErrListenerNotFound: // 监听器不存在，认为成功，从端口池释放
				log.Info("delete listener while listener not found, ignore")
				releasePort()
				continue
			case clb.ErrOtherListenerNotFound: // 因同一批次删除的其它监听器不存在导致删除失败，需重试
				log.Error(err, "delete listener failed cuz other listener not found, retry")
				result.Requeue = true
				return result, nil
			}
			if clb.IsLoadBalancerNotExistsError(e) { // lb 不存在，忽略
				releasePort()
				continue
			}
			if clb.IsRequestLimitExceededError(e) {
				log.Info("request limit exceeded, retry")
				result.RequeueAfter = time.Second
				return result, nil
			}
			return result, errors.Wrapf(err, "failed to delete listener (%s/%d/%s/%s)", binding.LoadbalancerId, binding.LoadbalancerPort, binding.Protocol, binding.ListenerId)
		} else { // 删除成功，释放端口
			releasePort()
		}
	}
	// 清理完成，检查 obj 是否是正常状态，如果是，通常是手动删除 CLBBinding 场景，此时触发一次 obj 对账，让被删除的 CLBBinding 重新创建出来
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
		var certSecretName *string
		if len(fields) >= 4 {
			options := fields[3]
			optionList := strings.Split(options, ",")
			for _, option := range optionList {
				kv := strings.Split(option, "=")
				if len(kv) == 1 {
					switch kv[0] {
					case "useSamePortAcrossPools":
						b := true
						useSamePortAcrossPools = &b
					}
				} else if len(kv) == 2 {
					key := kv[0]
					value := kv[1]
					switch key {
					case "certSecret":
						certSecretName = &value
					}
				}
			}
		}
		ports = append(ports, networkingv1alpha1.PortEntry{
			Port:                   port,
			Protocol:               protocol,
			Pools:                  pools,
			UseSamePortAcrossPools: useSamePortAcrossPools,
			CertSecretName:         certSecretName,
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

	bd := binding.GetObject()
	err = r.Get(ctx, client.ObjectKeyFromObject(obj), bd)
	// 获取 obj 的注解
	enablePortMappings := anno[constant.EnableCLBPortMappingsKey]
	switch enablePortMappings {
	case "true", "false": // 确保 CLBBinding 存在且符合预期
		// 获取 obj 对应的 CLBBinding
		if err != nil {
			if apierrors.IsNotFound(err) { // 不存在，自动创建
				// 没有 CLBBinding，自动创建
				bd.SetName(obj.GetName())
				bd.SetNamespace(obj.GetNamespace())
				// 生成期望的 CLBBindingSpec
				spec, err := generateCLBBindingSpec(portMappings, enablePortMappings)
				if err != nil {
					return result, errors.Wrapf(err, "failed to generate %s spec", binding.GetType())
				}
				*binding.GetSpec() = *spec
				// 给 CLBBinding 添加 OwnerReference，让 obj 被删除时，CLBBinding 也被清理，保留 IP 场景除外
				if obj.GetAnnotations()[constant.Ratain] != "true" {
					if err := controllerutil.SetOwnerReference(obj, bd, r.Scheme); err != nil {
						return result, errors.WithStack(err)
					}
				}
				log.FromContext(ctx).V(10).Info("create clbbinding", "binding", bd)
				if err := r.Create(ctx, bd); err != nil {
					r.Recorder.Eventf(obj, corev1.EventTypeWarning, "CreateCLBBinding", "create %s %s failed: %s", binding.GetType(), obj.GetName(), err.Error())
					return result, errors.WithStack(err)
				}
				r.Recorder.Eventf(obj, corev1.EventTypeNormal, "CreateCLBBinding", "create %s %s successfully", binding.GetType(), obj.GetName())
			} else { // 其它错误，直接返回错误
				return result, errors.WithStack(err)
			}
		} else { // 存在
			// 正在删除，重新入队（滚动更新场景，旧的解绑完，确保 CLBBinding 要重建出来）
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
			if objRecreated { // 检测到 obj 已被重建，CLBBinding 在等待被 GC 清理，重新入队，以便被 GC 清理后重新对账让新的 CLBBinding 被创建出来
				result.RequeueAfter = 3 * time.Second
				r.Recorder.Event(obj, corev1.EventTypeNormal, "WaitCLBBindingGC", "wait old clbbinding to be deleted")
				return result, nil
			}
			// CLBBinding 存在且没有被删除，对账 spec 是否符合预期
			spec, err := generateCLBBindingSpec(portMappings, enablePortMappings)
			if err != nil {
				return result, errors.Wrap(err, "failed to generate CLBBinding spec")
			}
			actualSpec := binding.GetSpec()
			if !reflect.DeepEqual(*actualSpec, *spec) { // spec 不一致，更新
				log.FromContext(ctx).Info("update clbbinding", "oldSpec", *actualSpec, "newSpec", *spec)
				*actualSpec = *spec
				if err := r.Update(ctx, bd); err != nil {
					r.Recorder.Eventf(obj, corev1.EventTypeWarning, "CLBBindingChanged", "update %s %s failed: %s", binding.GetType(), obj.GetName(), err.Error())
					return result, errors.WithStack(err)
				}
				r.Recorder.Eventf(obj, corev1.EventTypeNormal, "CLBBindingChanged", "update %s %s successfully", binding.GetType(), obj.GetName())
			}
		}
	default:
		// 没有配置注解，如发现有 CLBBinding，则删除掉
		if err == nil {
			r.Recorder.Eventf(obj, corev1.EventTypeNormal, "DeleteCLBBinding", "delete %s %s", binding.GetType(), obj.GetName())
			if err := r.Delete(ctx, bd); err != nil {
				return result, errors.WithStack(err)
			}
		}
	}
	if apierrors.IsNotFound(err) {
		return result, nil
	}
	return
}
