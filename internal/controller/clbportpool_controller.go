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

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/eventsource"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
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

func (r *CLBPortPoolReconciler) ensureExistedLB(ctx context.Context, pool *networkingv1alpha1.CLBPortPool, lbInfos map[string]*clb.CLBInfo) error {
	// 对账 clb
	lbIds := make(map[string]struct{})
	for _, lbStatus := range pool.Status.LoadbalancerStatuses {
		lbIds[lbStatus.LoadbalancerID] = struct{}{}
	}
	lbIdToAdd := []string{}
	for _, lbId := range pool.Spec.ExsistedLoadBalancerIDs {
		if _, exists := lbIds[lbId]; !exists {
			lbIdToAdd = append(lbIdToAdd, lbId)
		}
	}
	if len(lbIdToAdd) == 0 { // 没有新增已有 clb，忽略
		return nil
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
		status := networkingv1alpha1.LoadBalancerStatus{
			LoadbalancerID:   lbId,
			LoadbalancerName: lbInfo.LoadbalancerName,
			Ips:              lbInfo.Ips,
			Hostname:         lbInfo.Hostname,
		}
		lbToAdd = append(lbToAdd, status)
	}
	// 如果有需要加进来的，加到 status 中，并更新缓存
	if len(lbToAdd) > 0 {
		pool.Status.LoadbalancerStatuses = append(pool.Status.LoadbalancerStatuses, lbToAdd...)
		if err := r.Status().Update(ctx, pool); err != nil {
			return errors.WithStack(err)
		}
	}
	if len(lbNotExisted) > 0 {
		lbIds := strings.Join(lbNotExisted, ",")
		msg := fmt.Sprintf("clb %s not found", lbIds)
		r.Recorder.Event(pool, corev1.EventTypeWarning, "EnsureExistedLB", msg)
	}
	return nil
}

func (r *CLBPortPoolReconciler) ensureLbStatus(ctx context.Context, pool *networkingv1alpha1.CLBPortPool, lbInfos map[string]*clb.CLBInfo) error {
	quota := pool.Status.Quota
	lbStatuses := []networkingv1alpha1.LoadBalancerStatus{}
	allocatableLBs := []portpool.LBKey{}
	insufficientPorts := true
	autoCreatedLbNum := uint16(0)

	// 构造当前 lb status 列表
	for _, lbStatus := range pool.Status.LoadbalancerStatuses {
		lbId := lbStatus.LoadbalancerID
		// clb 不存在，通常是已删除，更新状态和端口池
		if info, ok := lbInfos[lbId]; ok {
			lbKey := portpool.NewLBKey(lbId, pool.GetRegion())
			lbStatus.State = networkingv1alpha1.LoadBalancerStateRunning
			lbStatus.Ips = info.Ips
			lbStatus.Hostname = info.Hostname
			lbStatus.LoadbalancerName = info.LoadbalancerName
			lbStatus.Allocated = portpool.Allocator.AllocatedPorts(pool.Name, lbKey)
			if insufficientPorts && quota-lbStatus.Allocated > 2 {
				insufficientPorts = false
			}
			if util.GetValue(lbStatus.AutoCreated) {
				autoCreatedLbNum++
			}
			allocatableLBs = append(allocatableLBs, lbKey)
		} else {
			if lbStatus.State != networkingv1alpha1.LoadBalancerStateNotFound {
				r.Recorder.Eventf(pool, corev1.EventTypeWarning, "GetLoadBalancer", "clb %s not found", lbId)
				lbStatus.State = networkingv1alpha1.LoadBalancerStateNotFound
			}
		}
		lbStatuses = append(lbStatuses, lbStatus)
	}

	// 确保所有可分配的 lb 在分配器缓存中
	if err := portpool.Allocator.EnsureLbIds(pool.Name, allocatableLBs); err != nil {
		return errors.WithStack(err)
	}

	// 如果 status 有变更就更新下
	if quota != pool.Status.Quota || !reflect.DeepEqual(lbStatuses, pool.Status.LoadbalancerStatuses) {
		pool.Status.LoadbalancerStatuses = lbStatuses
		pool.Status.Quota = quota
		if err := r.Status().Update(ctx, pool); err != nil {
			return errors.WithStack(err)
		}
	}

	if insufficientPorts { // 可分配端口不足，尝试扩容 clb
		if pool.Spec.AutoCreate != nil && pool.Spec.AutoCreate.Enabled { // 必须启用了 clb 自动创建
			if pool.Spec.AutoCreate.MaxLoadBalancers == nil || autoCreatedLbNum < *pool.Spec.AutoCreate.MaxLoadBalancers { // 满足可以自动创建clb的条件：没有限制自动创建的 clb 数量，或者自动创建的 clb 数量未达到限制
				if err := r.createCLB(ctx, pool); err != nil { // 创建 clb
					return errors.WithStack(err)
				}
			}
		}
	}
	return nil
}

