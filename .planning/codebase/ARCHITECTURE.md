# Architecture

**Analysis Date:** 2025-03-20

## Pattern Overview

**Overall:** Kubernetes Operator using controller-runtime with multi-layer reconciliation pattern

**Key Characteristics:**
- Multiple specialized reconcilers managing different Kubernetes resources and CRDs
- Unified binding abstraction for CLBPodBinding and CLBNodeBinding (generic CLBBindingReconciler)
- Global port allocator singleton managing distributed port assignment across multiple CLB instances
- Taints and Finalizer-based graceful deletion for all custom resources
- Event-driven reconciliation with field indexing for efficient lookups

## Layers

**Entry Point / Command Layer:**
- Purpose: Bootstrap the manager, parse flags, initialize cloud credentials and cluster info
- Location: `cmd/main.go`, `cmd/app/cmd.go`, `cmd/app/manager.go`
- Contains: Cobra CLI setup, Viper configuration binding, environment variable handling
- Depends on: All other layers through manager setup
- Used by: Container entrypoint

**Manager Setup / Initialization Layer:**
- Purpose: Initialize controller-runtime Manager, setup webhooks, configure schemes, register all controllers
- Location: `cmd/app/setup_manager.go`, `cmd/app/setup_controller.go`, `cmd/app/setup_webhook.go`
- Contains: Manager options, controller registration, webhook setup, discovery of optional schemes (e.g., Agones)
- Depends on: controller-runtime, API types, cloud API clients
- Used by: runManager() function

**API / CRD Definition Layer:**
- Purpose: Define Kubernetes custom resource types with validation rules
- Location: `api/v1alpha1/`
- Contains: 
  - `clbportpool_types.go`: CLBPortPool (cluster-scoped) - port pool configuration and status
  - `clbpodbinding_types.go`: CLBPodBinding (namespaced) - Pod CLB port mapping
  - `clbnodebinding_types.go`: CLBNodeBinding (cluster-scoped) - Node/HostPort CLB port mapping
  - `clbbinding_types.go`: Shared spec/status types for unified binding behavior
  - `api.go`: Client initialization
- Key validation: Immutability constraints on most spec fields, state machine validation
- Used by: All reconcilers and webhook validators

**Reconciliation / Control Logic Layer:**
- Purpose: Implement the core reconciliation logic - watching resources and driving desired state
- Location: `internal/controller/`
- Contains: 
  - `clbportpool_controller.go` (CLBPortPoolReconciler): Manages CLB instances, auto-creates CLBs, pre-creates listeners, scales pool capacity
  - `clbpodbinding_controller.go` (CLBPodBindingReconciler): Reconciles Pod CLB bindings
  - `clbnodebinding_controller.go` (CLBNodeBindingReconciler): Reconciles Node/HostPort CLB bindings
  - `pod_controller.go` (PodReconciler): Watches Pods, auto-creates CLBPodBindings based on annotations
  - `node_controller.go` (NodeReconciler): Watches Nodes, manages CLBNodeBindings for HostPort scenarios
  - `gameserverset_controller.go` (GameServerSetReconciler): Optional controller for OpenKruiseGame integration
  - `clbbinding.go` (CLBBindingReconciler[T]): Generic reconciliation logic for all CLB binding types
  - `util.go`: Shared reconciliation helpers (finalizers, state transitions, error handling)
- Depends on: Port allocator, CLB SDK, binding abstractions
- Used by: Kubernetes API watch/cache

**Binding Abstraction Layer:**
- Purpose: Provide unified interface for CLBPodBinding and CLBNodeBinding handling
- Location: `internal/clbbinding/`
- Contains:
  - `clbbinding.go`: CLBBinding interface and Backend interface defining shared contract
  - `clbpodbinding.go`: Pod-specific binding wrapper implementing CLBBinding
  - `clbnodebinding.go`: Node-specific binding wrapper implementing CLBBinding
  - `sort.go`: Comparison logic for binding ordering
- Pattern: Wraps API types and provides polymorphic behavior through interfaces
- Depends on: API types
- Used by: CLBBindingReconciler, Pod/Node controllers

