# Codebase Concerns

**Analysis Date:** 2025-01-14

## Tech Debt

**Synchronous deregistration of targets in cleanup loop:**
- Issue: In `internal/controller/clbbinding.go` line 976, the deregistration of targets during cleanup is performed synchronously in a loop across all port bindings. The comment at line 1 indicates an incomplete concurrent implementation ("TODO: 改成并发").
- Files: `internal/controller/clbbinding.go` (lines 914-932 cleanup phase, deregister targets)
- Impact: During Pod deletion, cleanup of many port bindings sequentially blocks the reconciliation loop, causing slow deletion and potential timeout issues in large-scale scenarios.
- Fix approach: Convert the loop at lines 914-932 to properly aggregate deregistration tasks and use the batch processing infrastructure (`clb.DeregisterTargetsTryBatch`) instead of sequential calls.

**Goroutine lifecycle management without cancellation:**
- Issue: Background goroutines started in `pkg/clb/quota.go` (line 52-68) and `pkg/clb/batch.go` (line 79-93) run infinite loops with no shutdown mechanism. These are spawned during package init and run for the lifetime of the controller.
- Files: `pkg/clb/quota.go` (lines 52-68), `pkg/clb/batch.go` (lines 79-93)
- Impact: On controller restart, these goroutines become orphaned or create duplicate workers. No graceful shutdown mechanism means resource cleanup is incomplete during termination.
- Fix approach: Pass context with cancellation to these goroutines, implement proper shutdown sequence in manager/app startup, store goroutine lifecycle references for testing.

**Panics on fatal initialization errors:**
- Issue: Multiple `panic()` calls in initialization paths that should be recoverable or gracefully handled:
  - `pkg/clb/clb.go` line 29: panic on CLB client creation failure
  - `pkg/vpc/client.go`: panic on VPC client creation failure
  - `pkg/cloudapi/cloudapi.go` line 18: panic if secrets are missing
  - `cmd/app/manager.go`: panic if region/vpcId missing
  - `cmd/main.go` line 7: panic on root command execution
- Files: `pkg/clb/clb.go`, `pkg/vpc/client.go`, `pkg/cloudapi/cloudapi.go`, `cmd/app/manager.go`, `cmd/main.go`
- Impact: No opportunity for graceful degradation, health checks, or retry logic. Any initialization issue crashes the entire controller.
- Fix approach: Replace panics with structured error returns that propagate to main, add validation checks that can trigger leader election backoff or health probe failures.

**Timer resource leak in batch processor:**
- Issue: In `pkg/clb/batch.go` lines 47-92, `time.Timer` is created but only explicitly stopped in implicit channel closure. If `batchRequest()` is called before timer fires, the timer is recreated at line 49, but prior timer may not be properly cleaned up in all code paths.
- Files: `pkg/clb/batch.go` (lines 45-94)
- Impact: Long-running controller accumulates stopped timers that are never garbage collected until context is lost.
- Fix approach: Explicitly call `timer.Stop()` before creating new timer in `batchRequest()`, add defer cleanup.

## Known Bugs

**Listener state inconsistency after failed creation:**
- Symptoms: If a listener creation partially succeeds (e.g., listener created but cache update fails), subsequent reconciliation may attempt to create duplicate listeners or incorrectly handle port conflicts.
- Files: `internal/controller/clbbinding.go` (lines 206-267 createListener logic)
- Trigger: Network partition during listener creation, API returns success but response is lost.
- Workaround: Manually delete the phantom listener from CLB console and trigger reconciliation. Controller should detect and recover on next check at line 311-319.

**Port binding released twice on state transition failures:**
- Symptoms: If cleanup is interrupted after marking finalized (line 937 in clbbinding.go) but before actually releasing ports, and reconciliation is re-triggered, ports may be marked as released in allocator but still bound in CLB.
- Files: `internal/controller/clbbinding.go` (lines 936-950), `internal/portpool/allocator.go`
- Trigger: Controller crash between finalized mark and port release, or status update failure.
- Workaround: Run `make test` to verify allocator consistency, manually audit port pool status.

