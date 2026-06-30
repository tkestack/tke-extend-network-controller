package util

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const MachineLabel = "node.tke.cloud.tencent.com/machine"

var machineGVK = schema.GroupVersionKind{
	Group:   "node.tke.cloud.tencent.com",
	Version: "v1beta1",
	Kind:    "Machine",
}

// 判断节点类型是否支持
func IsNodeTypeSupported(ctx context.Context, c client.Client, node *corev1.Node) bool {
	return IsServerlessNode(node) || IsNativeNode(ctx, c, node)
}

// 判断是否是serverless节点
func IsServerlessNode(node *corev1.Node) bool {
	return node.Labels["node.kubernetes.io/instance-type"] == "eklet"
}

// 判断是否是原生节点
func IsNativeNode(ctx context.Context, c client.Client, node *corev1.Node) bool {
	providerID := node.Spec.ProviderID
	// 传统原生节点: kn- 前缀
	if strings.HasPrefix(providerID, "tencentcloud://kn-") {
		return true
	}
	// CVM 加成的原生节点: ins- 前缀，需通过 Machine CR 确认
	if !strings.HasPrefix(providerID, "tencentcloud://ins-") {
		return false
	}
	machineName, ok := node.Labels[MachineLabel]
	if !ok || machineName == "" {
		return false
	}
	machine := &unstructured.Unstructured{}
	machine.SetGroupVersionKind(machineGVK)
	if err := c.Get(ctx, client.ObjectKey{Name: machineName}, machine); err != nil {
		if !apierrors.IsNotFound(err) {
			log.FromContext(ctx).Error(err, "failed to get machine for native node check", "machine", machineName, "node", node.Name)
		}
		return false
	}
	nodeRef, found, err := unstructured.NestedString(machine.Object, "status", "nodeRef", "name")
	if !found || err != nil {
		return false
	}
	return nodeRef == node.Name
}
