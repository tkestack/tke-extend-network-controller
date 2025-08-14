package controller

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"

	"github.com/go-logr/logr"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"
	"go.uber.org/multierr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/pkg/errors"
	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/internal/clbbinding"
	"github.com/tkestack/tke-extend-network-controller/internal/portpool"
	"github.com/tkestack/tke-extend-network-controller/pkg/clb"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/tkestack/tke-extend-network-controller/pkg/kube"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
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
		if err := r.ensureUnbound(ctx, bd); err != nil {
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
		switch errCause {
		case ErrLBNotFoundInPool: // lb 不存在于端口池中，通常是 lb 扩容了但还未将 lb 信息写入端口池的 status 中，重新入队重试
			log.FromContext(ctx).Info("lb info not found in pool yet, will retry", "err", err)
			result.RequeueAfter = 20 * time.Microsecond
			return result, nil
		case portpool.ErrPortPoolNotAllocatable: // 端口池不可用
			if err := r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStatePortPoolNotAllocatable); err != nil {
				return result, errors.WithStack(err)
			}
			return result, nil
		case portpool.ErrNoPortAvailable: // 端口不足
			r.Recorder.Event(bd.GetObject(), corev1.EventTypeWarning, "NoPortAvailable", "no port available in port pool, please add clb to port pool")
			if err := r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateNoPortAvailable); err != nil {
				return result, errors.WithStack(err)
			}
			return result, nil
		}
		// 分配端口时发现端口池不在分配器缓存中
		if e, ok := errCause.(*portpool.ErrPoolNotFound); ok {
			// 看是否有这个端口池对象
			poolName := e.Pool
			pp := &networkingv1alpha1.CLBPortPool{}
			if err := r.Client.Get(ctx, client.ObjectKey{Name: poolName}, pp); err != nil {
				if apierrors.IsNotFound(err) { // 端口池确实不存在，更新状态和 event
					r.Recorder.Eventf(bd.GetObject(), corev1.EventTypeWarning, "PoolNotFound", "port pool %q not found, please check the port pool name", poolName)
					if err := r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStatePortPoolNotFound); err != nil {
						return result, errors.WithStack(err)
					}
					return result, nil
				}
				return result, errors.WithStack(err)
			} else { // 端口池存在，但还没更新到分配器缓存，忽略，等待端口池就绪后会自动触发重新对账
				log.FromContext(ctx).Info("pool found but not exist in pool allocator yet, will retry", "pool", poolName)
				result.RequeueAfter = 20 * time.Microsecond
				return result, nil
			}
		}
		// 如果是被云 API 限流（默认每秒 20 qps 限制），1s 后重新入队
		if clb.IsRequestLimitExceededError(errCause) {
			result.RequeueAfter = time.Second
			log.FromContext(ctx).Info("requeue due to clb api request limit exceeded when reconciling", "err", err)
			return result, nil
		}

		if !apierrors.IsConflict(errCause) { // 其它非资源冲突的错误，将错误记录到 event 和状态中方便排障
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
				log.FromContext(ctx).Info("lbId not found, ignore")
				return result, nil
			}
		}
		// 其它错误
		return result, errors.WithStack(err)
	}
	return result, nil
}

