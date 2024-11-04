# 版本说明

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

