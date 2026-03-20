# 代码结构

**分析日期:** 2025-03-20

## 目录布局

```
tke-extend-network-controller/
├── api/                        # Kubernetes 自定义资源定义（CRD）
│   └── v1alpha1/              # v1alpha1 API 版本
│       ├── clbportpool_types.go          # CLBPortPool 资源定义
│       ├── clbpodbinding_types.go        # CLBPodBinding 资源定义
│       ├── clbnodebinding_types.go       # CLBNodeBinding 资源定义
│       ├── clbbinding_types.go           # CLBBinding 公共字段
│       ├── groupversion_info.go          # API 版本组信息
│       ├── api.go                        # 初始化函数和工具方法
│       └── zz_generated.deepcopy.go      # 自动生成的 DeepCopy 方法
│
├── internal/                   # 控制器实现（不导出）
│   ├── controller/            # 核心 Reconciler 实现
│   │   ├── clbportpool_controller.go           # CLBPortPool Reconciler
│   │   ├── clbportpool_controller_test.go      # 测试
│   │   ├── clbpodbinding_controller.go         # CLBPodBinding Reconciler
│   │   ├── clbpodbinding_controller_test.go    # 测试
│   │   ├── clbnodebinding_controller.go        # CLBNodeBinding Reconciler
│   │   ├── clbnodebinding_controller_test.go   # 测试
│   │   ├── pod_controller.go                   # Pod Reconciler
│   │   ├── pod_controller_test.go              # 测试
│   │   ├── node_controller.go                  # Node Reconciler
│   │   ├── node_controller_test.go             # 测试
│   │   ├── gameserverset_controller.go         # GameServerSet Reconciler (可选)
│   │   ├── gameserverset_controller_test.go    # 测试
│   │   ├── clbbinding.go                       # 通用绑定逻辑 + 泛型实现
│   │   ├── util.go                             # 控制器工具函数
│   │   └── suite_test.go                       # Ginkgo 测试套件
│   │
│   ├── clbbinding/            # 端口绑定实现层
│   │   ├── clbbinding.go                  # CLBBinding 接口定义
│   │   ├── clbpodbinding.go               # Pod 绑定实现
│   │   ├── clbnodebinding.go              # Node 绑定实现
│   │   └── sort.go                        # LB 挑选策略实现
│   │
│   ├── portpool/              # 端口分配管理
│   │   ├── allocator.go                   # 全局 PortAllocator 单例
│   │   ├── portpool.go                    # 单个 PortPool 实现
│   │   ├── portpools.go                   # PortPool 集合类型
│   │   ├── error.go                       # 端口分配错误类型
│   │   └── portpool_test.go               # 端口分配测试
│   │
│   ├── webhook/               # ValidatingWebhook 实现
│   │   └── v1alpha1/
│   │       ├── clbportpool_webhook.go           # CLBPortPool 验证 Webhook
│   │       ├── clbportpool_webhook_test.go      # 测试
│   │       └── webhook_suite_test.go            # 测试套件
│   │
│   └── constant/              # 常量定义
│       └── constant.go                    # 如注解键、状态值等常量
│
├── pkg/                        # 可导出的公共包
│   ├── clb/                   # 腾讯云 CLB SDK 封装
│   │   ├── clb.go                        # CLB 客户端初始化和全局变量
│   │   ├── api.go                        # API 调用日志记录
│   │   ├── instance.go                   # CLB 实例操作（查询、创建、删除）
│   │   ├── listener.go                   # 监听器操作（查询、创建）
│   │   ├── target.go                     # 后端目标操作（绑定、解除）
│   │   ├── batch-listener.go             # 批量监听器创建优化
│   │   ├── batch-target.go               # 批量后端配置优化
│   │   ├── cache.go                      # 监听器缓存（减少 API 调用）
│   │   ├── quota.go                      # 配额查询和管理
│   │   ├── rate-limit.go                 # 限速控制（支持突发）
│   │   ├── lock.go                       # 并发控制锁
│   │   ├── wait.go                       # 异步等待机制
│   │   ├── batch.go                      # 批处理框架
│   │   ├── param.go                      # CLB 参数类型定义
│   │   └── error.go                      # CLB 错误处理
│   │
│   ├── cloudapi/              # 腾讯云 API 凭证管理
│   │   ├── credential.go                 # 凭证初始化和获取
│   │   └── ...
│   │
│   ├── clusterinfo/           # 集群信息管理
│   │   ├── clusterinfo.go                # 全局集群信息（ID、VPC、Region、OKG 标志）
│   │   └── ...
│   │
│   ├── kube/                  # Kubernetes 工具
│   │   ├── kube.go                       # Kube 初始化和工具方法
│   │   └── ...
│   │
│   ├── userinfo/              # 用户信息初始化
│   │   ├── userinfo.go                   # OKG 支持检测等
│   │   └── ...
│   │
│   ├── eventsource/           # 事件源（触发 Reconcile）
│   │   ├── eventsource.go                # 事件通道定义
│   │   └── ...
│   │
│   ├── util/                  # 通用工具函数
│   │   ├── util.go                       # 字符串、数值、环境变量等工具
│   │   └── ...
│   │
│   └── vpc/                   # VPC 相关操作
│       └── ...
│
├── cmd/                        # 二进制入口
│   ├── main.go                           # 程序主入口
│   └── app/
│       ├── cmd.go                        # Cobra 命令定义及标志
│       ├── manager.go                    # Manager 初始化和启动
│       ├── setup_manager.go              # Manager 组件设置和启动缓存初始化
│       ├── setup_controller.go           # Reconciler 注册
│       ├── setup_webhook.go              # Webhook 注册
│       └── discovery.go                  # 集群能力发现
│
├── config/                     # Kubernetes YAML 配置
│   ├── crd/                   # CRD 定义 YAML（自动生成）
│   ├── rbac/                  # RBAC 角色定义
│   │   ├── role.yaml                    # ClusterRole 权限定义
│   │   ├── role_binding.yaml            # ClusterRoleBinding
│   │   ├── leader_election_role.yaml    # 选举角色
│   │   ├── metrics_*.yaml               # 指标访问角色
│   │   ├── clbportpool_*.yaml           # CRD 相关角色
│   │   ├── clbpodbinding_*.yaml         # CRD 相关角色
│   │   └── clbnodebinding_*.yaml        # CRD 相关角色
│   │
│   ├── webhook/               # Webhook 配置
│   │   ├── manifests.yaml               # ValidatingWebhookConfiguration
│   │   ├── service.yaml                 # Webhook 服务定义
│   │   └── kustomization.yaml
│   │
│   ├── manager/               # Manager 部署配置
│   │   ├── manager.yaml                 # Manager Pod 定义
│   │   └── service.yaml
│   │
│   └── kustomization.yaml               # Kustomize 入口
│
├── deploy/                     # 部署相关脚本和配置
│   └── ...
│
├── charts/                     # Helm Chart
│   └── tke-extend-network-controller/
│       ├── Chart.yaml                   # Chart 元数据
│       ├── values.yaml                  # 默认配置值
│       ├── templates/
│       │   ├── deployment.yaml          # 控制器 Deployment
│       │   ├── rbac.yaml                # RBAC 相关
│       │   ├── crd.yaml                 # CRD 定义
│       │   └── webhook.yaml             # Webhook 配置
│       └── ...
│
├── hack/                       # 代码生成和构建脚本
│   └── boilerplate.go.txt               # 文件头模板
│
├── test/                       # 测试代码
│   ├── e2e/                   # E2E 测试
│   │   ├── e2e_test.go                  # E2E 测试用例
│   │   └── e2e_suite_test.go            # Ginkgo 测试套件
│   │
│   └── utils/                 # 测试工具
│       └── utils.go                     # 通用测试工具函数
│
├── docs/                       # 文档
│   ├── api.md                           # API 文档（自动生成）
│   └── ...
│
├── Makefile                    # 构建自动化
├── go.mod / go.sum            # 模块依赖
├── PROJECT                     # kubebuilder 项目元数据
├── Dockerfile                  # 生产镜像
├── Dockerfile.cross           # 交叉编译镜像
├── .golangci.yml              # golangci-lint 配置
├── .gitignore                 # Git 忽略规则
├── CHANGELOG.md               # 版本更新日志
└── README.md                  # 项目说明
```