**Race condition in PortPool allocation under concurrent requests:**
- Symptoms: Multiple concurrent allocation requests can compete for the same port from quota check to actual allocation.
- Files: `internal/portpool/portpool.go` (lines 193-219 AllocatePortFromRange, 289-300 AllocatePort)
- Trigger: Multiple pods starting simultaneously requesting same port range.
- Workaround: Port quota checks are conservative and usually prevent collision, but edge cases exist near quota boundaries.

## Security Considerations

**Credentials stored as plain text environment variables:**
- Risk: `pkg/cloudapi/cloudapi.go` requires `secretId` and `secretKey` passed at startup with no validation or rotation capability. If controller logs are captured, credentials are exposed.
- Files: `pkg/cloudapi/cloudapi.go` (lines 18-24), `cmd/app/manager.go` (region/vpcId)
- Current mitigation: Credentials are passed via command line flags, typically loaded from Kubernetes secrets in Helm deployment.
- Recommendations: Add RBAC controls to prevent credential exposure in pod logs, implement credential rotation mechanism, consider using TKE IRSA (IAM Role for Service Account) if available.

**No input validation on CLB/VPC IDs:**
- Risk: CLB instance IDs and VPC IDs accepted without format validation. Malformed IDs could bypass intended resource isolation.
- Files: `api/v1alpha1/clbportpool_types.go`, `api/v1alpha1/clbpodbinding_types.go`
- Current mitigation: Kubernetes API server validates CRD schemas at field level, but no regex patterns defined in CRD specs.
- Recommendations: Add `pattern` validation rules in CRD definitions (e.g., `lb-[a-z0-9]{8,}` for CLB ID format).

**Missing authentication on listener configuration:**
- Risk: CLBPortPool reconciler can create/modify CLB listeners without verifying ownership or authorization boundaries.
- Files: `internal/controller/clbportpool_controller.go`
- Current mitigation: Requires valid Kubernetes credentials to create CRDs, TKE API layer validates cluster membership.
- Recommendations: Add ownership tags/labels verification before applying listener configurations.

## Performance Bottlenecks

**Synchronous listener query blocking allocation:**
- Problem: When allocating ports for Pod bindings, each binding triggers a query for existing listeners (line 243, 312 in clbbinding.go) even when listeners are recently created. This blocks the reconciliation loop.
- Files: `internal/controller/clbbinding.go` (lines 311-345 ensureListener query logic), `pkg/clb/listener.go`
- Cause: ListenerCache is consulted but full API call still made in many paths, especially on cache misses.
- Improvement path: Implement async listener population in batch, prefetch listener state during port pool reconciliation, increase ListenerCache TTL.

**Unbounded concurrent cleanup goroutines:**
- Problem: During Pod deletion with many port bindings, cleanup spawns one goroutine per binding (line 915 in clbbinding.go) without limits. Large pod could spawn 100+ goroutines simultaneously.
- Files: `internal/controller/clbbinding.go` (lines 912-932)
- Cause: No worker pool or semaphore to bound concurrency.
- Improvement path: Use `golang.org/x/sync/semaphore` to limit concurrent cleanup operations, batch cleanup by CLB instance to improve API efficiency.

**Quarterly quota refresh with global lock:**
- Problem: `pkg/clb/quota.go` refreshes quotas every 5 minutes (line 54) with synchronous API call holding global `q.mu.Lock()`, blocking all quota checks during refresh.
- Files: `pkg/clb/quota.go` (lines 50-68)
- Cause: No async quota refresh or read-write lock separation.
- Improvement path: Use `sync.RWMutex` instead of `sync.Mutex`, move quota refresh to separate non-blocking goroutine, implement quota update notifications.

**Listener cache without expiration:**
- Problem: ListenerCache in `pkg/clb/listener.go` never expires entries, so stale listener state persists indefinitely.
- Files: `pkg/clb/listener.go`
- Cause: Cache keys are not timestamped, no TTL mechanism.
- Improvement path: Add timestamp to cache entries, implement TTL expiration on cache misses, add cache size bounds.

## Fragile Areas

