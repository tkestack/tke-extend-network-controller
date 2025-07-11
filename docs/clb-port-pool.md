# 使用 CLB 端口池为 Pod 映射公网地址

## CLB 端口池介绍

使用 CLB 为 Pod 分配公网地址映射，首先需要定义 CLB 端口池。

端口池的设计：
1. 每个端口池使用一组相同属性的 CLB，用于为 Pod 分配端口映射。
2. 可以动态追加已有 CLB 实例 ID，也可以在端口不足时自动创建新的 CLB，自动新建的 CLB 的属性可以自定义。
3. 同一个 Pod 的同一个端口可以同时被不同的端口池映射，比如同时被多个单线 IP 的端口池映射来实现多线接入。
4. 支持使用 CLB 端口段映射（1 个 CLB 端口段监听器可映射多个 Pod），可支持大规模场景的映射。
5. 端口池不是 namespaced 资源，无需配置 namespace。

通过创建 CLBPortPool 这个自定义资源来声明端口池，完整的字段说明如下：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-test # 端口池名称，将会在 Pod 或 Node 注解中被引用
spec:
  startPort: 30000 # 端口池中 CLB 起始端口号
  endPort: 30100 # 可选，端口池的结束端口号。通常只在明确需要限制 CLB 最大端口号时才需要设置，默认情况下会根据当前监听器数量配额和端口分配情况来自动决定。
  segmentLength: 0 # 可选，端口段的长度。仅当值大于 1 时才有效，此时将使用 CLB 端口段监听器来映射（1 个 CLB 监听器可映射 segmentLength 个端口)，结合节点的 HostPort 可实现大规模场景的映射。
  exsistedLoadBalancerIDs: [lb-04iq85jh] # 指定已有的 CLB 实例 ID，可动态追加
  lbPolicy: Random # 可选，CLB 分配策略，单个端口池中有多个可分配 CLB ，分配端口时 CLB 的挑选策略。可选值：Uniform（均匀分配）、InOrder（顺序分配）、Random（随机分配）。默认值为 Random。若希望减小 DDoS 攻击的影响，建议使用 Uniform 策略，避免业务使用的 IP 过于集中；若希望提高CLB 的利用率，建议使用 InOrder 策略。
  lbBlacklist: [] # 可选，CLB 黑名单，负载均衡实例 ID 的数组，用于禁止某些 CLB 实例被分配端口，可动态追加和移除。如果发现某个 CLB 被 DDoS 攻击或其他原因导致不可用，可将该 CLB 的实例 ID 加入到黑名单中，避免后续端口分配使用该 CLB。
  listenerQuota: 50 # 可选，监听器数量配额。仅用在单独调整了指定 CLB 实例监听器数量配额的场景（TOTAL_LISTENER_QUOTA），控制器默认会获取账号维度的监听器数量配额作为端口分配的依据，如果 listenerQuota 不为空，将以它的值作为该端口池中所有 CLB 监听器数量配额覆盖账号维度的监听器数量配额。注意：如果指定了 listenerQuota，不支持启用 CLB 自动创建，且需自行保证该端口池中所有 CLB 实例的监听器数量配额均等于 listenerQuota 的值。
  region: ap-guangzhou # 可选，CLB 说在地域的地域代码，如 ap-chengdu，默认使用 TKE 集群说在地域。
  autoCreate: # 可选，自动创建 CLB 的配置，如果不配置，则不会自动创建 CLB
    enabled: true # 是否启用自动创建，如果启用将会在 CLB 端口不足时自动创建 CLB
    maxLoadBalancers: 10 # 可选，限制自动创建的最大负载均衡器数量，默认不限制。
    parameters: # 可选，自动创建 CLB 时购买 CLB 的参数，参考 CreateLoadBalancer 接口: https://cloud.tencent.com/document/api/214/30692
      # 负载均衡实例的网络类型：OPEN：公网属性， INTERNAL：内网属性。默认使用 OPEN（公网负载均衡）。
      loadBalancerType: OPEN
      # 在私有网络内购买内网负载均衡实例的情况下，必须指定子网 ID，内网负载均衡实例的 VIP 将从这个子网中产生。创建内网负载均衡实例时，此参数必填，创建公网IPv4负载均衡实例时，不支持指定该参数。
      subnetId: subnet-k57djpow
      # 负载均衡实例的名称。规则：1-60 个英文、汉字、数字、连接线“-”或下划线“_”。 注意：如果名称与系统中已有负载均衡实例的名称相同，则系统将会自动生成此次创建的负载均衡实例的名称。
      loadBalancerName: test
      # 仅适用于公网负载均衡。目前仅广州、上海、南京、济南、杭州、福州、北京、石家庄、武汉、长沙、成都、重庆地域支持静态单线 IP 线路类型，如需体验，请联系商务经理申请。申请通过后，即可选择中国移动（CMCC）、中国联通（CUCC）或中国电信（CTCC）的运营商类型，网络计费模式只能使用按带宽包计费(BANDWIDTH_PACKAGE)。 如果不指定本参数，则默认使用BGP。可通过 DescribeResources 接口查询一个地域所支持的Isp。
      vipIsp: CTCC
      # 带宽包ID，指定此参数时，网络计费方式（InternetAccessible.InternetChargeType）只支持按带宽包计费（BANDWIDTH_PACKAGE），带宽包的属性即为其结算方式。非上移用户购买的 IPv6 负载均衡实例，且运营商类型非 BGP 时 ，不支持指定具体带宽包id。
      bandwidthPackageId: bwp-40ykow69
      # 性能容量型规格（不同地域的可选规格列表可能不一样，以 CLB 购买页面展示的列表为准）。
      # 若需要创建性能容量型实例，则此参数必填，取值范围：
      # clb.c2.medium：标准型规格
      # clb.c3.small：高阶型1规格
      # clb.c3.medium：高阶型2规格
      # clb.c4.small：超强型1规格
      # clb.c4.medium：超强型2规格
      # clb.c4.large：超强型3规格
      # clb.c4.xlarge：超强型4规格
      # 若需要创建共享型实例，则无需填写此参数。
      slaType: clb.c4.xlarge # 
      # 负载均衡实例计费类型，取值：POSTPAID_BY_HOUR，PREPAID，默认是POSTPAID_BY_HOUR。
      lbChargeType: POSTPAID_BY_HOUR
      # 仅适用于公网负载均衡。负载均衡的网络计费模式。
      internetAccessible:
        # TRAFFIC_POSTPAID_BY_HOUR 按流量按小时后计费 ; BANDWIDTH_POSTPAID_BY_HOUR 按带宽按小时后计费; BANDWIDTH_PACKAGE 按带宽包计费;BANDWIDTH_PREPAID按带宽预付费。注意：此字段可能返回 null，表示取不到有效值。
        internetChargeType: TRAFFIC_POSTPAID_BY_HOUR
        # 最大出带宽，单位Mbps，仅对公网属性的共享型、性能容量型和独占型 CLB 实例、以及内网属性的性能容量型 CLB 实例生效。
        # - 对于公网属性的共享型和独占型 CLB 实例，最大出带宽的范围为1Mbps-2048Mbps。
        # - 对于公网属性和内网属性的性能容量型 CLB实例，最大出带宽的范围为1Mbps-61440Mbps。
        # （调用CreateLoadBalancer创建LB时不指定此参数则设置为默认值10Mbps。此上限可调整）
        internetMaxBandwidthOut: 61440
        # 带宽包的类型，如 SINGLEISP（单线）、BGP（多线）。
        bandwidthpkgSubType: BGP
      # 仅适用于公网负载均衡。IP版本，可取值：IPV4、IPV6、IPv6FullChain，不区分大小写，默认值 IPV4。说明：取值为IPV6表示为IPV6 NAT64版本；取值为IPv6FullChain，表示为IPv6版本。
      addressIPVersion: IPv4
      # 负载均衡后端目标设备所属的网络 ID，如vpc-12345678，可以通过 DescribeVpcs 接口获取。 不填此参数则默认为当前集群所在 VPC。创建内网负载均衡实例时，此参数必填。
      vpcId: vpc-091t4l6w
      # 购买负载均衡的同时，给负载均衡打上标签，最大支持20个标签键值对。
      tags:
      - tagKey: tag-key # 标签的键
        tagValue: tag-value # 标签的值
      # 负载均衡实例所属的项目 ID，可以通过 DescribeProject 接口获取。不填此参数则视为默认项目。
      projectId: 0
      # 仅适用于公网且IP版本为IPv4的负载均衡。设置跨可用区容灾时的主可用区ID，例如 100001 或 ap-guangzhou-1
      # 注：主可用区是需要承载流量的可用区，备可用区默认不承载流量，主可用区不可用时才使用备可用区。目前仅广州、上海、南京、北京、成都、深圳金融、中国香港、首尔、法兰克福、新加坡地域的 IPv4 版本的 CLB 支持主备可用区。可通过 DescribeResources 接口查询一个地域的主可用区的列表。【如果您需要体验该功能，请通过 工单申请】
      masterZoneId: ap-guangzhou-1
      # 仅适用于公网且IP版本为IPv4的负载均衡。可用区ID，指定可用区以创建负载均衡实例。
      zoneId: ap-guangzhou-1
      # Target是否放通来自CLB的流量。开启放通（true）：只验证CLB上的安全组；不开启放通（false）：需同时验证CLB和后端实例上的安全组。默认值为 true。
      loadBalancerPassToTarget: true
      # 是否创建域名化负载均衡（CLB 地址是域名，没有固定的 VIP）。
      dynamicVip: false
