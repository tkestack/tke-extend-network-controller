# API 设计与字段说明

* [DedicatedCLBService](#dedicatedclbservice)
* [DedicatedCLBListener](#dedicatedclblistener)

## DedicatedCLBService

为选中的每个 Pod 分配一个独立的 CLB 地址映射 (会将 `selector` 选中的所有 Pod 自动关联一个 `DedicatedCLBListener`，以实现为每个 Pod 绑定一个 CLB 监听器):

```yaml
apiVersion: networking.cloud.tencent.com/v1apha1
kind: DedicatedCLBService
metadata:
  namespace: demo
  name: gameserver
spec:
  lbRegion: ap-chengdu # 可选，CLB 所在地域，默认为集群所在地域
  minPort: 500 # 可选，在 CLB 自动创建监听器，每个 Pod 占用一个端口，默认端口号范围在 500-50000
  maxPort: 50000
  maxPod: 50 # 可选，限制最大 Pod/监听器 数量。
  listenerExtensiveParameters: | # 可选，指定创建监听器时的参数(JSON 格式)，完整参考 CreateListener 接口： https://cloud.tencent.com/document/api/214/30693 （由于是一个监听器只挂一个 Pod，通常不需要自定义监听器配置，因为健康检查、调度算法这些配置，对于只有一个 RS 的监听器没有意义）
    {
      "DeregisterTargetRst": true
    }
  selector:
    app: gameserver
  ports:
  - protocol: TCP # 端口监听的协议（TCP/UDP）
    targetPort: 9000 # 容器监听的端口
    addressPodAnnotation: networking.cloud.tencent.com/external-address-9000 # 可选，将外部地址自动注入到指定的 pod annotation 中
  - protocol: UDP
    targetPort: 8000
    addressPodAnnotation: networking.cloud.tencent.com/external-address-8080
  existedLbIds: # 如果复用已有的 CLB 实例，指定 CLB 实例 ID 的列表
    - lb-xxx
    - lb-yyy
    - lb-zzz
  lbAutoCreate:
    enable: true # 当 CLB 不足时，自动创建 CLB
    extensiveParameters: | # 购买 CLB 时的参数(JSON 字符串格式)：按流量计费，超强型4实例规格，带宽上限 60 Gbps （完整参数列表参考 CreateLoadBalancer 接口 https://cloud.tencent.com/document/api/214/30692）
      {
        "InternetAccessible": {
          "InternetChargeType": "TRAFFIC_POSTPAID_BY_HOUR",
          "InternetMaxBandwidthOut": 61440
        },
        "SlaType": "clb.c4.xlarge"
      }
```

## DedicatedCLBListener

为指定单个 Pod 分配一个独立的 CLB 地址映射:

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
  extensiveParameters: "" # 可选，指定创建监听器时的参数，JSON 格式，完整参考 CreateListener 接口： https://cloud.tencent.com/document/api/214/30693
  backendPod: # 可选，需绑定的后端Pod
    podName: gameserver-0 # 指定 backendPod 时必选，后端 Pod 名称
    port: 80 # 指定 backendPod 时必选，后端 Pod 监听的端口
status:
  listenerId: lbl-ku486mr3 # 监听器 ID
  state: Bound # 监听器状态，Pending (监听器创建中) | Bound (监听器已绑定Pod) | Available (监听器已创建但还未绑定Pod) | Deleting (监听器删除中) | Failed （失败）
  address: 139.135.64.53:8088 # 公网地址
```