**Port Allocation Layer:**
- Purpose: Manage global port assignment state and policies across all port pools
- Location: `internal/portpool/`
- Contains:
  - `allocator.go` (PortAllocator): Global singleton managing multiple port pools
  - `portpool.go` (PortPool): Single pool state - tracks allocated ports, implements allocation strategies (Uniform, InOrder, Random), manages per-CLB port state
  - `portpools.go`: Container type for multiple pools
  - `error.go`: Pool-specific error types
- Key abstractions:
  - ProtocolPort: (Port, EndPort, Protocol) tuple identifying unique port
  - LBPort: Port + CLB instance ID
  - PortAllocation: Tuple of (ProtocolPort, PoolName, LBKey) representing a lease
- Strategies: Uniform (spread across CLBs), InOrder (fill sequentially), Random (random selection)
- Depends on: API types
- Used by: CLBBindingReconciler when allocating ports

**CLB SDK Wrapper Layer:**
- Purpose: Encapsulate Tencent Cloud CLB API interactions with caching, batching, retry, and rate limiting
- Location: `pkg/clb/`
- Contains:
  - `clb.go`: Client factory with region-based caching
  - `api.go`: Thin wrapper for direct API calls
  - `listener.go`: Listener creation/deletion operations
  - `target.go`: Backend target registration/deregistration
  - `batch-listener.go`: Batch listener creation optimizations
  - `batch-target.go`: Batch target operations
  - `instance.go`: CLB instance management (create, delete, query)
  - `cache.go`: Listener information caching to reduce API calls
  - `quota.go`: Listener quota tracking and management
  - `param.go`: Parameter validation for CLB operations
  - `wait.go`: Polling operations for async CLB tasks
  - `rate-limit.go`: Request throttling (20 qps default)
  - `lock.go`: Distributed locking for concurrent operations
  - `error.go`: Cloud API error classification
- Depends on: tencentcloud-sdk-go, cloud API credentials
- Used by: Reconcilers when managing CLB resources

**Cloud Infrastructure Layer:**
- Purpose: Manage cloud provider credentials, cluster metadata, VPC information
- Location: `pkg/cloudapi/`, `pkg/clusterinfo/`, `pkg/vpc/`, `pkg/userinfo/`
- Contains:
  - `pkg/cloudapi/cloudapi.go`: Credential initialization and management
  - `pkg/clusterinfo/clusterinfo.go`: Cluster ID, VPC ID, Region constants
  - `pkg/vpc/client.go`: VPC SDK client wrapper
  - `pkg/userinfo/userinfo.go`: User account information
- Used by: CLB SDK, controller initialization

**Utility / Helper Layer:**
- Purpose: Provide common functions for pod operations, environment handling, retry logic, patching
- Location: `pkg/util/`, `pkg/kube/`
- Contains:
  - `pkg/util/`: Pointer helpers, slice operations, map utilities, retry logic, environment parsing, region detection
  - `pkg/kube/`: Pod finalizer operations, Pod IP lookup, patching, stripping managed fields, secret operations
- Used by: Controllers, CLB operations

**Webhook Validation Layer:**
- Purpose: Validate CRD mutations and enforce immutability constraints
- Location: `internal/webhook/v1alpha1/`
- Contains: `clbportpool_webhook.go` - validates CLBPortPool immutable fields
- Used by: Kubernetes API server during resource mutation

**Event Source / Trigger Layer:**
- Purpose: Integrate external event triggers into controller reconciliation
- Location: `pkg/eventsource/eventsource.go`
- Contains: Generic event source for triggering controller-runtime reconciliations
- Used by: Controllers to signal upstream events needing reconciliation

## Data Flow

**Pod CLB Binding Flow:**

1. User creates/updates Pod with annotation (e.g., `networking.cloud.tencent.com/enable-clb: "true"`)
2. PodReconciler watches Pod changes, detects annotation
3. PodReconciler creates CLBPodBinding CRD (namespaced) specifying port pool and pod reference
4. CLBPodBindingReconciler watches CLBPodBinding, delegates to generic CLBBindingReconciler[*CLBPodBinding]
5. CLBBindingReconciler calls portpool.Allocator.AllocatePort() for each protocol/port requirement
6. PortAllocator looks up PortPool by name, calls pool.Allocate() with policy (Uniform/InOrder/Random)
7. PortPool selects CLB instance(s) and returns PortAllocation (CLB ID + port + protocol)
8. CLBBindingReconciler calls clb.CreateListener() to create listener on selected CLB
9. CLBBindingReconciler calls clb.RegisterTarget() to bind Pod IP to listener
10. CLBBindingReconciler updates CLBPodBinding.Status with port mappings and state=Ready

