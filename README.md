# tke-extend-network-controller

针对 TKE 集群一些特殊场景的的网络控制器。

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
  - protocol: TCP
    targetPort: 9000
  extensiveParameters: '{"VipIsp":"CTCC"}' # 如果自动创建CLB，指定购买CLB接口的参数: https://cloud.tencent.com/document/product/214/30692
  existedLbIds: # 如果复用已有的 CLB 实例，指定 CLB 实例 ID 的列表
    - lb-xxx
    - lb-yyy
    - lb-zzz
```
