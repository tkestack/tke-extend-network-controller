package v1beta1

import (
	"context"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	backendPodNameField = "spec.targetPod.podName"
	lbPortField         = "spec.port"
	lbIdField           = "spec.lbId"
)

func indexFieldForDedicatedCLBListener(indexer client.FieldIndexer) error {
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
		context.TODO(), &DedicatedCLBListener{}, lbPortField,
		func(o client.Object) []string {
			lbPort := o.(*DedicatedCLBListener).Spec.Port
			return []string{strconv.Itoa(int(lbPort))}
		},
	); err != nil {
		return err
	}

	if err := indexer.IndexField(
		context.TODO(), &DedicatedCLBListener{}, lbIdField,
		func(o client.Object) []string {
			clbs := o.(*DedicatedCLBListener).Spec.CLBs
			ids := make([]string, len(clbs))
			for i, clb := range clbs {
				ids[i] = clb.LbId
			}
			return ids
		},
	); err != nil {
		return err
	}
	return nil
}
