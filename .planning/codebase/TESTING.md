# Testing Patterns

**Analysis Date:** 2025-01-10

## Test Framework

**Runner:**
- Ginkgo v2 (BDD-style Go testing framework)
- Version: `github.com/onsi/ginkgo/v2 v2.22.0`
- Config: Generated per spec in `internal/controller/suite_test.go`, `internal/webhook/v1alpha1/webhook_suite_test.go`

**Assertion Library:**
- Gomega v1.36.1
- BDD-style assertions: `Expect(...).To(Succeed())`, `Expect(...).NotTo(HaveOccurred())`
- Matchers: `HaveOccurred()`, `BeNil()`, `Equal()`, `Succeed()`

**Standard Library:**
- Go `testing` package for envtest integration
- `testing.T` entry point for Ginkgo suite
- `sync` package for concurrency testing (e.g., portpool tests)

**Run Commands:**
```bash
make test                          # Run all tests with envtest
KUBEBUILDER_ASSETS="..." go test ./...  # Manual with environment setup
go test ./internal/controller/... -run TestXxx -v  # Run specific test
make test-e2e                      # Run end-to-end tests
```

## Test File Organization

**Location:**
- Co-located with source code: `<source>_test.go` in same directory
- Controller tests: `internal/controller/<controller>_test.go` (e.g., `clbportpool_controller_test.go`)
- Webhook tests: `internal/webhook/v1alpha1/<webhook>_test.go`
- Unit tests: `pkg/<package>/<file>_test.go` and `internal/<package>/<file>_test.go`
- E2E tests: `test/e2e/e2e_test.go`, `test/e2e/e2e_suite_test.go`

**Naming:**
- Test files: `<name>_test.go`
- Suite files: `suite_test.go` (per package with tests)
- E2E suite: `e2e_suite_test.go`

**Structure:**
```
internal/controller/
├── clbportpool_controller.go
├── clbportpool_controller_test.go   # Tests for above
├── pod_controller.go
├── pod_controller_test.go
├── suite_test.go                    # Shared test setup (BeforeSuite, AfterSuite)
└── util.go
```

## Test Structure

**Suite Organization (Ginkgo BDD Style):**

Suite setup file (`internal/controller/suite_test.go`):
```go
func TestControllers(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
    // Initialize test environment
    // Set up logging
    // Create Kubernetes test environment (envtest)
    // Add schemes (networkingv1alpha1, corev1, etc.)
    // Initialize k8s client
})

var _ = AfterSuite(func() {
    // Cleanup test environment
    err := testEnv.Stop()
})
```

**Individual Test Structure:**

Controller test example from `internal/controller/clbportpool_controller_test.go`:
```go
var _ = Describe("CLBPortPool Controller", func() {
    Context("When reconciling a resource", func() {
        const resourceName = "test-resource"
        
        ctx := context.Background()
        typeNamespacedName := types.NamespacedName{
            Name:      resourceName,
            Namespace: "default",
        }
        
        BeforeEach(func() {
            By("creating the custom resource for the Kind CLBPortPool")
            // Test setup: create fixtures
        })
        
        AfterEach(func() {
            By("Cleanup the specific resource instance CLBPortPool")
            // Teardown: delete resources
        })
        
        It("should successfully reconcile the resource", func() {
            By("Reconciling the created resource")
            // Test execution
            Expect(...).To(Succeed())
        })
    })
})
```

**Patterns:**
- `Describe()`: test suite grouping
- `Context()`: contextual grouping (nested scenarios)
- `BeforeEach()`: setup before each test
- `AfterEach()`: cleanup after each test
- `It()`: individual test case
- `By()`: narrative step documentation
- `Expect(...).To(...)`: assertion

## Mocking

**Framework:** Ginkgo's built-in test utilities + controller-runtime envtest

**Patterns:**

**Real Kubernetes objects via envtest:**
```go
var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

// In BeforeSuite:
testEnv = &envtest.Environment{
    CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
    ErrorIfCRDPathMissing: true,
    BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s", fmt.Sprintf("1.30.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
}
cfg, err = testEnv.Start()
k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
```

