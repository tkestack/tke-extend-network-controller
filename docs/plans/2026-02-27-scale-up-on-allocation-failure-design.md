# CLB 扩容优化：分配失败触发扩容

## 问题

当用户设置了 `endPort` 且每个 Pod 需要大量端口时（如 100 个），第 1 个 Pod 分配成功后占满了 LB 上的端口范围，第 2 个 Pod 分配失败。原有的扩容判断基于 `CanAllocate` dry run，只检查能否分配 1 个 TCPUDP 端口，无法预知下一次分配需要多少端口，导致无法及时触发扩容。

## 方案

将扩容触发点从"CLBPortPool 主动预判"改为"分配失败时被动触发"。

### 核心机制

1. `PortPool` 新增 `scaleUpRequested atomic.Bool` 字段
2. Binding 分配失败时，调用 `RequestScaleUp`，内部用 `CAS(false→true)` 设标记，只有首次成功的才通过 eventsource 通知 CLBPortPool reconcile
3. CLBPortPool reconcile 时，用 `HasScaleUpRequest` 检查标记，满足 AutoCreate 条件则创建 CLB
4. 移除原有 `CanAllocate` dry run 逻辑

### 防重复扩容

大量 Pod 同时分配失败时（如 10000 个），`atomic.Bool` 的 CAS 保证只有第一个设标记成功并通知，其余跳过。每轮 CLBPortPool reconcile 只创建 1 个 CLB，逐步扩容直到满足需求或达到 `MaxLoadBalancers` 限制。

### 防竞态重复创建

CLB 创建完成后，存在一个竞态窗口：Binding 尚未成功分配（新 CLB 还没被 Binding 使用），又触发了 `RequestScaleUp`，导致多创建 CLB。通过以下机制解决：

1. **CLB 创建后立即加入分配器缓存**：调用 `EnsureLbIds` 将新 CLB 加入缓存，确保 Binding 能立刻使用
2. **一次性扩容完成标记**：`PortPool` 新增 `scaleUpJustCompleted atomic.Bool`，CLB 创建后调用 `MarkScaleUpCompleted()` 设为 true。下一次 `RequestScaleUp` 或 `HasScaleUpRequest` 时，用 CAS 消耗此标记并返回 false，只吸收一轮竞态请求。标记消耗后，后续真正的扩容请求正常通过。
3. **先创建后重置**：`ResetScaleUpRequest` 在 `createCLB` 之后调用，避免创建期间 Binding 重复设标记导致多次扩容

### 时序

```
Pod-N 分配失败 → CAS(false→true) 成功 → notifyPortPoolReconcile
Pod-N+1..M 分配失败 → CAS 失败 → 跳过

CLBPortPool reconcile:
  → HasScaleUpRequest 检查标记（若 scaleUpJustCompleted 为 true，吸收此轮，返回 false）
  → 检查 AutoCreate 启用 + MaxLoadBalancers 限制
  → 创建 1 个 CLB
  → EnsureLbIds 将新 CLB 加入分配器缓存
  → MarkScaleUpCompleted 设置一次性标记
  → ResetScaleUpRequest 重置标记
  → 状态更新 → 触发 NoPortAvailable 的 Binding 重新 reconcile
  → 部分成功，部分仍失败 → 失败的再次设标记（首次被吸收，之后正常触发）→ 下一轮扩容
```

## 改动文件

| 文件                                            | 改动                                                                                                                          |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `internal/portpool/portpool.go`                 | 新增 `scaleUpRequested`/`scaleUpJustCompleted` atomic.Bool，新增 `RequestScaleUp`/`HasScaleUpRequest`/`ResetScaleUpRequest`/`MarkScaleUpCompleted` 方法，移除 `CanAllocate`/`canAllocateFromLb` |
| `internal/portpool/allocator.go`                | 新增 `RequestScaleUp`/`HasScaleUpRequest`/`ResetScaleUpRequest`/`MarkScaleUpCompleted` 转发方法，移除 `CanAllocate`/`SetScaleUpCooldown` |
| `internal/controller/clbbinding.go`             | 分配失败时调用 `RequestScaleUp`，成功才 notify                                                                                |
| `internal/controller/clbportpool_controller.go` | `ensureLbStatus` 中用 `HasScaleUpRequest` 检查扩容请求，创建 CLB 后调用 `EnsureLbIds` + `MarkScaleUpCompleted` + `ResetScaleUpRequest` |
| `internal/portpool/portpool_test.go`            | 移除 `CanAllocate` 测试，新增 `RequestScaleUp` 竞态测试和一次性标记测试                                                       |