## 目录用途

**`api/v1alpha1/` - Kubernetes 资源定义:**
- 用途：定义控制器管理的三个 CRD 和共享的规范结构
- 文件特点：包含 kubebuilder 代码生成注解，支持自动生成 YAML 和 DeepCopy 方法
- 关键文件：
  - `clbportpool_types.go` - 端口池资源，ClusterScoped
  - `clbpodbinding_types.go` - Pod 绑定资源，NamespaceScoped
  - `clbnodebinding_types.go` - Node 绑定资源，ClusterScoped
  - `clbbinding_types.go` - Pod/Node 绑定的公共规范（Spec、Status）

**`internal/controller/` - 控制器实现:**
- 用途：Reconciler 实现，驱动资源状态收敛
- 文件模式：每个资源一个 `*_controller.go` 和对应的 `*_controller_test.go`
- 关键职责：
  - 实现 `Reconcile()` 入口方法
  - 定义 RBAC 权限注解
  - 配置事件源和索引
- `clbbinding.go` - 通用绑定逻辑的泛型实现，被 Pod/Node Reconciler 使用

**`internal/clbbinding/` - 端口绑定抽象:**
- 用途：统一 Pod 和 Node 绑定的通用逻辑
- 核心：`CLBBinding` 接口，支持泛型约束
- 实现：`CLBPodBinding` 和 `CLBNodeBinding` 分别适配 Pod 和 Node 后端

