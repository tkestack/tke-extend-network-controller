# API 设计与字段说明

* [DedicatedCLBService](#dedicatedclbservice)
* [DedicatedCLBListener](#dedicatedclblistener)
* [CLBListenerConfig](#clblistenerconfig)

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
  minPort: 501 # 在 CLB 自动创建监听器，每个 Pod 占用一个端口，端口号范围在 501-600
  maxPort: 600
  listenerConfig: "clblistenerconfig-sample" # 可选，指定监听器配置，引用 CLBListenerConfig。
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
  listenerConfig: "clblistenerconfig-sample" # 可选，指定监听器配置，引用 CLBListenerConfig。
  backendPod: # 可选，需绑定的后端Pod
    podName: gameserver-0 # 指定 backendPod 时必选，后端 Pod 名称
    port: 80 # 指定 backendPod 时必选，后端 Pod 监听的端口
status:
  listenerId: lbl-ku486mr3 # 监听器 ID
  state: Bound # 监听器状态，Pending (监听器创建中) | Bound (监听器已绑定Pod) | Available (监听器已创建但还未绑定Pod) | Deleting (监听器删除中) | Failed （失败）
  address: 139.135.64.53:8088 # 公网地址
```

## CLBListenerConfig

CLB 监听器配置，可被 `DedicatedCLBListener` 或 `DedicatedCLBService` 引用:

```yaml
apiVersion: networking.cloud.tencent.com/v1alpha1
kind: CLBListenerConfig
metadata:
  name: clblistenerconfig-sample
spec:
  healthcheck: # 可选，健康检查配置。 CLB API 文档: https://cloud.tencent.com/document/api/214/30694#HealthCheck
    healthSwitch: 0 # 可选，是否开启健康检查：1（开启）、0（关闭）。
    timeOut: 2 # 可选，健康检查的响应超时时间，可选值：2~60，默认值：2，单位：秒。响应超时时间要小于检查间隔时间。
    intervalTime: 10 # 可选，健康检查探测间隔时间，默认值：5，IPv4 CLB实例的取值范围为：2-300，IPv6 CLB 实例的取值范围为：5-300。单位：秒。
    healthNum: 3 # 可选，健康阈值，默认值：3，表示当连续探测三次健康则表示该转发正常，可选值：2~10，单位：次。
    unHealthNum: 3 # 可选，不健康阈值，默认值：3，表示当连续探测三次不健康则表示该转发异常，可选值：2~10，单位：次。
    httpCode: 32 # 可选，健康检查返回的 HTTP 状态码。健康检查状态码（仅适用于HTTP/HTTPS转发规则、TCP监听器的HTTP健康检查方式）。可选值：1~31，默认 31。1 表示探测后返回值 1xx 代表健康，2 表示返回 2xx 代表健康，4 表示返回 3xx 代表健康，8 表示返回 4xx 代表健康，16 表示返回 5xx 代表健康。若希望多种返回码都可代表健康，则将相应的值相加。
    httpCheckPath: / # 可选，健康检查路径。健康检查路径（仅适用于HTTP/HTTPS转发规则、TCP监听器的HTTP健康检查方式）。
    httpCheckDomain: "" # 可选，健康检查域名（仅适用于HTTP/HTTPS监听器和TCP监听器的HTTP健康检查方式。针对TCP监听器，当使用HTTP健康检查方式时，该参数为必填项）。
    httpCheckMethod: "HEAD" # 可选，健康检查方法（仅适用于HTTP/HTTPS转发规则、TCP监听器的HTTP健康检查方式），默认值：HEAD，可选值HEAD或GET。
    checkPort: 80 # 可选，自定义探测相关参数。健康检查端口，默认为后端服务的端口，除非您希望指定特定端口，否则建议留空。（仅适用于TCP/UDP监听器）。
    contextType: "" # 可选，自定义探测相关参数。健康检查协议CheckType的值取CUSTOM时，必填此字段，代表健康检查的输入格式，可取值：HEX或TEXT；取值为HEX时，SendContext和RecvContext的字符只能在0123456789ABCDEF中选取且长度必须是偶数位。（仅适用于TCP/UDP监听器）
    sendContext: "" # 可选，自定义探测相关参数。健康检查协议CheckType的值取CUSTOM时，必填此字段，代表健康检查发送的请求内容，只允许ASCII可见字符，最大长度限制500。（仅适用于TCP/UDP监听器）。
    recvContext: "" # 可选，自定义探测相关参数。健康检查协议CheckType的值取CUSTOM时，必填此字段，代表健康检查返回的结果，只允许ASCII可见字符，最大长度限制500。（仅适用于TCP/UDP监听器）。
    checkType: TCP # 可选，健康检查使用的协议。取值 TCP | HTTP | HTTPS | GRPC | PING | CUSTOM，UDP监听器支持PING/CUSTOM，TCP监听器支持TCP/HTTP/CUSTOM，TCP_SSL/QUIC监听器支持TCP/HTTP，HTTP规则支持HTTP/GRPC，HTTPS规则支持HTTP/HTTPS/GRPC。HTTP监听器默认值为HTTP;TCP、TCP_SSL、QUIC监听器默认值为TCP;UDP监听器默认为PING;HTTPS监听器的CheckType默认值与后端转发协议一致。
    httpVersion: "" # 可选，HTTP版本。健康检查协议CheckType的值取HTTP时，必传此字段，代表后端服务的HTTP版本：HTTP/1.0、HTTP/1.1；（仅适用于TCP监听器）。
    sourceIpType: 0 # 可选，健康检查源IP类型：0（使用LB的VIP作为源IP），1（使用100.64网段IP作为源IP）。
    extendedCode: 12 # 可选，GRPC健康检查状态码（仅适用于后端转发协议为GRPC的规则）。默认值为 12，可输入值为数值、多个数值、或者范围，例如 20 或 20,25 或 0-99
  certificate: # 可选，证书相关信息，此参数仅适用于TCP_SSL监听器和未开启SNI特性的HTTPS监听器。此参数和MultiCertInfo不能同时传入。
    sslMode: UNIDIRECTIONAL # 可选，认证类型，UNIDIRECTIONAL：单向认证，MUTUAL：双向认证
    certId: GMmA8Yjd # 可选，服务端证书的 ID，如果不填写此项则必须上传证书，包括 CertContent，CertKey，CertName。
    certCaId: GMnOUTcH # 可选，客户端证书的 ID，当监听器采用双向认证，即 SSLMode=MUTUAL 时，如果不填写此项则必须上传客户端证书，包括 CertCaContent，CertCaName。
  sessionExpireTime: 60 # 可选，会话保持时间，单位：秒。可选值：30~3600，默认 0，表示不开启。此参数仅适用于TCP/UDP监听器。
  scheduler: WRR # 可选，监听器转发的方式。可选值：WRR、LEAST_CONN，分别表示按权重轮询、最小连接数， 默认为 WRR。此参数仅适用于TCP/UDP/TCP_SSL/QUIC监听器。
  sniSwitch: 1 # 可选，是否开启SNI特性，此参数仅适用于HTTPS监听器。0表示开启，1表示未开启。
  targetType: NODE # 可选，后端目标类型，NODE表示绑定普通节点，TARGETGROUP表示绑定目标组。此参数仅适用于TCP/UDP监听器。七层监听器应在转发规则中设置。
  sessionType: NORMAL # 可选，会话保持类型。不传或传NORMAL表示默认会话保持类型。QUIC_CID 表示根据Quic Connection ID做会话保持。QUIC_CID只支持UDP协议。此参数仅适用于TCP/UDP监听器。七层监听器应在转发规则中设置。（若选择QUIC_CID，则Protocol必须为UDP，Scheduler必须为WRR，同时只支持ipv4）。
  keepaliveEnable: 0 # 可选，是否开启长连接，此参数仅适用于HTTP/HTTPS监听器，0:关闭；1:开启， 默认关闭。
  endPort: 1000 # 可选，创建端口段监听器时必须传入此参数，用以标识结束端口。同时，入参Ports只允许传入一个成员，用以标识开始端口。【如果您需要体验端口段功能，请通过 工单申请】。
  deregisterTargetRst: false # 可选，解绑后端目标时，是否发RST给客户端，此参数仅适用于TCP监听器。
  multiCertInfo: # 可选，证书信息，支持同时传入不同算法类型的多本服务端证书；此参数仅适用于未开启SNI特性的HTTPS。
    sslMode: UNIDIRECTIONAL # 必选，认证类型，UNIDIRECTIONAL：单向认证，MUTUAL：双向认证。
    certList:
      - certId: GMmA8Yjd # 可选，证书 ID。
  maxConn: -1 # 可选，监听器最大连接数，当前仅性能容量型实例且仅TCP/UDP/TCP_SSL/QUIC监听器支持，不传或者传-1表示监听器维度不限速。基础网络实例不支持该参数。
  maxCps: -1 # 可选，监听器最大新增连接数，当前仅性能容量型实例且仅TCP/UDP/TCP_SSL/QUIC监听器支持，不传或者传-1表示监听器维度不限速。基础网络实例不支持该参数。
  idleConnectTimeout: 900 # 可选，空闲连接超时时间，此参数仅适用于TCP监听器，单位：秒。默认值：900，取值范围：共享型实例和独占型实例支持：300～900，性能容量型实例支持：300~2000。如需设置超过2000s，请通过 工单申请,最大可设置到3600s。
  snatEnable: false # 可选，是否开启SNAT。
```