func (r *CLBBindingReconciler[T]) ensureUnbound(ctx context.Context, bd clbbinding.CLBBinding) error {
	for _, binding := range bd.GetStatus().PortBindings {
		lisId := binding.ListenerId
		if lisId == "" {
			continue
		}
		// TODO: 改成并发
		if err := clb.DeregisterAllTargetsTryBatch(ctx, binding.Region, binding.LoadbalancerId, lisId); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (r *CLBBindingReconciler[T]) ensureCLBBinding(ctx context.Context, bd clbbinding.CLBBinding) error {
	// 确保所有端口都被分配
	if err := r.ensurePortAllocated(ctx, bd); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有监听器都已创建并绑定到 backend
	if len(bd.GetStatus().PortBindings) > 0 { // 确保要分配到了端口
		if err := r.ensureBackendBindings(ctx, bd); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

var ErrListenerNotExpected = errors.New("listener not expected")

func (r *CLBBindingReconciler[T]) ensureListenerExpected(ctx context.Context, binding *networkingv1alpha1.PortBindingStatus, lis *clb.Listener) (*networkingv1alpha1.PortBindingStatus, error) {
	if lis.Port != int64(binding.LoadbalancerPort) || lis.EndPort != int64(util.GetValue(binding.LoadbalancerEndPort)) || lis.Protocol != binding.Protocol { // 不符预期，删除监听器
		if err := clb.DeleteListenerById(ctx, binding.Region, binding.LoadbalancerId, lis.ListenerId); err != nil {
			if clb.IsLoadBalancerNotExistsError(err) { // lb 不存在，删除 binding，清理缓存
				clb.DeleteListenerCache(clb.LBKey{LbId: binding.LoadbalancerId, Region: binding.Region})
				return nil, nil
			} else if clb.IsListenerNotFound(err) { // 监听器不存在，删除binding, 清理缓存
				clb.GetListenerCache(clb.LBKey{LbId: binding.LoadbalancerId, Region: binding.Region}).EnsureRemoved(ctx, binding.LoadbalancerPort, binding.Protocol)
				return nil, nil
			}
			// 删除失败，返回错误
			return nil, errors.WithStack(err)
		} else {
			// 删除成功，清理缓存
			clb.GetListenerCache(clb.LBKey{LbId: binding.LoadbalancerId, Region: binding.Region}).EnsureRemoved(ctx, binding.LoadbalancerPort, binding.Protocol)
		}
		return binding, ErrListenerNotExpected
	}
	// 符合预期
	if binding.ListenerId != lis.ListenerId { // 但监听器 ID 不一样，更新一下
		updateListener(binding, lis.ListenerId)
	}
	return binding, nil
}

func updateListener(binding *networkingv1alpha1.PortBindingStatus, lisId string) {
	binding.ListenerId = lisId
	lis := &clb.Listener{
		Port:         int64(binding.LoadbalancerPort),
		EndPort:      int64(util.GetValue(binding.LoadbalancerEndPort)),
		Protocol:     binding.Protocol,
		ListenerId:   lisId,
		ListenerName: clb.TkeListenerName,
	}
	clb.GetListenerCache(clb.LBKey{LbId: binding.LoadbalancerId, Region: binding.Region}).Set(lis)
}

func (r *CLBBindingReconciler[T]) createListener(ctx context.Context, bd clbbinding.CLBBinding, binding *networkingv1alpha1.PortBindingStatus, log logr.Logger) (*networkingv1alpha1.PortBindingStatus, error) {
	createListener := func() (lisId string, err error) {
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
			log.Error(err, "create listener failed", "listenerId", lisId)
			err = errors.WithStack(err)
		} else {
			log.V(2).Info("create listener success", "listenerId", lisId)
		}
		return
	}
	lisId, err := createListener()
	if err == nil { // 创建成功，记录最新的监听器 ID 并更新监听器缓存
		updateListener(binding, lisId)
		return binding, nil
	}
	// 创建失败，判断错误
	if clb.IsLoadBalancerNotExistsError(err) { // lb 不存在导致创建失败，移除该绑定以便重新分配
		r.Recorder.Eventf(
			bd.GetObject(), corev1.EventTypeWarning, "CLBDeleted",
			"clb %s has been deleted when create listener",
			binding.LoadbalancerId,
		)
		notifyPortPoolReconcile(binding.Pool) // 通知端口池重新对账 lb 状态
		return nil, errors.Wrapf(err, "lb %q not exists when create listener %d/%s", binding.LoadbalancerId, binding.LoadbalancerPort, binding.Protocol)
	}

	if clb.IsPortCheckFailedError(err) { // 已有监听器占用该端口，对比下监听器是否符合预期，更新缓存
		lis, err := clb.GetListenerByPort(ctx, binding.Region, binding.LoadbalancerId, int64(binding.LoadbalancerPort), binding.Protocol)
		if err != nil {
			return binding, errors.WithStack(err)
		}
		if lis == nil { // 不存在，再次尝试创建（一般不可能）
			log.Info("port check failed but no port exists, retry")
			if lisId, err := createListener(); err != nil {
				return binding, errors.WithStack(err)
			} else { // 重试创建成功，记录最新的监听器 ID
				updateListener(binding, lisId)
				log.Info("retry create listener successfully", "listenerId", lisId)
				return binding, nil
			}
		} else { // 存在
			// 检查是否符合预期
			binding, err = r.ensureListenerExpected(ctx, binding, lis)
			if err != nil {
				return binding, errors.WithStack(err)
			}
			return binding, nil
		}
	}
	// 其它错误，直接返回
	return binding, errors.WithStack(err)
}

// 对账单个监听器
func (r *CLBBindingReconciler[T]) ensureListener(ctx context.Context, bd clbbinding.CLBBinding, binding *networkingv1alpha1.PortBindingStatus) (*networkingv1alpha1.PortBindingStatus, error) {
	log := log.FromContext(ctx, "binding", binding)
	log.V(5).Info("ensureListener")
	// 如果 lb 已被移除，且当前还未绑定成功，则移除该端口绑定，等待重新分配端口（配错 lb导致一直无法绑定成功，更正后，可以触发重新分配以便能够成功绑定）
	if bd.GetStatus().State != networkingv1alpha1.CLBBindingStateBound && !portpool.Allocator.IsLbExists(binding.Pool, portpool.NewLBKeyFromBinding(binding)) {
		log.Info("remove allocated clbbinding due to lb not exists")
		r.Recorder.Eventf(bd.GetObject(), corev1.EventTypeNormal, "PortBindingRemoved", "lb %q not exists, remove port binding (lbPort:%s protocol:%s)", binding.LoadbalancerId, binding.LoadbalancerPort, binding.Protocol)
		return nil, nil
	}

	removeReason := ""
	removeMsg := ""

	// 确保关联的 pool 无误
	pool := portpool.Allocator.GetPool(binding.Pool)
	if pool == nil {
		// 如果端口池被删，且当前还未绑定成功，则移除该端口绑定
		if bd.GetStatus().State != networkingv1alpha1.CLBBindingStateBound {
			removeReason = "PortPoolDeleted"
			removeMsg = "port pool has been deleted"
		}
	} else {
		// 如果 lb 已被移除，且当前还未绑定成功，则移除该端口绑定。等待重新分配端口（配错 lb导致一直无法绑定成功，更正后，可以触发重新分配以便能够成功绑定）
		if bd.GetStatus().State != networkingv1alpha1.CLBBindingStateBound && !pool.IsLbExists(portpool.NewLBKeyFromBinding(binding)) {
			removeReason = "CLBDeleted"
			removeMsg = "clb has been removed"
		}
	}
	if removeMsg != "" {
		log.Info("remove allocated clbbinding due to " + removeMsg)
		r.Recorder.Eventf(bd.GetObject(), corev1.EventTypeWarning, removeReason, "%s (%s/%s/%d/%s)", removeMsg, binding.Pool, binding.LoadbalancerId, binding.LoadbalancerPort, binding.Protocol)
		// 确保监听器被清理
		if err := r.cleanupPortBinding(ctx, binding, log, pool.IsPrecreateListenerEnabled()); err != nil {
			return binding, errors.WithStack(err)
		}
		if portpool.Allocator.ReleaseBinding(binding) {
			notifyPortPoolReconcile(binding.Pool)
		}
		return nil, nil
	}

	if binding.ListenerId == "" { // 还没有记录监听器 ID，尝试从缓存中获取（可能之前创建了但记录到 status 时遇到 k8s api 冲突导致失败了）
		lis, err := clb.GetListener(ctx, binding.LoadbalancerId, binding.Region, uint16(binding.LoadbalancerPort), binding.Protocol, true)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if lis != nil { // 可能是之前创建了监听器，但记录到 status 失败了
			binding.ListenerId = lis.ListenerId
		}
	}

	if binding.ListenerId == "" { // 还没创建过监听器，尝试创建
		log.V(1).Info("no listener id found, try to create listener")
		binding, err := r.createListener(ctx, bd, binding, log)
		if err != nil {
			return binding, errors.WithStack(err)
		}
		log.V(1).Info("listener first time created")
		return binding, nil
	}

	// 已有监听器 ID，对账看是否符合预期
	lis, err := clb.GetListenerById(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId)
	if err != nil {
		if clb.IsLoadBalancerNotExistsError(err) { // lb 已删除，通知关联的端口池重新对账
			r.Recorder.Eventf(
				bd.GetObject(), corev1.EventTypeWarning, "CLBDeleted",
				"clb %s has been deleted when get listener %s",
				binding.LoadbalancerId, binding.ListenerId,
			)
			notifyPortPoolReconcile(binding.Pool)
			// lb 已删除，将此 binding 从 status 中移除
			return nil, errors.WithStack(err)
		}
		return binding, errors.WithStack(err)
	}

	if lis == nil { // 有记录监听器 ID 但没有查到相应的监听器
		log.V(3).Info("listener not found use listenerId")
		// 先通过端口和协议查找是否存在监听器
		lis, err = clb.GetListenerByPort(ctx, binding.Region, binding.LoadbalancerId, int64(binding.LoadbalancerPort), binding.Protocol)
		if err != nil {
			return binding, errors.WithStack(err)
		}
		if lis != nil { // 如果存在，检测已有监听器是否符合预期
			log.V(3).Info("listener found use port and protocol")
			binding, err = r.ensureListenerExpected(ctx, binding, lis)
			if err != nil {
				return binding, errors.WithStack(err)
			}
			return binding, nil
		} else { // 如果不存在，直接尝试新建
			binding, err = r.createListener(ctx, bd, binding, log)
			if err != nil {
				return binding, errors.WithStack(err)
			}
			return binding, nil
		}
	} else { // 通过 ID 查到了监听器，对比是否符合预期
		log.V(3).Info("found listener id, ensureListenerExpected", "lis", lis)
		binding, err = r.ensureListenerExpected(ctx, binding, lis)
		if err != nil {
			return binding, errors.WithStack(err)
		}
		return binding, nil
	}
}

// 确保监听器创建并绑定 rs
func (r *CLBBindingReconciler[T]) ensureBackendBindings(ctx context.Context, bd clbbinding.CLBBinding) error {
	needBind := true
	status := bd.GetStatus()
	backend, err := bd.GetAssociatedObject(ctx, r.Client)
	if err != nil {
		if apierrors.IsNotFound(errors.Cause(err)) { // 后端不存在，一般是网络隔离场景，保持待绑定状态（正常情况 clbbinding 的 OwnerReference 是 pod/node，它们被清理后 clbbinding 也会被 gc 自动清理）
			if err = r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateNoBackend); err != nil {
				return errors.WithStack(err)
			}
			log.FromContext(ctx).V(1).Info("not bind backend due to backend not found")
			needBind = false
		}
		// 其它错误，直接返回
		return errors.WithStack(err)
	}
	// 获取 rs
	node, err := backend.GetNode(ctx)
	if err != nil {
		if err == clbbinding.ErrNodeNameIsEmpty { // pod 还未调度，更新状态和 event
			r.Recorder.Event(bd.GetObject(), corev1.EventTypeNormal, "WaitBackend", "wait pod to be scheduled")
			if err = r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateWaitBackend); err != nil {
				return errors.WithStack(err)
			}
			log.FromContext(ctx).V(1).Info("not bind backend due to pod not scheduled")
			needBind = false
		}
		return errors.WithStack(err)
	}

	// rs 后端节点类型校验: 只支持原生节点和超级节点
	if !util.IsNodeTypeSupported(node) {
		if status.State != networkingv1alpha1.CLBBindingStateNodeTypeNotSupported { // 节点类型不支持
			msg := "current node type is not supported, please use super node or native node"
			// 给自定义资源（CLBNodeBinding/CLBPodBinding) 和关联的 K8S 对象（Node/Pod）都发送 event 告知原因
			r.Recorder.Event(bd.GetObject(), corev1.EventTypeWarning, "NodeNotSupported", msg)
			r.Recorder.Event(backend.GetObject(), corev1.EventTypeWarning, "NodeNotSupported", msg)
			// 更新 clbbinding 状态
			status.State = networkingv1alpha1.CLBBindingStateNodeTypeNotSupported
			status.Message = msg
			if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
				return errors.WithStack(err)
			}
		}
		log.FromContext(ctx).Info("not bind backend due to node type not supported")
		needBind = false
	}

	// 如果 rs 还没有分配到 IP，更新状态和 event
	if backend.GetIP() == "" { // 等待 backend (Node/Pod) 分配 IP
		r.Recorder.Event(bd.GetObject(), corev1.EventTypeNormal, "WaitBackend", "wait backend network to be ready")
		if err = r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateWaitBackend); err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).V(1).Info("not bind backend due to no pod ip")
		needBind = false
	}

	// rs 准备就绪，确保 CLB 监听器创建并绑定到 rs
	type Result struct {
		Binding *networkingv1alpha1.PortBindingStatus
		Err     error
	}
	result := make(chan Result)
	for i := range status.PortBindings { // 遍历所有 binding
		go func(binding *networkingv1alpha1.PortBindingStatus) {
			// 确保 listener 创建并符合预期
			binding, err := r.ensureListener(ctx, bd, binding)
			// 如果 listener 无误、当前 binding 不需要被清理、且 rs 有 IP，那么确保 listener 要绑定到 rs
			if needBind && err == nil && binding != nil {
				err = r.ensurePortBound(ctx, bd, backend, binding)
			}
			result <- Result{Binding: binding, Err: err}
		}(status.PortBindings[i].DeepCopy())
	}

	// 构造对账后的 bindings，如有变化，更新到 status
	bindings := []networkingv1alpha1.PortBindingStatus{}
	for range status.PortBindings {
		r := <-result
		if r.Err != nil {
			err = multierr.Append(err, r.Err)
		}
		if r.Binding != nil {
			bindings = append(bindings, *r.Binding)
		}
	}
	clbbinding.SortPortBindings(bindings)
	if !reflect.DeepEqual(bindings, status.PortBindings) { // 有变化，更新到 status
		err := util.RetryIfPossible(func() error { // 确保更新成功，避免丢失已创建的 listenerId，导致需要更多的查询判断，拖慢速度
			_, err := bd.FetchObject(ctx, r.Client)
			if err != nil {
				return err
			}
			bd.GetStatus().PortBindings = bindings
			return r.Status().Update(ctx, bd.GetObject())
		})
		if err != nil {
			return errors.WithStack(err)
		}
	}
	// 对账 bindings 有一个或多个 error 返回， 直接返回 error
	if err != nil {
		return errors.WithStack(err)
	}
	// 还不需要绑定，不可能更新 Bound 状态和写入注解，直接返回
	if !needBind {
		return nil
	}

	// 所有端口都已绑定成功，更新状态并将绑定信息写入 backend 注解
	if status.State != networkingv1alpha1.CLBBindingStateBound {
		cost := time.Since(bd.GetCreationTimestamp().Time)
		log.FromContext(ctx).V(1).Info("binding performance", "cost", cost.String())
		r.Recorder.Event(bd.GetObject(), corev1.EventTypeNormal, "AllBound", "all targets bound to listener")
		if err := util.RetryIfPossible(func() error {
			_, err := bd.FetchObject(ctx, r.Client)
			if err != nil {
				return err
			}
			status := bd.GetStatus()
			status.State = networkingv1alpha1.CLBBindingStateBound
			status.Message = ""
			return r.Status().Update(ctx, bd.GetObject())
		}); err != nil {
			return errors.WithStack(err)
		}
	}

	// 确保映射的结果写到 backend 资源的注解上
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
	Address                              string   `json:"address"`
}

