# 外部集成

**分析日期：** 2025-01-21

## APIs 与外部服务

**腾讯云服务：**

CLB (负载均衡器)
- SDK 包: `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb`
- 用途: 创建/管理 CLB 实例、创建监听器、注册/注销后端目标
- 关键 API:
  - `DescribeLoadBalancers` - 获取 CLB 列表
  - `CreateListener` - 创建监听器
  - `DescribeListeners` - 查询监听器
  - `BatchRegisterTargets` - 批量绑定后端目标
  - `BatchDeregisterTargets` - 批量解绑后端目标
  - `DescribeTargets` - 查询后端目标
  - `DeleteLoadBalancerListeners` - 删除监听器
  - `DescribeTaskStatus` - 查询异步任务状态
- 初始化: `pkg/cloudapi/cloudapi.go:Init()`
- 客户端获取: `pkg/clb/clb.go:GetClient(region)`

VPC (虚拟私有网络)
- SDK 包: `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc`
- 用途: 获取 VPC 相关信息
- 客户端获取: `pkg/vpc/client.go:GetClient(region)`

CAM (访问管理)
- SDK 包: `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam`
- 用途: 用户信息查询
- 初始化: `pkg/userinfo/userinfo.go:Init()`

Tag (标签服务)
- SDK 包: `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag`
- 用途: 资源标签管理

**云元数据服务：**
- 地址: `http://metadata.tencentyun.com/latest/meta-data/`
- 用途: 自动探测集群所在地域
- 实现: `pkg/util/region.go:GetCurrentRegion()`

## 数据存储

**数据库：**
- 无独立数据库
- 所有持久化通过 Kubernetes etcd 实现
- 资源存储：CLBPortPool、CLBPodBinding、CLBNodeBinding CRD

**文件存储：**
- Kubernetes Secret 存储证书 ID
  - 读取: `pkg/kube/secret.go:GetCertIdFromSecret()`
  - Secret 数据键: `qcloud_cert_id`

**缓存：**
- 内存缓存：`pkg/clb/cache.go:ListenerCache` - 监听器信息缓存
- 控制器运行时缓存: controller-runtime 的内存 cache
- 端口分配器缓存: `internal/portpool/allocator.go` - 全局端口分配状态

## 认证与授权

**认证提供商：**
- 腾讯云 API 认证
  - 凭证类型: 密钥认证(SecretID + SecretKey)
  - 配置: 命令行标志 `--secret-id`, `--secret-key`
  - 环境变量: `SECRET_ID`, `SECRET_KEY`
  - 初始化: `pkg/cloudapi/cloudapi.go:Init()`
  - 存储位置: 全局变量 `pkg/cloudapi/credential`

**Kubernetes 认证：**
- 使用 kubeconfig 进行 Kubernetes API 认证
- 支持的认证方式: 通过 `k8s.io/client-go/plugin/pkg/client/auth` 自动加载(包括 OIDC、Azure、GCP 等)

**RBAC 授权：**
- Kubernetes RBAC
  - ClusterRole: 控制器权限定义
  - RoleBinding: 服务账户权限绑定
  - 配置位置: `config/rbac/`
- Metrics 端点保护
  - 启用: `sigs.k8s.io/controller-runtime/pkg/metrics/filters:WithAuthenticationAndAuthorization`
  - 需要: RBAC 认证通过

## 监控与可观测性

**错误追踪：**
- 无专用错误追踪服务集成
- 通过日志输出记录错误

**日志：**
- 实现: controller-runtime 的 logr 接口
- 后端: `go.uber.org/zap`
- 格式: 可配置(json 或 console)
- 级别: 可配置(debug, info, error 或自定义 0+)
- 配置位置: Helm values 的 `log.level`, `log.encoder`
- 日志示例位置:
  - `pkg/clb/api.go:LogAPI()` - CLB API 调用日志
  - `pkg/clb/clb.go:clbLog` - CLB 操作日志

