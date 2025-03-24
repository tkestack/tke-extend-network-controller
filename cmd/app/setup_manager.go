package app

import (
	"context"
	"os"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/controller"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	builtinwebhook "github.com/imroc/tke-extend-network-controller/internal/webhook/v1"
	webhook "github.com/imroc/tke-extend-network-controller/internal/webhook/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var scheme = runtime.NewScheme()

func init() {
	// Add API to schema
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))     // k8s native resources
	utilruntime.Must(networkingv1alpha1.AddToScheme(scheme)) // tke-extend-network-controller CRDs
}

func SetupManager(mgr ctrl.Manager) {
	if err := mgr.Add(&initCache{mgr.GetClient()}); err != nil {
		setupLog.Error(err, "problem add init cache")
		os.Exit(1)
	}
	workers := viper.GetInt(workerCountFlag)
	if workers <= 0 {
		workers = 1
	}
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
	if err := (&networkingv1alpha1.DedicatedCLBListener{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "DedicatedCLBListener")
		os.Exit(1)
	}

	// CLBPortPool cotroller and webhook
	if err := (&controller.CLBPortPoolReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CLBPortPool")
		os.Exit(1)
	}
	if err := webhook.SetupCLBPortPoolWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CLBPortPool")
		os.Exit(1)
	}

	// CLBPodBinding cotroller and webhook
	if err := (&controller.CLBPodBindingReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CLBPodBinding")
		os.Exit(1)
	}
	if err := webhook.SetupCLBPodBindingWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CLBPodBinding")
		os.Exit(1)
	}

	// Pod controller and webhook
	if err := (&controller.PodReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pod")
		os.Exit(1)
	}
	if err := builtinwebhook.SetupPodWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Pod")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
}

type initCache struct {
	client.Client
}

func (i *initCache) NeedLeaderElection() bool {
	return true
}

func (i *initCache) Start(ctx context.Context) error {
	setupLog.Info("starting init cache")
	defer setupLog.Info("end init cache")
	// 初始化端口池
	ppl := &networkingv1alpha1.CLBPortPoolList{}
	if err := i.List(ctx, ppl); err != nil {
		return err
	}
	for _, pp := range ppl.Items {
		if err := portpool.Allocator.AddPool(
			pp.Name,
			util.GetRegionFromPtr(pp.Spec.Region),
			pp.Spec.StartPort,
			pp.Spec.EndPort,
			pp.Spec.SegmentLength,
			controller.GetCreateLoadBalancerFunc(i.Client, pp.Name),
		); err != nil {
			return err
		}
		lbIds := []string{}
		for _, lbStatus := range pp.Status.LoadbalancerStatuses {
			lbIds = append(lbIds, lbStatus.LoadbalancerID)
		}
		if err := portpool.Allocator.EnsureLbIds(pp.Name, lbIds); err != nil {
			return err
		}
	}

	// 初始化Pod端口绑定缓存
	pbl := &networkingv1alpha1.CLBPodBindingList{}
	if err := i.List(ctx, pbl); err != nil {
		return err
	}
	for _, pb := range pbl.Items {
		for _, podBinding := range pb.Status.PortBindings {
			if err := portpool.Allocator.MarkAllocated(podBinding.Pool, podBinding.LoadbalancerId, podBinding.LoadbalancerPort, podBinding.LoadbalancerEndPort, podBinding.Protocol); err != nil {
				return err
			}
		}
	}
	return nil
}
