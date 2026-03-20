# 架构设计

**分析日期:** 2025-03-20

## 模式概览

**总体模式:** Kubernetes Operator（控制器运算符）+ 管理层架构

**核心特点:**
- 基于 controller-runtime 框架的多控制器并行协调
- 声明式 Kubernetes API 资源驱动的网络配置管理
- 三层分离：CRD 定义层（API）→ 控制层（Reconciler）→ 实现层（端口池 + CLB SDK 封装）
- 专门化的端口分配器单例管理全局端口资源
- 通过 Webhook 进行验证和变更默认值

## 分层结构

**API 层 (`api/v1alpha1/`):**
- 作用：定义 Kubernetes 自定义资源的 API 规范
- 位置：`api/v1alpha1/`
- 包含：
  - `CLBPortPool` - 集群范围端口池资源，定义可分配的公网端口段
  - `CLBPodBinding` - 命名空间范围资源，为 Pod 声明式绑定 CLB 端口映射
  - `CLBNodeBinding` - 集群范围资源，为 Node HostPort 场景分配 CLB 端口映射
- 依赖于：Kubernetes API machinery
- 被依赖：所有控制器、Webhook、绑定实现

**控制层 (`internal/controller/`):**
- 作用：驱动期望状态向实际状态收敛的核心协调逻辑
- 位置：`internal/controller/`
- 包含六个 Reconciler：
  - `CLBPortPoolReconciler` - 管理端口池生命周期、CLB 实例维护、扩缩容
  - `CLBPodBindingReconciler` - 处理 Pod 端口绑定请求的直接调用
  - `CLBNodeBindingReconciler` - 处理 Node HostPort 绑定请求
  - `PodReconciler` - 监听 Pod 对象变化，自动创建/更新对应的 CLBPodBinding
  - `NodeReconciler` - 监听 Node 对象变化，处理 HostPort 场景的 CLBNodeBinding
  - `GameServerSetReconciler` - 可选，支持 OpenKruiseGame 的 GameServerSet 绑定
- 依赖于：Kubernetes client、端口分配器、CLB SDK、绑定实现
- 使用 Finalizer 实现安全的资源删除

**绑定实现层 (`internal/clbbinding/`):**
- 作用：抽象 Pod 和 Node 后端的端口绑定通用逻辑
- 位置：`internal/clbbinding/`
- 核心抽象：`CLBBinding` 接口，支持泛型实现
  - `CLBPodBinding` - Pod 后端绑定实现（包装 `networkingv1alpha1.CLBPodBinding`）
  - `CLBNodeBinding` - Node 后端绑定实现（包装 `networkingv1alpha1.CLBNodeBinding`）
- 绑定流程：端口分配 → CLB 监听器创建 → 后端绑定 → 状态写回
- 依赖于：端口分配器、CLB SDK、Kubernetes API
- 被依赖：Pod/Node Reconciler

**端口分配层 (`internal/portpool/`):**
- 作用：管理全局端口资源分配，支持多种 LB 挑选策略
- 位置：`internal/portpool/`
- 核心类型：
  - `PortAllocator` - 全局单例，管理所有端口池
  - `PortPool` - 单个端口池实例，管理分配状态和 LB 缓存
  - `LBKey` - 标识 CLB 实例的元组（LoadBalancer ID + Region）
- 支持三种 LB 挑选策略：
  - `Uniform` - 均匀分配（减小 DDoS 风险，IP 分散）
  - `InOrder` - 顺序分配（提高单 CLB 利用率）
  - `Random` - 随机分配（默认）
- 端口分配流程：获取端口池 → 选择 LB → 占用端口 → 返回 (LB, Port) 对
- 依赖于：CLBPortPool 对象模型
- 被依赖：CLB 绑定实现

