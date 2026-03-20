# Coding Conventions

**Analysis Date:** 2025-01-10

## Naming Patterns

**Files:**
- Lowercase with hyphens: `batch-listener.go`, `batch-target.go`
- Test files suffixed with `_test.go` in same package directory
- Controllers: `clbportpool_controller.go`, `pod_controller.go`
- Controllers suffixed with `_controller.go`
- Webhooks: `clbportpool_webhook.go`, `clbportpool_webhook_test.go`

**Functions:**
- PascalCase for exported functions (Go convention)
- camelCase for unexported functions
- Reconciler methods follow controller-runtime pattern: `Reconcile()`, `SetupWithManager()`
- Cleanup/sync methods: `cleanup()`, `sync()`, lowercase unexported
- Helper functions: `ReconcileWithFinalizer()`, `Reconcile()` (parameterized generic functions in `internal/controller/util.go`)

**Variables:**
- camelCase for local variables and parameters
- ALL_CAPS with underscores for constants (`EnableCLBPortMappingsKey`, `ProtocolTCP`)
- Receiver variables use single letter or short abbreviation: `r *CLBPortPoolReconciler`, `ctx context.Context`
- Client variable: `k8sClient`
- Logger variable: `logger` or `clbLog` (package-level logger)

**Types:**
- PascalCase for struct names: `CLBPortPoolReconciler`, `CLBPodBinding`, `PortPool`
- Interface names end in suffix or describe behavior: `ObjectWrapper`, `CustomDefaulter`, `CustomValidator`
- Receiver type alias pattern for generics: `CLBBindingReconciler[*clbbinding.CLBPodBinding]`

**Kubernetes Resource Naming:**
- CRD names use PascalCase: `CLBPortPool`, `CLBPodBinding`, `CLBNodeBinding`
- Annotation keys use full domain prefix: `networking.cloud.tencent.com/enable-clb-port-mapping`
- Status/condition suffixes: `/status`, `/finalizers`

## Code Style

**Formatting:**
- Tool: `gofmt` (enforced via make fmt)
- Import organization: goimports (configured in golangci-lint)
- Line length limit: default (lll linter enabled with exceptions)
- Indentation: tabs (Go standard)

**Linting:**
- Tool: `golangci-lint` with `.golangci.yml` configuration
- Enabled linters: `dupl`, `errcheck`, `exportloopref`, `ginkgolinter`, `goconst`, `gocyclo`, `gofmt`, `goimports`, `gosimple`, `govet`, `ineffassign`, `lll`, `misspell`, `nakedret`, `prealloc`, `revive`, `staticcheck`, `typecheck`, `unconvert`, `unparam`, `unused`
- Excluded rules for api/ paths: line length (lll), duplication (dupl)
- Excluded rules for internal/ paths: duplication (dupl), line length (lll)
- Revive rule enabled: `comment-spacings`

**Code Comments:**
- Start comments with function name: `// Reconcile is part of the main kubernetes reconciliation loop...`
- kubebuilder directives as special comments: `// +kubebuilder:rbac:groups=...`
- Marker comments for code generation: `// +kubebuilder:scaffold:imports`
- nolint directives for suppressing lint: `// nolint:unused` (as seen in webhook_test.go)
- Chinese comments used throughout codebase for implementation details: `// 清理端口池`, `// 拿到所有需要查询的 LbId`

## Import Organization

**Order (enforced by goimports):**
1. Standard library: `context`, `fmt`, `time`, etc.
2. External packages: `github.com/pkg/errors`, `github.com/go-logr/logr`, etc.
3. Internal packages: `github.com/tkestack/tke-extend-network-controller/...`

**Path Aliases:**
- Kubernetes core: `corev1 "k8s.io/api/core/v1"`
- Kubernetes apimachinery: `metav1 "k8s.io/apimachinery/apis/meta/v1"`
- Controller-runtime: `ctrl "sigs.k8s.io/controller-runtime"`
- Logging: `logf "sigs.k8s.io/controller-runtime/pkg/log"` (when using as package logger)
- API errors: `apierrors "k8s.io/apimachinery/pkg/api/errors"` (aliased to avoid confusion with pkg/errors)
- Project APIs: `networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"`

**Ginkgo/Gomega Test Imports:**
- Dot imports for BDD syntax: `. "github.com/onsi/ginkgo/v2"`, `. "github.com/onsi/gomega"`

## Error Handling

**Strategy:** Wrap errors with stack traces using `github.com/pkg/errors`