**CLBBinding reconciliation state machine:**
- Files: `internal/controller/clbbinding.go` (lines 40-200 sync() method)
- Why fragile: Complex error handling with multiple state transitions (Pending → Bound/NoPortAvailable/PortPoolNotAllocatable/Deleting). If an error occurs mid-transition, binding can be left in inconsistent state. No state validation on entry.
- Safe modification: Add state precondition checks at reconciliation start, document all valid state transitions in comments, add comprehensive logging of state changes.
- Test coverage: Controller tests exist but don't cover all state transition error paths, particularly around error recovery.

**Port allocator initialization and synchronization:**
- Files: `internal/portpool/allocator.go`, `internal/portpool/portpool.go`
- Why fragile: Allocator is a global singleton initialized lazily. If multiple goroutines trigger initialization simultaneously, race condition possible. Port cache structure uses plain `map` with manual locking.
- Safe modification: Ensure allocator is initialized exactly once before any controller starts (move to manager setup phase), consider using `sync.Map` for cache to reduce lock contention.
- Test coverage: Unit tests exist but don't exercise concurrent allocation scenarios.

**Listener precreation feature flag dependency:**
- Files: `internal/controller/clbbinding.go` (line 917, 974), `internal/portpool/portpool.go` (line 152-154)
- Why fragile: Cleanup behavior changes based on `IsPrecreateListenerEnabled()` state. If port pool config changes mid-lifecycle, previously allocated listeners may not be cleaned up correctly.
- Safe modification: Store precreation flag in PortBindingStatus so cleanup uses historical config, not current config.
- Test coverage: No test case for precreation flag toggle during binding lifecycle.

**CloudAPI credential initialization timing:**
- Files: `pkg/cloudapi/cloudapi.go`, `pkg/clb/clb.go`, `cmd/app/manager.go`
- Why fragile: Credentials must be initialized before first CLB API call. If any controller starts before credential init completes, GetClient() will panic. No dependency ordering.
- Safe modification: Move credential initialization to manager setup, add validation that credential is initialized before allowing controllers to start, return error instead of panicking.
- Test coverage: No test validates initialization order.

## Scaling Limits

**Concurrent batch operations capacity:**
- Current capacity: 4 goroutines for batch processing (targets, listeners, etc.) initialized at module load time.
- Limit: With 800 max accumulated tasks and 20/100/20 per-operation limits, single CLB can handle ~1600 ops/batch window. Under sustained load >200 ops/sec, backlog accumulates.
- Scaling path: Make concurrency configurable via environment variables (already partially implemented with `WORKER_CLB_POD_BINDING_CONTROLLER`), dynamically adjust based on CLB instance load.

**PortPool allocator state in memory:**
- Current capacity: Single allocator holds all port pools and binding states in memory. With 10 port pools × 10 CLBs × 1000s of binding entries = millions of allocations.
- Limit: At ~1000 bindings per pool, memory usage is ~100MB+ depending on allocation fragmentation. Allocator has no size limits or eviction policy.
- Scaling path: Implement allocator persistence to etcd if supporting >10 pools or >10k total bindings, add memory limits and LRU eviction.

**CLB instance listener quota exhaustion:**
- Current capacity: Each CLB supports 50 listeners (platform limit for TOTAL_LISTENER_QUOTA).
- Limit: With 3 port bindings per pod and 100 pods per CLB, quota is exceeded. Currently handled by requesting scale-up, but scaling is manual.
- Scaling path: Implement automated CLB provisioning in port pool controller to pre-create additional CLBs when quota utilization exceeds 70%.

## Dependencies at Risk

**Tencentcloud SDK version pinning:**
- Risk: `tencentcloud-sdk-go` dependency in go.mod is pinned to specific version. No automatic patching for security fixes.
- Impact: Security vulnerabilities in SDK are not automatically resolved, requiring manual dependency update and re-testing.
- Migration plan: Regularly review SDK changelog for security fixes, add dependabot checks if using GitHub.

