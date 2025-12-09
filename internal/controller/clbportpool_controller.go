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
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"
	"github.com/tkestack/tke-extend-network-controller/internal/portpool"
	"github.com/tkestack/tke-extend-network-controller/pkg/clb"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/tkestack/tke-extend-network-controller/pkg/eventsource"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
)

// CLBPortPoolReconciler reconciles a CLBPortPool object
type CLBPortPoolReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbportpools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbportpools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbportpools/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *CLBPortPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ReconcileWithFinalizer(ctx, req, r.Client, &networkingv1alpha1.CLBPortPool{}, r.sync, r.cleanup)
}

// 清理端口池
func (r *CLBPortPoolReconciler) cleanup(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) (result ctrl.Result, err error) {
	if err := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateDeleting); err != nil {
		return result, errors.WithStack(err)
	}
	// 从全局端口分配器缓存中移除该端口池
	portpool.Allocator.RemovePool(pool.Name)
	// 删除自动创建的 CLB
	for _, lb := range pool.Status.LoadbalancerStatuses {
		if !util.GetValue(lb.AutoCreated) {
			continue
		}
		if err := clb.Delete(ctx, pool.GetRegion(), lb.LoadbalancerID); err != nil {
			return result, errors.WithStack(err)
		}
	}
	return
}

