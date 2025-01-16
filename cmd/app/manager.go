package app

import (
	"context"
	"os"

	"github.com/spf13/viper"

	"github.com/imroc/tke-extend-network-controller/internal/controller"
	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	"github.com/imroc/tke-extend-network-controller/pkg/manager"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	networkingv1beta1 "github.com/imroc/tke-extend-network-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(networkingv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func runManager() {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(zapOptions)))

	region := viper.GetString(regionFlag)
	if region == "" {
		var err error
		setupLog.Info("no region specified, trying to get current region from metadata api")
		region, err = util.GetCurrentRegion()
		if err != nil {
			setupLog.Error(err, "failed to get current region")
			os.Exit(1)
		}
	}
	setupLog.Info("use region " + region)
	clb.Init(
		viper.GetString(secretIdFlag),
		viper.GetString(secretKeyFlag),
		region,
		viper.GetString(vpcIdFlag),
		viper.GetString(clusterIdFlag),
	)
	_, err := clb.SyncQuota(context.Background(), region)
	if err != nil {
		setupLog.Error(err, "failed to sync clb quota")
		os.Exit(1)
	}

	metricsAddr := viper.GetString(metricsBindAddressFlag)
	probeAddr := viper.GetString(healthProbeBindAddressFlag)
	enableLeaderElection := viper.GetBool(leaderElectFlag)
	workers := viper.GetInt(workerCountFlag)
	if workers <= 0 {
		workers = 1
	}

	mgr, err := ctrl.NewManager(
		ctrl.GetConfigOrDie(),
		manager.GetOptions(scheme, metricsAddr, probeAddr, enableLeaderElection),
	)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.DedicatedCLBServiceReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		APIReader: mgr.GetAPIReader(),
		Recorder:  mgr.GetEventRecorderFor("dedicatedclbservice-controller"),
	}).SetupWithManager(mgr, workers); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DedicatedCLBService")
		os.Exit(1)
	}

	if err = (&controller.DedicatedCLBListenerReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		APIReader: mgr.GetAPIReader(),
		Recorder:  mgr.GetEventRecorderFor("dedicatedclblistener-controller"),
	}).SetupWithManager(mgr, workers); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CLBListenerReconciler")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder
	networkingv1beta1.Init(setupLog, mgr)
	kube.Init(mgr)

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
