package app

import (
	"os"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/clbbinding"
	"github.com/imroc/tke-extend-network-controller/internal/controller"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupControllers(mgr ctrl.Manager) {
	// DedicatedCLBService controller
	if err := (&controller.DedicatedCLBServiceReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		APIReader: mgr.GetAPIReader(),
		Recorder:  mgr.GetEventRecorderFor("dedicatedclbservice-controller"),
	}).SetupWithManager(mgr, util.GetWorkerCount("WORKER_DEDICATED_CLB_SERVICE_CONTROLLER")); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DedicatedCLBService")
		os.Exit(1)
	}

	// DedicatedCLBListener controller and index and webhook
	if err := (&controller.DedicatedCLBListenerReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		APIReader: mgr.GetAPIReader(),
		Recorder:  mgr.GetEventRecorderFor("dedicatedclblistener-controller"),
	}).SetupWithManager(mgr, util.GetWorkerCount("WORKER_DEDICATED_CLB_LISTENER_CONTROLLER")); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CLBListenerReconciler")
		os.Exit(1)
	}
	indexer := mgr.GetFieldIndexer()
	if err := networkingv1alpha1.IndexFieldForDedicatedCLBListener(indexer); err != nil {
		setupLog.Error(err, "unable to index DedicatedCLBListener")
		os.Exit(1)
	}

	// CLBPortPool cotroller
	if err := (&controller.CLBPortPoolReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("clbportpool-controller"),
	}).SetupWithManager(mgr, util.GetWorkerCount("WORKER_CLB_PORT_POOL_CONTROLLER")); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CLBPortPool")
		os.Exit(1)
	}

	// CLBPodBinding cotroller
	if err := (&controller.CLBPodBindingReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("clbpodbinding-controller"),
	}).SetupWithManager(mgr, util.GetWorkerCount("WORKER_CLB_POD_BINDING_CONTROLLER")); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CLBPodBinding")
		os.Exit(1)
	}

	// CLNodeBinding cotroller
	if err := (&controller.CLBNodeBindingReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("clbnodebinding-controller"),
	}).SetupWithManager(mgr, util.GetWorkerCount("WORKER_CLB_NODE_BINDING_CONTROLLER")); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CLBNodeBinding")
		os.Exit(1)
	}

	// Pod controller
	if err := (&controller.PodReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("pod-controller"),
		CLBBindingReconciler: controller.CLBBindingReconciler[*clbbinding.CLBPodBinding]{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Recorder: mgr.GetEventRecorderFor("pod-controller"),
		},
	}).SetupWithManager(mgr, util.GetWorkerCount("WORKER_POD_CONTROLLER")); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pod")
		os.Exit(1)
	}

	// Node controller
	if err := (&controller.NodeReconciler{
		CLBBindingReconciler: controller.CLBBindingReconciler[*clbbinding.CLBNodeBinding]{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Recorder: mgr.GetEventRecorderFor("node-controller"),
		},
	}).SetupWithManager(mgr, util.GetWorkerCount("WORKER_NODE_CONTROLLER")); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Node")
		os.Exit(1)
	}

	// GameServerSet controller
	if err := (&controller.GameServerSetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GameServerSet")
		os.Exit(1)
	}
}
