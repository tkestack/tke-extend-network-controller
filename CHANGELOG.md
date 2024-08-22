# 版本说明

## v0.1.11 (2024-08-22)

* 完善和优化文档。
* 一些重构和小优化。

## v0.1.10 (2024-08-21)

* 若干优化，提升稳定性

## v0.1.9 (2024-08-19)

修复：
* 修复 clb 不存在时无法删除 DedicatedCLBListener (#1)。
 
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