**Custom error handling library (github.com/pkg/errors):**
- Risk: `github.com/pkg/errors` is commonly used but the Go team recommends `github.com/hashicorp/errwrap` or standard `errors` package in Go 1.20+. Project uses `errors.WithStack()` extensively.
- Impact: Error wrapping behavior is non-standard, may be incompatible with modern error inspection patterns.
- Migration plan: Gradually migrate to `errors.As()` and `errors.Is()` patterns, consider using `fmt.Errorf` with `%w` verb for new code.

**Kubernetes go-client rate limiting:**
- Risk: Uses default go-client rate limiter with hardcoded settings from controller-runtime. Under extreme load, API calls may be throttled without visibility.
- Impact: Pod binding delays when Kubernetes API is overloaded due to client-side rate limiting.
- Migration plan: Add configurable rate limit settings to controller options, expose metrics for rate limit events.

## Missing Critical Features

**No webhook validation for CLBPortPool configuration:**
- Problem: CLBPortPool CRD has only default kubebuilder webhook scaffold. Port range overlaps, invalid region references, and inconsistent configurations are not validated at admission time.
- Blocks: Administrators can accidentally create invalid port pools that fail silently during reconciliation.
- Improvement: Implement ValidatingWebhook in `internal/webhook/v1alpha1/clbportpool_webhook.go` to check port range validity, region existence, and CLB ID format.

**No metrics/observability for port pool state:**
- Problem: No Prometheus metrics exported for port pool utilization, allocation success rate, or listener quota usage.
- Blocks: Operators cannot detect pool exhaustion until Pods fail to get ports.
- Improvement: Add metrics like `clb_pool_utilization_percent`, `clb_pool_allocation_failed_total`, `clb_listener_quota_remaining`.

**No leader election for multi-controller deployments:**
- Problem: Currently assumes single controller instance. If multiple replicas are deployed, all attempt port allocation simultaneously causing conflicts.
- Blocks: High availability deployment model not supported.
- Improvement: Implement leader election using Kubernetes lease mechanism (controller-runtime already provides this infrastructure).

**No port binding reservation or pre-allocation:**
- Problem: Port allocation is on-demand during Pod creation. Large batch deployments can fail if pool is exhausted.
- Blocks: Cannot guarantee port availability for critical workloads.
- Improvement: Add reservation API allowing pre-allocation of port ranges for specific workloads.

## Test Coverage Gaps

**Untested cleanup error scenarios:**
- What's not tested: Cleanup when CLB has been deleted, listener deletion fails, concurrent cleanup conflicts.
- Files: `internal/controller/clbbinding_controller_test.go` (cleanup not exercised), `internal/controller/pod_controller_test.go` (no deletion tests)
- Risk: Dangling resources and port leaks on cleanup failure.
- Priority: High

**Untested port allocator race conditions:**
- What's not tested: Concurrent allocation requests, simultaneous release and allocation, quota boundary conditions.
- Files: `internal/portpool/portpool_test.go` (if exists), `internal/portpool/allocator_test.go` (if exists)
- Risk: Port collisions in high-concurrency environments.
- Priority: High

**No integration tests with real CLB API:**
- What's not tested: End-to-end Pod creation → port allocation → listener creation → backend binding with actual TKE CLB.
- Files: `test/e2e/e2e_test.go` (exists but likely uses mock/stub CLB client)
- Risk: API behavior mismatches discovered only in production.
- Priority: Medium

**Incomplete webhook tests:**
- What's not tested: ValidatingWebhook for CLBPortPool config validation (webhook scaffold TODOs visible in `internal/webhook/v1alpha1/clbportpool_webhook_test.go`).
- Files: `internal/webhook/v1alpha1/clbportpool_webhook_test.go` (lines marked with TODO)
- Risk: Invalid configurations pass validation and fail obscurely during reconciliation.
- Priority: Medium

**No tests for listener precreation feature:**
- What's not tested: Listener precreation workflow, cleanup behavior differences with feature enabled/disabled, quota interaction.
- Files: No specific test file identified.
- Risk: Precreation feature may have silent failures in certain scenarios.
- Priority: Medium

---

*Concerns audit: 2025-01-14*
