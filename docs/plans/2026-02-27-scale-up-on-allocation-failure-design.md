# CLB 扩容优化：分配失败触发扩容

## 问题

当用户设置了 `endPort` 且每个 Pod 需要大量端口时（如 100 个），第 1 个 Pod 分配成功后占满了 LB 上的端口范围，第 2 个 Pod 分配失败。原有的扩容判断基于 `CanAllocate` dry run，只检查能否分配 1 个 TCPUDP 端口，无法预知下一次分配需要多少端口，导致无法及时触发扩容。

## 方案

将扩容触发点从"CLBPortPool 主动预判"改为"分配失败时被动触发"。

### 核心机制

1. `PortPool` 新增 `scaleUpRequested atomic.Bool` 字段
2. Binding 分配失败时，调用 `RequestScaleUp`，内部用 `CAS(false→true)` 设标记，只有首次成功的才通过 eventsource 通知 CLBPortPool reconcile
3. CLBPortPool reconcile 时，用 `CAS(true→false)` 取出标记，满足 AutoCreate 条件则创建 CLB
4. 移除原有 `CanAllocate` dry run 逻辑

### 防重复扩容

大量 Pod 同时分配失败时（如 10000 个），`atomic.Bool` 的 CAS 保证只有第一个设标记成功并通知，其余跳过。每轮 CLBPortPool reconcile 只创建 1 个 CLB，逐步扩容直到满足需求或达到 `MaxLoadBalancers` 限制。

### 时序

```
Pod-N 分配失败 → CAS(false→true) 成功 → notifyPortPoolReconcile
Pod-N+1..M 分配失败 → CAS 失败 → 跳过

CLBPortPool reconcile:
  → CAS(true→false) 取出标记
  → 检查 AutoCreate 启用 + MaxLoadBalancers 限制
  → 创建 1 个 CLB
  → 状态更新 → 触发 NoPortAvailable 的 Binding 重新 reconcile
  → 部分成功，部分仍失败 → 失败的再次设标记 → 下一轮扩容
```

## 改动文件

| 文件                                            | 改动                                                                                                                          |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `internal/portpool/portpool.go`                 | 新增 `scaleUpRequested atomic.Bool`，新增 `RequestScaleUp`/`ResetScaleUpRequest` 方法，移除 `CanAllocate`/`canAllocateFromLb` |
| `internal/portpool/allocator.go`                | 新增 `RequestScaleUp` 转发方法，移除 `CanAllocate`                                                                            |
| `internal/controller/clbbinding.go`             | 分配失败时调用 `RequestScaleUp`，成功才 notify                                                                                |
| `internal/controller/clbportpool_controller.go` | `ensureLbStatus` 中用 `ResetScaleUpRequest` 替代 `CanAllocate` 判断                                                           |
| `internal/portpool/portpool_test.go`            | 移除 `CanAllocate` 测试，新增 `RequestScaleUp` 竞态测试                                                                       |
