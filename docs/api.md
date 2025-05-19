# API Reference

## Packages
- [networking.cloud.tencent.com/v1alpha1](#networkingcloudtencentcomv1alpha1)


## networking.cloud.tencent.com/v1alpha1


### Resource Types
- [CLBNodeBinding](#clbnodebinding)
- [CLBPodBinding](#clbpodbinding)
- [CLBPortPool](#clbportpool)
- [DedicatedCLBListener](#dedicatedclblistener)
- [DedicatedCLBService](#dedicatedclbservice)



#### AllocatableCLBInfo







_Appears in:_
- [DedicatedCLBServiceStatus](#dedicatedclbservicestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbId` _string_ | CLB 实例的 ID。 |  |  |
| `currentPort` _integer_ | CLB 当前已被分配的端口。 |  |  |
| `autoCreate` _boolean_ | 是否是自动创建的 CLB。如果是，删除 DedicatedCLBService 时，CLB 也会被清理。 |  |  |


#### AllocatedCLBInfo







_Appears in:_
- [DedicatedCLBServiceStatus](#dedicatedclbservicestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbId` _string_ | CLB 实例的 ID。 |  |  |
| `autoCreate` _boolean_ | 是否是自动创建的 CLB。如果是，删除 DedicatedCLBService 时，CLB 也会被清理。 |  |  |


#### AutoCreateConfig



AutoCreateConfig 定义自动创建CLB的配置



_Appears in:_
- [CLBPortPoolSpec](#clbportpoolspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | 是否启用自动创建 |  |  |
| `maxLoadBalancers` _integer_ | 自动创建的最大负载均衡器数量 |  |  |
| `parameters` _[CreateLBParameters](#createlbparameters)_ | 自动创建参数 |  |  |


#### CLBBindingSpec



CLBBindingSpec defines the desired state of CLBPodBinding.



_Appears in:_
- [CLBNodeBinding](#clbnodebinding)
- [CLBPodBinding](#clbpodbinding)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `disabled` _boolean_ | 网络隔离 |  |  |
| `ports` _[PortEntry](#portentry) array_ | 需要绑定的端口配置列表 |  |  |


#### CLBBindingState

_Underlying type:_ _string_





_Appears in:_
- [CLBBindingStatus](#clbbindingstatus)

| Field | Description |
| --- | --- |
| `Pending` |  |
| `Bound` |  |
| `NoBackend` |  |
| `WaitBackend` |  |
| `NodeTypeNotSupported` |  |
| `Disabled` |  |
| `Failed` |  |
| `PortPoolNotFound` |  |
| `NoPortAvailable` |  |
| `Deleting` |  |


#### CLBBindingStatus



CLBBindingStatus defines the observed state of CLBPodBinding.



_Appears in:_
- [CLBNodeBinding](#clbnodebinding)
- [CLBPodBinding](#clbpodbinding)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `state` _[CLBBindingState](#clbbindingstate)_ | 绑定状态 |  |  |
| `message` _string_ | 状态信息 |  |  |
| `portBindings` _[PortBindingStatus](#portbindingstatus) array_ | 端口绑定详情 |  |  |


#### CLBNodeBinding



CLBNodeBinding is the Schema for the clbnodebindings API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `CLBNodeBinding` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CLBBindingSpec](#clbbindingspec)_ |  |  |  |
| `status` _[CLBBindingStatus](#clbbindingstatus)_ |  |  |  |




#### CLBPodBinding



CLBPodBinding is the Schema for the clbpodbindings API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `CLBPodBinding` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CLBBindingSpec](#clbbindingspec)_ |  |  |  |
| `status` _[CLBBindingStatus](#clbbindingstatus)_ |  |  |  |


#### CLBPortPool



CLBPortPool is the Schema for the clbportpools API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `CLBPortPool` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CLBPortPoolSpec](#clbportpoolspec)_ |  |  |  |
| `status` _[CLBPortPoolStatus](#clbportpoolstatus)_ |  |  |  |


#### CLBPortPoolSpec



CLBPortPoolSpec defines the desired state of CLBPortPool.



_Appears in:_
- [CLBPortPool](#clbportpool)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `startPort` _integer_ | 端口池的起始端口号 |  |  |
| `endPort` _integer_ | 端口池的结束端口号 |  |  |
| `segmentLength` _integer_ | 端口段的长度 |  |  |
| `region` _string_ | 地域代码，如ap-chengdu |  |  |
| `exsistedLoadBalancerIDs` _string array_ | 已有负载均衡器ID列表 |  |  |
| `autoCreate` _[AutoCreateConfig](#autocreateconfig)_ | 自动创建配置 |  |  |


#### CLBPortPoolState

_Underlying type:_ _string_





_Appears in:_
- [CLBPortPoolStatus](#clbportpoolstatus)

| Field | Description |
| --- | --- |
| `Pending` |  |
| `Active` |  |
| `Scaling` |  |
| `Deleting` |  |


#### CLBPortPoolStatus



CLBPortPoolStatus defines the observed state of CLBPortPool.



_Appears in:_
- [CLBPortPool](#clbportpool)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `state` _[CLBPortPoolState](#clbportpoolstate)_ | 状态: Pending/Active/Scaling |  |  |
| `message` _string_ | 状态信息 |  |  |
| `loadbalancerStatuses` _[LoadBalancerStatus](#loadbalancerstatus) array_ | 负载均衡器状态列表 |  |  |


#### CreateLBParameters



CreateLBParameters 定义创建负载均衡器的参数



_Appears in:_
- [AutoCreateConfig](#autocreateconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vipIsp` _string_ | 仅适用于公网负载均衡。目前仅广州、上海、南京、济南、杭州、福州、北京、石家庄、武汉、长沙、成都、重庆地域支持静态单线 IP 线路类型，如需体验，请联系商务经理申请。申请通过后，即可选择中国移动（CMCC）、中国联通（CUCC）或中国电信（CTCC）的运营商类型，网络计费模式只能使用按带宽包计费(BANDWIDTH_PACKAGE)。 如果不指定本参数，则默认使用BGP。可通过 DescribeResources 接口查询一个地域所支持的Isp。 |  | Enum: [CMCC CUCC CTCC BGP] <br /> |
| `bandwidthPackageId` _string_ | 带宽包ID，指定此参数时，网络计费方式（InternetAccessible.InternetChargeType）只支持按带宽包计费（BANDWIDTH_PACKAGE），带宽包的属性即为其结算方式。非上移用户购买的 IPv6 负载均衡实例，且运营商类型非 BGP 时 ，不支持指定具体带宽包id。 |  |  |
| `addressIPVersion` _string_ | 仅适用于公网负载均衡。IP版本，可取值：IPV4、IPV6、IPv6FullChain，不区分大小写，默认值 IPV4。说明：取值为IPV6表示为IPV6 NAT64版本；取值为IPv6FullChain，表示为IPv6版本。 |  | Enum: [IPV4 IPV6 IPv6FullChain] <br /> |
| `loadBalancerPassToTarget` _boolean_ | Target是否放通来自CLB的流量。开启放通（true）：只验证CLB上的安全组；不开启放通（false）：需同时验证CLB和后端实例上的安全组。默认值为 true。 |  |  |
| `dynamicVip` _boolean_ | 是否创建域名化负载均衡。 |  |  |
| `vpcId` _string_ | 负载均衡后端目标设备所属的网络 ID，如vpc-12345678，可以通过 DescribeVpcs 接口获取。 不填此参数则默认为当前集群所在 VPC。创建内网负载均衡实例时，此参数必填。 |  |  |
| `vip` _string_ | 指定VIP申请负载均衡。此参数选填，不填写此参数时自动分配VIP。IPv4和IPv6类型支持此参数，IPv6 NAT64类型不支持。<br />注意：当指定VIP创建内网实例、或公网IPv6 BGP实例时，若VIP不属于指定VPC子网的网段内时，会创建失败；若VIP已被占用，也会创建失败。 |  |  |
| `tags` _[TagInfo](#taginfo) array_ | 购买负载均衡的同时，给负载均衡打上标签，最大支持20个标签键值对。 |  |  |
| `projectId` _integer_ | 负载均衡实例所属的项目 ID，可以通过 DescribeProject 接口获取。不填此参数则视为默认项目。 |  |  |
| `loadBalancerName` _string_ | 负载均衡实例的名称。规则：1-60 个英文、汉字、数字、连接线“-”或下划线“_”。 注意：如果名称与系统中已有负载均衡实例的名称相同，则系统将会自动生成此次创建的负载均衡实例的名称。 |  |  |
| `loadBalancerType` _string_ | 负载均衡实例的网络类型：OPEN：公网属性， INTERNAL：内网属性。默认使用 OPEN（公网负载均衡）。 |  | Enum: [OPEN INTERNAL] <br /> |
| `masterZoneId` _string_ | 仅适用于公网且IP版本为IPv4的负载均衡。设置跨可用区容灾时的主可用区ID，例如 100001 或 ap-guangzhou-1<br />注：主可用区是需要承载流量的可用区，备可用区默认不承载流量，主可用区不可用时才使用备可用区。目前仅广州、上海、南京、北京、成都、深圳金融、中国香港、首尔、法兰克福、新加坡地域的 IPv4 版本的 CLB 支持主备可用区。可通过 DescribeResources 接口查询一个地域的主可用区的列表。【如果您需要体验该功能，请通过 工单申请】 |  |  |
| `zoneId` _string_ | 仅适用于公网且IP版本为IPv4的负载均衡。可用区ID，指定可用区以创建负载均衡实例。 |  |  |
| `subnetId` _string_ | 在私有网络内购买内网负载均衡实例的情况下，必须指定子网 ID，内网负载均衡实例的 VIP 将从这个子网中产生。创建内网负载均衡实例时，此参数必填，创建公网IPv4负载均衡实例时，不支持指定该参数。 |  |  |
| `slaType` _string_ | 性能容量型规格。<br />若需要创建性能容量型实例，则此参数必填，取值范围：<br />clb.c2.medium：标准型规格<br />clb.c3.small：高阶型1规格<br />clb.c3.medium：高阶型2规格<br />clb.c4.small：超强型1规格<br />clb.c4.medium：超强型2规格<br />clb.c4.large：超强型3规格<br />clb.c4.xlarge：超强型4规格<br />若需要创建共享型实例，则无需填写此参数。 |  | Enum: [clb.c2.medium clb.c3.small clb.c3.medium clb.c4.small clb.c4.medium clb.c4.large clb.c4.xlarge] <br /> |
| `lbChargeType` _string_ | 负载均衡实例计费类型，取值：POSTPAID_BY_HOUR，PREPAID，默认是POSTPAID_BY_HOUR。 |  | Enum: [POSTPAID_BY_HOUR PREPAID] <br /> |
| `internetAccessible` _[InternetAccessible](#internetaccessible)_ | 仅适用于公网负载均衡。负载均衡的网络计费模式。 |  |  |


#### DedicatedCLBListener



DedicatedCLBListener is the Schema for the dedicatedclblisteners API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `DedicatedCLBListener` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[DedicatedCLBListenerSpec](#dedicatedclblistenerspec)_ |  |  |  |
| `status` _[DedicatedCLBListenerStatus](#dedicatedclblistenerstatus)_ |  |  |  |


#### DedicatedCLBListenerSpec



DedicatedCLBListenerSpec defines the desired state of DedicatedCLBListener



_Appears in:_
- [DedicatedCLBListener](#dedicatedclblistener)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbId` _string_ | CLB 实例的 ID。 |  |  |
| `lbRegion` _string_ | CLB 所在地域，不填则使用 TKE 集群所在的地域。 |  |  |
| `lbPort` _integer_ | CLB 监听器的端口号。 |  |  |
| `lbEndPort` _integer_ | CLB 端口段监听器的结束端口号。 |  |  |
| `protocol` _string_ | CLB 监听器的协议。 |  | Enum: [TCP UDP] <br /> |
| `extensiveParameters` _string_ | 创建监听器的参数，JSON 格式，详细参数请参考 CreateListener 接口：https://cloud.tencent.com/document/api/214/30693 |  |  |
| `targetPod` _[TargetPod](#targetpod)_ | CLB 监听器绑定的目标 Pod。 |  |  |


#### DedicatedCLBListenerStatus



DedicatedCLBListenerStatus defines the observed state of DedicatedCLBListener



_Appears in:_
- [DedicatedCLBListener](#dedicatedclblistener)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `listenerId` _string_ | CLB 监听器的 ID。 |  |  |
| `state` _string_ | CLB 监听器的状态。 |  | Enum: [Bound Available Pending Failed Deleting] <br /> |
| `message` _string_ | 记录 CLB 监听器的失败信息。 |  |  |
| `address` _string_ | CLB 监听器的外部地址。 |  |  |


#### DedicatedCLBService



DedicatedCLBService is the Schema for the dedicatedclbservices API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `DedicatedCLBService` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[DedicatedCLBServiceSpec](#dedicatedclbservicespec)_ |  |  |  |
| `status` _[DedicatedCLBServiceStatus](#dedicatedclbservicestatus)_ |  |  |  |


#### DedicatedCLBServicePort







_Appears in:_
- [DedicatedCLBServiceSpec](#dedicatedclbservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `protocol` _string_ | 端口协议，支持 TCP、UDP。 |  |  |
| `targetPort` _integer_ | 目标端口。 |  |  |
| `addressPodAnnotation` _string_ | Pod 外部地址的注解，如果设置，Pod 被映射的外部 CLB 地址将会被自动写到 Pod 的该注解中，Pod 内部可通过 Downward API 感知到自身的外部地址。 |  |  |


#### DedicatedCLBServiceSpec



DedicatedCLBServiceSpec defines the desired state of DedicatedCLBService



_Appears in:_
- [DedicatedCLBService](#dedicatedclbservice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbRegion` _string_ | CLB 所在地域，不填则使用 TKE 集群所在的地域。 |  |  |
| `vpcId` _string_ | CLB 所在 VPC ID，不填则使用 TKE 集群所在的 VPC 的 ID。 |  |  |
| `minPort` _integer_ | CLB 端口范围的最小端口号。 | 500 |  |
| `maxPort` _integer_ | CLB 端口范围的最大端口号。 | 50000 |  |
| `maxPod` _integer_ | 限制单个 CLB 的 Pod/监听器 的最大数量。 |  |  |
| `selector` _object (keys:string, values:string)_ | Pod 的标签选择器，被选中的 Pod 会被绑定到 CLB 监听器下。 |  |  |
| `ports` _[DedicatedCLBServicePort](#dedicatedclbserviceport) array_ | Pod 监听的端口。 |  |  |
| `listenerExtensiveParameters` _string_ | 创建监听器的参数，JSON 格式，详细参数请参考 CreateListener 接口：https://cloud.tencent.com/document/api/214/30693 |  |  |
| `existedLbIds` _string array_ | 复用的已有的 CLB ID，可动态追加。 |  |  |
| `lbAutoCreate` _[LbAutoCreate](#lbautocreate)_ | 启用自动创建 CLB 的功能。 |  |  |


#### DedicatedCLBServiceStatus



DedicatedCLBServiceStatus defines the observed state of DedicatedCLBService



_Appears in:_
- [DedicatedCLBService](#dedicatedclbservice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `allocatableLb` _[AllocatableCLBInfo](#allocatableclbinfo) array_ | 可分配端口的 CLB 列表 |  |  |
| `allocatedLb` _[AllocatedCLBInfo](#allocatedclbinfo) array_ | 已分配完端口的 CLB 列表 |  |  |


#### InternetAccessible



InternetAccessible 定义网络计费相关参数



_Appears in:_
- [CreateLBParameters](#createlbparameters)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `internetChargeType` _string_ | TRAFFIC_POSTPAID_BY_HOUR 按流量按小时后计费 ; BANDWIDTH_POSTPAID_BY_HOUR 按带宽按小时后计费; BANDWIDTH_PACKAGE 按带宽包计费;BANDWIDTH_PREPAID按带宽预付费。注意：此字段可能返回 null，表示取不到有效值。 |  | Enum: [TRAFFIC_POSTPAID_BY_HOUR BANDWIDTH_POSTPAID_BY_HOUR BANDWIDTH_PACKAGE BANDWIDTH_PREPAID] <br /> |
| `internetMaxBandwidthOut` _integer_ | 最大出带宽，单位Mbps，仅对公网属性的共享型、性能容量型和独占型 CLB 实例、以及内网属性的性能容量型 CLB 实例生效。<br />- 对于公网属性的共享型和独占型 CLB 实例，最大出带宽的范围为1Mbps-2048Mbps。<br />- 对于公网属性和内网属性的性能容量型 CLB实例，最大出带宽的范围为1Mbps-61440Mbps。<br />（调用CreateLoadBalancer创建LB时不指定此参数则设置为默认值10Mbps。此上限可调整） |  |  |
| `bandwidthpkgSubType` _string_ | 带宽包的类型，如 SINGLEISP（单线）、BGP（多线）。 |  | Enum: [SINGLEISP BGP] <br /> |


#### LbAutoCreate







_Appears in:_
- [DedicatedCLBServiceSpec](#dedicatedclbservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enable` _boolean_ | 是否启用自动创建 CLB 的功能，如果启用，当 CLB 不足时，会自动创建新的 CLB。 |  |  |
| `extensiveParameters` _string_ | 创建 CLB 时的参数，JSON 格式，详细参数请参考 CreateLoadBalancer 接口：https://cloud.tencent.com/document/api/214/30692 |  |  |


#### LoadBalancerState

_Underlying type:_ _string_





_Appears in:_
- [LoadBalancerStatus](#loadbalancerstatus)

| Field | Description |
| --- | --- |
| `Running` |  |
| `NotFound` |  |


#### LoadBalancerStatus



LoadBalancerStatus 定义负载均衡器状态



_Appears in:_
- [CLBPortPoolStatus](#clbportpoolstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `autoCreated` _boolean_ |  |  |  |
| `state` _[LoadBalancerState](#loadbalancerstate)_ |  |  |  |
| `loadbalancerID` _string_ |  |  |  |
| `loadbalancerName` _string_ |  |  |  |
| `ips` _string array_ |  |  |  |
| `hostname` _string_ |  |  |  |


#### PortBindingStatus



PortBindingStatus 描述单个端口的实际绑定情况



_Appears in:_
- [CLBBindingStatus](#clbbindingstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `port` _integer_ | 应用端口 |  |  |
| `protocol` _string_ | 协议类型 |  |  |
| `certId` _string_ | 服务端证书 ID（仅在 TCP_SSL 和 QUIC 协议下有效） |  |  |
| `pool` _string_ | 使用的端口池 |  |  |
| `region` _string_ | 地域信息 |  |  |
| `loadbalancerId` _string_ | 负载均衡器ID |  |  |
| `loadbalancerPort` _integer_ | 负载均衡器端口 |  |  |
| `loadbalancerEndPort` _integer_ | 负载均衡器端口段结束端口（当使用端口段时） |  |  |
| `listenerId` _string_ | 监听器ID |  |  |


#### PortEntry



PortEntry 定义单个端口的绑定配置



_Appears in:_
- [CLBBindingSpec](#clbbindingspec)
- [CLBNodeBindingSpec](#clbnodebindingspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `port` _integer_ | 应用监听的端口号 |  |  |
| `protocol` _string_ | 端口使用的协议 |  | Enum: [TCP UDP TCPUDP TCP_SSL QUIC] <br /> |
| `pools` _string array_ | 使用的端口池列表 |  |  |
| `useSamePortAcrossPools` _boolean_ | 是否跨端口池分配相同端口号 |  |  |
| `certSecretName` _string_ | 包含服务端证书的 ID 的 Secret 名称。仅对 TCP_SSL 和 QUIC 协议有效。 |  |  |


#### TagInfo



TagInfo 定义标签结构



_Appears in:_
- [CreateLBParameters](#createlbparameters)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tagKey` _string_ | 标签的键 |  |  |
| `tagValue` _string_ | 标签的值 |  |  |


#### TargetPod







_Appears in:_
- [DedicatedCLBListenerSpec](#dedicatedclblistenerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podName` _string_ | Pod 的名称。 |  |  |
| `targetPort` _integer_ | Pod 监听的端口。 |  |  |


