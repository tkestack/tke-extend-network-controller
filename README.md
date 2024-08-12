# tke-extend-network-controller

针对 TKE 集群一些特殊场景的的网络控制器。

[API 参考](docs/api.md)

## 支持房间类场景

目前主要支持会议、游戏战斗服等房间类场景的网络，即要求每个 Pod 都需要独立的公网地址，TKE 集群默认只支持 EIP 方案，但 EIP 资源有限，不仅是数量的限制，还有每日申请的数量限制，稍微上点规模，或频繁扩缩容更换EIP，可能很容易触达限制导致 EIP 分配失败；而如果保留 EIP，在 EIP 没被绑定前，又会收取额外的闲置费。

> TKE Pod 绑定 EIP 参考 [VPC-CNI: Pod 直接绑定弹性公网 IP 使用说明](https://cloud.tencent.com/document/product/457/64886) 与 [超级节点 Pod 绑定 EIP](https://cloud.tencent.com/document/product/457/44173#.E7.BB.91.E5.AE.9A-eip)。

如果不用 EIP，也可通过安装此插件来实现为每个 Pod 的指定端口都分配一个独立的公网地址映射 (公网 `IP:Port` 到内网 Pod `IP:Port` 的映射)。

## 前提条件

安装 `tke-extend-network-controller` 前请确保满足以下前提条件：
1. 创建了 [TKE](https://cloud.tencent.com/product/tke) 集群，且集群版本大于等于 1.26。
2. 集群中安装了 [cert-manager](https://cert-manager.io/docs/installation/) (webhook 依赖证书)。
3. 本地安装了 [helm](https://helm.sh) 命令，且能通过 helm 命令操作 TKE 集群（参考[本地 Helm 客户端连接集群](https://cloud.tencent.com/document/product/457/32731)）。
4. 需要一个腾讯云子账号的访问密钥(SecretID、SecretKey)，参考[子账号访问密钥管理](https://cloud.tencent.com/document/product/598/37140)，要求账号至少具有以下权限：
    ```json
    {
        "version": "2.0",
        "statement": [
            {
                "effect": "allow",
                "action": [
                    "clb:DescribeLoadBalancerBackends",
                    "clb:DescribeLoadBalancerListeners",
                    "clb:DescribeLoadBalancers",
                    "clb:CreateLoadBalancer",
                    "clb:DescribeTargets",
                    "clb:DeleteLoadBalancer",
                    "clb:DeleteLoadBalancerListeners",
                    "clb:BatchDeregisterTargets",
                    "clb:BatchRegisterTargets",
                    "clb:DeregisterTargets",
                    "clb:CreateLoadBalancerListeners",
                    "clb:CreateListener",
                    "clb:RegisterTargets",
                    "clb:DeleteLoadBalancers",
                    "clb:DescribeLoadBalancersDetail"
                ],
                "resource": [
                    "*"
                ]
            }
        ]
    }
    ```

## 使用 helm 安装

1. 添加 helm repo:

```bash
helm repo add tke-extend-network-controller https://imroc.github.io/tke-extend-network-controller
```

2. 创建 `values.yaml` 并配置:

```yaml
region: "" # TKE 集群所在地域，如 ap-guangzhou。全部地域列表参考: https://cloud.tencent.com/document/product/213/6091
vpcID: "" # TKE 集群所在 VPC ID (vpc-xxx)
clusterID: "" # TKE 集群 ID (cls-xxx)
secretID: "" # 腾讯云子账号的 SecretID
secretKey: "" # 腾讯云子账号的 SecretKey
```

3. 安装到 TKE 集群：
```bash
helm upgrade --install -f values.yaml \
  --namespace tke-extend-network-controller --create-namespace \
  tke-extend-network-controller tke-extend-network-controller/tke-extend-network-controller
```

> 1. 如果要升级版本，先执行 `helm repo update`，再重复执行上面的安装命令即可。
> 2. 如果要更改配置，直接修改 `values.yaml`，再重复执行上面的安装命令即可。

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

## 项目状态与版本说明

当前项目正处于活跃开发中，请及时更新版本以获得最新能力，参考 [版本说明](CHANGELOG.md)。
