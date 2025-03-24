package controller

import (
	"context"

	"github.com/imroc/tke-extend-network-controller/internal/constant"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ReconcileWithResult[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, syncFunc func(ctx context.Context, obj T) (ctrl.Result, error)) (ctrl.Result, error) {
	if err := apiClient.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	result, err := syncFunc(ctx, obj)
	if err != nil {
		if apierrors.IsConflict(err) {
			if !result.Requeue && result.RequeueAfter == 0 {
				result.Requeue = true
			}
			return result, nil
		}
		return result, errors.WithStack(err)
	}
	return result, nil
}

func Reconcile[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, syncFunc func(ctx context.Context, obj T) error) (ctrl.Result, error) {
	if err := apiClient.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err := syncFunc(ctx, obj); err != nil {
		if apierrors.IsConflict(err) || err == portpool.ErrLBCreated { // 资源冲突，或者分配端口时遇到端口不足新创建了 CLB，重新入队
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, errors.WithStack(err)
	}
	return ctrl.Result{}, nil
}

func ReconcileWithFinalizer[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, syncFunc func(ctx context.Context, obj T) error, cleanFunc func(ctx context.Context, obj T) error) (ctrl.Result, error) {
	return Reconcile(ctx, req, apiClient, obj, func(ctx context.Context, obj T) error {
		if obj.GetDeletionTimestamp().IsZero() { // 没有删除
			// 确保 finalizer 存在
			if !controllerutil.ContainsFinalizer(obj, constant.Finalizer) {
				controllerutil.AddFinalizer(obj, constant.Finalizer)
				if err := apiClient.Update(ctx, obj); err != nil {
					return errors.WithStack(err)
				}
			}
			if err := syncFunc(ctx, obj); err != nil {
				return errors.WithStack(err)
			}
		} else { // 正在删除
			if err := cleanFunc(ctx, obj); err != nil {
				return errors.WithStack(err)
			}
			// 移除 finalizer
			controllerutil.RemoveFinalizer(obj, constant.Finalizer)
			if err := apiClient.Update(ctx, obj); err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	})
}

func ReconcilePodWithFinalizer[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, syncFunc func(ctx context.Context, obj T) error, cleanFunc func(ctx context.Context, obj T) error) (ctrl.Result, error) {
	return Reconcile(ctx, req, apiClient, obj, func(ctx context.Context, obj T) error {
		if obj.GetDeletionTimestamp().IsZero() { // 没有删除
			// 确保 finalizer 存在
			if !controllerutil.ContainsFinalizer(obj, constant.Finalizer) {
				if err := kube.AddPodFinalizer(ctx, obj, constant.Finalizer); err != nil {
					return errors.WithStack(err)
				}
			}
			if err := syncFunc(ctx, obj); err != nil {
				return errors.WithStack(err)
			}
		} else { // 正在删除
			if err := cleanFunc(ctx, obj); err != nil {
				return errors.WithStack(err)
			}
			// 移除 finalizer
			if err := kube.RemovePodFinalizer(ctx, obj, constant.Finalizer); err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	})
}