func lbStatusKey(poolName, lbId string) string {
	return fmt.Sprintf("%s/%s", poolName, lbId)
}

type LBStatusGetter struct {
	cache  map[string]*networkingv1alpha1.LoadBalancerStatus
	client client.Client
}

func NewLBStatusGetter(client client.Client) *LBStatusGetter {
	return &LBStatusGetter{
		cache:  make(map[string]*networkingv1alpha1.LoadBalancerStatus),
		client: client,
	}
}

var ErrLBNotFoundInPool = errors.New("loadbalancer not found in port pool")

func (g *LBStatusGetter) Get(ctx context.Context, poolName, lbId string) (*networkingv1alpha1.LoadBalancerStatus, error) {
	key := lbStatusKey(poolName, lbId)
	status, exists := g.cache[key]
	if exists {
		return status, nil
	}
	pp := &networkingv1alpha1.CLBPortPool{}
	if err := g.client.Get(ctx, client.ObjectKey{Name: poolName}, pp); err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range pp.Status.LoadbalancerStatuses {
		status := &pp.Status.LoadbalancerStatuses[i]
		g.cache[lbStatusKey(poolName, status.LoadbalancerID)] = status
	}
	status, exists = g.cache[key]
	if exists {
		return status, nil
	}
	return nil, errors.Wrapf(ErrLBNotFoundInPool, "loadbalancer %s not found in port pool %s", lbId, poolName)
}

