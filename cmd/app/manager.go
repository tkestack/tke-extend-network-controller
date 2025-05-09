package app

import (
	"context"
	"os"

	"github.com/spf13/viper"

	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/imroc/tke-extend-network-controller/pkg/cloudapi"
	"github.com/imroc/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	"github.com/imroc/tke-extend-network-controller/pkg/manager"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var setupLog = ctrl.Log.WithName("setup")

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
	clusterinfo.ClusterId = viper.GetString(clusterIdFlag)
	clusterinfo.VpcId = viper.GetString(vpcIdFlag)
	clusterinfo.Region = viper.GetString(regionFlag)
	if clusterinfo.Region == "" || clusterinfo.VpcId == "" {
		panic("default region and vpcId is required")
	}
	if clusterinfo.ClusterId == "" {
		panic("clusterId is required")
	}
	cloudapi.Init(
		viper.GetString(secretIdFlag),
		viper.GetString(secretKeyFlag),
	)
	clb.Init(
		viper.GetString(secretIdFlag),
		viper.GetString(secretKeyFlag),
	)
	_, err := clb.SyncQuota(context.Background(), region)
	if err != nil {
		setupLog.Error(err, "failed to sync clb quota")
		os.Exit(1)
	}

	metricsAddr := viper.GetString(metricsBindAddressFlag)
	probeAddr := viper.GetString(healthProbeBindAddressFlag)
	enableLeaderElection := viper.GetBool(leaderElectFlag)

	mgr, err := ctrl.NewManager(
		ctrl.GetConfigOrDie(),
		manager.GetOptions(scheme, metricsAddr, probeAddr, enableLeaderElection),
	)
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	// setup manager
	SetupManager(mgr)

	// TODO: remove the tricky code in the future
	networkingv1alpha1.Init(mgr)
	kube.Init(mgr)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
