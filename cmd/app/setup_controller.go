package app

import (
	"os"

	"github.com/tkestack/tke-extend-network-controller/internal/clbbinding"
	"github.com/tkestack/tke-extend-network-controller/internal/controller"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupControllers(mgr ctrl.Manager) {
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
	if clusterinfo.OKGSupported {
		if err := (&controller.GameServerSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "GameServerSet")
			os.Exit(1)
		}
	}
}
