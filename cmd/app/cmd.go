package app

import (
	"flag"
	"fmt"
	"strings"

	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var RootCommand = cobra.Command{
	Use:   "tke-extend-network-controller",
	Short: "A network controller for TKE",
	Run: func(cmd *cobra.Command, args []string) {
		clb.Init(
			viper.GetString(secretId),
			viper.GetString(secretKey),
			viper.GetString(region),
			viper.GetString(vpcId),
			viper.GetString(clusterId),
		)
		runManager()
	},
}

const (
	metricsBindAddress     = "metrics-bind-address"
	leaderElect            = "leader-elect"
	healthProbeBindAddress = "health-probe-bind-address"
	secretId               = "secret-id"
	secretKey              = "secret-key"
	region                 = "region"
	vpcId                  = "vpcid"
	clusterId              = "cluster-id"
	workerCount            = "worker-count"
)

var (
	envReplacer = strings.NewReplacer("-", "_")
	zapOptions  = &zap.Options{}
)

func init() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(envReplacer)

	flags := RootCommand.Flags()
	zapOptions.BindFlags(flag.CommandLine)
	flags.AddGoFlagSet(flag.CommandLine)
	addIntFlag(flags, workerCount, 1, "The worker count of each controller.")
	addStringFlag(flags, metricsBindAddress, "0", "The address the metrics endpoint binds to. Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	addStringFlag(flags, healthProbeBindAddress, ":8081", "The address the probe endpoint binds to.")
	addBoolFlag(flags, leaderElect, false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	addStringFlag(flags, secretId, "", "Secret ID")
	addStringFlag(flags, secretKey, "", "Secret Key")
	addStringFlag(flags, region, "", "The region of TKE cluster")
	addStringFlag(flags, vpcId, "", "The VPC ID of TKE cluster")
}

func addStringFlag(flags *pflag.FlagSet, name, value, usage string) {
	flags.String(name, value, wrapUsage(name, usage))
	viper.BindPFlag(name, flags.Lookup(name))
}

func addIntFlag(flags *pflag.FlagSet, name string, value int, usage string) {
	flags.Int(name, value, wrapUsage(name, usage))
	viper.BindPFlag(name, flags.Lookup(name))
}

func addBoolFlag(flags *pflag.FlagSet, name string, value bool, usage string) {
	flags.Bool(name, value, wrapUsage(name, usage))
	viper.BindPFlag(name, flags.Lookup(name))
}

func wrapUsage(name, usage string) string {
	envName := strings.ToUpper(envReplacer.Replace(name))
	return fmt.Sprintf("%s (Env: %s)", usage, envName)
}
