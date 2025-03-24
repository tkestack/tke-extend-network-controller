package v1alpha1

import (
	"context"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	backendPodNameField = "spec.backendPod.podName"
	backendPortField    = "spec.backendPod.port"
	lbPortField         = "spec.lbPort"
	lbIdField           = "spec.lbId"
	protocolField       = "spec.protocol"
	stateField          = "status.state"
)

func IndexFieldForDedicatedCLBListener(indexer client.FieldIndexer) error {
	if err := indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, backendPodNameField,
		func(o client.Object) []string {
			targetPod := o.(*DedicatedCLBListener).Spec.TargetPod
			if targetPod != nil {
				return []string{targetPod.PodName}
			}
			return nil
		},
	); err != nil {
		return err
	}
	if err := indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, backendPortField,
		func(o client.Object) []string {
			targetPod := o.(*DedicatedCLBListener).Spec.TargetPod
			if targetPod != nil {
				return []string{strconv.Itoa(int(targetPod.TargetPort))}
			}
			return nil
		},
	); err != nil {
		return err
	}
	if err := indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, lbPortField,
		func(o client.Object) []string {
			lbPort := o.(*DedicatedCLBListener).Spec.LbPort
			return []string{strconv.Itoa(int(lbPort))}
		},
	); err != nil {
		return err
	}
	if err := indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, protocolField,
		func(o client.Object) []string {
			protocol := o.(*DedicatedCLBListener).Spec.Protocol
			return []string{protocol}
		},
	); err != nil {
		return err
	}

	if err := indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, lbIdField,
		func(o client.Object) []string {
			lbId := o.(*DedicatedCLBListener).Spec.LbId
			return []string{lbId}
		},
	); err != nil {
		return err
	}

	if err := indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, stateField,
		func(o client.Object) []string {
			state := o.(*DedicatedCLBListener).Status.State
			return []string{state}
		},
	); err != nil {
		return err
	}
	return nil
}