**CLB SDK 封装层 (`pkg/clb/`):**
- 作用：对腾讯云 CLB 服务的统一封装，提供高阶操作
- 位置：`pkg/clb/`
- 核心能力：
  - `instance.go` - CLB 实例的查询、创建、删除
  - `listener.go` - 监听器的创建、查询，包含配额管理
  - `target.go` - 后端绑定的配置、删除
  - `batch-listener.go` - 批量创建监听器优化
  - `batch-target.go` - 批量配置后端
  - `cache.go` - 监听器信息缓存减少 API 调用
  - `rate-limit.go` - 限速控制（支持突发，避免频率限制）
  - `quota.go` - 账号维度和 CLB 维度配额查询与管理
- 错误重试和幂等性保证
- 依赖于：腾讯云 CLB SDK（`tencentcloud-sdk-go`）、凭证管理
- 被依赖：CLB 绑定实现、CLBPortPool Reconciler

**基础设施层 (`pkg/`):**
- `cloudapi/` - 腾讯云 API 凭证初始化和管理
- `clusterinfo/` - 全局集群信息（ID、VPC ID、Region、OKG 支持标志）
- `kube/` - Kubernetes 集群交互工具
- `userinfo/` - 用户信息初始化（OpenKruiseGame 支持检测）
- `util/` - 通用工具函数
- `eventsource/` - 事件源，用于在 CLB 变化时触发 Controller Reconcile

## 数据流

**Pod 端口绑定流程:**

```
1. [触发] Pod 创建/更新 或 用户创建 CLBPodBinding
   ↓
2. [监听] PodReconciler 或 CLBPodBindingReconciler 收到事件
   ↓
3. [绑定实现] CLBPodBinding.Reconcile() 执行：
   - 检查 Pod 是否有 annotation: "cloud.tencent.com/enable-clb-port-mappings"
   - 从 CLBPodBinding Spec 中读取端口池名称和需求的端口
   ↓
4. [分配端口] PortAllocator.AllocatePorts() 执行：
   - 从指定端口池获取 PortPool 实例
   - 根据 LB 挑选策略选择 CLB 实例
   - 在该 CLB 中占用端口
   - 返回 [(LB_ID, Port, Protocol), ...]
   ↓
5. [创建监听器] 若监听器不存在，调用 clb.CreateListener()：
   - 向 CLB 实例添加监听器
   - 缓存监听器信息
   ↓
6. [绑定后端] clb.ConfigureTarget() 执行：
   - 将 Pod IP 作为后端绑定到监听器
   - 记录绑定关系
   ↓
7. [更新状态] CLBPodBinding Status 写入绑定结果：
   - status.state = "Bound"
   - status.allocations = [(ip, port, protocol), ...]
   - 记录到 Pod Annotation
```

**端口池扩缩容流程:**

```
1. [监听扩容请求] 当端口不足时，CLB 绑定实现调用 PortAllocator.RequestScaleUp()
   ↓
2. [标记扩容] PortPool 内部状态标记需要扩容（带一轮吸收机制）
   ↓
3. [触发 Reconcile] CLBPortPool Reconciler 检测到扩容标记
   ↓
4. [扩容决策] 若启用 AutoCreate 配置，且未达上限：
   - 检查当前 CLB 实例数
   - 验证创建新 CLB 是否超过上限
   ↓
5. [创建 CLB] clb.CreateLoadBalancer() 执行：
   - 创建新的负载均衡器实例
   - 预创建固定数量的监听器（若配置启用）
   ↓
6. [更新状态] CLBPortPool Status 记录新的 LB 信息：
   - status.loadbalancerStatuses 追加新 LB
   - 标记 AutoCreated = true
   ↓
7. [更新分配器] PortAllocator 内存缓存同步新增的 LB
   ↓
8. [清除标记] ResetScaleUpRequest() 清除扩容标记，继续服务新请求
```

**资源删除流程:**