**Patterns:**
- Always use `errors.WithStack(err)` when returning errors from internal functions
- Use `errors.New()` for sentinel/static errors: `ErrListenerNotFound = errors.New("listener not found")`
- Use `fmt.Errorf()` for formatted errors in validation/webhook contexts
- Use `apierrors.IsNotFound()`, `apierrors.IsConflict()` for Kubernetes API error checks
- Conflict handling: on `IsConflict` error, automatic requeue with 20ms delay (see `internal/controller/util.go`)
- Error wrapping in reconciliation: errors are wrapped and propagated up to controller-runtime

**Example pattern from `internal/controller/util.go`:**
```go
if err := apiClient.Get(ctx, req.NamespacedName, obj); err != nil {
    err = client.IgnoreNotFound(err)
    if err != nil {
        return ctrl.Result{}, errors.WithStack(err)
    } else {
        return ctrl.Result{}, nil
    }
}
```

**In cleanup/sync methods:**
```go
if err := r.ensureState(ctx, pool, networkingv1alpha1.CLBPortPoolStateDeleting); err != nil {
    return result, errors.WithStack(err)
}
```

## Logging

**Framework:** `github.com/go-logr/logr` via controller-runtime

**Patterns:**
- Package-level logger: `var clbLog = ctrl.Log.WithName("clb")` (in `pkg/clb/clb.go`)
- Context logger: `logger := log.FromContext(ctx)` (in reconciliation functions)
- Structured logging with key-value pairs: `logger.Info("message", "key", value)`
- Conditional logging based on verbosity: `logger.V(1)` for verbose (read-only) operations

**Example from `pkg/clb/clb.go`:**
```go
logger.Info("CLB API Call", "api", apiName, "request", req, "response", resp, "cost", cost.String(), "error", err)
```

**Verbosity levels:**
- `logger.Info()`: normal operations, reconciliation events
- `logger.Error()`: errors and failures
- `logger.V(1)`: verbose logging for read-only API operations
- Conditional logging: check `logger.GetV() > 2` for detailed logging

## Comments

**When to Comment:**
- Functions with public/exported names: document purpose at function start
- Complex logic blocks: explain why, not what
- Kubernetes annotations and constants: document purpose and usage
- kubebuilder directives: used extensively for RBAC, webhooks, validation rules
- TODO comments: minimal, mostly in generated/template code

**JSDoc/GoDoc:**
- Standard Go doc comments: `// FunctionName does...`
- No special JSDoc format; Go uses simple comment style
- Package-level documentation: `// Package controller...`
- Method documentation: `// MethodName does X and returns Y`

**Example from controller:**
```go
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *CLBPortPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ...
}
```

## Function Design

**Size:** No strict limit, but typically 30-100 lines in reconciliation functions

**Parameters:**
- Always include context as first parameter: `func(..., ctx context.Context, ...)`
- Use receiver methods for reconciler functions: `func (r *CLBPortPoolReconciler) Reconcile(...)`
- Generic type parameters for reusable logic: `func Reconcile[T client.Object](...)`

**Return Values:**
- Reconcilers return `(ctrl.Result, error)` (standard controller-runtime)
- Helper functions return `(result ctrl.Result, err error)` with named returns
- Errors wrapped with `errors.WithStack()`
- Void functions typically return `error` if operations can fail

**Error Propagation:**
- Return early on error: `if err != nil { return result, errors.WithStack(err) }`
- Use `errors.WithStack()` consistently to capture stack traces

## Module Design

**Exports:**
- Exported types: `type CLBPortPoolReconciler struct { ... }`
- Exported methods for controller registration: `SetupWithManager(mgr ctrl.Manager, workers int) error`
- Public reconciliation methods: `Reconcile()`, `Default()`, `ValidateCreate()`
- Internal functions use lowercase: `cleanup()`, `sync()`, `ensureState()`

**Barrel Files (Package Organization):**
- No barrel files (index.go patterns) in use
- Each controller in separate file: `clbportpool_controller.go`, `clbpodbinding_controller.go`
- Utility functions in `util.go`: generic `Reconcile[T]` and `ReconcileWithFinalizer[T]` helpers
- Constants in separate package: `internal/constant/constant.go`
- CLB operations in separate package: `pkg/clb/` with specialized modules (batch-listener.go, batch-target.go, listener.go)

**Package Structure:**
- `api/v1alpha1/`: CRD types and webhooks
- `cmd/app/`: application setup and configuration
- `internal/controller/`: reconciler implementations
- `internal/webhook/`: webhook validators and defaulters
- `internal/portpool/`: port allocation logic
- `pkg/clb/`: CLB SDK wrapper and operations
- `pkg/cloudapi/`: credential management
- `pkg/clusterinfo/`: cluster metadata
- `pkg/util/`: utility functions
- `internal/constant/`: constant definitions

---

*Convention analysis: 2025-01-10*
