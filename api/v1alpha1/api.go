package v1alpha1

import (
	"os"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var apiClient client.Client

func Init(log logr.Logger, mgr ctrl.Manager) {
	apiClient = mgr.GetClient()
	indexer := mgr.GetFieldIndexer()
	if err := indexFieldForDedicatedCLBListener(indexer); err != nil {
		log.Error(err, "unable to index DedicatedCLBListener")
		os.Exit(1)
	}
	if err := (&DedicatedCLBListener{}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create webhook", "webhook", "DedicatedCLBListener")
		os.Exit(1)
	}
}
