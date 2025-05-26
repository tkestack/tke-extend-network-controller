package v1alpha1

// PortEntry 定义单个端口的绑定配置
type PortEntry struct {
	// 应用监听的端口号
	Port uint16 `json:"port"`
	// 端口使用的协议
	// +kubebuilder:validation:Enum=TCP;UDP;TCPUDP;TCP_SSL;QUIC
	Protocol string `json:"protocol"`
	// 使用的端口池列表
	Pools []string `json:"pools"`
	// 是否跨端口池分配相同端口号
	// +optional
	UseSamePortAcrossPools *bool `json:"useSamePortAcrossPools,omitempty"`
	// 包含服务端证书的 ID 的 Secret 名称。仅对 TCP_SSL 和 QUIC 协议有效。
	// +optional
	CertSecretName *string `json:"certSecretName,omitempty"`
}

type CLBBindingState string

const (
	CLBBindingStatePending              CLBBindingState = "Pending"
	CLBBindingStateBound                CLBBindingState = "Bound"
	CLBBindingStateNoBackend            CLBBindingState = "NoBackend"
	CLBBindingStateWaitBackend          CLBBindingState = "WaitBackend"
	CLBBindingStateNodeTypeNotSupported CLBBindingState = "NodeTypeNotSupported"
	CLBBindingStateDisabled             CLBBindingState = "Disabled"
	CLBBindingStateFailed               CLBBindingState = "Failed"
	CLBBindingStatePortPoolNotFound     CLBBindingState = "PortPoolNotFound"
	CLBBindingStateNoPortAvailable      CLBBindingState = "NoPortAvailable"
	CLBBindingStateDeleting             CLBBindingState = "Deleting"
)

// CLBBindingStatus defines the observed state of CLBPodBinding.
type CLBBindingStatus struct {
	// 绑定状态
	State CLBBindingState `json:"state"`
	// 状态信息
	Message string `json:"message,omitempty"`
	// 端口绑定详情
	PortBindings []PortBindingStatus `json:"portBindings,omitempty"`
}

// PortBindingStatus 描述单个端口的实际绑定情况
type PortBindingStatus struct {
	// 应用端口
	Port uint16 `json:"port"`
	// 协议类型
	Protocol string `json:"protocol"`
	// 服务端证书 ID（仅在 TCP_SSL 和 QUIC 协议下有效）
	// +optional
	CertId *string `json:"certId,omitempty"`
	// 使用的端口池
	Pool string `json:"pool"`
	// 地域信息
	Region string `json:"region"`
	// 负载均衡器ID
	LoadbalancerId string `json:"loadbalancerId"`
	// 负载均衡器端口
	LoadbalancerPort uint16 `json:"loadbalancerPort"`
	// 负载均衡器端口段结束端口（当使用端口段时）
	// +optional
	LoadbalancerEndPort *uint16 `json:"loadbalancerEndPort,omitempty"`
	// 监听器ID
	ListenerId string `json:"listenerId"`
}

// CLBBindingSpec defines the desired state of CLBPodBinding.
type CLBBindingSpec struct {
	// 网络隔离
	// +optional
	Disabled *bool `json:"disabled,omitempty"`
	// 需要绑定的端口配置列表
	Ports []PortEntry `json:"ports"`
}