// 确保映射的结果写到 backend 资源的注解上
func (r *CLBBindingReconciler[T]) ensureBackendStatusAnnotation(ctx context.Context, bd clbbinding.CLBBinding, backend clbbinding.Backend) error {
	lbStatuses := NewLBStatusGetter(r.Client)
	statuses := []PortBindingStatus{}
	bdStatus := bd.GetStatus()
	for _, binding := range bdStatus.PortBindings {
		status, err := lbStatuses.Get(ctx, binding.Pool, binding.LoadbalancerId)
		if err != nil {
			return errors.WithStack(err)
		}
		var endPort *uint16
		if !util.IsZero(binding.LoadbalancerEndPort) {
			val := binding.Port + (*binding.LoadbalancerEndPort - binding.LoadbalancerPort)
			endPort = &val
		}
		address := util.GetValue(status.Hostname)
		if address == "" && len(status.Ips) > 0 {
			address = status.Ips[0]
		}
		if address != "" {
			address = fmt.Sprintf("%s:%d", address, binding.LoadbalancerPort)
		}
		statuses = append(statuses, PortBindingStatus{
			PortBindingStatus: binding,
			EndPort:           endPort,
			Hostname:          status.Hostname,
			Ips:               status.Ips,
			Address:           address,
		})
	}
	val, err := json.Marshal(statuses)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := patchResult(ctx, r.Client, backend.GetObject(), string(val), false); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func patchResult(ctx context.Context, c client.Client, obj client.Object, result string, hostPort bool) error {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	var resultKey, statusKey string
	if hostPort {
		resultKey = constant.CLBHostPortMappingResultKey
		statusKey = constant.CLBHostPortMappingStatuslKey
	} else {
		resultKey = constant.CLBPortMappingResultKey
		statusKey = constant.CLBPortMappingStatuslKey
	}
	if annotations[constant.CLBPortMappingResultKey] != string(result) {
		patchMap := map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]string{
					resultKey: string(result),
					statusKey: "Ready",
				},
			},
		}
		if err := kube.PatchMap(ctx, c, obj, patchMap); err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).V(1).Info("patch clb port mapping result success", "value", string(result))
	}

	if !clusterinfo.AgonesSupported { // 没安装 agones，不做后续处理
		return nil
	}

	// 如果关联了 agones 的 gameserver，也给把 result 注解 patch 到 gameserver 上
	labels := obj.GetLabels()
	if labels == nil {
		return nil
	}
	if gsName := labels[constant.AgonesGameServerLabelKey]; gsName != "" {
		gs := &agonesv1.GameServer{}
		if err := c.Get(ctx, client.ObjectKey{
			Namespace: obj.GetNamespace(),
			Name:      gsName,
		}, gs); err != nil {
			if apierrors.IsNotFound(err) { // 实际不存在 gameserver，忽略
				return nil
			}
			return errors.WithStack(err)
		}
		// 存在对应的 gameserver，patch result 注解
		gsAnnotations := gs.GetAnnotations()
		if gsAnnotations == nil || gsAnnotations[resultKey] != string(result) {
			patchMap := map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{
						resultKey: string(result),
					},
				},
			}
			if err := kube.PatchMap(ctx, c, gs, patchMap); err != nil {
				return errors.WithStack(err)
			}
		}
		log.FromContext(ctx).V(1).Info("patch clb port mapping result to agones gameserver success", "value", string(result))
	}
	return nil
}