**Prometheus 指标：**
- SDK: `github.com/prometheus/client_golang v1.22.0`
- OpenTelemetry 集成:
  - 包: `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
  - 包: `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`
- 指标端点:
  - 绑定地址: 通过 `--metrics-bind-address` 配置
  - 默认值: `:8443` (HTTPS) 或 `:8080` (HTTP) 或 `0` (禁用)
  - 自动启用: controller-runtime 内置

**健康检查：**
- Liveness probe:
  - 端点: `GET /healthz`
  - 端口: 8081
- Readiness probe:
  - 端点: `GET /readyz`
  - 端口: 8081
- 绑定地址: 通过 `--health-probe-bind-address` 配置(默认 :8081)

## CI/CD 与部署

**托管平台：**
- 任意 Kubernetes 集群(最低 1.26+)
- 腾讯云 TKE

**CI 流水线：**
- 未检测到 CI/CD 配置
- 支持的发布方式:
  - Docker 镜像推送到镜像仓库
  - Helm Chart 部署

**部署工具：**
- Kustomize: CRD 和资源部署
- Helm: 完整部署解决方案

## 环境配置

**必需的环境变量：**
- `SECRET_ID` - 腾讯云 API 密钥 ID
- `SECRET_KEY` - 腾讯云 API 密钥
- `REGION` - TKE 集群地域(可选，自动检测)
- `VPCID` - VPC ID
- `CLUSTER_ID` - TKE 集群 ID

**可选环境变量：**
- `METRICS_BIND_ADDRESS` - 指标端点地址
- `HEALTH_PROBE_BIND_ADDRESS` - 健康检查端点地址
- `LEADER_ELECT` - 启用领导者选举

**密钥位置：**
- 方式 1: 命令行标志(启动参数)
- 方式 2: 环境变量
- 方式 3: Helm values (部署时通过 values.yaml 注入)
- Kubernetes Secret (用于证书 ID 存储)

**配置验证：**
- 启动时验证:
  - `cloudapi.Init()` 检查 SecretID 和 SecretKey 非空
  - `clb.Quota.Get()` 验证凭证有效性
  - 区域、VPC ID、集群 ID 验证

## Webhooks 与回调

**入站 Webhooks：**
- 无入站外部 Webhook

**Kubernetes Webhooks (内部)：**
- 验证性 Webhook:
  - `/validate-networking-cloud-tencent-com-v1alpha1-clbportpool` - CLBPortPool 验证
- 变更性 Webhook:
  - `/mutate-networking-cloud-tencent-com-v1alpha1-clbportpool` - CLBPortPool 默认值注入

**出站 Webhooks：**
- 无出站 Webhook 调用
- 只有腾讯云 API 调用

## API 速率限制

**腾讯云 API 限速：**
- 实现: `pkg/clb/rate-limit.go`
- 配置位置: Helm values 的 `apiRateLimit`
- 默认 QPS 配置:
  - `DescribeLoadBalancers`: 20
  - `CreateListener`: 20
  - `DescribeListeners`: 20
  - `DeleteLoadBalancerListeners`: 20
  - `BatchRegisterTargets`: 20
  - `DescribeTargets`: 20
  - `BatchDeregisterTargets`: 20
  - `DescribeTaskStatus`: 20
- 限频超出重试逻辑:
  - 检测错误: `IsRequestLimitExceededError()`
  - 重试策略: 指数退避(每次睡眠 1 秒)
  - 检查 context 是否已撤销防止无限重试

## 并发配置

**控制器并发度 (Helm values)：**
- `clbPodBindingController`: 20
- `clbNodeBindingController`: 20
- `podController`: 20
- `nodeController`: 20
- `clbPortPoolController`: 10

**领导者选举：**
- 启用标志: `--leader-elect`
- ID: `0ce7cb44.cloud.tencent.com`
- 用途: 多副本部署中只有一个活跃控制器

---

*集成审计时间：2025-01-21*