**Node HostPort Binding Flow:**

1. User creates Node with spec specifying HostPort requirements or applies CLBNodeBinding CRD
2. NodeReconciler watches Node changes, creates CLBNodeBinding (cluster-scoped) if needed
3. CLBNodeBindingReconciler delegates to generic CLBBindingReconciler[*CLBNodeBinding]
4. Same as Pod flow but binds Node internal IP instead of Pod IP

**Port Pool Lifecycle:**

1. User creates CLBPortPool CRD specifying:
   - Port range (startPort, endPort)
   - Existing CLBs (exsistedLoadBalancerIDs) or auto-create config
   - Allocation policy (LbPolicy)
2. CLBPortPoolReconciler watches creation
3. CLBPortPoolReconciler calls clb.DescribeLoadBalancers() for each existing CLB
4. CLBPortPoolReconciler creates PortPool object and adds to portpool.Allocator
5. If autoCreate enabled and ports insufficient:
   - CLBPortPoolReconciler calls clb.CreateLoadBalancer()
   - Updates CLBPortPool.Status.LoadbalancerStatuses with new CLB ID
   - Triggers listener pre-creation (if configured)
6. portpool.Allocator.ResetScaleUpRequest() clears scale-up flag after CLB added
7. On deletion:
   - CLBPortPoolReconciler calls clb.Delete() for auto-created CLBs
   - Removes pool from allocator cache
   - Finalizer removed allowing resource deletion

**State Management:**

- CLBPortPool.Status.State: Creating → Ready → Deleting
- CLBBinding.Status.State: Pending → Ready (or error states: Failed, NoPortAvailable, PortPoolNotFound, PortPoolNotAllocatable, Disabled)
- PortPool tracks: available ports per CLB, allocated ports by binding ref, scale-up request flag
- All state transitions validated in reconcilers before update

## Key Abstractions

**CLBBinding Interface (polymorphism):**
- Purpose: Allows CLBPodBinding and CLBNodeBinding to be handled uniformly
- File: `internal/clbbinding/clbbinding.go`
- Methods: GetSpec(), GetStatus(), GetAssociatedObject(), GetAssociatedObjectByIP(), GetObject(), GetType(), FetchObject()
- Implementations: `CLBPodBinding`, `CLBNodeBinding`
- Pattern: Wraps API type, provides contract for generic CLBBindingReconciler

**PortAllocator Singleton:**
- Purpose: Global coordinator for all port assignment decisions
- File: `internal/portpool/allocator.go`
- Instance: `portpool.Allocator` (initialized once at startup)
- Key operations: GetPool(), RequestScaleUp(), AllocatePort(), ReleasePort()
- Thread-safety: RWMutex protected

**PortPool State Machine:**
- Tracks per-CLB port availability
- Supports fragmented port ranges (TCP/UDP, TCP_SSL/QUIC protocols)
- Implements three LB selection strategies: Uniform (round-robin among CLBs), InOrder (fill first, then next), Random (random selection)
- Ref tracking: maintains map of allocated ports → binding references for cleanup

**Reconcile Helpers (util.go):**
- `Reconcile[T]()`: Base reconciliation loop - fetch object, call sync, return requeue on conflict
- `ReconcileWithFinalizer[T]()`: Adds finalizer logic for graceful deletion - blocks deletion, calls cleanup before removal
- Error handling: Custom error types for pool-specific errors (ErrPoolNotFound, ErrNoPortAvailable, ErrLBNotFoundInPool)

**Event Recording:**
- Each reconciler has EventRecorder injected from manager
- Records state transitions, allocation failures, pool exhaustion as Kubernetes Events
- Events appear in `kubectl describe` output for troubleshooting

## Entry Points

**Binary Entrypoint:**
- Location: `cmd/main.go`
- Triggers: Container startup
- Responsibilities: Call app.RootCommand.Execute()