func (r *CLBBindingReconciler[T]) ensurePortBound(ctx context.Context, bd clbbinding.CLBBinding, backend clbbinding.Backend, binding *networkingv1alpha1.PortBindingStatus) error {
	log := log.FromContext(ctx, "binding", binding)
	log.V(3).Info("ensurePortBound", "binding", *binding)
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
		for _, target := range targetToDelete {
			ip := target.TargetIP
			backend, err := bd.GetAssociatedObjectByIP(ctx, r.Client, ip)
			if err != nil {
				return errors.WithStack(err)
			}
			if backend != nil {
				msg := fmt.Sprintf("port conflict due to %s:%d/%s is already bound to %s", binding.LoadbalancerId, binding.LoadbalancerPort, binding.Protocol, backend.GetName())
				r.Recorder.Event(bd.GetObject(), corev1.EventTypeWarning, "OtherTargetBound", msg)
				return nil
			}
		}
		r.Recorder.Eventf(bd.GetObject(), corev1.EventTypeNormal, "DeregisterTarget", "remove unexpected target: %v", targetToDelete)
		if err := clb.DeregisterTargetsForListener(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, targetToDelete...); err != nil {
			return errors.WithStack(err)
		}
	}
	// 绑定后端
	if !alreadyAdded {
		if err := clb.RegisterTarget(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, backendTarget); err != nil {
			return errors.WithStack(err)
		}
	}
	// 到这里，能确保后端 已绑定到所有 lb 监听器
	return nil
}