func (r *CLBPortPoolReconciler) createCLB(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) (err error) {
	r.Recorder.Event(pool, corev1.EventTypeNormal, "CreateLoadBalancer", "try to create clb")
	if err = r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateScaling); err != nil {
		err = errors.WithStack(err)
		return
	}
	pool.Status.State = networkingv1alpha1.CLBPortPoolStateScaling
	if err = r.Status().Update(ctx, pool); err != nil {
		err = errors.WithStack(err)
		return
	}
	lbId, err := clb.CreateCLB(ctx, pool.GetRegion(), clb.ConvertCreateLoadBalancerRequest(pool.Spec.AutoCreate.Parameters))
	if err != nil { // 创建失败，记录 event，回滚 state
		r.Recorder.Eventf(pool, corev1.EventTypeWarning, "CreateLoadBalancer", "create clb failed: %s", err.Error())
		if e := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateActive); e != nil {
			err = multierr.Append(err, e)
		}
		return errors.WithStack(err)
	}
	// 创建成功，记录 event，更新 state 并记录 lbId
	r.Recorder.Eventf(pool, corev1.EventTypeNormal, "CreateLoadBalancer", "create clb success: %s", lbId)
	lbStatuses := append(pool.Status.LoadbalancerStatuses, networkingv1alpha1.LoadBalancerStatus{
		LoadbalancerID: lbId,
		AutoCreated:    util.GetPtr(true),
	})
	addLbIdToStatus := func() error {
		pool.Status.State = networkingv1alpha1.CLBPortPoolStateActive // 创建成功，状态改为 Active，以便再次可分配端口
		pool.Status.LoadbalancerStatuses = lbStatuses
		if err := r.Status().Update(ctx, pool); err != nil {
			return err
		}
		return nil
	}
	if err = util.RetryIfPossible(addLbIdToStatus); err != nil {
		err = errors.WithStack(err)
		return
	}
	return
}

func (r *CLBPortPoolReconciler) ensureLb(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	// 查询 lb 信息并保存到 context，以供后面的 ensureXXX 对账使用
	info, err := r.getCLBInfo(ctx, pool)
	if err != nil {
		return errors.WithStack(err)
	}
	// 确保已有 lb 被添加到 status 中
	if err := r.ensureExistedLB(ctx, pool, info); err != nil {
		return errors.WithStack(err)
	}
	// 同步 clb 信息到 status
	if err := r.ensureLbStatus(ctx, pool, info); err != nil {
		return errors.WithStack(err)
	}
	// lb 准备就绪，确保状态为 Active
	if err := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateActive); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// 同步端口池
func (r *CLBPortPoolReconciler) sync(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) (result ctrl.Result, err error) {
	// 确保分配器缓存中存在该 port pool，放在最开头，避免同时创建 CLBPortPool 和 CLBBinding 导致分配端口时找不到 pool
	portpool.Allocator.AddPoolIfNotExists(pool.Name)

	needUpdateStatus := false

	// 确定 quota
	if pool.Status.Quota == 0 {
		quota := util.GetValue(pool.Spec.ListenerQuota)
		if quota == 0 {
			q, err := clb.Quota.GetQuota(ctx, pool.GetRegion(), clb.TOTAL_LISTENER_QUOTA)
			if err != nil {
				return result, errors.WithStack(err)
			}
			quota = uint16(q)
		}
		pool.Status.Quota = quota
		needUpdateStatus = true
	}

	// 初始化状态
	if pool.Status.State == "" {
		pool.Status.State = networkingv1alpha1.CLBPortPoolStatePending
		needUpdateStatus = true
	}

	if needUpdateStatus {
		if err := r.Status().Update(ctx, pool); err != nil {
			return result, errors.WithStack(err)
		}
	}

	// 确保 lb 列表可用
	if err := r.ensureLb(ctx, pool); err != nil {
		return result, errors.WithStack(err)
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