```
1. [删除触发] 用户删除 CLBPodBinding / CLBNodeBinding / CLBPortPool
   ↓
2. [Finalizer 保护] Reconciler 检测到 DeletionTimestamp 不为空
   ↓
3. [资源清理] 调用 cleanup() 方法：
   - CLBPodBinding/CLBNodeBinding: 解除后端绑定，释放端口，标记删除状态
   - CLBPortPool: 从全局分配器缓存移除，删除自动创建的 CLB
   ↓
4. [移除 Finalizer] 清理完成后调用 controllerutil.RemoveFinalizer()
   ↓
5. [确认删除] Kubernetes API Server 完成对象删除
```

**状态机:**

```
CLBBinding 状态转换：
  Pending → Bound → Disabled (可选)
              ↓
         NoPortAvailable (端口不足)
              ↓
         PortPoolNotAllocatable (端口池不可用)
              ↓
         PortPoolNotFound (端口池不存在)
         
CLBPortPool 状态：
  Active (运行中)
     ↓
  Deleting (删除进行中，资源清理中)
```

## 核心抽象

**CLBBinding 接口 (`internal/clbbinding/clbbinding.go`):**
- 目的：统一 Pod 和 Node 的端口绑定实现逻辑
- 方法：
  - `GetSpec()` - 获取绑定规范（端口需求、端口池选择）
  - `GetStatus()` - 获取绑定状态（当前分配的端口）
  - `GetObject()` - 获取底层 Kubernetes 对象
  - `GetType()` - 返回绑定类型标识（"CLBPodBinding" / "CLBNodeBinding"）
- 优势：CLBBindingReconciler 泛型实现可同时处理两种绑定

**Backend 接口系列（分别在 `clbbinding/clbpodbinding.go` 和 `clbbinding/clbnodebinding.go`）:**
- `podBackend` - 封装 Pod 对象，提供获取 IP、Node 等能力
- `nodeBackend` - 封装 Node 对象，提供获取 IP、HostPort 能力
- 用途：解耦后端资源的获取和绑定逻辑

**PortPool 抽象:**
- 单个端口池的内存表示，包含：
  - 端口范围和段长度信息
  - LB 列表和对应的端口占用状态
  - 扩容请求标记和一轮吸收计数器
- 支持原子性的端口分配操作

**LB 挑选策略:**
- 在 `internal/portpool/sort.go` 中实现
- `Uniform` - 按 CLB 可用端口数排序，优先选择可用端口最少的（均衡负载）
- `InOrder` - 按顺序遍历，选择第一个可分配的
- `Random` - 随机选择可分配的 CLB
- 策略选择通过 `CLBPortPool.Spec.lbPolicy` 字段配置

## 入口点

**二进制入口 (`cmd/main.go`):**
- 位置：`cmd/main.go`
- 触发：`./manager` 或 `make run`
- 职责：初始化 rootCommand 并执行

**命令入口 (`cmd/app/cmd.go`):**
- 位置：`cmd/app/cmd.go`
- 职责：Cobra 命令定义，支持的标志：
  - `--region` - 腾讯云地域代码
  - `--cluster-id` - TKE 集群 ID
  - `--vpc-id` - VPC ID
  - `--secret-id` / `--secret-key` - 腾讯云 API 凭证
  - `--metrics-bind-address` - 指标服务地址
  - `--health-probe-bind-address` - 健康检查地址
  - `--leader-elect` - 启用选举模式
- 调用 `runManager()` 启动控制器

**Manager 初始化 (`cmd/app/manager.go`):**
- 位置：`cmd/app/manager.go`
- 触发：`runManager()` 中调用
- 职责：
  1. 初始化日志系统（Zap）
  2. 获取和验证配置参数（region, cluster-id, vpc-id）
  3. 初始化腾讯云 API 凭证（`cloudapi.Init`）
  4. 验证 CLB 配额可用性
  5. 初始化用户信息（OKG 支持检测）
  6. 创建 controller-runtime Manager
  7. 调用 `SetupManager()` 挂载各个组件

