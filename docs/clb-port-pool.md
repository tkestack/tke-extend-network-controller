# 使用 CLB 端口池为 Pod 映射公网地址（预览阶段）

## 抢先体验

本文中的功能将在 2.0.0 版本中正式发布，目前处于预览阶段，可通过以下 helm 安装方式测试和体验：

```bash
```bash
helm repo add tke-extend-network-controller https://tkestack.github.io/tke-extend-network-controller
helm upgrade --install --devel -f values.yaml \
  --namespace tke-extend-network-controller --create-namespace \
  tke-extend-network-controller tke-extend-network-controller/tke-extend-network-controller
```

`--devel` 参数很重要，其中 `values.yaml` 需配置好以下必要的参数:

```yaml
vpcID: "" # TKE 集群所在 VPC ID (vpc-xxx)
region: "" # TKE 集群所在地域，如 ap-guangzhou
clusterID: "" # TKE 集群 ID (cls-xxx)
secretID: "" # 腾讯云子账号的 SecretID
secretKey: "" # 腾讯云子账号的 SecretKey
```

## 创建端口池 (CLBPortPool)

使用 CLB 为 Pod 分配公网地址映射，需要先创建端口池，每个端口池对应一组相同属性的 CLB，可以动态追加已有 CLB 实例 ID，也可以在端口不足时自动创建新的 CLB。

通过创建 CLBPortPool 这个自定义资源来声明端口池，示例：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-test
spec:
  startPort: 30000 # 端口池中 CLB 起始端口号
  exsistedLoadBalancerIDs: [lb-04iq85jh] # 指定已有的 CLB 实例 ID，可动态追加
  autoCreate:
    enabled: true # 是否启用在 CLB 端口不足时自动创建 CLB
```

## 指定 Pod 注解映射公网地址

在 Pod Template 中指定注解，声明从 CLB 端口池为 Pod 分配公网地址映射，可以是任意类型的工作负载，比如：
1. Kubernetes 自带的 Deployment、Statusfulset。
2. OpenKruise 的 Advanced Deployment 或 Advanced Statusfulset。
3. 开源的游戏专用工作负载，如 OpenKruiseGame 的 GameServerSet、Agones 的 Fleet。

下面以 OpenKruiseGame 的 GameServerSet 为例，指定 Pod 注解：

```yaml
apiVersion: game.kruise.io/v1alpha1
kind: GameServerSet
metadata:
  name: gameserver
  namespace: default
spec:
  replicas: 3
  updateStrategy:
    rollingUpdate:
      podUpdatePolicy: InPlaceIfPossible
  gameServerTemplate:
    annotations:
        networking.cloud.tencent.com/enable-clb-port-mapping: "true"
        networking.cloud.tencent.com/clb-port-mapping: |-
          8000 UDP pool-test
    spec:
      containers:
        - image: your-gameserver-image
          name: gameserver
```

1. 指定注解 `networking.cloud.tencent.com/enable-clb-port-mapping` 为 `true` 开启使用 CLB 端口池为 Pod 映射公网地址。
2. 指定注解 `networking.cloud.tencent.com/clb-port-mapping` 配置映射规则，`8000 TCP pool-test`，其中 `8000` 表示 Pod 监听的端口号，`UDP` 表示端口协议（支持 TCP、UDP 和 TCPUDP），`pool-test` 表示 CLB 端口池名称，可指定多行配置多个端口映射。

## 通过 Downward API 获取 Pod 映射公网地址

当配置好 Pod 注解后，会为 Pod 自动分配 CLB 公网地址的映射，并将映射的结果写到 Pod 注解中：

```yaml
annotations:
    networking.cloud.tencent.com/clb-port-mapping-result: '[{"port":8000,"protocol":"TCP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-04iq85jh","loadbalancerPort":30210,"listenerId":"lbl-dt94u61x","hostname":"lb-04iq85jh-w49ru3xpmdynoigk.clb.cd-tencentclb.work"},{"port":8000,"protocol":"UDP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-04iq85jh","loadbalancerPort":30210,"listenerId":"lbl-467wodtz","hostname":"lb-04iq85jh-w49ru3xpmdynoigk.clb.cd-tencentclb.work"}]'
    networking.cloud.tencent.com/clb-port-mapping-status: Ready
```

可以将注解的内容通过 Downward API 挂载到容器中，然后在容器中读取注解内容，获取 Pod 映射的公网地址：


```yaml
    spec:
      containers:
        - ...
          volumeMounts:
            - name: podinfo
              mountPath: /etc/podinfo
      volumes:
        - name: podinfo
          downwardAPI:
            items:
              - path: "clb-port-mapping"
                fieldRef:
                  fieldPath: metadata.annotations['networking.cloud.tencent.com/clb-port-mapping-result']
```

进程启动时可轮询指定文件（本例中文件路径为 `/etc/podinfo/clb-port-mapping`），当文件内容为空说明此时 Pod 还未绑定到 CLB，当读取到内容时说明已经绑定成功，其内容格式为 JSON 数组，每个元素代表一个端口映射，可参考上面给出的注解示例。

## 单 Pod 多 DS 映射

如果单个 Pod 中运行了多个 DS（比如 10 个），使用 CLB 的端口段特性可节约 CLB 监听器数量（默认情况下，1 个 CLB 只能创建 50 个监听器）。

