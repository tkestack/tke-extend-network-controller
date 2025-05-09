package app

import (
	"context"
	"os"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/portpool"
	portpoolutil "github.com/imroc/tke-extend-network-controller/internal/portpool/util"
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
		if err := portpool.Allocator.AddPool(portpoolutil.NewPortPool(pp, i.Client)); err != nil {
			return err
		}
		lbIds := []string{}
		for _, lbStatus := range pp.Status.LoadbalancerStatuses {
			if lbStatus.State != networkingv1alpha1.LoadBalancerStateNotFound {
				lbIds = append(lbIds, lbStatus.LoadbalancerID)
			}
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
		for _, bd := range pb.Status.PortBindings {
			portpool.Allocator.MarkAllocated(bd.Pool, bd.LoadbalancerId, bd.LoadbalancerPort, bd.LoadbalancerEndPort, bd.Protocol)
		}
	}
	npbl := &networkingv1alpha1.CLBNodeBindingList{}
	if err := i.List(ctx, npbl); err != nil {
		return err
	}
	for _, pb := range npbl.Items {
		for _, bd := range pb.Status.PortBindings {
			portpool.Allocator.MarkAllocated(bd.Pool, bd.LoadbalancerId, bd.LoadbalancerPort, bd.LoadbalancerEndPort, bd.Protocol)
		}
	}
	return nil
}
