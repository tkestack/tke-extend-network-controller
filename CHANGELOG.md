# 版本说明

## v2.0.3 (2025-05-27)

- 大幅提升多线接入场景下的绑定性能。

## v2.0.2 (2025-05-23)

- 支持手动指定实例维度监听器数量配额（大规模场景下提工单调到很大的配额时通常只能实例维度，而非账号维度）。

## v2.0.1 (2025-05-21)

- 支持 TCP_SSL 和 QUIC 协议。
- 优化 clb quota 同步机制。
- 一些其他小优化。

## v2.0.0 (2025-05-16)

- 正式发布端口池 API，用法参考： [使用 CLB 端口池为 Pod 映射公网地址](./docs/clb-port-pool.md)。

## v2.0.0-beta.8 (2025-05-15)

- 修复一些小问题。

## v2.0.0-beta.7 (2025-05-14)

- 一些小优化。
- 输出更多事件日志，方便排查问题。

## v2.0.0-beta.6 (2025-05-13)

- 优化端口分配器。
- 优化自动创建 CLB 逻辑，避免并发对账时 CLB 多创问题。

## v2.0.0-beta.5 (2025-05-12)

- 优化 pod webhook：避免 controller 异常时导致集群范围内 pod 无法正常创建。
- 优化 controller 部署：默认双副本，打散调度。
- 一些 bug fix。

## v2.0.0-beta.4 (2025-05-12)

- 支持 CLB + EIP 映射。
- 一些 bug fix 和优化。
- 优化文档。

## v2.0.0-beta.3 (2025-04-27)

- 完善文档。
- 修复端口段监听器名称。
- 清理监听器时，如果因其它监听器不存在影响本监听器删除失败（批量删除监听器接口），直接重新对账重试，不报错。

## v2.0.0-beta.2 (2025-04-24)

- 进一步优化新 API（端口池）在大规模场景扩容、缩容、发版重建时 CLB 端口映射的性能（所有监听器、后端 rs 的查询、删除、写入的操作请求全部支持自动合并批量操作）。

## v2.0.0-beta.1 (2025-04-23)

- 完善文档。
- 优化大规模快速扩容场景：支持合并请求，通过单个 CLB 云 API 调用实现批量创建监听器和绑定 rs。
- 支持自定义各个 controller 的 worker 并发数量，大规模场景下可根据需求进行调整。

## v2.0.0-beta.0 (2025-04-21)

- 修复若干 bug。
- 当 lb 被删除，自动联动 CLBPortPool 及其关联的 CLBPodBinding 和 CLBNodeBinding 对象，触发 CLB 自动创建和端口重新分配。
- 优化 CLBPortPool 的 status 更新。

## v2.0.0-alpha.5 (2025-04-11)

- 修复端口分配器缓存初始化逻辑，避免 controller 重启后新增 Node 的端口映射分配冲突。

## v2.0.0-alpha.4 (2025-04-11)

- 修复了一些 bug。
- 完善文档。

## v2.0.0-alpha.3 (2025-04-10)

- 支持 HostPort 端口映射，Pod 注解声明 `networking.cloud.tencent.com/enable-clb-hostport-mapping: "true"` 后自动为 container 中声明的 hostPort 从节点已被分配的端口段中算出所有被映射的端口并将映射回写到 pod 注解 `networking.cloud.tencent.com/clb-hostport-mapping-result` 中。

## v2.0.0-alpha.2 (2025-03-27)

- 支持通过定义 CLBNodeBinding 实现从 CLB 端口池中为 Node 端口映射，并将映射结果写入 Node 注解。
- 支持通过定义 Node 注解自动生成 CLBNodeBinding，可结合节点池的 Annotation 配置，为存量和增量节点自动配置注解。

## v2.0.0-alpha.1 (2025-03-25)

- 支持 CLBPortPool 定义 CLB 端口池，用于分配端口映射，支持端口段用单个监听器映射多个端口。
- 支持通过定义 CLBPodBinding 实现从 CLB 端口池中为 Pod 端口映射，并将映射结果写入 Pod 注解。
- 支持通过定义 Pod 注解自动生成 CLBPodBinding，可结合任意工作负载类型使用，实现自动映射。
- 抢先体验参考 [使用 CLB 端口池为 Pod 映射公网地址](./docs/clb-port-pool.md)。

## v1.1.2 (2024-11-4)

* 修复：解决DedicatedCLBListener的TargetPod置为nil后导致pod finalizer泄露问题。

## v1.1.1 (2024-10-28)

* DedicatedCLBListener 校验端口重复时不考虑正在删除的 DedicatedCLBListener。
* 完善文档。

## v1.1.0 (2024-09-09)

* 限制节点类型：超级节点和原生节点
* 内部缓存监听器上限的配额，定期更新，创建新监听器的判断条件也加上监听器配额的判断。
* `DedicatedCLBService` 支持通过 `maxPod` 限制绑定 Pod 数量，minPort/maxPort 支持默认值，使用大范围的端口区间。

## v1.0.1 (2024-09-04)

* 支持将地域作为可选配置，默认通过 TKE 的 metadata 内部接口获取当前地域信息。

## v1.0.0 (2024-08-28)

* 不兼容变更：`DedicatedCLBService` 的 `status` 有不兼容变更。
* 优化：做了大量重构和优化，代码更简洁，性能更优。
* 修复：解决更新资源冲突导致的一些问题。

## v0.1.13 (2024-08-27)

* 新功能：DedicatedCLBService 支持自动创建CLB。
* 修复：解决 DedicatedCLBService 的状态写入失败后状态不一致问题。

## v0.1.12 (2024-08-26)

* clb 接口改为内网调用(以支持 controller 调度到没有公网的节点或超级节点)。
* 展示 DedicatedCLBListener 时增加 Pod 列（增加可读性）。

## v0.1.11 (2024-08-22)

* 完善和优化文档。
* 一些重构和小优化。

## v0.1.10 (2024-08-21)

* 若干优化，提升稳定性

## v0.1.9 (2024-08-19)

修复：
* 修复 clb 不存在时无法删除 DedicatedCLBListener ([#1](https://github.com/imroc/tke-extend-network-controller/issues/1))。
 
优化：
* 一些文档优化。

## v0.1.8 (2024-08-12)

优化：
* 增加CLB并发锁(CLB的写接口有实例锁，并发写会导致报错重新入队重试，这里利用golang的锁与CLB接口的锁对齐锁逻辑，避免不必要的报错与重试)。
* 优化 `DedicatedCLBListener` 的事件消息打印。

不兼容变更: 
* `DedicatedCLBListener` 中的 `backendPod` 字段改为 `targetPod`。

修复：
* chart修复：将crd使用template渲染，避免升级时无法更新 crd。
  > 参考 [helm官方说明](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#some-caveats-and-explanations)。
  >
  > There is no support at this time for upgrading or deleting CRDs using
  > Helm. This was an explicit decision after much community discussion due
  > to the danger for unintentional data loss. Furthermore, there is
  > currently no community consensus around how to handle CRDs and their
  > lifecycle. As this evolves, Helm will add support for those use cases.