var (
	ErrCertIdNotFound = errors.New("no cert id found from secret")
	ErrPoolNotReady   = errors.New("pool not ready")
)

func (r *CLBBindingReconciler[T]) ensurePortAllocated(ctx context.Context, bd clbbinding.CLBBinding) error {
	status := bd.GetStatus()
	bindings := make(map[portKey]*networkingv1alpha1.PortBindingStatus)
	newBindings := []networkingv1alpha1.PortBindingStatus{}
	for i := range status.PortBindings {
		binding := &status.PortBindings[i]
		key := portKey{
			Port:     binding.Port,
			Protocol: binding.Protocol,
			Pool:     binding.Pool,
		}
		bindings[key] = binding
		newBindings = append(newBindings, *binding)
	}

	var allocatedPorts portpool.PortAllocations
	releasePorts := func() {
		if len(allocatedPorts) > 0 {
			allocatedPorts.Release()
		}
		allocatedPorts = nil
	}
	spec := bd.GetSpec()
	poolsShouldReconcile := make(map[string]struct{})
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
		for _, key := range keys {
			if _, exists := bindings[key]; exists { // 已分配端口，跳过
				continue LOOP_PORT
			}
		}
		// 未分配端口，先检查证书配置
		var certId *string
		if secretName := port.CertSecretName; secretName != nil && *secretName != "" {
			id, err := kube.GetCertIdFromSecret(ctx, r.Client, client.ObjectKey{
				Namespace: bd.GetNamespace(),
				Name:      *secretName,
			})
			if err != nil {
				releasePorts()
				if apierrors.IsNotFound(errors.Cause(err)) {
					r.Recorder.Eventf(bd.GetObject(), corev1.EventTypeWarning, "CertNotFound", "cert secret %q not found", *secretName)
					return errors.Wrapf(ErrCertIdNotFound, "cert secret %q not found", *secretName)
				}
				return errors.WithStack(err)
			}
			certId = &id
		}
		// 配置无误，执行分配
		before := time.Now()
		allocated, err := portpool.Allocator.Allocate(ctx, port.Pools, port.Protocol, util.GetValue(port.UseSamePortAcrossPools))
		cost := time.Since(before)
		log.FromContext(ctx).V(3).Info("allocate port", "cost", cost.String(), "allocated", allocated.String(), "protocol", port.Protocol, "pools", port.Pools, "useSamePortAcrossPools", util.GetValue(port.UseSamePortAcrossPools), "err", err)
		if err != nil {
			return errors.WithStack(err)
		}

		// 要么全部分配成功，要么无法分配
		if len(allocated) > 0 { // 分配成功
			for _, allocatedPort := range allocated {
				poolsShouldReconcile[allocatedPort.Name] = struct{}{}
				binding := networkingv1alpha1.PortBindingStatus{
					Port:             port.Port,
					Protocol:         allocatedPort.Protocol,
					CertId:           certId,
					Pool:             allocatedPort.Name,
					LoadbalancerId:   allocatedPort.LbId,
					LoadbalancerPort: allocatedPort.Port,
					Region:           allocatedPort.Region,
				}
				if allocatedPort.EndPort > 0 {
					binding.LoadbalancerEndPort = &allocatedPort.EndPort
				}
				newBindings = append(newBindings, binding)
			}
			allocatedPorts = append(allocatedPorts, allocated...)
		} else { // 只要有一个端口分配失败就认为失败
			releasePorts()                               // 为保证事务性，释放已分配的端口
			for poolName := range poolsShouldReconcile { // 通知关联的端口池对账，如果启用自动创建 clb，可触发 lb 扩容
				notifyPortPoolReconcile(poolName)
			}
			return portpool.ErrNoPortAvailable
		}
	}

	// 将已分配的端口写入 status
	clbbinding.SortPortBindings(newBindings)
	if !reflect.DeepEqual(newBindings, status.PortBindings) {
		status.PortBindings = newBindings
		if len(allocatedPorts) > 0 { // 有分配到端口，更新 state 为 Allocated
			status.State = networkingv1alpha1.CLBBindingStateAllocated
		}
		if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
			// 更新状态失败，释放已分配端口
			releasePorts()
			return errors.WithStack(err)
		}
		for _, pool := range allocatedPorts.Pools() {
			notifyPortPoolReconcile(pool)
		}
	}
	return nil
}

