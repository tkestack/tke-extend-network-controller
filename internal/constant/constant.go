package constant

const (
	EnableCLBPortMappingsKey     = "networking.cloud.tencent.com/enable-clb-port-mapping"
	CLBPortMappingsKey           = "networking.cloud.tencent.com/clb-port-mapping"
	CLBPortMappingStatuslKey     = "networking.cloud.tencent.com/clb-port-mapping-status"
	CLBHostPortMappingStatuslKey = "networking.cloud.tencent.com/clb-hostport-mapping-status"
	CLBPortMappingResultKey      = "networking.cloud.tencent.com/clb-port-mapping-result"
	CLBHostPortMappingResultKey  = "networking.cloud.tencent.com/clb-hostport-mapping-result"
	EnableCLBHostPortMapping     = "networking.cloud.tencent.com/enable-clb-hostport-mapping"
	Finalizer                    = "networking.cloud.tencent.com/finalizer"
	Ratain                       = "networking.cloud.tencent.com/retain"
	LastUpdateTime               = "networking.cloud.tencent.com/last-update-time"
	FinalizedKey                 = "networking.cloud.tencent.com/finalized"
	ProtocolTCP                  = "TCP"
	ProtocolUDP                  = "UDP"
	ProtocolTCPUDP               = "TCPUDP"
	OKGNetworkType               = "TencentCloud-CLB"
	AgonesGameServerLabelKey     = "agones.dev/gameserver"

	// 均匀端口分配策略，每次找已分配数最少的 lb 来分配端口
	LbPolicyUniform = "Uniform"
	// 按固定顺序分配
	LbPolicyInOrder = "InOrder"
	// 随机分配策略，每次随机找一个 lb 来分配端口
	LbPolicyRandom         = "Random"
	CLBPortPoolTagKey      = "clbportpool"
	TkeClusterIDTagKey     = "tke-clusterId"
	TkeCreatedFlagTagKey   = "tke-createdBy-flag"
	TkeCreatedFlagYesValue = "yes"
)