**Reconciler Testing:**
Create reconciler instance directly and call methods:
```go
controllerReconciler := &CLBPortPoolReconciler{
    Client: k8sClient,
    Scheme: k8sClient.Scheme(),
}

_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
    NamespacedName: typeNamespacedName,
})
Expect(err).NotTo(HaveOccurred())
```

**Standard Library Testing (non-Ginkgo):**
Unit tests in `internal/portpool/portpool_test.go` use Go's standard `testing.T`:
```go
func TestRequestScaleUp(t *testing.T) {
    t.Run("首次请求成功", func(t *testing.T) {
        pp := &PortPool{Name: "test-pool"}
        if !pp.RequestScaleUp() {
            t.Error("首次 RequestScaleUp() 应返回 true")
        }
    })
}
```

**What to Mock:**
- External cloud APIs (CLB SDK calls) - typically not mocked in unit tests
- File system operations - use real files in test
- Kubernetes cluster state - use envtest (real in-memory cluster)
- Time/delays - use real delays in integration tests

**What NOT to Mock:**
- Kubernetes client operations (use real envtest cluster)
- CRD operations (use real cluster)
- Context operations (use real context)

## Fixtures and Factories

**Test Data:**

Ginkgo-style fixture creation (from test):
```go
resource := &networkingv1alpha1.CLBPortPool{
    ObjectMeta: metav1.ObjectMeta{
        Name:      resourceName,
        Namespace: "default",
    },
}
Expect(k8sClient.Create(ctx, resource)).To(Succeed())
```

**Factory Pattern:**
No dedicated factory package. Test objects created directly in tests:
```go
obj := &networkingv1alpha1.CLBPortPool{}
oldObj := &networkingv1alpha1.CLBPortPool{}
```

**Location:**
- Fixtures created inline in test functions
- Test utilities in `test/utils/utils.go`
- CRD definitions in `config/crd/bases/`

**Cleanup Pattern (from fixture in `clbportpool_controller_test.go`):**
```go
AfterEach(func() {
    resource := &networkingv1alpha1.CLBPortPool{}
    err := k8sClient.Get(ctx, typeNamespacedName, resource)
    Expect(err).NotTo(HaveOccurred())
    By("Cleanup the specific resource instance CLBPortPool")
    Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
})
```

## Coverage

**Requirements:** No strict target enforced in CI

**Current State:**
- Coverage file generated: `cover.out` (195KB, indicates reasonable test coverage)
- Coverage tracking enabled: `go test ... -coverprofile cover.out`

**View Coverage:**
```bash
go tool cover -html=cover.out
# or
go test ./... -coverprofile=cover.out && go tool cover -html=cover.out
```

**Make Target:**
```bash
make test  # Generates cover.out
```

## Test Types

**Unit Tests:**
- Scope: Individual functions/methods in isolation
- Framework: Go `testing` package (standard library)
- Example: `internal/portpool/portpool_test.go`
- Approach: No external dependencies, test port allocation logic, scale-up request state
- Pattern: Test concurrency with goroutines (1000 concurrent requests tested)

**Unit Test Example (`portpool_test.go`):**
```go
func TestRequestScaleUp(t *testing.T) {
    t.Run("首次请求成功", func(t *testing.T) {
        pp := &PortPool{Name: "test-pool"}
        if !pp.RequestScaleUp() {
            t.Error("首次 RequestScaleUp() 应返回 true")
        }
    })
    
    t.Run("并发请求只有一个成功", func(t *testing.T) {
        pp := &PortPool{Name: "test-pool"}
        goroutines := 1000
        successCount := 0
        var mu sync.Mutex
        var wg sync.WaitGroup
        wg.Add(goroutines)
        for i := 0; i < goroutines; i++ {
            go func() {
                defer wg.Done()
                if pp.RequestScaleUp() {
                    mu.Lock()
                    successCount++
                    mu.Unlock()
                }
            }()
        }
        wg.Wait()
        if successCount != 1 {
            t.Errorf("并发 %d 个请求，应只有 1 个成功，实际 %d 个成功", goroutines, successCount)
        }
    })
}
```

