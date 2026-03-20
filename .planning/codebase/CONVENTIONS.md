# 代码规范

**分析日期:** 2025-01-09

## 命名规范

**文件名:**
- controller 相关: `*_controller.go`，例如 `clbportpool_controller.go`
- 测试文件: `*_test.go`，例如 `clbportpool_controller_test.go`
- Webhook: `*_webhook.go`
- 辅助文件: `util.go`、`sort.go`、`batch.go` 等按功能命名

**包名:**
- 小写、无下划线，例如 `controller`、`webhook`、`clbbinding`
- 按功能组织到对应目录

**函数名和方法名:**
- 采用 PascalCase（大驼峰），例如 `Reconcile`、`SetupWithManager`、`GetCLBInfo`
- 遵循 Go 导出规则：导出函数首字母大写，私有函数首字母小写
- Reconciler 相关方法通常为 `Reconcile`、`sync`、`cleanup`、`SetupWithManager`
- 辅助方法前缀表示操作：`ensure*`（确保状态）、`get*`（获取）、`create*`（创建）、`find*`（查找）、`cleanup*`（清理）

**变量名:**
- 使用 camelCase（小驼峰），例如 `lbId`、`clbPortPool`
- 简明扼要，例如 `ctx` 代替 `context`
- 布尔值通常以 `is`、`has` 或 `should` 开头，例如 `isServerlessNode`
- 错误变量统一为 `err`

**类型名:**
- 采用 PascalCase，例如 `CLBPortPool`、`PortBindingStatus`
- 接口名通常以 `-er` 结尾，例如 `CLBBinding`、`Backend`

**常量和枚举:**
- 采用 UPPER_SNAKE_CASE，例如 `TOTAL_LISTENER_QUOTA`（CRD 字段注释中）
- 在 `internal/constant` 中定义常量，例如 `constant.EnableCLBPortMappingsKey`、`constant.Finalizer`

## 代码风格

**格式化工具:**
- 使用 `go fmt` 进行代码格式化
- 使用 `goimports` 自动调整导入顺序
- 在 Makefile 中通过 `make fmt` 执行

**Linting:**
- 使用 `golangci-lint` v1.57.2，配置文件: `.golangci.yml`
- 启用的主要检查器: `errcheck`、`gofmt`、`goimports`、`gosimple`、`govet`、`staticcheck`、`typecheck`、`unused`、`revive`、`ginkgolinter`
- 在 Makefile 中通过 `make lint` 和 `make lint-fix` 执行
- api 目录和 internal 目录排除某些检查（如 `lll`、`dupl`）

**最大行长:**
- 通常不超过 100 字符（由 lll 检查器限制）
- api 定义和 internal 逻辑代码中排除行长检查

## 导入组织

**导入顺序（从上到下）:**

1. 标准库导入，例如 `context`、`fmt`、`time`
2. 第三方导入（按包名字母顺序），例如 `github.com/pkg/errors`、`github.com/onsi/ginkgo/v2`
3. 腾讯云 SDK 导入，例如 `github.com/tencentcloud/tencentcloud-sdk-go/...`
4. Kubernetes 相关导入，例如 `k8s.io/api`、`k8s.io/apimachinery`、`k8s.io/client-go`、`sigs.k8s.io/controller-runtime`
5. 项目内部导入，例如 `github.com/tkestack/tke-extend-network-controller/api/v1alpha1`

**示例（来自 `clbportpool_controller.go`）:**
```go
import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"
	"github.com/tkestack/tke-extend-network-controller/internal/portpool"
	"github.com/tkestack/tke-extend-network-controller/pkg/clb"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	"github.com/tkestack/tke-extend-network-controller/pkg/eventsource"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
)
```

## 错误处理

**错误包装:**
- 使用 `github.com/pkg/errors` 进行错误包装，保留调用栈信息
- 始终使用 `errors.WithStack(err)` 而不是 `fmt.Errorf`，便于调试
- 示例：`return errors.WithStack(err)`

**错误处理模式:**
- 立即检查错误，不延迟处理
- 在 Reconcile 方法中处理 API conflict 错误，自动 requeue：
  ```go
  if apierrors.IsConflict(err) {
      if !result.Requeue && result.RequeueAfter == 0 {
          result.RequeueAfter = 20 * time.Millisecond
      }
      return result, nil
  }
  ```
- 使用 `client.IgnoreNotFound(err)` 忽略资源不存在的错误
- 检查 `apierrors.IsNotFound(err)` 和 `apierrors.IsConflict(err)` 处理特殊 API 错误

**返回值:**
- 通常返回 `(ctrl.Result, error)` 对
- Reconcile 方法签名: `(ctx context.Context, req ctrl.Request) (ctrl.Result, error)`
- 辅助方法通常使用具名返回值，便于提前返回：`(result ctrl.Result, err error)`

## 日志记录

**日志框架:**
- 使用 `github.com/go-logr/logr` 接口和 `sigs.k8s.io/controller-runtime/pkg/log`
- 项目中统一从 context 获取 logger: `log.FromContext(ctx)`