**`internal/portpool/` - 端口资源管理:**
- 用途：管理全局和单个端口池的分配状态，支持多种 LB 挑选策略
- 核心概念：
  - `PortAllocator` - 全局单例，线程安全的多端口池管理
  - `PortPool` - 单个端口池内存表示
  - `LB 挑选策略` - Uniform/InOrder/Random 策略
- 接口定义在 `clbbinding.go`，实现在 `sort.go`

**`internal/webhook/` - API 验证:**
- 用途：Kubernetes Webhook 实现
- 当前：仅 CLBPortPool 的 ValidatingWebhook，验证字段不可变性
- 扩展点：可添加 MutatingWebhook 进行默认值设置

**`internal/constant/` - 常量定义:**
- 用途：集中管理常量，避免魔术字符串
- 包含：注解键、状态值、标签键等

**`pkg/clb/` - CLB SDK 封装:**
- 用途：对腾讯云 CLB 服务的统一封装
- 核心职责：
  - CLB 实例操作（创建、查询、删除）
  - 监听器操作（创建、配置、缓存）
  - 后端绑定（配置、删除）
  - 配额管理和监控
  - 限速和重试
- 特点：
  - 批量操作优化（`batch-listener.go` / `batch-target.go`）
  - 监听器缓存减少 API 调用
  - 错误分类和重试策略

**`pkg/cloudapi/` - API 凭证管理:**
- 用途：初始化和管理腾讯云 API 凭证
- 接口：`Init(secretId, secretKey)` 初始化凭证，`GetCredential()` 获取凭证

**`pkg/clusterinfo/` - 集群全局信息:**
- 用途：存储集群级别的全局配置
- 包含：
  - `ClusterId` - TKE 集群 ID
  - `VpcId` - VPC ID
  - `Region` - 腾讯云地域代码
  - `OKGSupported` - 是否支持 OpenKruiseGame

**`pkg/kube/` - Kubernetes 工具:**
- 用途：Kubernetes 集群交互工具函数
- 功能：Pod/Node 查询、字段索引设置等

**`pkg/userinfo/` - 用户信息初始化:**
- 用途：启动时检测集群能力
- 功能：OKG 支持检测，影响是否注册 GameServerSet Reconciler

