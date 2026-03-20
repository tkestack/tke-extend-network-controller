# 技术栈

**分析日期：** 2025-01-21

## 编程语言

**主要语言：**
- Go 1.24.0 - 核心控制器实现
- Go 1.24.4 (toolchain) - 构建工具链

**辅助语言：**
- YAML - 配置、CRD、Helm Chart 定义
- Makefile - 构建脚本

## 运行时

**环境：**
- Kubernetes 1.26+ (Helm Chart 最低要求)
- TKE (腾讯云容器服务)

**包管理器：**
- Go modules
- go.mod / go.sum 锁文件（已提交）

## 框架

**核心框架：**
- `sigs.k8s.io/controller-runtime v0.21.0` - Kubernetes 控制器运行时基础
- `k8s.io/api v0.33.0` - Kubernetes API 资源定义
- `k8s.io/apimachinery v0.33.0` - Kubernetes 通用数据结构和转换
- `k8s.io/client-go v0.33.0` - Kubernetes 客户端库

**测试框架：**
- `github.com/onsi/ginkgo/v2 v2.22.0` - 行为驱动测试框架
- `github.com/onsi/gomega v1.36.1` - 测试断言库

**构建工具：**
- kubebuilder (controller-gen v0.15.0) - CRD 生成、代码生成
- kustomize v5.4.1 - Kubernetes 资源定制化管理
- golangci-lint v1.57.2 - Go 代码检查

## 关键依赖

**腾讯云 SDK：**
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb v1.0.1120` - CLB(负载均衡)服务 SDK
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc v1.0.1161` - VPC 服务 SDK
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam v1.1.3` - 访问管理 SDK
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag v1.1.0` - 标签服务 SDK
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.1.3` - 通用 SDK

**可选游戏相关：**
- `agones.dev/agones v1.49.0` - Google Agones 游戏服务器支持(可选)
- `github.com/openkruise/kruise-game v0.10.0` - OpenKruise Game 集成(可选)

**CLI 与配置：**
- `github.com/spf13/cobra v1.8.1` - 命令行框架
- `github.com/spf13/viper v1.19.0` - 配置管理
- `github.com/spf13/pflag v1.0.5` - POSIX 命令行标志处理

**日志与监控：**
- `go.uber.org/zap v1.27.0` - 高性能结构化日志(controller-runtime 集成)
- `go.uber.org/multierr v1.11.0` - 多错误聚合
- `github.com/prometheus/client_golang v1.22.0` - Prometheus 指标
- `go.opentelemetry.io/*` - OpenTelemetry 可观测性套件

**工具库：**
- `github.com/pkg/errors v0.9.1` - 堆栈跟踪错误包装
- `golang.org/x/time v0.9.0` - 速率限制支持
- `google.golang.org/grpc v1.72.0` - gRPC 支持
- `gopkg.in/yaml.v3 v3.0.1` - YAML 解析

## 配置

**环境变量配置：**
- `SECRET_ID` - 腾讯云账户密钥 ID
- `SECRET_KEY` - 腾讯云账户密钥
- `REGION` - TKE 集群地域(可选，自动检测)
- `VPCID` - VPC ID
- `CLUSTER_ID` - TKE 集群 ID
- `METRICS_BIND_ADDRESS` - Prometheus 指标端点绑定地址(默认 0)
- `HEALTH_PROBE_BIND_ADDRESS` - 健康检查端点(默认 :8081)
- `LEADER_ELECT` - 领导者选举开关(默认 false)

**运行时配置来源：**
1. 命令行标志(最高优先级)
2. 环境变量(通过 viper 的 `AutomaticEnv()`)
3. 默认值

**文件配置：**
- `Helm values.yaml` - 部署配置
  - 副本数、资源限制
  - 并发配置(各控制器并发度)
  - API 限频配置(QPS 管理)
  - 日志级别和格式

## 构建与部署

**本地构建：**
```bash
make build              # 输出: bin/manager
make docker-build       # Docker 镜像构建(linux/amd64)
make docker-buildx      # 多平台构建(linux/amd64, arm64, s390x, ppc64le)
```

**Docker 镜像：**
- 基础镜像: `golang:1.24` (构建阶段)
- 运行时镜像: `gcr.io/distroless/static:nonroot`
- 镜像仓库: `imroc/tke-extend-network-controller`
- 镜像标签: Git 标签(去掉 v 前缀)或最短提交哈希

**部署方式：**
- Kubernetes 清单 (Kustomize)
- Helm Chart v2.4.1
  - 最低 Kubernetes 版本: 1.26.0
  - Chart 仓库位置: `charts/tke-extend-network-controller/`

## 平台要求

**开发环境：**
- Go 1.24+
- Docker 或兼容的容器工具(podman 等)
- kubectl
- 可选：Kustomize(包含在 Makefile 工具)

**运行时环境：**
- TKE Kubernetes 集群 v1.26+
- 访问腾讯云 API 的凭证
- CLB 权限
- VPC 信息

**特殊环境：**
- E2E 测试: Kind Kubernetes 集群
- 本地测试: envtest (k8s 1.30.0 资产)

---

*栈分析时间：2025-01-21*
