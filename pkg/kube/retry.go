package kube

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Update(ctx context.Context, obj client.Object, mutate func()) error {
	return update(ctx, obj, mutate, false)
}

func UpdateStatus(ctx context.Context, obj client.Object, mutate func()) error {
	return update(ctx, obj, mutate, true)
}

func update(ctx context.Context, obj client.Object, mutate func(), updateStatus bool) error {
	return wait.ExponentialBackoff(retry.DefaultBackoff, func() (done bool, err error) {
		key := client.ObjectKeyFromObject(obj)
		err = apiClient.Get(ctx, key, obj)
		if err != nil {
			return false, err
		}
		mutate()
		if updateStatus {
			err = apiClient.Status().Update(ctx, obj)
		} else {
			err = apiClient.Update(ctx, obj)
		}
		if err == nil {
			return true, nil // 正常更新，直接返回
		}
		if apierrors.IsConflict(err) || apierrors.IsTooManyRequests(err) {
			return false, nil // 只针对冲突和限流错误进行重试
		}
		return false, err // 其它错误不重试
	})
}
