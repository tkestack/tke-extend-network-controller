package app

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StripPodUnusedFields is the transform function for shared pod informers,
// it removes unused fields from objects before they are stored in the cache to save memory.
func StripPodUnusedFields(obj any) (any, error) {
	t, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		// shouldn't happen
		return obj, nil
	}
	// ManagedFields is large and we never use it
	t.GetObjectMeta().SetManagedFields(nil)
	// only container ports can be used
	if pod := obj.(*corev1.Pod); pod != nil {
		containers := []corev1.Container{}
		for _, c := range pod.Spec.Containers {
			if len(c.Ports) > 0 {
				containers = append(containers, corev1.Container{
					Ports: c.Ports,
				})
			}
		}
		oldSpec := pod.Spec
		newSpec := corev1.PodSpec{
			Containers:  containers,
			NodeName:    oldSpec.NodeName,
			HostNetwork: oldSpec.HostNetwork,
		}
		pod.Spec = newSpec
		pod.Status = corev1.PodStatus{
			Phase:      pod.Status.Phase,
			Conditions: pod.Status.Conditions,
			PodIP:      pod.Status.PodIP,
			Reason:     pod.Status.Reason,
			Message:    pod.Status.Message,
			PodIPs:     pod.Status.PodIPs,
		}
		var newMeta metav1.ObjectMeta
		newMeta.Namespace = pod.Namespace
		newMeta.Name = pod.Name
		pod.ObjectMeta = newMeta
	}

	return obj, nil
}
