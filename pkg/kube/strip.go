package kube

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StripPodUnusedFields is the transform function for shared pod informers,
// it removes unused fields from objects before they are stored in the cache to save memory.
func StripPodUnusedFields(obj any) (any, error) {
	pod := obj.(*corev1.Pod)
	if pod == nil {
		return obj, nil
	}
	containers := []corev1.Container{}
	for _, c := range pod.Spec.Containers {
		if len(c.Ports) > 0 {
			containers = append(containers, corev1.Container{
				Ports: c.Ports,
			})
		}
	}

	pod.Spec = corev1.PodSpec{
		Containers:  containers,
		NodeName:    pod.Spec.NodeName,
		HostNetwork: pod.Spec.HostNetwork,
	}

	pod.Status = corev1.PodStatus{
		Phase:      pod.Status.Phase,
		Conditions: pod.Status.Conditions,
		PodIP:      pod.Status.PodIP,
		Reason:     pod.Status.Reason,
		Message:    pod.Status.Message,
		PodIPs:     pod.Status.PodIPs,
	}

	pod.ObjectMeta = metav1.ObjectMeta{
		UID:               pod.UID,
		Namespace:         pod.Namespace,
		Name:              pod.Name,
		DeletionTimestamp: pod.DeletionTimestamp,
		Finalizers:        pod.Finalizers,
		ResourceVersion:   pod.ResourceVersion,
		Labels:            pod.Labels,
		Annotations:       pod.Annotations,
		OwnerReferences:   pod.OwnerReferences,
	}

	return obj, nil
}

func StripNodeUnusedFields(obj any) (any, error) {
	node := obj.(*corev1.Node)
	if node == nil {
		return obj, nil
	}

	node.ObjectMeta = metav1.ObjectMeta{
		Namespace:         node.Namespace,
		Name:              node.Name,
		DeletionTimestamp: node.DeletionTimestamp,
		ResourceVersion:   node.ResourceVersion,
		Labels:            node.Labels,
	}

	node.Spec = corev1.NodeSpec{
		ProviderID: node.Spec.ProviderID,
	}
	return obj, nil
}
