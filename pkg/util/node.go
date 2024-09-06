package util

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// 判断节点类型是否支持
func IsNodeTypeSupported(node *corev1.Node) bool {
	return IsServerlessNode(node) || IsNativeNode(node)
}

// 判断是否是serverless节点
func IsServerlessNode(node *corev1.Node) bool {
	return node.Labels["node.kubernetes.io/instance-type"] == "eklet"
}

// 判断是否是原生节点
func IsNativeNode(node *corev1.Node) bool {
	return strings.HasPrefix(node.Spec.ProviderID, "tencentcloud://kn-")
}
