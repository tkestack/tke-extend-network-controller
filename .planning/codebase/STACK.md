# Technology Stack

**Analysis Date:** 2025-01-17

## Languages

**Primary:**
- Go 1.24.0 (toolchain 1.24.4) - All controller logic, APIs, and command-line tools

## Runtime

**Environment:**
- Linux x86_64 (primary), ARM64, s390x, ppc64le (multi-arch support)

**Package Manager:**
- Go modules (`go.mod`, `go.sum`)
- Lockfile: Present (`go.sum`)

## Frameworks

**Core:**
- `sigs.k8s.io/controller-runtime` v0.21.0 - Kubernetes controller framework
- `k8s.io/api` v0.33.0 - Kubernetes API types
- `k8s.io/apimachinery` v0.33.0 - Kubernetes common types and utilities
- `k8s.io/client-go` v0.33.0 - Kubernetes client library
- `sigs.k8s.io/controller-tools` v0.15.0 (code-gen) - CRD and RBAC generation

**CLI & Configuration:**
- `github.com/spf13/cobra` v1.8.1 - Command-line interface (`cmd/app/cmd.go`)
- `github.com/spf13/viper` v1.19.0 - Configuration management (env var binding)
- `github.com/spf13/pflag` v1.0.5 - Command-line flag parsing

**Logging:**
- `github.com/go-logr/logr` v1.4.2 - Structured logging interface
- `sigs.k8s.io/controller-runtime/pkg/log/zap` - Zap logger implementation

**Testing:**
- `github.com/onsi/ginkgo/v2` v2.22.0 - BDD-style testing framework (`internal/controller/*_test.go`)
- `github.com/onsi/gomega` v1.36.1 - Matcher library
- `sigs.k8s.io/controller-runtime/pkg/envtest` - Kubernetes API server emulation for tests

**Build/Dev:**
- `sigs.k8s.io/kustomize/kustomize/v5` v5.4.1 - Kubernetes manifest overlays
- `sigs.k8s.io/controller-tools/cmd/controller-gen` v0.15.0 - Code generation
- `github.com/golangci/golangci-lint` v1.57.2 - Linting

## Key Dependencies

**Critical - Game Server Integration:**
- `github.com/openkruise/kruise-game` v0.10.0 - OpenKruise GameServerSet CRD integration (optional)
- `agones.dev/agones` v1.49.0 - Agones game server framework (optional)

**Critical - Tencent Cloud APIs:**
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb` v1.0.1120 - CLB (Cloud Load Balancer) API
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common` v1.1.3 - Common SDK utilities
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc` v1.0.1161 - VPC API (optional)
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam` v1.1.3 - CAM (Identity & Access Management) API (optional)
- `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag` v1.1.0 - Tag management API

**Infrastructure:**
- `golang.org/x/time` v0.9.0 - Rate limiting primitives (`pkg/clb/rate-limit.go`)
- `golang.org/x/oauth2` v0.27.0 - OAuth2 support (transitive via client-go)
- `go.uber.org/multierr` v1.11.0 - Error handling utilities
- `go.uber.org/zap` v1.27.0 - High-performance logging (via controller-runtime)
- `github.com/pkg/errors` v0.9.1 - Error wrapping with stack traces

**Observability:**
- `go.opentelemetry.io/otel` v1.34.0 - OpenTelemetry tracing (transitive)
- `go.opentelemetry.io/otel/sdk` v1.34.0 - OpenTelemetry SDK (transitive)
- `github.com/prometheus/client_golang` v1.22.0 - Prometheus metrics (transitive via controller-runtime)

## Configuration

**Environment:**
- Configuration via command-line flags or environment variables (via Viper)
- Key flags: `--secret-id`, `--secret-key`, `--region`, `--vpcid`, `--cluster-id`, `--leader-elect`, `--metrics-bind-address`, `--health-probe-bind-address`
- Environment variable conversion: hyphens (`-`) converted to underscores (`_`) and uppercased
  - Example: `--secret-id` → `SECRET_ID`

**Build:**
- `Makefile` - Primary build automation
- `Dockerfile` - Multi-stage build using `golang:1.24` builder and `gcr.io/distroless/static:nonroot` runtime
- `docker-buildx` support for multi-platform builds

**Code Generation:**
- `.kubebuilder` project configuration in `PROJECT` file
- CRD manifests generated to `config/crd/bases/`
- RBAC roles generated to `config/rbac/`
- Webhook configurations generated to `config/webhook/`

## Platform Requirements

**Development:**
- Go 1.24+
- `kubectl` for Kubernetes interaction
- `docker` or `podman` for container builds
- `kubebuilder` compatible environment (v4 layout)

**Testing:**
- Kubebuilder envtest (v1.30.0) - Downloaded automatically via `make test`

**Production:**
- Kubernetes 1.30+ cluster (uses v0.33.0 API)
- TKE cluster with CLB support
- Tencent Cloud credentials (Secret ID, Secret Key)
- Deployed to `kube-system` namespace as Deployment with 1-2 replicas

**Container:**
- Base image: `gcr.io/distroless/static:nonroot`
- Non-root user (UID: 65532)
- No shell, no package manager

## API Versioning

**CRD Version:** v1alpha1
- Domain: `cloud.tencent.com`
- Group: `networking`
- Resources: `CLBPortPool`, `CLBPodBinding`, `CLBNodeBinding`

## Metrics & Monitoring

**Prometheus Integration:**
- Metrics endpoint: `:8080/metrics` (default) or configured via `--metrics-bind-address`
- ServiceMonitor: `config/prometheus/monitor.yaml` (requires Prometheus Operator)
- Health probes: `/healthz` and `/readyz` on port 8081 (configurable)

## Summary

This is a **kubebuilder-based Kubernetes operator** written in Go, tightly integrated with **Tencent Cloud CLB APIs** for dynamic port mapping in TKE clusters. It uses industry-standard components (controller-runtime, kustomize, Ginkgo/Gomega) with multi-platform container support and comprehensive monitoring capabilities.

---

*Stack analysis: 2025-01-17*
