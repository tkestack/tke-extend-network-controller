package kube

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//	func RemovePodAnnotation(ctx context.Context, pod *corev1.Pod, name string) error {
//		return update(ctx, pod, func() {
//			if pod.Annotations == nil {
//				return
//			}
//			delete(pod.Annotations, name)
//		}, false, false)
//	}
func SetPodAnnotation(ctx context.Context, pod *corev1.Pod, name, value string) error {
	return update(ctx, pod, func() {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[name] = value
	}, false, false)
}

func PatchPodLastUpdateTime(ctx context.Context, obj client.Object) error {
	pod := &corev1.Pod{}
	if err := apiReader.Get(ctx, client.ObjectKeyFromObject(obj), pod); err != nil {
		return errors.WithStack(err)
	}
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[constant.LastUpdateTime] = time.Now().String()
	return apiClient.Update(ctx, pod)
}

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
