package kube

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func AddPodFinalizer(ctx context.Context, pod client.Object, finalizerName string) error {
	return updatePodFinalizer(ctx, pod, finalizerName, true)
}

func RemovePodFinalizer(ctx context.Context, pod client.Object, finalizerName string) error {
	return updatePodFinalizer(ctx, pod, finalizerName, false)
}

func updatePodFinalizer(ctx context.Context, cachedPod client.Object, finalizerName string, add bool) error {
	pod := &corev1.Pod{}
	if err := apiReader.Get(ctx, client.ObjectKeyFromObject(cachedPod), pod); err != nil {
		return err
	}
	if add { // 添加 finalizer
		if controllerutil.AddFinalizer(pod, finalizerName) {
			if err := apiClient.Update(ctx, pod); err != nil {
				return err
			}
		}
	} else { // 删除 finalizer
		if controllerutil.RemoveFinalizer(pod, finalizerName) {
			if err := apiClient.Update(ctx, pod); err != nil {
				return err
			}
		}
	}
	return nil
}
