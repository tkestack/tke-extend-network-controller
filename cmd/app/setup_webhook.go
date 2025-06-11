package app

import (
	"os"

	webhook "github.com/imroc/tke-extend-network-controller/internal/webhook/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhooks(mgr ctrl.Manager) {
	if err := webhook.SetupCLBPortPoolWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CLBPortPool")
		os.Exit(1)
	}
}
