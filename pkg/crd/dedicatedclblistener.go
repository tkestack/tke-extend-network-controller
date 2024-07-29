package crd

import (
	"context"
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime/schema"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
		context.TODO(), &networkingv1alpha1.DedicatedCLBListener{}, backendPodNameField,
		func(o client.Object) []string {
			backendPod := o.(*networkingv1alpha1.DedicatedCLBListener).Spec.BackendPod
			if backendPod != nil {
				return []string{backendPod.PodName}
			}
			return []string{""}
		},
	)
	indexer.IndexField(
		context.TODO(), &networkingv1alpha1.DedicatedCLBListener{}, lbPortField,
		func(o client.Object) []string {
			lbPort := o.(*networkingv1alpha1.DedicatedCLBListener).Spec.LbPort
			return []string{strconv.Itoa(int(lbPort))}
		},
	)
	indexer.IndexField(
		context.TODO(), &networkingv1alpha1.DedicatedCLBListener{}, protocolField,
		func(o client.Object) []string {
			protocol := o.(*networkingv1alpha1.DedicatedCLBListener).Spec.Protocol
			return []string{protocol}
		},
	)
	indexer.IndexField(
		context.TODO(), &networkingv1alpha1.DedicatedCLBListener{}, lbIdField,
		func(o client.Object) []string {
			lbId := o.(*networkingv1alpha1.DedicatedCLBListener).Spec.LbId
			return []string{lbId}
		},
	)
}

func ValidateLbPort(lis *networkingv1alpha1.DedicatedCLBListener) error {
	list := &networkingv1alpha1.DedicatedCLBListenerList{}
	err := apiClient.List(
		context.Background(), list,
		client.MatchingFields{
			lbIdField:   lis.Spec.LbId,
			lbPortField: strconv.Itoa(int(lis.Spec.LbPort)),
		},
	)
	if err != nil {
		return err
	}
	if len(list.Items) > 0 {
		lbPortPath := field.NewPath("spec").Child("lbPort")
		var allErrs field.ErrorList
		allErrs = append(
			allErrs,
			field.Invalid(
				lbPortPath, lis.Spec.LbPort,
				fmt.Sprintf("lbPort is already used by other DedicatedCLBListener (%s/%s)", list.Items[0].Namespace, list.Items[0].Name),
			),
		)
		return apierrors.NewInvalid(
			schema.GroupKind{Group: "networking.cloud.tencent.com", Kind: "DedicatedCLBListener"},
			lis.Name,
			allErrs,
		)
	}
	return nil
}

func FindDedicatedCLBListenerByBackendPod(ctx context.Context, pod *corev1.Pod, port int64, protocol string) ([]networkingv1alpha1.DedicatedCLBListener, error) {
	list := &networkingv1alpha1.DedicatedCLBListenerList{}
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
