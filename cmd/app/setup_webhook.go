package app

import (
	"os"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	webhook "github.com/imroc/tke-extend-network-controller/internal/webhook/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhooks(mgr ctrl.Manager) {
	if err := webhook.SetupCLBPortPoolWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CLBPortPool")
		os.Exit(1)
	}
	if err := (&networkingv1alpha1.DedicatedCLBListener{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "DedicatedCLBListener")
		os.Exit(1)
	}
}
