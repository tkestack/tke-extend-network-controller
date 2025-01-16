package v1beta1

import (
	"os"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Init(log logr.Logger, mgr ctrl.Manager) {
	indexer := mgr.GetFieldIndexer()
	if err := indexFieldForDedicatedCLBListener(indexer); err != nil {
		log.Error(err, "unable to index DedicatedCLBListener")
		os.Exit(1)
	}
}