```

> 更详细的 API 说明请参考 [API 参考](api.md#clbportpool)

## 使用约束

- 需使用原生节点或超级节点部署 Pod，否则将无法为 Pod 映射 (从事件日志中可以看到 warning 信息)。
- 部分场景需向 CLB 提工单开通特性（如端口段）以及调整配额（如单个 CLB 监听器的数量上限），可根据文中的指引进行操作。

## 创建 CLB 端口池

下面给出一些常见的 CLB 端口池示例。

1. 测试阶段，端口池使用内网 CLB 分配映射地址：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-internal
spec:
  startPort: 30000
  exsistedLoadBalancerIDs: [lb-04iq85jh] # 指定已有的内网 CLB
  autoCreate:
    enabled: true # 端口不足时自动创建内网 CLB
    parameters:
      loadBalancerType: INTERNAL # 指定内网 CLB 类型
      subnetId: subnet-k57djpow # 创建内网 CLB 必须指定子网 ID
```

2. 生产阶段，端口池使用高规格的性能容量型 CLB：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-prod
spec:
  startPort: 30000
  exsistedLoadBalancerIDs: [lb-04iq85jh] # 指定已有的 CLB
  autoCreate:
    enabled: true # 端口不足时自动创建 CLB
    parameters:
      slaType: clb.c4.xlarge #  clb.c4.xlarge：超强型4规格
      internetAccessible:
        internetChargeType: TRAFFIC_POSTPAID_BY_HOUR  # 按流量按小时后计费
        internetMaxBandwidthOut: 61440 # 最大出带宽 60 Gbps（不同规格的 CLB 的最大出带宽上限不一样，参考 https://cloud.tencent.com/document/product/214/84689）
```

3. 端口池使用指定运营商类型的 CLB：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-ctcc
spec:
  startPort: 30000
  autoCreate:
    maxLoadBalancers: 3 # 最多自动创建 3 个电信 CLB
    enabled: true
    parameters:
      vipIsp: CTCC # 使用电信运营商的 IP（单线 IP）
      bandwidthPackageId: bwp-40ykow69 # 指定电信类型的共享带宽包 ID（需提前创建，参考 https://cloud.tencent.com/document/product/684/39942）
```

## 使用注解为 Pod 分配 CLB 映射公网地址

在 Pod Template 中指定注解，声明从 CLB 端口池为 Pod 分配公网地址映射，可以是任意类型的工作负载，比如：
1. Kubernetes 自带的 Deployment、Statusfulset。
2. OpenKruise 的 Advanced Deployment 或 Advanced Statusfulset。
3. 开源的游戏专用工作负载，如 OpenKruiseGame 的 GameServerSet、Agones 的 Fleet。

Pod 注解配置方法：
1. 指定注解 `networking.cloud.tencent.com/enable-clb-port-mapping` 为 `true` 开启使用 CLB 端口池为 Pod 映射公网地址。
2. 指定注解 `networking.cloud.tencent.com/clb-port-mapping` 配置映射规则，比如 `8000 UDP pool-test`，其中 `8000` 表示 Pod 监听的端口号，`UDP` 表示端口协议（支持 TCP、UDP、TCP_SSL、QUIC 和 TCPUDP，其中 TCPUDP 表示该端口同时监听了 TCP 和 UDP），`pool-test` 表示 CLB 端口池名称，可指定多行来配置多个端口映射。

