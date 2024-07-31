package kube

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	apiClient client.Client
	apiReader client.Reader
)

func Init(mgr ctrl.Manager) {
	apiClient = mgr.GetClient()
	apiReader = mgr.GetAPIReader()
}