func (r *CLBPortPoolReconciler) ensureState(ctx context.Context, pool *networkingv1alpha1.CLBPortPool, state networkingv1alpha1.CLBPortPoolState) error {
	if pool.Status.State != state {
		pool.Status.State = state
		pool.Status.Message = nil
		if err := r.Status().Update(ctx, pool); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (r *CLBPortPoolReconciler) getCLBInfo(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) (info map[string]*clb.CLBInfo, err error) {
	// 拿到所有需要查询的 LbId
	lbIdMap := make(map[string]bool)
	for _, lbId := range pool.Spec.ExsistedLoadBalancerIDs {
		lbIdMap[lbId] = true
	}
	for _, lb := range pool.Status.LoadbalancerStatuses {
		if lb.State != networkingv1alpha1.LoadBalancerStateNotFound && !lbIdMap[lb.LoadbalancerID] {
			lbIdMap[lb.LoadbalancerID] = true
		}
	}
	if len(lbIdMap) == 0 {
		return nil, nil
	}
	lbIds := []string{}
	for lbId := range lbIdMap {
		lbIds = append(lbIds, lbId)
	}
	info, err = clb.BatchGetClbInfo(ctx, lbIds, util.GetRegionFromPtr(pool.Spec.Region))
	if err != nil {
		err = errors.WithStack(err)
	}
	return
}

func (r *CLBPortPoolReconciler) ensureExistedLB(pool *networkingv1alpha1.CLBPortPool, lbInfos map[string]*clb.CLBInfo, status *networkingv1alpha1.CLBPortPoolStatus) {
	// 对账 clb
	lbIds := make(map[string]struct{})
	for _, lbStatus := range status.LoadbalancerStatuses {
		lbIds[lbStatus.LoadbalancerID] = struct{}{}
	}
	lbIdToAdd := []string{}
	for _, lbId := range pool.Spec.ExsistedLoadBalancerIDs {
		if _, exists := lbIds[lbId]; !exists {
			lbIdToAdd = append(lbIdToAdd, lbId)
		}
	}
	if len(lbIdToAdd) == 0 { // 没有新增已有 clb，忽略
		return
	}
	lbToAdd := []networkingv1alpha1.LoadBalancerStatus{}
	lbNotExisted := []string{}
	// 确保所有已有 CLB 在 status 中存在
	for _, lbId := range lbIdToAdd {
		// 校验 CLB 是否存在
		lbInfo, ok := lbInfos[lbId]
		if !ok {
			lbNotExisted = append(lbNotExisted, lbId)
			continue
		}
		// CLB 存在，准备加进来
		lbStatus := networkingv1alpha1.LoadBalancerStatus{
			LoadbalancerID:   lbId,
			LoadbalancerName: lbInfo.LoadbalancerName,
			Ips:              lbInfo.Ips,
			Hostname:         lbInfo.Hostname,
		}
		lbToAdd = append(lbToAdd, lbStatus)
	}
	// 如果有需要加进来的，加到 status 中，并更新缓存
	if len(lbToAdd) > 0 {
		status.LoadbalancerStatuses = append(status.LoadbalancerStatuses, lbToAdd...)
	}
	if len(lbNotExisted) > 0 {
		lbIds := strings.Join(lbNotExisted, ",")
		msg := fmt.Sprintf("clb %s not found", lbIds)
		r.Recorder.Event(pool, corev1.EventTypeWarning, "EnsureExistedLB", msg)
	}
	return
}

func (r *CLBPortPoolReconciler) ensureLbStatus(ctx context.Context, pool *networkingv1alpha1.CLBPortPool, lbInfos map[string]*clb.CLBInfo, status *networkingv1alpha1.CLBPortPoolStatus) error {
	lbStatuses := []networkingv1alpha1.LoadBalancerStatus{}
	allocatableLBs := []portpool.LBKey{}
	insufficientPorts := true
	autoCreatedLbNum := uint16(0)

	existedLbIds := make(map[string]struct{})
	for _, lbId := range pool.Spec.ExsistedLoadBalancerIDs {
		existedLbIds[lbId] = struct{}{}
	}

	// 反查未记录的自动创建 CLB，并补充记录
	autoCreatedCLBs, err := clb.ListCLBsByTags(ctx, pool.GetRegion(), map[string]string{
		constant.TkeClusterIDTagKey:   clusterinfo.ClusterId,
		constant.CLBPortPoolTagKey:    pool.Name,
		constant.TkeCreatedFlagTagKey: constant.TkeCreatedFlagYesValue,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	recordedLbIds := make(map[string]struct{})
	for _, lbStatus := range status.LoadbalancerStatuses {
		recordedLbIds[lbStatus.LoadbalancerID] = struct{}{}
	}
	for _, lb := range autoCreatedCLBs {
		lbId := *lb.LoadBalancerId
		if _, exists := recordedLbIds[lbId]; !exists {
			// 发现未记录的自动创建 CLB，补充记录
			lbStatus := networkingv1alpha1.LoadBalancerStatus{
				LoadbalancerID:   lbId,
				LoadbalancerName: *lb.LoadBalancerName,
				AutoCreated:      util.GetPtr(true),
				State:            networkingv1alpha1.LoadBalancerStateRunning,
			}
			if util.GetValue(lb.LoadBalancerDomain) != "" {
				lbStatus.Hostname = lb.LoadBalancerDomain
			} else {
				lbStatus.Ips = util.ConvertPtrSlice(lb.LoadBalancerVips)
			}
			lbStatuses = append(lbStatuses, lbStatus)
			lbKey := portpool.NewLBKey(lbId, pool.GetRegion())
			allocatableLBs = append(allocatableLBs, lbKey)
			r.Recorder.Eventf(pool, corev1.EventTypeNormal, "EnsureAutoCreatedCLB", "found auto-created CLB %s missing in status, add it to status", lbId)
		}
	}

	// 1. 确保自动创建的 CLB 有正确的标签
	// 2. 多余的 lb 也自动移除
	// 3. 更新 lb 信息与状态
	expectedTags := map[string]string{
		constant.CLBPortPoolTagKey:    pool.Name,
		constant.TkeClusterIDTagKey:   clusterinfo.ClusterId,
		constant.TkeCreatedFlagTagKey: constant.TkeCreatedFlagYesValue,
	}
	for _, lbStatus := range status.LoadbalancerStatuses {
		lbId := lbStatus.LoadbalancerID
		if info, ok := lbInfos[lbId]; ok { // clb 存在，更新lb相关信息
			lbKey := portpool.NewLBKey(lbId, pool.GetRegion())
			lbStatus.State = networkingv1alpha1.LoadBalancerStateRunning
			lbStatus.Ips = info.Ips
			lbStatus.Hostname = info.Hostname
			lbStatus.LoadbalancerName = info.LoadbalancerName
			lbStatus.Allocated = portpool.Allocator.AllocatedPorts(pool.Name, lbKey)
			if insufficientPorts && status.Quota-lbStatus.Allocated > 2 {
				insufficientPorts = false
			}
			if util.GetValue(lbStatus.AutoCreated) { // 自动创建的 lb，确保标签正确，计算数量（用于自动创建最大clb数量限制的校验）
				autoCreatedLbNum++
				missingTags := make(map[string]string)
				if lbInfo, ok := lbInfos[lbId]; ok {
					for k, v := range expectedTags {
						if lbInfo.Tags[k] != v {
							missingTags[k] = v
						}
					}
				}
				if len(missingTags) > 0 {
					if err := clb.EnsureCLBTags(ctx, pool.GetRegion(), lbStatus.LoadbalancerID, missingTags); err != nil {
						r.Recorder.Eventf(pool, corev1.EventTypeWarning, "EnsureCLBTags", "failed to ensure tags for CLB %s: %s", lbStatus.LoadbalancerID, err.Error())
					}
				}
			} else { // 已有的 lb，如果被手动移除，可能是 lb 有误（比如错误的 vpc），也将其从status和分配器中移除
				if _, exists := existedLbIds[lbId]; !exists {
					if portpool.Allocator.RemoveLB(pool.Name, portpool.NewLBKey(lbId, pool.GetRegion())) {
						r.Recorder.Eventf(pool, corev1.EventTypeNormal, "RemoveLoadBalancer", "remove existed clb %s from pool", lbId)
					}
					continue
				}
			}
			allocatableLBs = append(allocatableLBs, lbKey)
		} else { // clb 不存在，通常是已删除，更新状态和端口池
			if lbStatus.State != networkingv1alpha1.LoadBalancerStateNotFound {
				r.Recorder.Eventf(pool, corev1.EventTypeWarning, "GetLoadBalancer", "clb %s not found", lbId)
				lbStatus.State = networkingv1alpha1.LoadBalancerStateNotFound
			}
		}
		lbStatuses = append(lbStatuses, lbStatus)
	}

	status.LoadbalancerStatuses = lbStatuses

	// 确保所有可分配的 lb 在分配器缓存中
	if err := portpool.Allocator.EnsureLbIds(pool.Name, allocatableLBs); err != nil {
		return errors.WithStack(err)
	}

	if insufficientPorts { // 可分配端口不足，尝试扩容 clb
		if pool.Spec.AutoCreate != nil && pool.Spec.AutoCreate.Enabled { // 必须启用了 clb 自动创建
			if pool.Spec.AutoCreate.MaxLoadBalancers == nil || autoCreatedLbNum < *pool.Spec.AutoCreate.MaxLoadBalancers { // 满足可以自动创建clb的条件：没有限制自动创建的 clb 数量，或者自动创建的 clb 数量未达到限制
				if err := r.createCLB(ctx, pool, status); err != nil { // 创建 clb
					return errors.WithStack(err)
				}
			}
		}
	}
	return nil
}

func (r *CLBPortPoolReconciler) createCLB(ctx context.Context, pool *networkingv1alpha1.CLBPortPool, status *networkingv1alpha1.CLBPortPoolStatus) (err error) {
	r.Recorder.Event(pool, corev1.EventTypeNormal, "CreateLoadBalancer", "try to create clb")
	lbId, err := clb.CreateCLB(ctx, pool.GetRegion(), clb.ConvertCreateLoadBalancerRequest(pool.Spec.AutoCreate.Parameters, pool.Name))
	if err != nil { // 创建失败，记录 event，回滚 state
		r.Recorder.Eventf(pool, corev1.EventTypeWarning, "CreateLoadBalancer", "create clb failed: %s", err.Error())
		if e := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateActive); e != nil {
			err = multierr.Append(err, e)
		}
		return errors.WithStack(err)
	}
	// 创建成功，记录 event，更新 state 并记录 lbId
	r.Recorder.Eventf(pool, corev1.EventTypeNormal, "CreateLoadBalancer", "create clb success: %s", lbId)
	status.LoadbalancerStatuses = append(status.LoadbalancerStatuses, networkingv1alpha1.LoadBalancerStatus{
		LoadbalancerID: lbId,
		AutoCreated:    util.GetPtr(true),
	})
	return
}

func (r *CLBPortPoolReconciler) ensureLb(ctx context.Context, pool *networkingv1alpha1.CLBPortPool, status *networkingv1alpha1.CLBPortPoolStatus) error {
	// 查询 lb 信息并保存到 context，以供后面的 ensureXXX 对账使用
	info, err := r.getCLBInfo(ctx, pool)
	if err != nil {
		return errors.WithStack(err)
	}
	// 确保已有 lb 被添加到 status 中
	r.ensureExistedLB(pool, info, status)
	// 同步 clb 信息到 status
	if err := r.ensureLbStatus(ctx, pool, info, status); err != nil {
		return errors.WithStack(err)
	}
	// 同步监听器缓存
	if err := r.ensureListenerCache(ctx, pool); err != nil {
		return errors.WithStack(err)
	}
	// 预创建监听器
	if pool.Spec.ListenerPrecreate != nil && pool.Spec.ListenerPrecreate.Enabled {
		if err := r.ensureListenerPrecreate(ctx, pool, pool.Spec.ListenerPrecreate, status); err != nil {
			return errors.WithStack(err)
		}
	}
	// lb 准备就绪，确保状态为 Active
	status.State = networkingv1alpha1.CLBPortPoolStateActive
	return nil
}

func (r *CLBPortPoolReconciler) ensureListenerCache(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	for _, lb := range pool.Status.LoadbalancerStatuses {
		if lb.State != networkingv1alpha1.LoadBalancerStateRunning {
			continue
		}
		lbKey := clb.LBKey{LbId: lb.LoadbalancerID, Region: pool.GetRegion()}
		if err := clb.GetListenerCache(lbKey).EnsureInit(ctx); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// 确保预创建的监听器都被创建
func (r *CLBPortPoolReconciler) ensureListenerPrecreate(ctx context.Context, pool *networkingv1alpha1.CLBPortPool, lpc *networkingv1alpha1.ListenerPrecreateConfig, status *networkingv1alpha1.CLBPortPoolStatus) error {
	tcpNum := util.GetValue(lpc.TCP)
	udpNum := util.GetValue(lpc.UDP)
	startPort := pool.Spec.StartPort
	segmentLength := util.GetValue(pool.Spec.SegmentLength)
	ensureListenerCreated := func(lbId, protocol string, num uint16) error {
		var ports, endPorts []int64
		for i := uint16(0); i < num; i++ {
			port := startPort + i
			if lis, err := clb.GetListener(ctx, lbId, pool.GetRegion(), port, protocol, true); err != nil {
				return errors.WithStack(err)
			} else {
				if lis == nil { // 监听器不存在，创建
					ports = append(ports, int64(port))
					if segmentLength > 1 {
						endPorts = append(endPorts, int64(port+segmentLength-1))
					}
				}
			}
		}
		if len(ports) == 0 { // 所有监听器都已存在
			return nil
		}
		// 有监听器需要创建
		lisIds, err := clb.BatchCreateListener(ctx, pool.GetRegion(), lbId, protocol, ports, endPorts)
		if err != nil {
			return errors.WithStack(err)
		}
		lbKey := clb.LBKey{LbId: lbId, Region: pool.GetRegion()}
		lisCache := clb.GetListenerCache(lbKey)
		for i, port := range ports {
			lis := &clb.Listener{
				Port:         port,
				Protocol:     protocol,
				ListenerName: clb.TkeListenerName,
				ListenerId:   lisIds[i],
			}
			if len(endPorts) > 0 {
				lis.EndPort = endPorts[i]
			}
			lisCache.Set(lis)
		}
		return nil
	}
	for i := range status.LoadbalancerStatuses {
		lbStatus := &status.LoadbalancerStatuses[i]
		if tcpNum > 0 {
			if err := ensureListenerCreated(lbStatus.LoadbalancerID, "TCP", tcpNum); err != nil {
				return errors.WithStack(err)
			}
		}
		if udpNum > 0 {
			if err := ensureListenerCreated(lbStatus.LoadbalancerID, "UDP", udpNum); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}

var ErrNoListenerNum = errors.New("no listener number specified when listenerPrecreate is enabled")

func (r *CLBPortPoolReconciler) ensureQuota(ctx context.Context, pool *networkingv1alpha1.CLBPortPool, status *networkingv1alpha1.CLBPortPoolStatus) error {
	var quota uint16
	if lcp := pool.Spec.ListenerPrecreate; lcp != nil && lcp.Enabled { // 预创建监听器，quota 为所有类型的预创建监听器数量之和
		quota = util.GetValue(lcp.TCP) + util.GetValue(lcp.UDP)
		if quota == 0 {
			return ErrNoListenerNum
		}
	} else { // 动态创建监听器，quota 为 CLB 监听器数量的配额限制
		quota = util.GetValue(pool.Spec.ListenerQuota)
		if quota == 0 {
			q, err := clb.Quota.GetQuota(ctx, pool.GetRegion(), clb.TOTAL_LISTENER_QUOTA)
			if err != nil {
				return errors.WithStack(err)
			}
			quota = uint16(q)
		}
	}
	status.Quota = quota
	return nil
}

// 同步端口池
func (r *CLBPortPoolReconciler) sync(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) (result ctrl.Result, err error) {
	// 确保分配器缓存中存在该 port pool，放在最开头，避免同时创建 CLBPortPool 和 CLBBinding 导致分配端口时找不到 pool
	portpool.Allocator.EnsurePool(pool)

	// 初始化状态
	if pool.Status.State == "" {
		if err := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStatePending); err != nil {
			return result, errors.WithStack(err)
		}
	}

	status := pool.Status.DeepCopy()

	// 同步 quota
	if err := r.ensureQuota(ctx, pool, status); err != nil {
		return result, errors.WithStack(err)
	}
	// 确保 lb 列表可用
	if err := r.ensureLb(ctx, pool, status); err != nil {
		return result, errors.WithStack(err)
	}
	// 对比 status 是否有变化，如果有变化则更新
	if !reflect.DeepEqual(*status, pool.Status) { // 确保更新成功，否则一直重试（避免自动创建的 lb id 丢失）
		if err := util.RetryIfPossible(func() error {
			cpp := &networkingv1alpha1.CLBPortPool{}
			if err := r.Get(ctx, client.ObjectKeyFromObject(pool), cpp); err != nil {
				return err
			}
			if reflect.DeepEqual(*status, cpp.Status) { // dubble check
				return nil
			}
			cpp.Status = *status
			if err := r.Status().Update(ctx, cpp); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return result, errors.WithStack(err)
		}
	}
	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLBPortPoolReconciler) SetupWithManager(mgr ctrl.Manager, workers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBPortPool{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workers,
		}).
		WatchesRawSource(source.Channel(eventsource.PortPool, &handler.EnqueueRequestForObject{})).
		Named("clbportpool").
		Complete(r)
}

func notifyPortPoolReconcile(poolName string) {
	pp := &networkingv1alpha1.CLBPortPool{}
	pp.Name = poolName
	eventsource.PortPool <- event.TypedGenericEvent[client.Object]{
		Object: pp,
	}
}