**日志记录模式:**
- 在 context 中获取 logger：`log := log.FromContext(ctx)`
- 调用日志方法：`log.Info(msg, key, value, ...)`、`log.Error(err, msg, key, value, ...)`
- 支持日志级别控制，使用 `.V(verbosity).Info(...)` 用于详细日志
- 示例（来自 `clbbinding.go`）：
  ```go
  log := log.FromContext(ctx, "binding", binding)
  log.FromContext(ctx).Info("lb info not found in pool yet, will retry", "err", err)
  log.FromContext(ctx).V(1).Info("not bind backend due to backend not found")
  ```

**webhook 中的日志:**
- 定义包级别的 logger: `var clbportpoollog = logf.Log.WithName("clbportpool-resource")`
- 在 webhook 方法中使用：`clbportpoollog.Info("Defaulting for CLBPortPool", "name", clbportpool.GetName())`

## 注释规范

**文档注释:**
- 导出的函数和类型必须有文档注释，以函数名或类型名开头
- 格式: `// FunctionName 描述`
- 示例：`// CLBPortPoolReconciler reconciles a CLBPortPool object`、`// Reconcile is part of the main kubernetes reconciliation loop...`

**kubebuilder 注释（CRD 生成）:**
- 使用 `+kubebuilder:` 前缀注释控制 CRD 生成
- 示例：
  - `// +kubebuilder:rbac:groups=networking.cloud.tencent.com,resources=clbportpools,verbs=get;list;watch;create;update;patch;delete`
  - `// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="Value is immutable"`
  - `// +kubebuilder:webhook:path=/mutate-networking-cloud-tencent-com-v1alpha1-clbportpool,mutating=true,failurePolicy=fail,...`

**代码注释:**
- 中文注释用于解释复杂的业务逻辑和边界情况
- 注释应该说明"为什么"而不是"做什么"
- 示例（来自代码）：
  ```go
  // 查询 lb 信息并保存到 context，以供后面的 ensureXXX 对账使用
  // 拿到所有需要查询的 LbId
  // 清理端口池
  // 从全局端口分配器缓存中移除该端口池
  // 删除自动创建的 CLB
  ```

**TODO 和 FIXME:**
- 使用格式: `// TODO(user): description` 或 `// FIXME: description`
- 示例（自动生成的测试模板中）: `// TODO(user): Specify other spec details if needed.`
- 实际代码中的 TODO: `// TODO: 改成并发` （表示改进方向）

## 代码组织模式

**Reconciler 结构体:**
- 必须嵌入 `client.Client` 和包含 `Scheme *runtime.Scheme` 及 `Recorder record.EventRecorder`
- 示例（`CLBPortPoolReconciler`）：
  ```go
  type CLBPortPoolReconciler struct {
      client.Client
      Scheme   *runtime.Scheme
      Recorder record.EventRecorder
  }
  ```

**Reconcile 流程:**
- 标准模式: `ReconcileWithFinalizer` 或 `Reconcile` 分离同步和清理逻辑
- 包装函数 `ReconcileWithFinalizer` 自动处理 finalizer 和删除流程
- Reconcile 方法调用同步和清理函数：`syncFunc`、`cleanupFunc`

**SetupWithManager 方法:**
- 所有 Reconciler 都需要实现此方法，将其注册到 Manager
- 接收 `ctrl.Manager` 和可选的 worker 数量
- 示例：
  ```go
  func (r *CLBPortPoolReconciler) SetupWithManager(mgr ctrl.Manager, workers int) error {
      return ctrl.NewControllerManagedBy(mgr).
          For(&networkingv1alpha1.CLBPortPool{}).
          Owns(&networkingv1alpha1.CLBPodBinding{}).
          Watches(...).
          WithOptions(controller.Options{MaxConcurrentReconciles: workers}).
          Complete(r)
  }
  ```

**泛型 Reconciler:**
- 某些 Reconciler 使用泛型以支持多种对象类型
- 示例：`CLBBindingReconciler[T]` 是泛型类型，可以处理不同的绑定资源

**接口设计:**
- 定义接口表示抽象概念，例如 `CLBBinding`、`Backend`
- 接口方法通过嵌入 `client.Object` 获得 Kubernetes 对象能力
- 示例（`clbbinding/clbbinding.go`）：
  ```go
  type CLBBinding interface {
      client.Object
      GetSpec() *networkingv1alpha1.CLBBindingSpec
      GetStatus() *networkingv1alpha1.CLBBindingStatus
      GetAssociatedObject(context.Context, client.Client) (Backend, error)
      GetObject() client.Object
      GetType() string
  }
  ```

## Go 版本和特性

**Go 版本:** 1.24.0（go.mod 指定 `go 1.24.0`）

**使用的现代 Go 特性:**
- 泛型（Generic Types）：`CLBBindingReconciler[T]`、`Reconcile[T]` 等
- 类型参数：多个 utility 函数使用，例如 `GetPtr[T]`、`GetValue[T]`
- Context 作为第一参数：所有涉及 I/O 的函数都接收 `context.Context`

## 跨领域关注

**验证:**
- CRD 字段验证使用 `+kubebuilder:validation:XValidation` 规则
- 不可变字段通过 CEL 规则实现：`rule="self == oldSelf", message="Value is immutable"`
- Webhook 中实现自定义验证逻辑

**认证与授权:**
- 在 Reconcile 方法上使用 `// +kubebuilder:rbac` 注释声明权限要求
- 权限包括 get、list、watch、create、update、patch、delete 等操作

---

*规范分析: 2025-01-09*
