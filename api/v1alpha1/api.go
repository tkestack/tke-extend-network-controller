package v1alpha1

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var apiClient client.Client

func Init(mgr ctrl.Manager) {
	apiClient = mgr.GetClient()
}
