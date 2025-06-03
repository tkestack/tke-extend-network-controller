package controller

import (
	"context"
	"time"

	"github.com/imroc/tke-extend-network-controller/internal/constant"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ObjectWrapper interface {
	GetObject() client.Object
}

func Reconcile[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, sync func(ctx context.Context, obj T) (ctrl.Result, error)) (ctrl.Result, error) {
	if err := apiClient.Get(ctx, req.NamespacedName, obj); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			return ctrl.Result{}, errors.WithStack(err)
		} else {
			return ctrl.Result{}, nil
		}
	}
	result, err := sync(ctx, obj)
	if err != nil {
		if apierrors.IsConflict(err) {
			if !result.Requeue && result.RequeueAfter == 0 {
				log.FromContext(ctx).Info("requeue due to k8s api conflict", "err", err)
				result.RequeueAfter = 20 * time.Millisecond
			}
			return result, nil
		}
		return result, errors.WithStack(err)
	}
	return result, nil
}

func ReconcileWithFinalizer[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, syncFunc func(ctx context.Context, obj T) (ctrl.Result, error), cleanupFunc func(ctx context.Context, obj T) (ctrl.Result, error)) (ctrl.Result, error) {
	result, err := Reconcile(ctx, req, apiClient, obj, func(ctx context.Context, obj T) (ctrl.Result, error) {
		if obj.GetDeletionTimestamp().IsZero() { // 没有删除
			// 确保 finalizer 存在，阻塞资源删除
			if !controllerutil.ContainsFinalizer(obj, constant.Finalizer) {
				if controllerutil.AddFinalizer(obj, constant.Finalizer) {
					if err := apiClient.Update(ctx, obj); err != nil {
						return ctrl.Result{}, errors.WithStack(err)
					}
				}
			}
			// 执行同步函数
			result, err := syncFunc(ctx, obj)
			if err != nil {
				return result, errors.WithStack(err)
			}
			return result, nil
		} else { // 正在删除
			// 如果没有 finalizer，说明已经清理过了，直接返回避免重复清理
			if !controllerutil.ContainsFinalizer(obj, constant.Finalizer) {
				return ctrl.Result{}, nil
			}
			// 执行清理函数
			result, err := cleanupFunc(ctx, obj)
			if err != nil {
				return result, errors.WithStack(err)
			}
			if result.Requeue || result.RequeueAfter > 0 {
				return result, nil
			}
			// 移除 finalizer，让资源最终被删除
			if controllerutil.RemoveFinalizer(obj, constant.Finalizer) {
				if err = apiClient.Update(ctx, obj); err != nil {
					return result, errors.WithStack(err)
				}
			}
			return result, nil
		}
	})
	if err != nil {
		return result, errors.WithStack(err)
	}
	return result, nil
}

func ReconcilePodWithFinalizer[T client.Object](ctx context.Context, req ctrl.Request, apiClient client.Client, obj T, syncFunc func(ctx context.Context, obj T) (ctrl.Result, error), cleanFunc func(ctx context.Context, obj T) (ctrl.Result, error)) (ctrl.Result, error) {
	result, err := Reconcile(ctx, req, apiClient, obj, func(ctx context.Context, obj T) (ctrl.Result, error) {
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
	if err != nil {
		return result, errors.WithStack(err)
	}
	return result, nil
}
