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

	clbsdk "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	Scheme *runtime.Scheme
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
func (r *CLBPortPoolReconciler) cleanup(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) (err error) {
	portpool.Allocator.RemovePool(pool.Name)
	return
}

func GetCreateLoadBalancerFunc(apiClient client.Client, poolName string) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		pool := &networkingv1alpha1.CLBPortPool{}
		if err := apiClient.Get(context.Background(), client.ObjectKey{Name: poolName}, pool); err != nil {
			return "", err
		}

		if util.IsValueEqual(pool.Status.State, "Active") && pool.Spec.AutoCreate != nil && pool.Spec.AutoCreate.Enabled { // 启用了 CLB 自动创建
			if !util.IsZero(pool.Spec.AutoCreate.MaxLoadBalancers) { // 限制了最大创建数量
				// 检查是否已创建了足够的 CLB
				num := uint16(0)
				for _, lbStatus := range pool.Status.LoadbalancerStatuses {
					if lbStatus.AutoCreated != nil && *lbStatus.AutoCreated {
						num++
					}
				}
				// 如果已创建数量已满，则直接返回
				if num >= *pool.Spec.AutoCreate.MaxLoadBalancers {
					return "", nil
				}
			}
			lbId, err := clb.CreateCLB(ctx, util.GetRegionFromPtr(pool.Spec.Region), pool.Spec.AutoCreate.Parameters)
			if err != nil {
				return "", err
			}
			pool.Status.LoadbalancerStatuses = append(pool.Status.LoadbalancerStatuses, networkingv1alpha1.LoadBalancerStatus{
				LoadbalancerID: lbId,
				AutoCreated:    util.GetPtr(true),
			})
			if err := apiClient.Status().Update(ctx, pool); err != nil {
				return "", err
			}
			return lbId, nil
		} else { // 没启用自动创建，直接返回
			return "", nil
		}
	}
}

// 同步端口池
func (r *CLBPortPoolReconciler) sync(ctx context.Context, pool *networkingv1alpha1.CLBPortPool) (err error) {
	defer func() {
		if err != nil && !apierrors.IsConflict(err) {
			pool.Status.State = util.GetPtr("Error")
			pool.Status.Message = util.GetPtr(err.Error())
			if e := r.Status().Update(ctx, pool); e != nil {
				err = e
				return
			}
		} else {
			// 确保分配器缓存中存在该 port pool
			if !portpool.Allocator.IsPoolExists(pool.Name) { // 分配器缓存中不存在，则添加
				if e := portpool.Allocator.AddPool(
					pool.Name,
					util.GetRegionFromPtr(pool.Spec.Region),
					pool.Spec.StartPort,
					pool.Spec.EndPort,
					pool.Spec.SegmentLength,
					GetCreateLoadBalancerFunc(r.Client, pool.Name),
				); e != nil {
					err = e
					return
				}
			}
		}
	}()
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
			lb, err = clb.GetClb(ctx, lbId, util.GetRegionFromPtr(pool.Spec.Region))
			if err != nil {
				return
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
		if err = r.Status().Update(ctx, pool); err != nil {
			return
		}
	}

	// 同步lbId列表到端口池
	allLbIds := []string{}
	for _, lbStatus := range pool.Status.LoadbalancerStatuses {
		allLbIds = append(allLbIds, lbStatus.LoadbalancerID)
	}
	portpool.Allocator.EnsureLbIds(pool.Name, allLbIds)

	// 确保状态为 Active
	if pool.Status.State == nil || *pool.Status.State != "Active" {
		pool.Status.State = util.GetPtr("Active")
		pool.Status.Message = nil
		if err := r.Status().Update(ctx, pool); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CLBPortPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.CLBPortPool{}).
		Named("clbportpool").
		Complete(r)
}
