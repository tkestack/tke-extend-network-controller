# tke-extend-network-controller

针对 TKE 集群一些特殊场景的的网络控制器。

[API 参考](docs/api.md)

## 房间类场景

类似会议、游戏战斗服等房间类场景，每个 Pod 都需要一个独立的公网地址 (IP:Port)，而 TKE 集群默认不提供这个能力，可通过安装此插件来实现为每个 Pod 的某个端口都分配一个独立的公网地址。

## 使用 CLB 为 Pod 分配公网地址映射

通过自动为 CLB 创建监听器并绑定单个 Pod 来实现为 Pod 分配独立的公网地址：

```yaml
apiVersion: networking.cloud.tencent.com/v1apha1
kind: DedicatedCLBService
metadata:
  namespace: demo
  name: gameserver
spec:
  lbRegion: ap-chengdu # 可选，CLB 所在地域，默认为集群所在地域
  minPort: 501 # 在CLB自动创建监听器，每个Pod占用一个端口，端口号范围在 501-600
  maxPort: 600
  selector:
    app: gameserver
  ports:
  - protocol: TCP # CLB 监听器协议（TCP/UDP）
    targetPort: 9000 # 容器监听的端口
    addressPodAnnotation: networking.cloud.tencent.com/external-address-9000 # 可选，将外部地址注入到pod的annotation中
  - protocol: UDP
    targetPort: 8000
    addressPodAnnotation: networking.cloud.tencent.com/external-address-8080
  existedLbIds: # 如果复用已有的 CLB 实例，指定 CLB 实例 ID 的列表
    - lb-xxx
    - lb-yyy
    - lb-zzz
  # 暂未实现：extensiveParameters: '{"VipIsp":"CTCC"}' # 如果自动创建CLB，指定购买CLB接口的参数: https://cloud.tencent.com/document/product/214/30692
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
  state: Bound # 监听器状态，Pending (监听器创建中) | Bound (监听器已绑定Pod) | Available (监听器已创建但还未绑定Pod) | Deleting (监听器删除中)
  address: 139.135.64.53:8088 # 公网地址
```

然后 controller 根据 `DedicatedCLBListener` 进行对账，自动将 Pod 绑定到对应的 CLB 监听器上。

## 使用 NAT 网关为 Pod 分配公网地址映射

TODO

## FAQ

### 为什么不直接用 EIP，TKE 本身支持，每个 Pod 绑一个 EIP 就行了?

技术上可行，但EIP资源有限，不仅有数量限制，还有每日申请的数量限制，稍微上点规模，或频繁扩缩容更换EIP，可能很容易触达限制，如果保留EIP，在EIP没被绑定前，又会收取闲置费。
