# 测试模式

**分析日期:** 2025-01-09

## 测试框架

**测试运行器:**
- Ginkgo v2.22.0 - BDD 风格的 Go 测试框架
- 配置文件: Makefile 中定义，无单独配置文件
- 项目中测试命令通过 `go test` 运行，Ginkgo 用作测试描述和断言库

**断言库:**
- Gomega v1.36.1 - 与 Ginkgo 配合使用的匹配和断言库
- 导入方式: `. "github.com/onsi/gomega"` （点导入用于 BDD 语法）

**运行命令:**
```bash
make test              # 运行单元测试（执行 envtest，排除 e2e）
make test-e2e          # 运行 E2E 测试（针对真实 Kind Kubernetes 集群）
go test ./internal/controller/... -run TestXxx -v  # 运行单个测试
```

## 测试文件组织

**位置:**
- 单元测试与被测文件在同一目录
- 测试文件名: `*_test.go`
- 例如: `internal/controller/clbportpool_controller.go` 对应 `internal/controller/clbportpool_controller_test.go`

**命名:**
- 测试函数通过 Ginkgo `Describe`/`Context`/`It` 组织，而非传统的 `TestXxx` 函数
- 套件测试文件: `suite_test.go`（例如 `internal/controller/suite_test.go`）
- Webhook 测试: `internal/webhook/v1alpha1/webhook_suite_test.go`

**测试文件计数:**
- 项目共 27 个测试文件

## 测试结构

**Ginkgo 测试布局:**

```go
var _ = Describe("CLBPortPool Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		clbportpool := &networkingv1alpha1.CLBPortPool{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind CLBPortPool")
			err := k8sClient.Get(ctx, typeNamespacedName, clbportpool)
			if err != nil && errors.IsNotFound(err) {
				resource := &networkingv1alpha1.CLBPortPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &networkingv1alpha1.CLBPortPool{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CLBPortPool")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &CLBPortPoolReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
```

**结构要点:**
- `Describe` - 顶级测试套件描述
- `Context` - 测试上下文分组，用于描述特定场景
- `BeforeEach` - 每个测试前执行的设置
- `AfterEach` - 每个测试后执行的清理
- `It` - 单个测试用例
- `By` - 在测试执行中打印步骤描述（用于日志输出）

## 套件测试配置

**Suite 初始化（`internal/controller/suite_test.go`）:**

```go
var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.30.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = networkingv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = gamekruiseiov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
```

**要点:**
- `envtest.Environment` - Kubernetes API 服务器的本地测试环境
- CRD 目录指向 `config/crd/bases`，用于加载自定义资源定义
- Kubernetes 版本: 1.30.0（通过 `ENVTEST_K8S_VERSION` 在 Makefile 中定义）
- 日志使用 Zap 配置，支持详细模式
- 注册所有需要的 CRD Scheme（`networkingv1alpha1`、`corev1`、`gamekruiseiov1alpha1`）

## 测试的 Mock 和 Fixture

**Mock 模式:**
- 项目中主要通过 `envtest` 使用真实的 Kubernetes 客户端进行集成测试
- 不依赖专门的 Mock 框架（如 testify/mock）
- 通过创建真实的测试资源对象进行测试

**测试数据创建:**
- 直接创建 Kubernetes 对象实例（例如 `CLBPortPool`）
- 设置必要的元数据和 spec 字段
- 使用 `k8sClient.Create()` 将对象写入测试集群
- 示例：
  ```go
  resource := &networkingv1alpha1.CLBPortPool{
      ObjectMeta: metav1.ObjectMeta{
          Name:      resourceName,
          Namespace: "default",
      },
  }
  Expect(k8sClient.Create(ctx, resource)).To(Succeed())
  ```

**Fixture 位置:**
- 示例资源配置位于 `config/samples/`
- 测试中通常硬编码创建测试对象，而非从文件加载

## 覆盖率

**覆盖率要求:**
- 未显式指定覆盖率目标
- Makefile 中 `make test` 生成 `cover.out` 文件用于覆盖率追踪

**查看覆盖率:**
```bash
make test                    # 生成 cover.out
go tool cover -html=cover.out # 在浏览器中查看
```

## 测试类型

**单元测试:**
- 范围: `internal/controller/*_test.go` 和 `internal/webhook/*_test.go`
- 使用 envtest 在隔离的 Kubernetes 环境中测试 Controller 逻辑
- 测试 Reconcile 循环、状态管理、事件处理等
- 示例: `clbportpool_controller_test.go`、`clbpodbinding_controller_test.go`
- 所有单元测试排除后缀 `/e2e` 的包：`go list ./... | grep -v /e2e`

**集成测试:**
- 范围: `internal/controller/*_test.go` 使用 envtest 进行集成测试
- 与 Kubernetes 客户端、CRD 和其他 controller 交互
- 验证完整的 reconciliation 流程

**E2E 测试:**
- 位置: `test/e2e/`（例如 `test/e2e/e2e_test.go`、`test/e2e/e2e_suite_test.go`）
- 针对真实的 Kind Kubernetes 集群运行
- 命令: `make test-e2e`
- 用于验证完整的端到端场景