**`pkg/eventsource/` - 事件源:**
- 用途：自定义事件通道，触发 Controller Reconcile
- 用法：当 CLB 配置变化时，通过事件源唤醒相关 Controller

**`cmd/main.go` - 二进制入口:**
- 位置：`cmd/main.go`
- 功能：加载并执行 rootCommand（Cobra 命令）

**`cmd/app/cmd.go` - 命令定义:**
- 位置：`cmd/app/cmd.go`
- 功能：Cobra 命令行定义，支持的标志（--region、--cluster-id、--secret-id 等）
- 调用：执行 `runManager()` 启动控制器

**`cmd/app/manager.go` - Manager 初始化:**
- 位置：`cmd/app/manager.go`
- 职责：
  1. 初始化日志（Zap）
  2. 加载和验证配置
  3. 初始化腾讯云凭证
  4. 创建 controller-runtime Manager
  5. 调用各个 Setup 函数
  6. 启动 Manager

**`cmd/app/setup_manager.go` - 组件设置:**
- 位置：`cmd/app/setup_manager.go`
- 职责：
  1. 启动 `initCache` 组件（需要选举）
  2. 调用 `SetupControllers()` 注册 Reconciler
  3. 调用 `SetupWebhooks()` 注册 Webhook
  4. 配置健康检查

**`cmd/app/setup_controller.go` - Reconciler 注册:**
- 位置：`cmd/app/setup_controller.go`
- 职责：创建并注册 6 个 Reconciler，配置工作线程数和索引

**`cmd/app/setup_webhook.go` - Webhook 注册:**
- 位置：`cmd/app/setup_webhook.go`
- 职责：注册 ValidatingWebhook（目前仅 CLBPortPool）

**`cmd/app/discovery.go` - 集群发现:**
- 位置：`cmd/app/discovery.go`
- 职责：可选的集群能力发现（如 OKG 支持）

**`config/` - Kubernetes 配置:**
- 用途：存储控制器部署所需的 YAML 文件
- 子目录：
  - `crd/` - CRD 定义（自动生成）
  - `rbac/` - RBAC 角色和绑定
  - `webhook/` - Webhook 配置
  - `manager/` - 控制器 Pod 和服务定义

**`test/` - 测试代码:**
- 用途：单元测试和 E2E 测试
- `test/e2e/` - 集成环境下的完整功能测试
- 测试框架：Ginkgo + Gomega

## 关键位置导航

**API 资源定义:**
- `CLBPortPool` - `api/v1alpha1/clbportpool_types.go`
- `CLBPodBinding` - `api/v1alpha1/clbpodbinding_types.go`
- `CLBNodeBinding` - `api/v1alpha1/clbnodebinding_types.go`
- 共享 Spec/Status - `api/v1alpha1/clbbinding_types.go`

**Reconciler 实现:**
- 端口池 - `internal/controller/clbportpool_controller.go`
- Pod 绑定 - `internal/controller/clbpodbinding_controller.go`
- Node 绑定 - `internal/controller/clbnodebinding_controller.go`
- Pod 监听 - `internal/controller/pod_controller.go`
- Node 监听 - `internal/controller/node_controller.go`
- GameServerSet - `internal/controller/gameserverset_controller.go` (可选)

**通用绑定逻辑:**
- 泛型 Reconciler - `internal/controller/clbbinding.go`
- Pod 绑定适配 - `internal/clbbinding/clbpodbinding.go`
- Node 绑定适配 - `internal/clbbinding/clbnodebinding.go`

**端口分配:**
- 全局分配器 - `internal/portpool/allocator.go`
- 单个端口池 - `internal/portpool/portpool.go`
- LB 挑选策略 - `internal/portpool/sort.go`
- 错误定义 - `internal/portpool/error.go`

**CLB 操作:**
- 客户端初始化 - `pkg/clb/clb.go`
- 实例操作 - `pkg/clb/instance.go`
- 监听器操作 - `pkg/clb/listener.go`
- 后端操作 - `pkg/clb/target.go`
- 批量优化 - `pkg/clb/batch-listener.go` / `pkg/clb/batch-target.go`
- 监听器缓存 - `pkg/clb/cache.go`
- 配额管理 - `pkg/clb/quota.go`
- 限速控制 - `pkg/clb/rate-limit.go`