配置方法是定义端口池时指定端口段长度（segmentLength），指定后，分配的 CLB 监听器会使用端口段，即 1 个监听器映射后端连续的 segmentLength 个端口，示例：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-test
spec:
  startPort: 30000
  segmentLength: 10 # 端口段长度，10 表示 1 个 CLB 监听器映射 Pod 中连续的 10 个端口，即 10 个 DS 的地址
  exsistedLoadBalancerIDs: [lb-04iq85jh]
  autoCreate:
    enabled: true
```

**注意**：CLB 端口段特性需通过 [工单申请](https://console.cloud.tencent.com/workorder/category?level1_id=6&level2_id=163&source=14&data_title=%E8%B4%9F%E8%BD%BD%E5%9D%87%E8%A1%A1&step=1) 开通使用。

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

![](./images/tcpudp.png)

> Pod 的一个端口同时监听 TCP 和 UDP 协议，CLB 映射公网地址时，会分别使用 TCP 和 UDP 两个相同端口号的不同监听器进行映射。

自动生成的映射结果的注解示例如下：

```yaml
annotations:
    networking.cloud.tencent.com/clb-port-mapping-result: '[{"port":8000,"protocol":"TCP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-04iq85jh","loadbalancerPort":30170,"loadbalancerEndPort":30179,"listenerId":"lbl-bjoyr92j","endPort":8009,"hostname":"lb-04iq85jh-w49ru3xpmdynoigk.clb.cd-tencentclb.work"},{"port":8000,"protocol":"UDP","pool":"pool-test","region":"ap-chengdu","loadbalancerId":"lb-04iq85jh","loadbalancerPort":30170,"loadbalancerEndPort":30179,"listenerId":"lbl-6dg9wfs5","endPort":8009,"hostname":"lb-04iq85jh-w49ru3xpmdynoigk.clb.cd-tencentclb.work"}]'
    networking.cloud.tencent.com/clb-port-mapping-status: Ready
```

## 多线接入

CLB 默认使用 BGP 多运营商接入，带宽成本较高，游戏场景通常要消耗巨大的带宽资源，为节约成本，可以考虑使用 CLB 的单线接入，通过多个 CLB 来实现多线接入（电信玩家连上电信 CLB，联通玩家连上联通 CLB，移动玩家连上移动 CLB），这样可以节约大量带宽成本。

下面介绍配置方法，首先创建多个端口池，一个运营商一个端口池，这里以电信、联通、移动三个运营商为例，创建各自的 CLB 端口池：

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBPortPool
metadata:
  name: pool-ctcc # 电信 CLB 端口池
spec:
  startPort: 30000 # 端口池中 CLB 起始端口号
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
  startPort: 30000 # 端口池中 CLB 起始端口号
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
  startPort: 30000 # 端口池中 CLB 起始端口号
  exsistedLoadBalancerIDs: [lb-cxxc6xup, lb-mq3rs6h9] # 指定已有的移动 CLB 实例 ID，可动态追加
  autoCreate:
    enabled: true # 联通 CLB 端口不足时自动创建电信 CLB
    parameters: # 指定联通 CLB 创建参数
      vipIsp: CUCC # 指定运营商为联通
      bandwidthPackageId: bwp-97yjlal5 # 指定联通带宽包 ID
```

然后在 Pod 注解中配置端口映射：

```yaml
apiVersion: game.kruise.io/v1alpha1
kind: GameServerSet
metadata:
  name: gameserver
  namespace: default
spec:
  replicas: 2
  updateStrategy:
    rollingUpdate:
      podUpdatePolicy: InPlaceIfPossible
  gameServerTemplate:
    annotations:
        networking.cloud.tencent.com/enable-clb-port-mapping: "true"
        networking.cloud.tencent.com/clb-port-mapping: |-
          8000 TCPUDP pool-ctcc,pool-cmcc,pool-cucc useSamePortAcrossPools
    spec:
      containers:
        - image: your-gameserver-image
          name: gameserver
```

映射效果如下：

![](./images/three-isp.png)


解释：

1. Pod 端口同时监听 TCP 和 UDP，映射规则中的协议指定为 `TCPUDP`，CLB 映射公网地址时，会分别使用 TCP 和 UDP 两个相同端口号的不同监听器进行映射。
2. 使用多个端口池进行映射，用逗号隔开，每个端口池分别都会为 Pod 映射各自公网地址。
3. 追加 useSamePortAcrossPools 选项表示最终每个端口池分配相同的端口号。
4. 综上，最终每个 Pod 的每个端口会被映射三个公网地址，算上 TCP 和 UDP 同时监听，每个 Pod 端口使用 6 个 CLB 监听器映射公网地址；玩家连上自己运营商对应的 CLB 映射地址，如果玩家的网络环境 UDP 无法正常工作，自动 fallback 到 TCP 协议进行通信。

## TODO

- CLB端口段+HostPort 形成的端口池实现大规模单 Pod 单 DS 端口映射。
- 与 Agones 和 OKG 联动，映射信息写入 GameServer CR。
- 通过 EIP、NATGW 等方式映射。
- 优雅停机，避免缩容导致游戏中断。

## CRD 字段参考

关于 CRD 字段的详细说明，请参考 [API 参考](./api.md)。
