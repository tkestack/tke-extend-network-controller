package v1alpha1

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	backendPodNameField = "spec.backendPod.podName"
	lbPortField         = "spec.lbPort"
	lbIdField           = "spec.lbId"
	protocolField       = "spec.protocol"
)

func indexFieldForDedicatedCLBListener(indexer client.FieldIndexer) {
	indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, backendPodNameField,
		func(o client.Object) []string {
			backendPod := o.(*DedicatedCLBListener).Spec.BackendPod
			if backendPod != nil {
				return []string{backendPod.PodName}
			}
			return []string{""}
		},
	)
	indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, lbPortField,
		func(o client.Object) []string {
			lbPort := o.(*DedicatedCLBListener).Spec.LbPort
			return []string{strconv.Itoa(int(lbPort))}
		},
	)
	indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, protocolField,
		func(o client.Object) []string {
			protocol := o.(*DedicatedCLBListener).Spec.Protocol
			return []string{protocol}
		},
	)
	indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, lbIdField,
		func(o client.Object) []string {
			lbId := o.(*DedicatedCLBListener).Spec.LbId
			return []string{lbId}
		},
	)
}

func FindDedicatedCLBListenerByBackendPod(ctx context.Context, pod *corev1.Pod, port int64, protocol string) ([]DedicatedCLBListener, error) {
	list := &DedicatedCLBListenerList{}
	if err := apiClient.List(
		ctx, list, client.InNamespace(pod.Namespace),
		client.MatchingFields{
			backendPodNameField: pod.Name,
			protocolField:       protocol,
			lbPortField:         strconv.Itoa(int(port)),
		},
	); err != nil {
		return nil, err
	}
	return list.Items, nil
}