**Manager 组件设置 (`cmd/app/setup_manager.go`):**
- 位置：`cmd/app/setup_manager.go`
- 职责：
  1. 启动 `initCache` 组件进行启动初始化（需要选举领导权）
     - 加载所有 CLBPortPool 对象到 PortAllocator
     - 初始化 Pod 端口绑定缓存
  2. 调用 `SetupControllers()` 注册所有 Reconciler
  3. 调用 `SetupWebhooks()` 注册 Webhook
  4. 配置健康检查端点

**Reconciler 注册 (`cmd/app/setup_controller.go`):**
- 位置：`cmd/app/setup_controller.go`
- 职责：创建并注册 6 个 Reconciler：
  - CLBPortPoolReconciler
  - CLBPodBindingReconciler
  - CLBNodeBindingReconciler
  - PodReconciler（带事件源过滤特定注解的 Pod）
  - NodeReconciler（带事件源过滤）
  - GameServerSetReconciler（条件注册）
- 配置：
  - 控制器工作线程数（通过环境变量 `WORKER_*_CONTROLLER` 配置）
  - 事件记录器
  - 索引器配置（Pod 按 `.status.podIP` 索引，Node 按 `.status.nodeIP` 索引）

**Webhook 设置 (`cmd/app/setup_webhook.go`):**
- 位置：`cmd/app/setup_webhook.go`
- 职责：注册验证和变更 Webhook
  - `CLBPortPool` ValidatingWebhook - 验证字段不可变性和配置合法性

## 错误处理

**策略:** 分层错误处理 + 重试 + 状态反馈

**模式:**

1. **CLB API 错误** (`pkg/clb/error.go`):
   - 区分临时错误（重试）和永久错误（返回错误）
   - 常见错误映射：
     - 监听器已存在 → 幂等性处理（忽略并继续）
     - 配额不足 → 返回 ErrResourceSoldOut（触发扩容）
     - 频率限制 → 指数退避重试

2. **端口分配错误** (`internal/portpool/error.go`):
   - `ErrPortPoolNotAllocatable` - 端口池不可分配（标记 CLBBinding 状态为 PortPoolNotAllocatable）
   - `ErrNoPortAvailable` - 端口不足（触发扩容请求并标记状态为 NoPortAvailable）
   - `ErrPoolNotFound` - 端口池不存在（标记状态为 PortPoolNotFound）

3. **Reconciler 错误处理** (`internal/controller/clbbinding.go` 的 `sync()` 方法):
   - 捕获特定错误类型进行状态转换
   - 分类处理：
     - 临时错误（如 LB 未准备好）→ RequeueAfter（延迟重试）
     - 配置错误（如端口池不存在）→ 状态标记 + Event + 不重试
     - 资源错误（如端口不足）→ 状态标记 + 扩容请求
   - 所有错误都用 `errors.WithStack()` 保留堆栈跟踪

4. **Event 记录:**
   - 重要事件通过 `record.EventRecorder` 写入 Kubernetes Event
   - Pod/Node 无法找到时记录 Warning Event

## 跨模块关键点

**日志:** 
- 基于 controller-runtime 的 logr 接口
- 通过 Zap 后端实现
- 各模块通过 `ctrl.Log.WithName()` 创建命名日志
- CLB API 调用通过 `LogAPI()` 统一记录，支持不同详细级别

**验证:**
- CRD 层通过 `+kubebuilder:validation:XValidation` 实现字段不可变性验证
- Webhook 阶段进行更深入的业务逻辑验证

**认证:**
- 基于 RBAC 角色，配置在 `config/rbac/` 下的 YAML 文件
- Reconciler 通过 kubebuilder 注解自动声明所需权限

**状态管理:**
- 所有受控资源维护明确的 `.status` 子资源
- 状态更新通过 `client.Status().Update()` 原子性执行
- CLBBinding 状态包含分配的端口列表和当前状态

---

*架构分析日期: 2025-03-20*
