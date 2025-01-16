package app

import (
	"os"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	networkingv1beta1 "github.com/imroc/tke-extend-network-controller/api/v1beta1"
	webhookv1beta1 "github.com/imroc/tke-extend-network-controller/internal/webhook/v1beta1"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupAPI(mgr ctrl.Manager) {
	networkingv1beta1.Init(setupLog, mgr)
	networkingv1alpha1.Init(setupLog, mgr)
	if err := webhookv1beta1.SetupDedicatedCLBListenerWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to setup webhook for DedicatedCLBListener v1beta1")
		os.Exit(1)
	}
	kube.Init(mgr)
}