**启动初始化:**
- 二进制入口 - `cmd/main.go`
- 命令定义 - `cmd/app/cmd.go`
- Manager 启动 - `cmd/app/manager.go`
- 组件设置 - `cmd/app/setup_manager.go`
- Reconciler 注册 - `cmd/app/setup_controller.go`
- Webhook 注册 - `cmd/app/setup_webhook.go`

**全局状态:**
- 集群信息 - `pkg/clusterinfo/clusterinfo.go`
- API 凭证 - `pkg/cloudapi/credential.go`
- 全局分配器 - `internal/portpool/allocator.go`（单例 `Allocator`）
- CLB 客户端缓存 - `pkg/clb/clb.go`（`clients` 映射按 region 缓存）

**配置和部署:**
- Helm Chart - `charts/tke-extend-network-controller/`
- RBAC 配置 - `config/rbac/`
- CRD 定义 - `config/crd/`
- Webhook 配置 - `config/webhook/`

## 命名约定

**文件命名:**
- `*_types.go` - Kubernetes 资源类型定义（含 kubebuilder 注解）
- `*_controller.go` - Reconciler 实现
- `*_controller_test.go` - 单元测试
- `*_webhook.go` - Webhook 实现
- `*_webhook_test.go` - Webhook 测试
- `*_suite_test.go` - Ginkgo 测试套件（setup/teardown）
- `zz_generated.*.go` - 自动生成的代码

**目录命名:**
- `api/` - API 资源定义
- `internal/` - 非导出的内部实现
- `pkg/` - 导出的公共包
- `cmd/` - 二进制程序
- `config/` - Kubernetes YAML 配置
- `test/` - 测试代码
- `deploy/` - 部署相关
- `charts/` - Helm Chart
- `docs/` - 文档
- `hack/` - 构建脚本和工具

**变量和常量命名:**
- `Allocator` - 全局端口分配器单例
- `scheme` - 全局 Kubernetes runtime.Scheme
- `setupLog` - 启动日志记录器
- `*Reconciler` - Reconciler 结构体
- `*Binding` - 绑定对象包装器
- `*Pool` - 端口池对象
- `Spec`/`Status` - CRD 规范和状态字段

## 新增代码位置指导

**新增 Reconciler (控制器):**
- 实现：`internal/controller/{resource}_controller.go`
- 测试：`internal/controller/{resource}_controller_test.go`
- 注册：修改 `cmd/app/setup_controller.go` 的 `SetupControllers()` 函数

**新增绑定类型:**
- API 定义：`api/v1alpha1/{resource}binding_types.go`
- 绑定实现：`internal/clbbinding/{resource}binding.go`
- 适配器注册：修改 `internal/clbbinding/clbbinding.go` 的接口实现

**新增业务逻辑工具:**
- 通用工具：`pkg/util/{功能}.go`（需导出）
- 特定模块工具：`{package}/util.go`（可选，内部使用）

**新增测试:**
- 单元测试：与被测文件同目录，命名为 `{file}_test.go`
- 测试数据：`test/utils/` 下的共用工具函数

**新增文档:**
- API 文档：自动从 CRD 生成至 `docs/api.md`（运行 `make generate-docs`）
- 设计文档：`docs/{设计主题}.md`

## 特殊目录

**`.planning/` - 代码审计和规划输出:**
- 生成来源：GSD 代码审计工具
- 用途：存储架构分析、结构分析等文档
- 特点：不提交到 git（在 `.gitignore` 中）

**`.worktrees/` - Git worktree 管理:**
- 用途：用于并行处理多个分支
- 特点：不提交到 git

**`bin/` - 编译输出:**
- 位置：`bin/manager` - 编译后的二进制文件
- 生成方式：`make build`

**`debug-roc/` - 调试和实验:**
- 用途：本地开发调试文件
- 特点：不提交到 git

---

*结构分析日期: 2025-03-20*