**单元测试的通用模式:**

测试文件通常包含以下模式（部分测试为 kubebuilder 生成的模板）：

```go
var _ = Describe("CLBPortPool Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		clbportpool := &networkingv1alpha1.CLBPortPool{}

		BeforeEach(func() {
			// 创建测试资源
		})

		AfterEach(func() {
			// 清理测试资源
		})

		It("should successfully reconcile the resource", func() {
			// 测试 Reconcile 逻辑
		})
	})
})
```

## 关键 API 及测试模式

**Ginkgo 断言（使用 Gomega）:**
- `Expect(value).To(Succeed())` - 断言没有错误
- `Expect(err).NotTo(HaveOccurred())` - 断言错误为 nil
- `Expect(err).To(HaveOccurred())` - 断言发生了错误
- `Expect(k8sClient.Get(...)).To(Succeed())` - 断言获取成功
- `Expect(k8sClient.Delete(...)).To(Succeed())` - 断言删除成功
- `Expect(value).To(Equal(expected))` - 值相等性判断

**Context 和测试客户端:**
```go
ctx := context.Background()
k8sClient.Get(ctx, namespacedName, obj)      // 获取资源
k8sClient.Create(ctx, obj)                   // 创建资源
k8sClient.Delete(ctx, obj)                   // 删除资源
k8sClient.Update(ctx, obj)                   // 更新资源
k8sClient.Status().Update(ctx, obj)          // 更新状态子资源
```

**Reconcile 调用:**
```go
controllerReconciler := &CLBPortPoolReconciler{
    Client: k8sClient,
    Scheme: k8sClient.Scheme(),
}

result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
    NamespacedName: typeNamespacedName,
})
Expect(err).NotTo(HaveOccurred())
```

## 测试清理模式

**资源清理:**
- 所有 `BeforeEach` 中创建的资源在 `AfterEach` 中删除
- 标准模式：获取资源 → 验证无错误 → 删除资源
- 示例（来自测试文件）：
  ```go
  AfterEach(func() {
      resource := &networkingv1alpha1.CLBPortPool{}
      err := k8sClient.Get(ctx, typeNamespacedName, resource)
      Expect(err).NotTo(HaveOccurred())

      By("Cleanup the specific resource instance CLBPortPool")
      Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
  })
  ```

**Suite 清理:**
- `AfterSuite` 中调用 `testEnv.Stop()` 停止 Kubernetes 测试环境
- 自动清理所有测试资源和临时配置

## 快速测试执行

**测试前准备:**
- 所有测试依赖于 `make manifests generate fmt vet` 的输出
- Makefile 中 `make test` 会自动执行这些前置步骤

**命令集合:**
```bash
make test          # 完整流程：生成 → 格式化 → lint → 运行测试
make test-e2e      # E2E 测试
make lint-fix       # 修复代码风格问题后再测试
```

## 测试覆盖的组件

**Controller 测试:**
- `internal/controller/clbportpool_controller_test.go`
- `internal/controller/clbpodbinding_controller_test.go`
- `internal/controller/clbnodebinding_controller_test.go`
- `internal/controller/pod_controller_test.go`
- `internal/controller/node_controller_test.go`
- `internal/controller/gameserverset_controller_test.go`

**Webhook 测试:**
- `internal/webhook/v1alpha1/clbportpool_webhook_test.go`
- `internal/webhook/v1alpha1/webhook_suite_test.go`

**工具函数测试:**
- `pkg/util/pointer_test.go`
- `pkg/util/slice_test.go`

## 常见测试模式的示例

**测试 Reconcile 流程:**

```go
It("should successfully reconcile the resource", func() {
    By("creating the custom resource")
    resource := &networkingv1alpha1.CLBPortPool{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-resource",
            Namespace: "default",
        },
    }
    Expect(k8sClient.Create(ctx, resource)).To(Succeed())

    By("reconciling the created resource")
    controllerReconciler := &CLBPortPoolReconciler{
        Client: k8sClient,
        Scheme: k8sClient.Scheme(),
    }

    _, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
        NamespacedName: types.NamespacedName{
            Name:      "test-resource",
            Namespace: "default",
        },
    })
    Expect(err).NotTo(HaveOccurred())
})
```

**异步测试模式:**
- Ginkgo 原生支持异步测试和超时控制
- 使用 `Eventually` 进行轮询断言
- 示例（Ginkgo v2 模式）：
  ```go
  Eventually(func() error {
      return k8sClient.Get(ctx, namespacedName, obj)
  }, timeout, interval).Should(Succeed())
  ```

## 测试的环境变量

**Kubernetes 版本:**
- `ENVTEST_K8S_VERSION = 1.30.0` （在 Makefile 中定义）
- `KUBEBUILDER_ASSETS` - 在 `make test` 执行时由 envtest 设置

**覆盖率路径:**
- `cover.out` - 生成的覆盖率文件

---

*测试分析: 2025-01-09*
