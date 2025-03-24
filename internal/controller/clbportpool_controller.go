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
	"reflect"

	"github.com/pkg/errors"
	clbsdk "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
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

func GetNotifyCreateLoadBalancerFunc(apiClient client.Client, poolName string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		pool := &networkingv1alpha1.CLBPortPool{}
		if err := apiClient.Get(context.Background(), client.ObjectKey{Name: poolName}, pool); err != nil {
			return err
		}
		if !pool.CanCreateLB() || pool.Status.State == networkingv1alpha1.CLBPortPoolStateScaling { // 忽略未启用自动创建或正在扩容的端口池
			return nil
		}
		pool.Status.State = networkingv1alpha1.CLBPortPoolStateScaling
		if err := apiClient.Status().Update(ctx, pool); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
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

func (r *CLBPortPoolReconciler) ensureExistedLB(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	// 对账 clb
	lbIds := make(map[string]struct{})
	for _, lbStatus := range pool.Status.LoadbalancerStatuses {
		lbIds[lbStatus.LoadbalancerID] = struct{}{}
	}
	lbToAdd := []networkingv1alpha1.LoadBalancerStatus{}
	// 确保所有已有 CLB 在 status 中存在
	for _, lbId := range pool.Spec.ExsistedLoadBalancerIDs {
		// 如果在 status 中不存在，则准备校验并加进端口池
		if _, exists := lbIds[lbId]; !exists { // TODO: 当前是 status 中如果不存在，只有首次查 clb 然后写入 status，后续改成每次对账（lb 信息变动可同步过来）
			// 校验 CLB 是否存在
			var lb *clbsdk.LoadBalancer
			lb, err := clb.GetClb(ctx, lbId, util.GetRegionFromPtr(pool.Spec.Region))
			if err != nil {
				return errors.WithStack(err)
			}
			// CLB 存在，准备加进来
			status := networkingv1alpha1.LoadBalancerStatus{
				LoadbalancerID:   lbId,
				LoadbalancerName: *lb.LoadBalancerName,
			}
			if len(lb.LoadBalancerVips) > 0 {
				status.Ips = util.ConvertPtrSlice(lb.LoadBalancerVips)
			}
			if lb.Domain != nil && *lb.Domain != "" {
				status.Hostname = lb.Domain
			}
			lbToAdd = append(lbToAdd, status)
		}
	}
	// 如果有需要加进来的，加到 status 中，并更新缓存
	if len(lbToAdd) > 0 {
		pool.Status.LoadbalancerStatuses = append(pool.Status.LoadbalancerStatuses, lbToAdd...)
		if err := r.Status().Update(ctx, pool); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (r *CLBPortPoolReconciler) ensureLbInAllocator(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	// 同步lbId列表到端口池
	allLbIds := []string{}
	for _, lbStatus := range pool.Status.LoadbalancerStatuses {
		allLbIds = append(allLbIds, lbStatus.LoadbalancerID)
	}
	if err := portpool.Allocator.EnsureLbIds(pool.Name, allLbIds); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CLBPortPoolReconciler) ensureLbStatus(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	needUpdate := false
	for i := range pool.Status.LoadbalancerStatuses {
		lbStatus := &pool.Status.LoadbalancerStatuses[i]
		lbId := lbStatus.LoadbalancerID
		lb, err := clb.GetClb(ctx, lbId, pool.GetRegion())
		if err != nil {
			return errors.WithStack(err)
		}
		ips := util.ConvertPtrSlice(lb.LoadBalancerVips)
		if !reflect.DeepEqual(ips, lbStatus.Ips) {
			lbStatus.Ips = ips
			needUpdate = true
		}
		if util.GetValue(lb.Domain) != util.GetValue(lbStatus.Hostname) {
			lbStatus.Hostname = lb.Domain
			needUpdate = true
		}
		if util.GetValue(lb.LoadBalancerName) != lbStatus.LoadbalancerName {
			lbStatus.LoadbalancerName = *lb.LoadBalancerName
			needUpdate = true
		}
	}
	if needUpdate {
		if err := r.Status().Update(ctx, pool); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (r *CLBPortPoolReconciler) ensureLb(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	// 确保已有 lb 被添加到 status 中
	if err := r.ensureExistedLB(ctx, pool); err != nil {
		return err
	}
	// 确保状态为 Active
	if err := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateActive); err != nil {
		return errors.WithStack(err)
	}
	// 确保所有 lb 都被添加到分配器的缓存中
	if err := r.ensureLbInAllocator(ctx, pool); err != nil {
		return err
	}
	// 同步 clb 信息到 status
	if err := r.ensureLbStatus(ctx, pool); err != nil {
		return err
	}
	return nil
}

// 扩容 CLB
func (r *CLBPortPoolReconciler) createLB(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	if !pool.CanCreateLB() { // 触发自动创建 CLB，但当前却无法扩容，一般不可能发生，除非代码 BUG
		r.Recorder.Event(pool, corev1.EventTypeWarning, "CreateLoadBalancer", "auto create is not enabled while scale clb is been tiggered")
		if err := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateActive); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	// 创建 CLB
	lbId, err := clb.CreateCLB(ctx, pool.GetRegion(), pool.Spec.AutoCreate.Parameters.ExportCreateLoadBalancerRequest())
	if err != nil {
		r.Recorder.Eventf(pool, corev1.EventTypeWarning, "CreateLoadBalancer", "create clb failed: %s", err.Error())
		return errors.WithStack(err)
	}
	r.Recorder.Eventf(pool, corev1.EventTypeNormal, "CreateLoadBalancer", "create clb success: %s", lbId)
	addLbId := func() error {
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
	if err := util.RetryIfPossible(addLbId); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CLBPortPoolReconciler) ensureAllocatorCache(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) error {
	if !portpool.Allocator.IsPoolExists(pool.Name) { // 分配器缓存中不存在，则添加
		if err := portpool.Allocator.AddPool(
			pool.Name,
			pool.GetRegion(),
			pool.Spec.StartPort,
			pool.Spec.EndPort,
			pool.Spec.SegmentLength,
			GetNotifyCreateLoadBalancerFunc(r.Client, pool.Name),
		); err != nil {
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
func (r *CLBPortPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBPortPool{}).
		Named("clbportpool").
		Complete(r)
}