**Manager Reconciliation Loop:**
- Location: `cmd/app/manager.go` runManager()
- Triggers: Manager.Start() in controller-runtime
- Responsibilities:
  1. Parse flags (region, cluster-id, vpc-id, secret credentials)
  2. Initialize cloud API client, cluster info, user info
  3. Create controller-runtime Manager
  4. Register all controllers and webhooks
  5. Start event loop watching Kubernetes resources

**Pod/Node Resource Watch:**
- Location: `internal/controller/pod_controller.go`, `node_controller.go`
- Triggers: Any Pod/Node creation/update/deletion
- Watches: All Pods/Nodes cluster-wide
- Field indexes: `status.podIP`, `status.nodeIP` for efficient lookups

**CLB Binding Reconciliation:**
- Location: `internal/controller/clbpodbinding_controller.go`, `clbnodebinding_controller.go`
- Triggers: Any CLBPodBinding/CLBNodeBinding creation/update/deletion
- Queue: One queue per controller with configurable worker count (WORKER_CLB_POD_BINDING_CONTROLLER env var)

**Port Pool Reconciliation:**
- Location: `internal/controller/clbportpool_controller.go`
- Triggers: Any CLBPortPool creation/update/deletion
- Responsibilities: Manage CLB instances, handle auto-creation, pre-create listeners, scale capacity

## Error Handling

**Strategy:** Multi-layer error classification with specific recovery behaviors

**Patterns:**

- **Pool not found:** Log and retry with 20μs delay (waiting for pool to sync to allocator)
- **No port available:** Transition state to NoPortAvailable, emit warning event, don't retry
- **Port pool not allocatable:** Transition to PortPoolNotAllocatable state
- **Cloud API rate limit (RequestLimitExceeded):** Requeue after 1 second
- **LB not found in pool status:** Requeue after 20μs (waiting for pool status update)
- **LB ID not found on cloud (InvalidParameter.LBIdNotFound):** Log and stop retrying (LB deleted)
- **Resource conflict (409 Conflict):** Requeue after 20ms (API server conflict)
- **Other errors:** Record event, transition to Failed state, attempt partial cleanup, log for debugging

**Error Types (pkg/clb/error.go):**
- IsLbIdNotFoundError(): Check for InvalidParameter.LBIdNotFound
- IsLoadBalancerNotExistsError(): Check for "LoadBalancer not exist" or "LB not exist"
- IsRequestLimitExceededError(): Check for RequestLimitExceeded
- IsPortCheckFailedError(): Check for InvalidParameter.PortCheckFailed
- IsListenerNotFound(): Check for "some ListenerId...not found"

**Finalizer-based Cleanup:**
- On deletion: Block deletion with finalizer until cleanup complete
- Cleanup: Deregister all binding targets from CLB, release allocated ports
- Remove finalizer: Allow Kubernetes to delete resource after cleanup succeeds
- Pod special handling: Use custom finalizer addition/removal (kube.AddPodFinalizer, RemovePodFinalizer) to handle pod deletion during reconciliation

## Cross-Cutting Concerns

**Logging:**
- Framework: logr with controller-runtime zap integration
- Verbosity: V(0)=Warnings/Errors, V(1)=API calls, V(2)+=Debug details
- Context-aware: log.FromContext(ctx) carries request ID and resource info

**Validation:**
- Spec immutability: Enforced via kubebuilder XValidation rules on CRD types
- Pod IP uniqueness: Field indexed for fast lookup
- Node IP uniqueness: Field indexed for fast lookup
- Port range validation: Done in PortPool.Allocate() before assignment

**Authentication:**
- Cloud provider: Tencent Cloud credentials (secret-id, secret-key) passed at startup
- Kubernetes: In-cluster service account (KUBECONFIG or mounted token)
- Both: Initialized before reconciliation starts (runManager)

**Concurrency:**
- Reconciler queues: One per controller type with per-controller worker count (e.g., WORKER_CLB_PORT_POOL_CONTROLLER=10)
- PortAllocator: RWMutex protects pools map
- PortPool: Mutex protects allocated ports and LB state
- CLB operations: Rate-limited at 20 qps (pkg/clb/rate-limit.go)
- Distributed locking: pkg/clb/lock.go supports optional distributed locks via Kubernetes ConfigMaps (experimental)

---

*Architecture analysis: 2025-03-20*
