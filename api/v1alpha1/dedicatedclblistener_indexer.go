package v1alpha1

import (
	"context"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	backendPodNameField = "spec.backendPod.podName"
	lbPortField         = "spec.lbPort"
	lbIdField           = "spec.lbId"
	protocolField       = "spec.protocol"
	stateField          = "status.state"
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
	indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, stateField,
		func(o client.Object) []string {
			state := o.(*DedicatedCLBListener).Status.State
			return []string{state}
		},
	)
}
