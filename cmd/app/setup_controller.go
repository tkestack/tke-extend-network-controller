package app

import (
	"os"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/clbbinding"
	"github.com/imroc/tke-extend-network-controller/internal/controller"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupControllers(mgr ctrl.Manager, workers int) {
	// DedicatedCLBService controller
	if err := (&controller.DedicatedCLBServiceReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		APIReader: mgr.GetAPIReader(),
		Recorder:  mgr.GetEventRecorderFor("dedicatedclbservice-controller"),
	}).SetupWithManager(mgr, workers); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DedicatedCLBService")
		os.Exit(1)
	}

	// DedicatedCLBListener controller and index and webhook
	if err := (&controller.DedicatedCLBListenerReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		APIReader: mgr.GetAPIReader(),
		Recorder:  mgr.GetEventRecorderFor("dedicatedclblistener-controller"),
	}).SetupWithManager(mgr, workers); err != nil {
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
	}).SetupWithManager(mgr, workers); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CLBPortPool")
		os.Exit(1)
	}

	// CLBPodBinding cotroller
	if err := (&controller.CLBPodBindingReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("clbpodbinding-controller"),
	}).SetupWithManager(mgr, workers); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CLBPodBinding")
		os.Exit(1)
	}

	// CLNodeBinding cotroller
	if err := (&controller.CLBNodeBindingReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("clbnodebinding-controller"),
	}).SetupWithManager(mgr); err != nil {
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
	}).SetupWithManager(mgr, workers); err != nil {
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
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Node")
		os.Exit(1)
	}
}
