# 版本说明

## v2.3.4 (2025-08-07)

- 修复：避免当 clb 底层性能出问题时绑定 rs 失败（没有错误码）误以为绑定成功而将映射结果写入 pod 注解的问题。

## v2.3.3 (2025-08-04)

- 优化chart：vpc 接口调用走内部地址，不依赖公网。

## v2.3.2 (2025-07-06)

- 修复端口分配冲突问题。
- 修复当端口分配冲突可能导致的 worker 卡住问题。

## v2.3.1 (2025-06-25)

- 改进一些日志打印。
- go modules 重命名为当前 github 仓库地址。
- 完善文档。

## v2.3.0 (2025-06-23)

- 端口池增加 `lbPolicy` 字段，即 CLB 分配策略，控制分配端口时的 CLB 挑选策略（均匀分配策略可减小 DDoS 攻击的影响范围；按顺序分配策略可提升 LB 利用率；无特殊要求可用随机分配策略）。
- 端口池增加 `lbBlacklist` 字段，即 CLB 黑名单，控制分配端口时 CLB 的黑名单，避免某些 CLB 被分配（可用于临时屏蔽被 DDos 攻击的 IP 被分配）。

## v2.2.3 (2025-06-20)

- 优化：当端口池的 CLB 配置错误时（如创建的 CLB 的所在 VPC 不正确导致无法绑定成功），用户可直接更正 CLBPortPool 中的 exsistedLoadBalancerIDs 字段，错误的 CLB 会自动从分配器中移除，已分配的且没有绑定成功的会自动释放并重新对账，然后会自动重新分配正确的 CLB 并绑定成功。

## v2.2.2 (2025-06-19)

- chart：默认使用 tke-extend-network-controller 作为应用名称，与 release 名称不挂钩。
- 修复：动态检查 agones 和 okg 是否安装，仅在安装时才启用对 agones 和 okg 的支持，避免因集群找不到对应的 crd 定义而报错。
- 一些其它小优化。

## v2.2.1 (2025-06-13)

- 修复 `v2.2.0` 引入的 node controller 无法获取 node annotation 问题以及端口段+hostPort映射场景的 annotation key 不正确问题。

## v2.2.0 (2025-06-13)

- 废弃旧版 API（DedicatedCLBListener/DedicatedCLBService），请使用新版的端口池替代，参考 [使用 CLB 端口池为 Pod 映射公网地址](./docs/clb-port-pool.md)。
- 支持将映射信息写入 Agones 的 GameServer，以便通过 GameServerAllocation API 分配 GameServer 时也能直接获取到映射信息。参考视频教程 [在 TKE 使用 Agones 部署游戏服并通过 CLB 为每个游戏服映射独立的公网地址](https://www.bilibili.com/video/BV1vaMAzqEGL/)。
- 移除暂时不需要的 pod webhook。
- 更新 API 文档。
- 安装文档：安装 chart 到 kube-system。
- chart 优化：避免名称拼接导致的长度超限。
- 升级 golang 到 v1.24。
- 升级 kubebuilder 到 v4.6.0。
- 升级 controller-runtime 到 v0.21.0。

## v2.1.1 (2025-06-10)

- enable 注解值为 false 时解绑 rs 但保留监听器实现网络隔离。
- OKG GameServerSet 删除时，清理关联的 CLBPortPool。
- 修复某些删除场景的监听器泄露情况。

## v2.1.0 (2025-06-09)

- 支持为 OKG 的 GameServerSet 的 TencentCloud-CLB 模式的 GameServer 自动创建对应的端口池。
- 自动创建 CLB 默认使用固定 VIP，通过可选的 `dynamicVip` 选项可以改为域名化的 CLB。

## v2.0.9 (2025-06-05)

- 映射结果信息更详细。
- 优化一些 edge case。
- 升级 golang 版本。
- 完善文档。

## v2.0.8 (2025-06-05)

- 重构和优化端口池的端口分配逻辑和 CLB 自动创建逻辑。
- 优化性能。
- 极大的提升了稳定性。

## v2.0.7 (2025-06-03)

- 支持手动配置云 API 限速，避免大规模场景因调用过快导致频繁 API 限速，降低整体对账速度。
- 提升稳定性。

## v2.0.6 (2025-06-03)

- 修复一些 bug

## v2.0.5 (2025-05-29)

- 性能优化：大幅提升 CLBPodBinding 和 CLBNodeBinding 的回收速度。
- 小范围重构：提升健壮性

## v2.0.4 (2025-05-27)

- 修复因 k8s api conflict 导致已创建的监听器 ID 没被记录上后过多失败的重试（在一次调谐周期内尽量保证重要信息记录上，避免下次调谐因信息不对做额外更多的错误重试）。

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
* 修复 clb 不存在时无法删除 DedicatedCLBListener ([#1](https://github.com/tkestack/tke-extend-network-controller/issues/1))。
 
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

