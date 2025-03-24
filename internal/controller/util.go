package controller

import (
	"context"

	"github.com/imroc/tke-extend-network-controller/internal/constant"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Reconcile[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, sync func(ctx context.Context, obj T) (ctrl.Result, error)) (ctrl.Result, error) {
	if err := apiClient.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	result, err := sync(ctx, obj)
	if err != nil {
		if apierrors.IsConflict(err) {
			if !result.Requeue && result.RequeueAfter == 0 {
				result.Requeue = true
			}
			return result, nil
		}
	}
	return result, nil
}

func ReconcileWithFinalizer[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, syncFunc func(ctx context.Context, obj T) (ctrl.Result, error), cleanupFunc func(ctx context.Context, obj T) (ctrl.Result, error)) (ctrl.Result, error) {
	return Reconcile(ctx, req, apiClient, obj, func(ctx context.Context, obj T) (ctrl.Result, error) {
		if obj.GetDeletionTimestamp().IsZero() { // 没有删除
			// 确保 finalizer 存在，阻塞资源删除
			if !controllerutil.ContainsFinalizer(obj, constant.Finalizer) {
				controllerutil.AddFinalizer(obj, constant.Finalizer)
				if err := apiClient.Update(ctx, obj); err != nil {
					return ctrl.Result{}, errors.WithStack(err)
				}
			}
			// 执行同步函数
			result, err := syncFunc(ctx, obj)
			if err != nil {
				return result, errors.WithStack(err)
			}
			return result, nil
		} else { // 正在删除
			// 执行清理函数
			result, err := cleanupFunc(ctx, obj)
			if err != nil {
				return result, errors.WithStack(err)
			}
			// 移除 finalizer，让资源最终被删除
			controllerutil.RemoveFinalizer(obj, constant.Finalizer)
			if err = apiClient.Update(ctx, obj); err != nil {
				return result, errors.WithStack(err)
			}
			return result, nil
		}
	})
}

func ReconcilePodWithFinalizer[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, syncFunc func(ctx context.Context, obj T) (ctrl.Result, error), cleanFunc func(ctx context.Context, obj T) (ctrl.Result, error)) (ctrl.Result, error) {
	return Reconcile(ctx, req, apiClient, obj, func(ctx context.Context, obj T) (ctrl.Result, error) {
		if obj.GetDeletionTimestamp().IsZero() { // 没有删除
			// 确保 finalizer 存在
			if !controllerutil.ContainsFinalizer(obj, constant.Finalizer) {
				if err := kube.AddPodFinalizer(ctx, obj, constant.Finalizer); err != nil {
					return ctrl.Result{}, errors.WithStack(err)
				}
			}
			result, err := syncFunc(ctx, obj)
			if err != nil {
				return result, errors.WithStack(err)
			}
			return result, nil
		} else { // 正在删除
			result, err := cleanFunc(ctx, obj)
			if err != nil {
				return result, errors.WithStack(err)
			}
			// 移除 finalizer
			if err = kube.RemovePodFinalizer(ctx, obj, constant.Finalizer); err != nil {
				return result, errors.WithStack(err)
			}
			return result, nil
		}
	})
}
