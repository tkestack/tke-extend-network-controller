# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

TKE Extend Network Controller 是一个 TKE (腾讯云 Kubernetes) 集群的网络控制器，主要用于游戏房间、会议等房间类场景，为每个 Pod/Node 分配独立的公网地址映射（通过 CLB 端口池实现）。

## 常用命令

### 构建与运行

```bash
make build              # 构建二进制 (bin/manager)
make run                # 本地运行控制器
make docker-build       # 构建 Docker 镜像
make docker-push        # 推送 Docker 镜像
```

### 代码生成

```bash
make generate           # 生成 DeepCopy 等方法实现
make manifests          # 生成 CRD、RBAC、Webhook YAML
make generate-docs      # 生成 API 文档 (docs/api.md)
```

### 测试

```bash
make test               # 运行单元测试 (使用 envtest)
make test-e2e           # 运行 E2E 测试
go test ./internal/controller/... -run TestXxx -v  # 运行单个测试
```

### 代码检查

```bash
make fmt                # 格式化代码
make vet                # go vet 检查
make lint               # golangci-lint 检查
make lint-fix           # lint 并自动修复
```

### 部署

```bash
make install            # 安装 CRD 到集群
make deploy             # 部署控制器到集群
make uninstall          # 卸载 CRD
make undeploy           # 卸载控制器
```

## 架构概览

### 核心 CRD (api/v1alpha1/)

| CRD | 简称 | 作用域 | 用途 |
|-----|------|--------|------|
| CLBPortPool | cpp | Cluster | 定义 CLB 端口池，管理可分配的公网端口资源 |
| CLBPodBinding | cpb | Namespaced | 为 Pod 分配 CLB 端口映射 |
| CLBNodeBinding | cnb | Cluster | 为 Node (HostPort) 分配 CLB 端口映射 |

### 控制器 (internal/controller/)

- **CLBPortPoolReconciler**: 管理端口池的 CLB 实例、同步配额、自动创建/扩容 CLB、预创建监听器
- **CLBPodBindingReconciler**: 处理 Pod 的端口绑定请求
- **CLBNodeBindingReconciler**: 处理 Node 的端口绑定请求
- **PodReconciler**: 监听 Pod 变化，为带有特定 annotation 的 Pod 自动创建/更新 CLBPodBinding
- **NodeReconciler**: 监听 Node 变化，处理 HostPort 场景的 CLBNodeBinding
- **GameServerSetReconciler**: 可选，支持 OpenKruiseGame (OKG) 的 GameServerSet

### 端口分配器 (internal/portpool/)

- **Allocator**: 全局端口分配器单例
- **PortPool**: 管理单个端口池的分配状态，支持三种 LB 分配策略：Uniform（均匀）、InOrder（顺序）、Random（随机）
- 支持 TCPUDP 协议同时分配 TCP 和 UDP 端口

### CLB SDK 封装 (pkg/clb/)

- 封装腾讯云 CLB SDK，提供批量操作、监听器缓存、限速、重试等能力
- **ListenerCache**: 监听器信息缓存，减少 API 调用
- **batch-listener.go / batch-target.go**: 批量创建监听器和绑定后端

### 关键包

- **pkg/cloudapi/**: 腾讯云 API 凭证管理
- **pkg/clusterinfo/**: 集群信息（ID、Region）
- **pkg/eventsource/**: 用于触发 Controller Reconcile 的事件源

## 测试

使用 Ginkgo/Gomega 测试框架，配合 controller-runtime 的 envtest 进行控制器测试。测试文件与被测文件在同一目录，以 `_test.go` 结尾。

## 开发注意事项

- 本项目使用 kubebuilder 生成，基于 controller-runtime 框架
- CRD 定义中使用 `+kubebuilder:validation:XValidation` 实现字段不可变性校验
- 修改 CRD 后需运行 `make manifests` 和 `make generate`
- Helm Chart 位于 `charts/tke-extend-network-controller/`