`StatefulSet` 配置示例：

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: gameserver
spec:
  selector:
    matchLabels:
      app: gameserver
  serviceName: gameserver
  replicas: 10
  template:
    metadata:
      annotations:
        networking.cloud.tencent.com/enable-clb-port-mapping: "true"
        networking.cloud.tencent.com/clb-port-mapping: |-
          8000 UDP pool-test
          7000 TCPUDP pool-test
      labels:
        app: gameserver
    spec:
      containers:
      - name: gameserver
        image: your-gameserver-image
```

`OpenKruiseGame` 的 `GameServerSet` 配置示例：

```yaml
apiVersion: game.kruise.io/v1alpha1
kind: GameServerSet
metadata:
  name: gameserver
spec:
  replicas: 10
  updateStrategy:
    rollingUpdate:
      podUpdatePolicy: InPlaceIfPossible
  gameServerTemplate:
    metadata:
      annotations:
        networking.cloud.tencent.com/enable-clb-port-mapping: "true"
        networking.cloud.tencent.com/clb-port-mapping: |-
          8000 UDP pool-test
          7000 TCPUDP pool-test
    spec:
      containers:
        - image: your-gameserver-image
          name: gameserver
```

`Agones` 的 `Fleet` 配置示例：

```yaml
apiVersion: agones.dev/v1
kind: Fleet
metadata:
  name: gameserver
spec:
  replicas: 10
  template:
    spec:
      template:
        metadata:
          annotations:
            networking.cloud.tencent.com/enable-clb-port-mapping: "true"
            networking.cloud.tencent.com/clb-port-mapping: |-
              7654 TCPUDP pool-test
        spec:
          containers:
          - name: gameserver
            image: your-gameserver-image
```

## TCP 和 UDP 同端口号接入

有些情况下，玩家的网络环境 UDP 可能无法正常工作，游戏客户端自动 fallback 到 TCP 协议进行通信。

用 CLB 端口池为 Pod 映射公网地址时，可以同时监听 TCP 和 UDP 协议，最终映射的公网地址 TCP 和 UDP 使用相同端口号。

配置方法是在 Pod 注解中指定端口协议时使用 `TCPUDP` 即可：

```yaml
annotations:
    networking.cloud.tencent.com/clb-port-mapping: |-
      8000 TCPUDP pool-test
```

映射效果如下：

![](images/tcpudp.jpg)

> Pod 的一个端口同时监听 TCP 和 UDP 协议，CLB 映射公网地址时，会分别使用 TCP 和 UDP 两个相同端口号的不同监听器进行映射。

自动生成的映射结果的注解示例如下：

```yaml
annotations:
    networking.cloud.tencent.com/clb-port-mapping-result: '[{"port":8000,"protocol":"TCP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-04iq85jh","loadbalancerPort":30170,"loadbalancerEndPort":30179,"listenerId":"lbl-bjoyr92j","endPort":8009,"hostname":"lb-04iq85jh-w49ru3xpmdynoigk.clb.cd-tencentclb.work"},{"port":8000,"protocol":"UDP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-04iq85jh","loadbalancerPort":30170,"loadbalancerEndPort":30179,"listenerId":"lbl-6dg9wfs5","endPort":8009,"hostname":"lb-04iq85jh-w49ru3xpmdynoigk.clb.cd-tencentclb.work"}]'
    networking.cloud.tencent.com/clb-port-mapping-status: Ready
```


## 大规模场景下的端口映射

### 大规模场景下存在的问题

为了给某个 Pod 监听的某个端口映射出唯一的公网地址，可以使用 CLB 的监听器只绑 1 个 Pod，这样每个 Pod 都被绑定到不同的 CLB 监听器上，Pod 独占 CLB 监听器，所有 Pod 最终在 CLB 上对应的公网地址也就不会冲突：

![](images/dedicated-listener-for-pod.jpg)

但在大规模场景下，需要消耗的 CLB 监听器数量会非常大，比如 1000 Pod 需要 1000 个 CLB 监听器，而 CLB 默认的监听器配额上限为 50，（参考[CLB 使用约束](https://cloud.tencent.com/document/product/214/6187) 中的 `一个实例可添加的监听器数量`），完成 1000 个 Pod 的端口映射就需要 20 个 CLB，费用和管理成本都非常高。

如何优化？请看下面几种方案。

### 方案一：提升 CLB 监听器的配额

根据需求 [提工单](https://console.cloud.tencent.com/workorder/category?level1_id=6&level2_id=163&source=14&data_title=%E8%B4%9F%E8%BD%BD%E5%9D%87%E8%A1%A1&step=1) 申请提升 CLB `一个实例可添加的监听器数量` 的配额限制（调整方法参考 FAQ 中的 [如何提升 CLB 的监听器数量配额？](#如何提升-clb-的监听器数量配额))。

配额调整分为账号维度和实例维度，如果希望调整到很大（比如 2000），通常只能在实例维度调整。如果在实例维度调整，在定义端口池时只能添加已有 CLB 实例的方式（不能启用自动创建），且需要手动指定下 `listenerQuota` 的值，与申请到的配额需一致：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-quota
spec:
  startPort: 30000
  exsistedLoadBalancerIDs: [lb-cxxc6xup, lb-mq3rs6h9] # 指定已申请调整配额的 CLB 实例
  listenerQuota: 2000 # 指定调整后的配额值
```

另外需要注意的是，用多端口池映射时（多线接入场景），每个端口池配置的 `listenerQuota` 值必须一致。

如果调整的是账号维度的配额，则没有上述约束，也无需显式配置 `listenerQuota`。

### 方案二：单 Pod 多 DS + CLB 端口段

CLB 支持端口段类型的监听器，可以实现用一个监听器映射连续的多个端口，比如创建了一个CLB 监听器，它的端口是 `30000~30009`，给它绑定一个后端 Pod，端口为 `8000`，那么该 CLB 的 `30000~30009` 端口将映射到 这个 Pod 的 `8000~8009` 端口。再具体一点，这个 CLB 的 30000 端口的流量将转发给这个 Pod 的 8000 端口，30001 端口的流量将转发给这个 Pod 的 8001 端口，以此类推：

![](images/clb-port-range-for-pod.jpg)

单个 Pod 中可以运行多个 DS（DedicatedServer），每个 DS 监听的端口号不一样，但它们是连续的，比如运行 10 个 DS，一共监听 `8000~8009` 这 10 个端口。这样，运行 1000 个 DS 只需要 100 个 Pod，消耗 100 个 CLB 监听器。