func (r *CLBBindingReconciler[T]) ensureState(ctx context.Context, bd clbbinding.CLBBinding, state networkingv1alpha1.CLBBindingState) error {
	status := bd.GetStatus()
	if status.State == state {
		return nil
	}
	status.State = state
	status.Message = ""
	log.FromContext(ctx).V(5).Info("ensure state", "state", state)
	if err := r.Status().Update(ctx, bd.GetObject()); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// 清理 CLBBinding
func (r *CLBBindingReconciler[T]) cleanup(ctx context.Context, bd T) (result ctrl.Result, err error) {
	anno := bd.GetAnnotations()
	if anno == nil {
		anno = make(map[string]string)
	}
	if anno[constant.FinalizedKey] == "true" {
		return result, nil
	}
	log := log.FromContext(ctx)
	status := bd.GetStatus()
	log.Info("cleanup "+bd.GetType(), "bindings", len(status.PortBindings))
	if err = r.ensureState(ctx, bd, networkingv1alpha1.CLBBindingStateDeleting); err != nil {
		return result, errors.WithStack(err)
	}
	ch := make(chan error)
	controllerutil.ContainsFinalizer(bd.GetObject(), constant.Finalizer)
	for _, binding := range status.PortBindings {
		go func(binding *networkingv1alpha1.PortBindingStatus) {
			isListenerPrecreated := false
			if pool := portpool.Allocator.GetPool(binding.Pool); pool != nil && pool.IsPrecreateListenerEnabled() {
				isListenerPrecreated = true
			}
			if err := r.cleanupPortBinding(ctx, binding, log, isListenerPrecreated); err != nil {
				ch <- err
			} else {
				ch <- nil
			}
		}(&binding)
	}
	for range status.PortBindings {
		e := <-ch
		if e != nil {
			err = multierr.Append(err, e)
		}
	}
	if err != nil {
		return result, errors.WithStack(err)
	}
	// 全部解绑完成，打上标记，避免重复释放端口导致冲突（比如刚释放完端口又被其它 pod 分配，然后再次进入cleanup时又被清理，有可能再次分配给其它 pod，导致相同ip:port被重复分配）
	anno[constant.FinalizedKey] = "true"
	bd.SetAnnotations(anno)
	if err := r.Update(ctx, bd.GetObject()); err != nil {
		return result, errors.WithStack(err)
	}
	// 释放已分配端口
	log.V(5).Info("release allocated ports", "bindings", status.PortBindings)
	pools := make(map[string]struct{})
	for _, binding := range status.PortBindings {
		if portpool.Allocator.ReleaseBinding(&binding) {
			pools[binding.Pool] = struct{}{}
		}
	}
	for pool := range pools {
		notifyPortPoolReconcile(pool)
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

// 解绑：
// 1）预创建监听器场景：解绑 rs
// 2）其它：删除监听器
func (r *CLBBindingReconciler[T]) cleanupPortBinding(ctx context.Context, binding *networkingv1alpha1.PortBindingStatus, log logr.Logger, isListenerPrecreated bool) error {
	log.V(2).Info("cleanupPortBinding")
	if isListenerPrecreated { // 预创建监听器，仅解绑 rs
		if binding.ListenerId != "" {
			clb.DeregisterAllTargetsTryBatch(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId)
		}
		return nil
	}
	// 非预创建监听器，直接删除监听器并清理缓存
	clb.GetListenerCache(clb.LBKey{Region: binding.Region, LbId: binding.LoadbalancerId}).EnsureRemoved(ctx, binding.LoadbalancerPort, binding.Protocol)
	err := clb.DeleteListenerByIdOrPort(ctx, binding.Region, binding.LoadbalancerId, binding.ListenerId, int64(binding.LoadbalancerPort), binding.Protocol)
	if err != nil {
		errCause := errors.Cause(err)
		switch errCause {
		case clb.ErrListenerNotFound: // 监听器不存在，忽略
			log.Info("delete listener while listener not found, ignore")
			return nil
		default:
			if clb.IsLoadBalancerNotExistsError(errCause) { // lb 不存在，忽略
				log.Info("lb not found, ignore when cleanup listener")
				return nil
			}
		}
		// 其它错误，不释放端口，返回错误
		return errors.WithStack(err)
	} else { // 没有错误，删除成功
		return nil
	}
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
				log.FromContext(ctx).V(5).Info("create clbbinding", "binding", bd)
				if err := r.Create(ctx, bd); err != nil {
					if apierrors.IsAlreadyExists(err) { // 已存在，通常是刚刚创建了但 cache 还没更新导致，忽略错误
						return result, nil
					} else {
						r.Recorder.Eventf(obj, corev1.EventTypeWarning, "CreateCLBBinding", "create %s %s failed: %s", binding.GetType(), obj.GetName(), err.Error())
						return result, errors.WithStack(err)
					}
				}
				r.Recorder.Eventf(obj, corev1.EventTypeNormal, "CreateCLBBinding", "create %s %s successfully", binding.GetType(), obj.GetName())
			} else { // 其它错误，直接返回错误
				return result, errors.WithStack(err)
			}
		} else { // 存在
			// 正在删除，重新入队（滚动更新场景，旧的解绑完，确保 CLBBinding 要重建出来）
			if !binding.GetDeletionTimestamp().IsZero() {
				log.FromContext(ctx).Info("wait old clbbinding to be deleted, will retry", "binding", binding)
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
		// 没有配置注解
		if err == nil { // 没有错误，说明获取 CLBBinding 成功，删除掉这个多余的 CLBBinding
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

func shouldNotify(portpool client.Object, spec networkingv1alpha1.CLBBindingSpec, status networkingv1alpha1.CLBBindingStatus) bool {
	switch status.State {
	case "", networkingv1alpha1.CLBBindingStatePending, // 还未分配端口的状态，触发对账分配端口
		networkingv1alpha1.CLBBindingStateNoPortAvailable,        // 分配过端口但当时端口不足，触发一次对账重新分配
		networkingv1alpha1.CLBBindingStatePortPoolNotFound,       // 之前端口池不存在，但现在有了，触发一次对账以便分配端口。通常是 apply yaml 场景，端口池和工作负载同时创建，先后顺序不固定导致
		networkingv1alpha1.CLBBindingStatePortPoolNotAllocatable: // 之前端口池不可分配，但现在可以分配了，触发一次对账以便分配端口。通常是端口池还未就绪，等待就绪后自动触发对账重新分配端口
		for _, port := range spec.Ports {
			if slices.Contains(port.Pools, portpool.GetName()) {
				return true
			}
		}
	}
	return false
}
