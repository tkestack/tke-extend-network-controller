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

	portpoolutil "github.com/imroc/tke-extend-network-controller/internal/portpool/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CLBPortPool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
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

type lbInfoKeyType int

const lbInfoKey lbInfoKeyType = iota

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

func (r *CLBPortPoolReconciler) ensureExistedLB(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
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
	lbInfos := getCLBInfoFromContext(ctx)
	log.FromContext(ctx).V(10).Info("getCLBInfoFromContext", "lbInfos", lbInfos)
	if lbInfos == nil {
		return nil
	}
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

func (r *CLBPortPoolReconciler) ensureLbInAllocator(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	// 同步lbId列表到端口池
	allLbIds := []string{}
	for _, lbStatus := range pool.Status.LoadbalancerStatuses {
		if lbStatus.State != networkingv1alpha1.LoadBalancerStateNotFound {
			allLbIds = append(allLbIds, lbStatus.LoadbalancerID)
		}
	}
	if err := portpool.Allocator.EnsureLbIds(pool.Name, allLbIds); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CLBPortPoolReconciler) ensureLbStatus(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	lbInfos := getCLBInfoFromContext(ctx)
	needUpdate := false
	for i := range pool.Status.LoadbalancerStatuses {
		lbStatus := &pool.Status.LoadbalancerStatuses[i]
		lbId := lbStatus.LoadbalancerID
		// clb 不存在，通常是已删除，更新状态和端口池
		if info, ok := lbInfos[lbId]; !ok {
			if lbStatus.State != networkingv1alpha1.LoadBalancerStateNotFound {
				r.Recorder.Eventf(pool, corev1.EventTypeWarning, "GetLoadBalancer", "clb %s not found", lbId)
				portpool.Allocator.ReleaseLb(pool.Name, lbId)
				lbStatus.State = networkingv1alpha1.LoadBalancerStateNotFound
				needUpdate = true
			}
			continue
		} else {
			if lbStatus.State == "" {
				lbStatus.State = networkingv1alpha1.LoadBalancerStateRunning
				needUpdate = true
			}
			if !reflect.DeepEqual(info.Ips, lbStatus.Ips) {
				lbStatus.Ips = info.Ips
				needUpdate = true
			}
			if !reflect.DeepEqual(info.Hostname, lbStatus.Hostname) {
				lbStatus.Hostname = info.Hostname
				needUpdate = true
			}
			if !reflect.DeepEqual(info.LoadbalancerName, lbStatus.LoadbalancerName) {
				lbStatus.LoadbalancerName = info.LoadbalancerName
				needUpdate = true
			}
		}
	}
	if needUpdate {
		if err := r.Status().Update(ctx, pool); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func getCLBInfoFromContext(ctx context.Context) map[string]*clb.CLBInfo {
	info, ok := ctx.Value(lbInfoKey).(map[string]*clb.CLBInfo)
	if ok {
		return info
	}
	return make(map[string]*clb.CLBInfo)
}

func (r *CLBPortPoolReconciler) ensureLb(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	// 查询 lb 信息并保存到 context，以供后面的 ensureXXX 对账使用
	if info, err := r.getCLBInfo(ctx, pool); err != nil {
		return errors.WithStack(err)
	} else {
		ctx = context.WithValue(ctx, lbInfoKey, info)
		log.FromContext(ctx).V(10).Info("set lb info", "info", info)
	}
	// 确保已有 lb 被添加到 status 中
	if err := r.ensureExistedLB(ctx, pool); err != nil {
		return errors.WithStack(err)
	}
	// 同步 clb 信息到 status
	if err := r.ensureLbStatus(ctx, pool); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有 lb 都被添加到分配器的缓存中
	if err := r.ensureLbInAllocator(ctx, pool); err != nil {
		return errors.WithStack(err)
	}
	// lb 准备就绪，确保状态为 Active
	if err := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateActive); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// 扩容 CLB
func (r *CLBPortPoolReconciler) createLB(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	if !portpoolutil.CanCreateLB(ctx, pool) { // 触发自动创建 CLB，但当前却无法扩容，一般不可能发生，除非代码 BUG
		r.Recorder.Event(pool, corev1.EventTypeWarning, "CreateLoadBalancer", "not able to scale clb while scale is been tiggered")
		if err := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateActive); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	// 创建 CLB
	r.Recorder.Event(pool, corev1.EventTypeNormal, "CreateLoadBalancer", "try to create clb")
	lbId, err := clb.CreateCLB(ctx, pool.GetRegion(), clb.ConvertCreateLoadBalancerRequest(pool.Spec.AutoCreate.Parameters))
	if err != nil {
		r.Recorder.Eventf(pool, corev1.EventTypeWarning, "CreateLoadBalancer", "create clb failed: %s", err.Error())
		return errors.WithStack(err)
	}
	r.Recorder.Eventf(pool, corev1.EventTypeNormal, "CreateLoadBalancer", "create clb success: %s", lbId)
	if err := portpool.Allocator.AddLbId(pool.Name, lbId); err != nil {
		return errors.WithStack(err)
	}
	addLbIdToStatus := func() error {
		p := &networkingv1alpha1.CLBPortPool{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(pool), p); err != nil {
			return errors.WithStack(err)
		}
		p.Status.State = networkingv1alpha1.CLBPortPoolStateActive // 创建成功，状态改为 Active，以便再次可分配端口
		p.Status.LoadbalancerStatuses = append(p.Status.LoadbalancerStatuses, networkingv1alpha1.LoadBalancerStatus{
			LoadbalancerID: lbId,
			AutoCreated:    util.GetPtr(true),
		})
		if err := r.Status().Update(ctx, p); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	if err := util.RetryIfPossible(addLbIdToStatus); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CLBPortPoolReconciler) ensureAllocatorCache(_ context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	if !portpool.Allocator.IsPoolExists(pool.Name) { // 分配器缓存中不存在，则添加
		if err := portpool.Allocator.AddPool(portpoolutil.NewPortPool(pool, r.Client)); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// 同步端口池
func (r *CLBPortPoolReconciler) sync(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) (result ctrl.Result, err error) {
	// 初始化状态
	if pool.Status.State == "" {
		pool.Status.State = networkingv1alpha1.CLBPortPoolStatePending
		if err := r.Status().Update(ctx, pool); err != nil {
			return result, errors.WithStack(err)
		}
	}

	// 被通知扩容CLB时，执行扩容操作
	if pool.Status.State == networkingv1alpha1.CLBPortPoolStateScaling {
		if err := r.createLB(ctx, pool); err != nil { // 执行扩容
			return result, errors.WithStack(err)
		}
		// 扩容成功，重新对账
		result.Requeue = true
		return result, nil
	}

	// 确保分配器缓存中存在该 port pool
	if err := r.ensureAllocatorCache(ctx, pool); err != nil {
		return result, errors.WithStack(err)
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