具体如何配置呢？参考以下方法。

1. 首先，CLB 端口段特性需通过 [工单申请](https://console.cloud.tencent.com/workorder/category?level1_id=6&level2_id=163&source=14&data_title=%E8%B4%9F%E8%BD%BD%E5%9D%87%E8%A1%A1&step=1) 开通使用。

2. 然后，在定义端口池时指定端口段长度（`segmentLength`），指定后，分配的 CLB 监听器会使用端口段，即 1 个监听器映射后端连续的 `segmentLength` 个端口，示例：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-port-range
spec:
  startPort: 30000
  segmentLength: 10 # 端口段长度，10 表示 1 个 CLB 监听器映射 Pod 中连续的 10 个端口，即 10 个 DS 的地址
  exsistedLoadBalancerIDs: [lb-04iq85jh]
  autoCreate:
    enabled: true
```

3. 最后，在 Pod 注解中引用该端口池即可：

```yaml
networking.cloud.tencent.com/enable-clb-port-mapping: "true"
networking.cloud.tencent.com/clb-port-mapping: |-
  7000 UDP pool-port-range
```

> 单 Pod 多 DS 场景下 Pod 规格通常较大，推荐使用超级节点以获得更好的弹性和成本优势。

### 方案三：HostPort + CLB 端口段

使用方案二在一定程度上可以解决大规模场景 CLB 监听器不够用的问题，但存在以下限制：
1. 需游戏开发自行实现在单个 Pod 内管理多个 DS 进程，每个 DS 监听器一个端口，且端口需连续。
2. 通常单个 Pod 内能运行的 DS 进程数量不会太多（比如 10 个以内），如果在规模非常大的情况下（比如上万个 DS），仍然需要消耗大量的 CLB 监听器数量。

使用本插件还可以利用 CLB 端口段 + HostPort 来实现大规模单 Pod 单 DS 的端口映射：

![](images/port-range-for-hostport.jpg)

解释：

1. Pod 通过 HostPort 暴露端口，此 HostPort 是在节点上某个端口区间自动分配（Kubernetes 自身不支持 HostPort 随机分配，参考 [这个 issue](https://github.com/kubernetes/kubernetes/issues/49792)，需借助第三方工作负载类型来支持该能力，如 Agones 的 Fleet、OpenKruiseGame 的 GameServerSet）。
2. 插件为 CLB 创建端口段监听器并绑定节点，端口段区间大小通常为 HostPort 自动分配的端口范围大小。
3. Pod 调度到节点，插件根据 Pod 所调度到的节点被 CLB 端口段监听器的绑定情况和 Pod 被分配的 HostPort，自动计算出 Pod 在 CLB 中对外映射的端口号，完成映射。

相比之下，方案二是 1 个端口段监听器映射 1 个 Pod 中所有 DS 监听器端口，而方案三则是 1 个端口段监听器映射 1 个节点中所有 Pod 的 DS 监听的端口，因此在使用相同监听器数量的情况下可映射的 Pod 数量方案三远大于方案二，不过也会带来一些限制：
1. **必须**使用**原生节点**调度 Pod（需使用 HostPort，而超级节点是虚拟节点，没有 HostPort）。
2. **必须**使用支持 HostPort 动态分配的工作负载类型（Agones 的 Fleet 和 OpenKruiseGame 的 GameServerSet）。
3. 如果需要实现 Pod 的端口同时用 TCP 和 UDP 协议对外暴露且保持端口号相同，还依赖动态分配 HostPort 的工作负载类型支持分配 TCP 和 UDP 相同端口号的 HostPort，目前已知 Agones 支持（定义 port 的 protocol 时指定 `TCPUDP`，参考 [`#1532`](https://github.com/googleforgames/agones/issues/1523)），OpenKruiseGame 在 `v1.0` 之后支持（也是在定义 port 的 protocol 时指定 `TCPUDP`，参考 [`#244`](https://github.com/openkruise/kruise-game/pull/244)）。

具体如何配置呢？参考以下方法。

> **注意**：由于需要使用 HostPort，而超级节点没有 HostPort，所以这种方式不支持超级节点。

1. 首先需要选择支持自动分配 HostPort 的工作负载，Agones 的 Fleet 和 OpenKruiseGame 的 GameServerSet 都可以支持，Agones 默认会为每个 GameServer 的 Pod 分配 `7000~8000` 的 HostPort，而 OpenKruiseGame 默认是 `8000~9000`（每个节点共 1001 个 HostPort 可能被分配给 GameServer）。

2. 然后创建一个端口池，`segmentLength` 为 1001（每个 CLB 端口段的监听器都可以映射一个节点的所有 GameServer 可能使用的 HostPort）：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-test
spec:
  startPort: 30000
  segmentLength: 1001
  autoCreate:
    enabled: true
```

3. 使用 TKE 节点池创建节点，并为节点池中所有节点配置注解，启用 CLB 端口映射并指定映射规则，可直接编辑节点池来配置（Node 注解与 Pod 注解配置格式完全一致，Pod 注解用于 CLB 绑定 Pod，Node 注解用于 CLB 绑定 Node）：

![](images/node-pool-annotation.png)

注解示例：

```yaml
networking.cloud.tencent.com/enable-clb-port-mapping: "true"
networking.cloud.tencent.com/clb-port-mapping: "7000 TCPUDP pool-test2"
```

> 端口号为与工作负载分配 HostPort 范围的最小端口号，Agones 默认是 7000，OpenKruiseGame 默认是 8000。

4. 使用选择的工作负载类型来部署游戏服，声明需要的端口和协议配置，并为 Pod 指定注解 `networking.cloud.tencent.com/enable-clb-hostport-mapping: "true"` 以启用根据 Pod 所在节点 HostPort 被映射的 CLB 地址自动回写到 Pod 注解。
    - 如果是 Agones 的 Fleet，需声明监听的端口（假设同时监听了 TCP 和 UDP）：
      ```yaml
      apiVersion: agones.dev/v1
      kind: Fleet
      metadata:
        name: simple-game-server
      spec:
        replicas: 5
        template:
          spec:
            ports:
              protocol: TCPUDP # 指定 TCPUDP 可保证分配的 TCP 和 UDP 两个 HostPort 端口号相同
              containerPort: 7654
            template:
              metadata:
                annotations:
                  networking.cloud.tencent.com/enable-clb-hostport-mapping: "true"
              spec:
                containers:
                - name: gameserver
                  image: docker.io/imroc/simple-game-server:0.36
      ```
    - 如果是 OpenKruiseGame 的 GameServerSet，需声明使用 `Kubernetes-HostPort` 网络插件，并在 `ContainerPorts` 参数中声明要分配 HostPort 的容器端口和协议（假设同时监听了 TCP 和 UDP）：
      ```yaml
      apiVersion: game.kruise.io/v1alpha1
      kind: GameServerSet
      metadata:
        name: nginx
      spec:
        replicas: 5
        updateStrategy:
          rollingUpdate:
            podUpdatePolicy: InPlaceIfPossible
        network:
          networkType: Kubernetes-HostPort
          networkConf:
          - name: ContainerPorts
            value: "nginx:80/TCP,80/UDP"
        gameServerTemplate:
          metadata:
            annotations:
              networking.cloud.tencent.com/enable-clb-hostport-mapping: "true"
          spec:
            containers:
            - image: nginx:latest
              name: nginx
              ports:
              - containerPort: 80
                protocol: UDP
              - containerPort: 80
                protocol: TCP
      ```

5. 最后，在 Pod 注解 `networking.cloud.tencent.com/clb-hostport-mapping-result` 可以看到被映射的 CLB 地址（容器内获取可通过 Downward API 挂载）：

```yaml
networking.cloud.tencent.com/clb-hostport-mapping-result: '[{"containerPort":7654,"hostPort":7106,"protocol":"UDP","pool":"pool-test2","region":"ap-chengdu","loadbalancerId":"lb-e9bt8x65","loadbalancerPort":31107,"listenerId":"lbl-rvunrb65"},{"containerPort":7654,"hostPort":7157,"protocol":"TCP","pool":"pool-test2","region":"ap-chengdu","loadbalancerId":"lb-e9bt8x65","loadbalancerPort":31107,"listenerId":"lbl-av86rekp"}]'
networking.cloud.tencent.com/clb-hostport-mapping-status: Ready
networking.cloud.tencent.com/enable-clb-hostport-mapping: "true"
```

### 其它注意事项

使用 `tke-extend-network-controller` 的默认部署配置，没有指定 request 和 limit，在大规模场景下，`tke-extend-network-controller` 消耗的资源也更多。如果调度到超级节点，默认 Pod 规格是 1C2G；如果调度到非超级节点上，实际可使用的资源也无法确定。

所以建议在规模较大的场景下一定根据实际情况合理设置 request 和 limit（对应 `values.yaml` 中的 `resources` 字段），具体大小可根据实际压测结合监控数据确定。

## 多线接入场景使用多端口池映射

### 为什么要使用多线接入

CLB 默认使用 BGP 多运营商接入，带宽成本较高，游戏场景通常要消耗巨大的带宽资源，为节约成本，可以考虑使用 CLB 的单线 IP，通过多个不同运营商 IP 的 CLB 来实现多线接入（电信玩家连上电信 CLB，联通玩家连上联通 CLB，移动玩家连上移动 CLB，其它的 fallback 到 BGP CLB），这样可以节约大量带宽成本。

### 多线接入场景的配置方法

下面介绍配置方法，首先创建多个端口池，一个运营商一个端口池，分别为电信、联通、移动创建各自的 CLB 端口池，另外再创建一个 BGP 类型的端口池：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-ctcc # 电信 CLB 端口池
spec:
  startPort: 30000
  exsistedLoadBalancerIDs: [lb-04i895jh, lb-04i87jjk] # 指定已有的电信 CLB 实例 ID，可动态追加
  autoCreate:
    enabled: true # 电信 CLB 端口不足时自动创建电信 CLB
    parameters: # 指定电信 CLB 创建参数
      vipIsp: CTCC # 指定运营商为电信
      bandwidthPackageId: bwp-40ykow69 # 指定电信带宽包 ID
---
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-cmcc # 移动 CLB 端口池
spec:
  startPort: 30000
  exsistedLoadBalancerIDs: [lb-jjgsqldb, lb-08jk7hh] # 指定已有的移动 CLB 实例 ID，可动态追加
  autoCreate:
    enabled: true # 移动 CLB 端口不足时自动创建电信 CLB
    parameters: # 指定移动 CLB 创建参数
      vipIsp: CMCC # 指定运营商为移动
      bandwidthPackageId: bwp-97yjlal5 # 指定移动带宽包 ID
---
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-cucc # 联通 CLB 端口池
spec:
  startPort: 30000
  exsistedLoadBalancerIDs: [lb-cxxc6xup, lb-mq3rs6h9] # 指定已有的联通 CLB 实例 ID，可动态追加
  autoCreate:
    enabled: true # 联通 CLB 端口不足时自动创建电信 CLB
    parameters: # 指定联通 CLB 创建参数
      vipIsp: CUCC # 指定运营商为联通
      bandwidthPackageId: bwp-97yjlal5 # 指定联通带宽包 ID
---
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-bgp # BGP CLB 端口池
spec:
  startPort: 30000
  exsistedLoadBalancerIDs: [lb-cx8c6xxa, lb-mq8rs9hk] # 指定已有的 BGP CLB 实例 ID，可动态追加
  autoCreate:
    enabled: true # 不指定运营商，默认创建 BGP 类型的 CLB
```

然后在 Pod 注解中配置端口映射，将这些端口池都加上去，表示从每个端口池都各自分配一个公网映射：

```yaml
metadata:
  annotations:
    networking.cloud.tencent.com/enable-clb-port-mapping: "true"
    networking.cloud.tencent.com/clb-port-mapping: |-
      8000 TCPUDP pool-ctcc,pool-cmcc,pool-cucc,pool-bgp useSamePortAcrossPools
```

### 映射效果与解释

映射效果如下：

![](images/multi-isp.jpg)

解释：

1. Pod 端口同时监听 TCP 和 UDP，映射规则中的协议指定为 `TCPUDP`，CLB 映射公网地址时，会分别使用 TCP 和 UDP 两个相同端口号的不同监听器进行映射。
2. 使用多个端口池进行映射，用逗号隔开，每个端口池分别都会为 Pod 映射各自公网地址。
3. 追加 `useSamePortAcrossPools` 选项表示同一个 Pod 在所有端口池中分配到的端口号相同。可用于简化游戏客户端的连接游戏服务端公网地址的 fallback 逻辑（只需决定连接哪个 IP，不需要关心不同 IP 连接不同端口的情况）。
4. 综上，最终每个 Pod 的每个端口会被映射四个公网地址，算上 TCP 和 UDP 同时监听，每个 Pod 端口使用 8 个 CLB 监听器映射公网地址。
5. 每个游戏服既同时提供 TCP 和 UDP 协议，又同时提供多个 ISP 的公网地址，游戏客户端可根据玩家网络环境实现灵活的自动 fallback 能力：玩家的游戏客户端优先连上当前网络运营商对应的 CLB 映射地址以节约带宽成本，如果是其它运营商，再自动 fallback 到通用的 BGP CLB 地址；如果玩家的网络环境 UDP 无法正常工作，游戏客户端再自动 fallback 到 TCP 协议进行通信。

### 大规模场景下的问题

这种多线接入场景如果直接用 CLB 监听器绑定 Pod，消耗的 CLB 监听器数量会非常大，比如上面的例子中，一个端口同时监听 TCP 和 UDP，每个端口池要分 2 个 CLB 监听器，4 个端口池就要分 8 个监听器才能完成映射；如果 Pod 中有 2 个这样的端口就要消耗 16 个监听器才能完成 1 个 Pod 的端口映射。

### 使用 CLB 端口段 + HostPort 优化大规模场景下的多线接入

如何解决多线接入场景下 CLB 监听器数量消耗过多的问题？可参考 [大规模场景下的端口映射方案三：HostPort + CLB 端口段](#使用-clb-端口段--hostport-优化大规模场景下的多线接入) （前提是接受其中提到的限制）。

如果要保证 TCP 和 UDP 对外同端口号暴露，目前只有使用 Agones 的 Fleet 来部署 GameServer （只有 Agones 支持分配同端口号的 TCP + UDP 的 HostPort）。

以下是配置方法：

1. 创建多个端口池，与前面 **多线接入场景的配置方法** 中的方法基本相同，唯一不同的是要指定 `segmentLength`，表示分配 CLB 监听器时使用端口段监听器，且端口段大小为 1001：
  ```yaml
  apiVersion: networking.cloud.tencent.com/v1alpha1
  kind: CLBPortPool
  metadata:
    name: pool-ctcc # 电信 CLB 端口池
  spec:
    startPort: 30000
    segmentLength: 10001
    exsistedLoadBalancerIDs: [lb-04i895jh, lb-04i87jjk] # 指定已有的电信 CLB 实例 ID，可动态追加
    autoCreate:
      enabled: true # 电信 CLB 端口不足时自动创建电信 CLB
      parameters: # 指定电信 CLB 创建参数
        vipIsp: CTCC # 指定运营商为电信
        bandwidthPackageId: bwp-40ykow69 # 指定电信带宽包 ID
  ---
  apiVersion: networking.cloud.tencent.com/v1alpha1
  kind: CLBPortPool
  metadata:
    name: pool-cmcc # 移动 CLB 端口池
  spec:
    startPort: 30000
    segmentLength: 10001
    exsistedLoadBalancerIDs: [lb-jjgsqldb, lb-08jk7hh] # 指定已有的移动 CLB 实例 ID，可动态追加
    autoCreate:
      enabled: true # 移动 CLB 端口不足时自动创建电信 CLB
      parameters: # 指定移动 CLB 创建参数
        vipIsp: CMCC # 指定运营商为移动
        bandwidthPackageId: bwp-97yjlal5 # 指定移动带宽包 ID
  ---
  apiVersion: networking.cloud.tencent.com/v1alpha1
  kind: CLBPortPool
  metadata:
    name: pool-cucc # 联通 CLB 端口池
  spec:
    startPort: 30000
    segmentLength: 10001
    exsistedLoadBalancerIDs: [lb-cxxc6xup, lb-mq3rs6h9] # 指定已有的联通 CLB 实例 ID，可动态追加
    autoCreate:
      enabled: true # 联通 CLB 端口不足时自动创建电信 CLB
      parameters: # 指定联通 CLB 创建参数
        vipIsp: CUCC # 指定运营商为联通
        bandwidthPackageId: bwp-97yjlal5 # 指定联通带宽包 ID
  ---
  apiVersion: networking.cloud.tencent.com/v1alpha1
  kind: CLBPortPool
  metadata:
    name: pool-bgp # BGP CLB 端口池
  spec:
    startPort: 30000
    segmentLength: 10001
    exsistedLoadBalancerIDs: [lb-cx8c6xxa, lb-mq8rs9hk] # 指定已有的 BGP CLB 实例 ID，可动态追加
    autoCreate:
      enabled: true # 不指定运营商，默认创建 BGP 类型的 CLB
  ```
3. 使用原生节点池，为节点配置以下注解 (Agones 分配的 HostPort 默认区间是7000-8000，所以这里要绑定的 Node 端口是 7000，结合端口池中配置的 segmentLength 为 1001 可覆盖 7000-8000 范围的 HostPort 映射)：
  ```yaml
  networking.cloud.tencent.com/enable-clb-port-mapping: "true"
  networking.cloud.tencent.com/clb-port-mapping: "7000 TCPUDP pool-ctcc,pool-cmcc,pool-cucc,pool-bgp useSamePortAcrossPools"
  ```
3. 使用 Agones Fleet 部署 GameServer，声明端口时，协议指定 `TCPUDP` 以便让 Agones 分配同端口号的 TCP 和 UDP 两个 HostPort 端口号；并加 Pod 注解 `networking.cloud.tencent.com/enable-clb-hostport-mapping: "true"` 声明开启 CLB + HostPort 端口映射：
  ```yaml
  apiVersion: agones.dev/v1
  kind: Fleet
  metadata:
    name: simple-game-server
  spec:
    replicas: 5
    scheduling: Packed
    template:
      spec:
        ports:
        - containerPort: 7654
          protocol: TCPUDP # 指定 TCPUDP 可保证分配的 TCP 和 UDP 两个 HostPort 端口号相同
        template:
          metadata:
            annotations:
              networking.cloud.tencent.com/enable-clb-hostport-mapping: "true" # 开启 CLB + HostPort 端口映射
          spec:
            containers:
            - image: docker.io/imroc/simple-game-server:0.36
              name: gameserver
  ```

以下是操作演示：


[![](https://image-host-1251893006.cos.ap-chengdu.myqcloud.com/videos/agones-clb-hostport-mapping.png)](https://image-host-1251893006.cos.ap-chengdu.myqcloud.com/videos/agones-clb-hostport-mapping.mp4)

## 内网 CLB 绑 EIP 映射

在某些场景下，可能需要使用内网 CLB 支持绑定 EIP 这种方式来为 Pod 进行端口映射。

> 内网 CLB 绑定弹性公网 IP 功能处于内测阶段，如需使用，请提交 [工单申请](https://console.cloud.tencent.com/workorder/category?level1_id=6&level2_id=660&source=0&data_title=%E5%BC%B9%E6%80%A7%E5%85%AC%E7%BD%91%20IP&step=1)。

用法很简单，可参考 [内网负载均衡实例绑定 EIP](https://cloud.tencent.com/document/product/214/65682) 准备好 CLB 实例，然后通过添加已有 CLB 的方式加入将其端口池即可：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-eip
spec:
  startPort: 30000
  exsistedLoadBalancerIDs:  # 将绑定了 EIP 的内网 CLB 实例 ID 添加到这里，可动态追加
  - lb-04iq85jh
```

> 控制器会自动检测内网 CLB 是否绑定了 EIP，如果绑定 EIP 就认为此 CLB 的 VIP 为绑定的 EIP，映射结果也会使用 EIP 地址。

## 使用 TLS

有些场景可能会用到 TLS，比如 websocket H5 小游戏，这时你可以根据需求使用 CLB 的 TCP_SSL 或 QUIC 协议来接入，下面介绍配置方法。

1. 首先，在 [证书管理](https://console.cloud.tencent.com/clb/certificate) 创建好证书并复制证书 ID。
2. 然后，在要为 Pod 映射端口的命名空间中创建一个 Secret，写入 `qcloud_cert_id` 字段，值为证书 ID：

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cert-secret
type: Opaque
stringData:
  qcloud_cert_id: "O6TkzGNJ"
```

> 创建 Secret 时，如果数据 `stringData` 保存，证书 ID 的值无需 base64 编码；用 `data` 保存则需要手动进行 base64 编码后再写入。

3. 最后，在声明端口映射的注解中指定使用相应的协议，并指定包含证书 ID 的 Secret 名称：

```yaml
networking.cloud.tencent.com/enable-clb-port-mapping: "true"
networking.cloud.tencent.com/clb-port-mapping: |-
  8000 TCP_SSL pool-test certSecret=cert-secret
```

> `certSecret` 选项表示要挂载的证书 Secret 名称，Secret 中必须包含 `qcloud_cert_id` 字段，值为证书 ID。

## 通过 Downward API 获取 Pod 映射公网地址

当配置好 Pod 注解后，会为 Pod 自动分配 CLB 公网地址的映射，并将映射的结果写到 Pod 的 `networking.cloud.tencent.com/clb-port-mapping-result` 注解中：

```yaml
metadata:
  annotations:
    networking.cloud.tencent.com/enable-clb-port-mapping: "true"
    networking.cloud.tencent.com/clb-port-mapping: 80 TCPUDP pool-test
    networking.cloud.tencent.com/clb-port-mapping-status: Ready
    networking.cloud.tencent.com/clb-port-mapping-result: '[{"port":80,"protocol":"TCP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-6phn6qgb","loadbalancerPort":30000,"listenerId":"lbl-qvjvdt9n","ips":["111.231.210.160"],"address":"111.231.210.160:30000"},{"port":80,"protocol":"UDP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-6phn6qgb","loadbalancerPort":30000,"listenerId":"lbl-bbxvltnv","ips":["111.231.210.160"],"address":"111.231.210.160:30000"}]'
```

如果用 [大规模场景下的端口映射方案三：HostPort + CLB 端口段](#使用-clb-端口段--hostport-优化大规模场景下的多线接入) 的方式映射，映射结果会写到 Pod 的 `networking.cloud.tencent.com/clb-hostport-mapping-result` 注解中：

```yaml
metadata:
  annotations:
    networking.cloud.tencent.com/enable-clb-hostport-mapping: "true"
    networking.cloud.tencent.com/clb-hostport-mapping-result: '[{"containerPort":80,"hostPort":8344,"protocol":"TCP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-cb92uxex","loadbalancerPort":30344,"listenerId":"lbl-knmgdwb1","address":"111.231.211.104:30344","ips":["111.231.211.104"]},{"containerPort":80,"hostPort":8713,"protocol":"UDP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-cb92uxex","loadbalancerPort":30713,"listenerId":"lbl-i4n8f78h","address":"111.231.211.104:30713","ips":["111.231.211.104"]}]'
    networking.cloud.tencent.com/clb-hostport-mapping-status: Ready
```

映射结果为 JSON 数组，每个元素代表一个端口，具体字段参考上面给出的示例。

如果 CLB 是用的域名化的 CLB（只有 CLB 域名，没有 VIP），会有 `hostname` 字段替代 `ips` 字段，表示的 CLB 域名，示例：

```yaml
networking.cloud.tencent.com/clb-port-mapping-result: '[{"port":80,"protocol":"TCP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-6phn6qgb","loadbalancerPort":30000,"listenerId":"lbl-qvjvdt9n","hostname":"lb-6phn6qgb-8gcf1q6kd092nxsl.clb.cd-tencentclb.work","address":"lb-6phn6qgb-8gcf1q6kd092nxsl.clb.cd-tencentclb.work:30000"},{"port":80,"protocol":"UDP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-6phn6qgb","loadbalancerPort":30000,"listenerId":"lbl-bbxvltnv","hostname":"lb-6phn6qgb-8gcf1q6kd092nxsl.clb.cd-tencentclb.work","address":"lb-6phn6qgb-8gcf1q6kd092nxsl.clb.cd-tencentclb.work:30000"}]'
networking.cloud.tencent.com/clb-hostport-mapping-result: '[{"containerPort":80,"hostPort":8344,"protocol":"TCP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-cb92uxex","loadbalancerPort":30344,"listenerId":"lbl-knmgdwb1","address":"lb-cb92uxex-8gcf1q6kd092nxsl.clb.cd-tencentclb.work:30344","hostname":"lb-cb92uxex-8gcf1q6kd092nxsl.clb.cd-tencentclb.work"},{"containerPort":80,"hostPort":8713,"protocol":"UDP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-cb92uxex","loadbalancerPort":30713,"listenerId":"lbl-i4n8f78h","address":"lb-cb92uxex-8gcf1q6kd092nxsl.clb.cd-tencentclb.work:30713","hostname":"lb-cb92uxex-8gcf1q6kd092nxsl.clb.cd-tencentclb.work"}]'
```

可以将注解的内容通过 Downward API 挂载到容器中，然后在容器中读取注解内容，获取 Pod 映射的公网地址：

```yaml
spec:
  containers:
    - name: gameserver
      image: your-gameserver-image
      volumeMounts:
        - name: podinfo
          mountPath: /etc/podinfo
  volumes:
    - name: podinfo
      downwardAPI:
        items:
          - path: "clb-mapping"
            fieldRef:
              fieldPath: metadata.annotations['networking.cloud.tencent.com/clb-port-mapping-result']
```

同理，如果用 [大规模场景下的端口映射方案三：HostPort + CLB 端口段](#使用-clb-端口段--hostport-优化大规模场景下的多线接入) 的方式映射，由于注解名称不同，需微调下 downwardAPI：

```yaml
spec:
  containers:
    - name: gameserver
      image: your-gameserver-image
      volumeMounts:
        - name: podinfo
          mountPath: /etc/podinfo
  volumes:
    - name: podinfo
      downwardAPI:
        items:
          - path: "clb-mapping"
            fieldRef:
              fieldPath: metadata.annotations['networking.cloud.tencent.com/clb-hostport-mapping-result']
```

进程启动时可轮询指定文件（本例中文件路径为 `/etc/podinfo/clb-mapping`），当文件内容为空说明此时 Pod 还未绑定到 CLB，当读取到内容时说明已经绑定成功，其内容格式为 JSON 数组，每个元素代表一个端口映射，内容格式可参考前面给出的注解示例。

如果希望保证在业务容器在启动前完成 CLB 的端口映射，可利用 `initContainers` 来轮询检测该文件中的内容，直到有文件内容后退出，然后才会拉起业务容器。`initContainers` 示例：

```yaml
spec:
  initContainers:
  - name: get-clb-mapping
    image: busybox
    imagePullPolicy: IfNotPresent
    command: ["sh", "-c"]
    args:
    - |
      while true; do
        if [[ -e /etc/podinfo/clb-mapping ]]; then
            mapping=$(cat /etc/podinfo/clb-mapping);
            if [ $mapping ];then
                echo "found clb mapping: ${mapping}";
                break;
            else
                echo "wait clb mapping";
            fi;
        else
            echo "clb mapping file not found";
        fi;
        sleep 1;
      done;
    volumeMounts:
    - mountPath: /etc/podinfo
      name: podinfo
  volumes:
    - name: podinfo
      downwardAPI:
        items:
          - path: "clb-mapping"
            fieldRef:
              fieldPath: metadata.annotations['networking.cloud.tencent.com/clb-port-mapping-result']
```

## FAQ

### 如何提升 CLB 的监听器数量配额？

1. [提工单](https://console.cloud.tencent.com/workorder/category?level1_id=6&level2_id=163&source=14&data_title=%E8%B4%9F%E8%BD%BD%E5%9D%87%E8%A1%A1&step=1) 到负载均衡。
2. **问题类型** 选 **配额/白名单**。
3. 点击 **创建工单**。
4. **问题描述** 中填写：
  - 如果是调整账号维度的配额，内容填写：`申请调整负载均衡"一个实例可添加的监听器数量"的配额到 XX`（XX 为期望的监听器数量，可预估一个期望值，然后 CLB 侧会评估是否可以调整）。
  - 如果是调整实例维度的配额，内容填写：`申请调整以下 CLB 实例的"一个实例可添加的监听器数量"的配额到 XX：lb-xxx lb-yyy`（同理，XX 为期望的监听器数量，可预估一个期望值，实例维度的配额更容易调整，相比账号维度的配额能够调到更高；最后附上需要调整配额的实例 ID 列表）。
5. 提交工单。

如果本身有腾讯云工作人员对接您，也可以直接联系他们进行调整。

需要注意的是，如果调整的是实例维度的监听器数量配额，务必确保对应的端口池中所有的 CLB 实例都做了相同的配额调整，且需要显式指定 `listenerQuota` 字段，值为调整后的监听器数量配额。

### 端口池自动扩容 CLB 的条件是什么？

如果端口池配置了自动创建 CLB（`spec.clb.autoCreate` 为 `true`），则会在可分配的 CLB 监听器数量不足时自动创建新的 CLB 实例并加入端口池。

可分配的 CLB 监听器数量不足的条件又是什么？是端口池中所有 CLB 实例的分配监听器数量均小于 2 的时候。

那可分配的 CLB 监听器数量是多少呢？是 CLB 的监听器数量配额减去已分配的监听器数量。 CLB 的监听器数量配额默认是 50（参考 [CLB 使用约束](https://cloud.tencent.com/document/product/214/6187) 中的 `一个实例可添加的监听器数量`）；已分配的监听器数量可通过查看 CLBPortPool 对象中 status 里的 `allocated` 字段：`kubectl get clbportpool xxx -o yaml`。

为什么是监听器数量小于 2 时扩容？因为 `TCPUDP` 协议一个端口会消耗 2 个监听器（2 个相同端口号的监听器，一个 TCP 协议，一个 UDP 协议）如果数量小于 1 才扩容，可能导致无法扩容。

## 视频教程（更新中）

以下是相关视频教程，可点击封面跳转播放，持续更新中。

### tke-extend-network-controller 介绍与快速上手

[![](https://i1.hdslb.com/bfs/archive/a1710675e075f3daadeb5df5dfde1e5180c02580.jpg)](https://www.bilibili.com/video/BV1RVMSzKE19/?share_source=copy_web&vd_source=471575ae03d431e29c4355f21f275ab4)

### 在 TKE 使用 Agones 部署游戏服并通过 CLB 为每个游戏服映射独立的公网地址

[![](https://i1.hdslb.com/bfs/archive/74870cc81a785ebdad7936c0dcdfac77efabbefc.jpg)](https://www.bilibili.com/video/BV1vaMAzqEGL/?share_source=copy_web&vd_source=471575ae03d431e29c4355f21f275ab4)

### 在 TKE 使用 Agones 大规模部署游戏服时如何用CLB映射地址？

[![](https://i1.hdslb.com/bfs/archive/63623361abe6917c8633b6338b887ef5b3c0d545.jpg)](https://www.bilibili.com/video/BV1odMkzWEDu/?share_source=copy_web&vd_source=471575ae03d431e29c4355f21f275ab4)

### LB 分配策略和 LB 黑名单功能演示

[![](https://i2.hdslb.com/bfs/archive/b9df61fcaccd59b4e4cf1aee4e96408497038e78.jpg)](https://www.bilibili.com/video/BV11zK8zfET7/?vd_source=aec72c2053794e8a3e75e07931c8b440)

## CRD 字段参考

关于 CRD 字段的详细说明，请参考 [API 参考](./api.md)。
