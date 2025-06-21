package app

import (
	"context"
	"os"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var scheme = runtime.NewScheme()

func init() {
	// Add API to schema
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))     // k8s native resources
	utilruntime.Must(networkingv1alpha1.AddToScheme(scheme)) // tke-extend-network-controller CRDs
	// utilruntime.Must(kruisegamev1alpha1.AddToScheme(scheme)) // OKG CRDs
	// utilruntime.Must(agonesv1.AddToScheme(scheme))           // Agones CRDs
}

func SetupManager(mgr ctrl.Manager, opts manager.Options) {
	if err := mgr.Add(&initCache{mgr.GetClient(), mgr.GetEventRecorderFor("clbportpool-controller")}); err != nil {
		setupLog.Error(err, "problem add init cache")
		os.Exit(1)
	}

	SetupControllers(mgr)
	SetupWebhooks(mgr)

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
	record.EventRecorder
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
	for index := range ppl.Items {
		pp := &ppl.Items[index]
		portpool.Allocator.AddPoolIfNotExists(pp)

		lbKeys := []portpool.LBKey{}
		for _, lbStatus := range pp.Status.LoadbalancerStatuses {
			if lbStatus.State != networkingv1alpha1.LoadBalancerStateNotFound {
				lbKeys = append(lbKeys, portpool.NewLBKey(lbStatus.LoadbalancerID, pp.GetRegion()))
			}
		}
		if err := portpool.Allocator.EnsureLbIds(pp.Name, lbKeys); err != nil {
			return err
		}
	}

	// 初始化Pod端口绑定缓存
	pbl := &networkingv1alpha1.CLBPodBindingList{}
	if err := i.List(ctx, pbl); err != nil {
		return err
	}
	for _, pb := range pbl.Items {
		for _, bd := range pb.Status.PortBindings {
			portpool.Allocator.MarkAllocated(bd.Pool, portpool.NewLBKey(bd.LoadbalancerId, bd.Region), bd.LoadbalancerPort, bd.LoadbalancerEndPort, bd.Protocol)
		}
	}
	npbl := &networkingv1alpha1.CLBNodeBindingList{}
	if err := i.List(ctx, npbl); err != nil {
		return err
	}
	for _, pb := range npbl.Items {
		for _, bd := range pb.Status.PortBindings {
			portpool.Allocator.MarkAllocated(bd.Pool, portpool.NewLBKey(bd.LoadbalancerId, bd.Region), bd.LoadbalancerPort, bd.LoadbalancerEndPort, bd.Protocol)
		}
	}
	return nil
}
