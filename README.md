# tke-extend-network-controller

针对 TKE 集群一些特殊场景的的网络控制器。

[API 参考](docs/api.md)

## 游戏场景

大多游戏都有战斗服，每个 Pod 都需要一个独立的公网地址 (IP:Port)，而 TKE 集群默认不提供这个能力，可通过安装此插件来实现为每个 Pod 都分配一个独立的公网地址。

## 使用 NAT 网关为 Pod 分配专属地址

通过自动为 NAT 网关添加 DNAT 规则(端口转发) 来实现为 Pod 分配独立的公网地址：

```yaml
apiVersion: networking.cloud.tencent.com/v1apha1
kind: DedicatedNatgwService
metadata:
  namespace: demo
  name: gameserver
spec:
  minPort: 500 # 在NAT网关自动创建端口转发，每个 Pod 占用 NAT 网关的一个 IP:Port，端口号范围在 500-600
  maxPort: 600
  selector:
    app: gameserver
  ports:
  - protocol: TCP
    targetPort: 9000
  extensiveParameters: '{"InternetMaxBandwidthOut":5000, "NatProductVersion":2}' # 如果自动创建NAT，指定购买NAT接口的参数: https://cloud.tencent.com/document/api/215/36721
  existedNatgwIds: # 如果复用已有的 NAT 网关实例，指定 NAT 网关实例 ID 的列表
    - nat-xxx
    - nat-yyy
    - nat-zzz
```

## 使用 CLB 为 Pod 分配专属地址

通过自动为 CLB 创建监听器并绑定单个 Pod 来实现为 Pod 分配独立的公网地址：

```yaml
apiVersion: networking.cloud.tencent.com/v1apha1
kind: DedicatedCLBService
metadata:
  namespace: demo
  name: gameserver
spec:
  minPort: 500 # 在CLB自动创建监听器，每个Pod占用一个端口，端口号范围在 500-600
  maxPort: 600
  selector:
    app: gameserver
  ports:
  - protocol: TCP # CLB 监听器协议（TCP/UDP）
    targetPort: 9000 # 容器监听的端口
  - protocol: UDP
    targetPort: 8000
  lbRegion: ap-chengdu # 可选，CLB 所在地域，默认为集群所在地域
  extensiveParameters: '{"VipIsp":"CTCC"}' # 如果自动创建CLB，指定购买CLB接口的参数: https://cloud.tencent.com/document/product/214/30692
  existedLbIds: # 如果复用已有的 CLB 实例，指定 CLB 实例 ID 的列表
    - lb-xxx
    - lb-yyy
    - lb-zzz
```

controller 会自动为关联的所有 Pod 自动创建 `DedicatedCLBListener`:

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: DedicatedCLBListener
metadata:
  name: gameserver-0
spec:
  lbId: lb-xxx # 必选，CLB 的实例 ID
  lbRegion: ap-chengdu # 可选，CLB 所在地域，默认为集群所在地域
  lbPort: 8088 # 必选，监听器端口
  protocol: TCP # 必选，监听器协议。TCP | UDP
  listenerConfig: clblistenerconfig-sample # 可选，创建监听器的配置
  backendPod: # 可选，需绑定的后端Pod
    podName: gameserver-0 # 指定 backendPod 时必选，后端 Pod 名称
    port: 80 # 指定 backendPod 时必选，后端 Pod 监听的端口
status:
  listenerId: lbl-ku486mr3 # 监听器 ID
  state: Occupied # 监听器状态，Pending (监听器创建中) | Occupied (监听器已绑定Pod) | Available (监听器已创建但还未绑定Pod) | Deleting (监听器删除中)
```

然后 controller 根据 `CLBPodBinding` 进行对账，自动将 Pod 绑定到对应的 CLB 监听器上。