**Integration Tests:**
- Scope: Controllers reconciling Kubernetes resources
- Framework: Ginkgo v2 with envtest (Kubernetes in-memory cluster)
- Example: `internal/controller/clbportpool_controller_test.go`
- Approach: Create real CRDs, run reconciler, verify state changes
- Pattern: Use `BeforeEach`/`AfterEach` for fixture management

**Integration Test Example (from `clbportpool_controller_test.go`):**
```go
var _ = Describe("CLBPortPool Controller", func() {
    Context("When reconciling a resource", func() {
        BeforeEach(func() {
            By("creating the custom resource for the Kind CLBPortPool")
            resource := &networkingv1alpha1.CLBPortPool{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-resource",
                    Namespace: "default",
                },
            }
            Expect(k8sClient.Create(ctx, resource)).To(Succeed())
        })
        
        It("should successfully reconcile the resource", func() {
            reconciler := &CLBPortPoolReconciler{
                Client: k8sClient,
                Scheme: k8sClient.Scheme(),
            }
            _, err := reconciler.Reconcile(ctx, reconcile.Request{
                NamespacedName: typeNamespacedName,
            })
            Expect(err).NotTo(HaveOccurred())
        })
    })
})
```

**E2E Tests:**
- Framework: Ginkgo v2
- Files: `test/e2e/e2e_test.go`, `test/e2e/e2e_suite_test.go`
- Approach: Real Kubernetes cluster (Kind or TKE)
- Not yet implemented (template structure exists)
- Run: `make test-e2e`

**Webhook Tests:**
- Framework: Ginkgo v2
- Files: `internal/webhook/v1alpha1/clbportpool_webhook_test.go`, `webhook_suite_test.go`
- Pattern: Test defaulting and validation webhooks
- Current state: Mostly template structure (TODO comments)

## Common Patterns

**Async Testing (Goroutines in Tests):**

From `portpool_test.go`:
```go
t.Run("并发请求只有一个成功", func(t *testing.T) {
    pp := &PortPool{Name: "test-pool"}
    goroutines := 1000
    var wg sync.WaitGroup
    wg.Add(goroutines)
    for i := 0; i < goroutines; i++ {
        go func() {
            defer wg.Done()
            // Test operation
        }()
    }
    wg.Wait()
    // Verify results
})
```

**Error Testing (from reconciler tests):**
```go
It("should handle error scenarios", func() {
    // Trigger error condition
    _, err := reconciler.Reconcile(ctx, request)
    Expect(err).NotTo(HaveOccurred())  // or To(HaveOccurred()) for failure tests
})
```

**Fixture Lifecycle:**

```go
var _ = Describe("Resource Controller", func() {
    var resource *CustomResource
    
    BeforeEach(func() {
        // Create fresh fixture
        resource = &CustomResource{}
        Expect(k8sClient.Create(ctx, resource)).To(Succeed())
    })
    
    AfterEach(func() {
        // Cleanup
        Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
    })
    
    It("should verify behavior", func() {
        // Test uses resource
    })
})
```

## Test Organization Guidelines

**Where to Add Tests:**

**New Controller Tests:**
- File: `internal/controller/<controller>_test.go`
- Add to same package as controller
- Include in existing suite (`suite_test.go`)
- Use Ginkgo BDD style

**New Unit Tests:**
- File: `<package>/<file>_test.go` (co-located with source)
- Can use standard Go testing package or Ginkgo
- For pure unit tests without Kubernetes: use `testing.T`

**New Webhook Tests:**
- File: `internal/webhook/v1alpha1/<webhook>_test.go`
- Include in webhook suite (`webhook_suite_test.go`)
- Test validation and defaulting logic

**Test Configuration:**
- Envtest binaries downloaded to: `bin/k8s/1.30.0-<os>-<arch>/`
- CRD manifests referenced from: `config/crd/bases/`
- Scheme setup in suite file (BeforeSuite)

---

*Testing analysis: 2025-01-10*
