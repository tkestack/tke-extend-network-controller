# tke-extend-network-controller

针对 TKE 集群一些特殊场景的的网络控制器。

## 支持房间类场景

目前主要支持会议、游戏战斗服等房间类场景的网络，即要求每个 Pod 都需要独立的公网地址，TKE 集群默认只支持 EIP 方案，但 EIP 资源有限，不仅是数量的限制，还有每日申请的数量限制，稍微上点规模，或频繁扩缩容更换EIP，可能很容易触达限制导致 EIP 分配失败；而如果保留 EIP，在 EIP 没被绑定前，又会收取额外的闲置费。

> TKE Pod 绑定 EIP 参考 [VPC-CNI: Pod 直接绑定弹性公网 IP 使用说明](https://cloud.tencent.com/document/product/457/64886) 与 [超级节点 Pod 绑定 EIP](https://cloud.tencent.com/document/product/457/44173#.E7.BB.91.E5.AE.9A-eip)。

如果不用 EIP，也可通过安装此插件来实现为每个 Pod 的指定端口都分配一个独立的公网地址映射 (公网 `IP:Port` 到内网 Pod `IP:Port` 的映射)。

## 文档

- [安装](./docs/install.md)
- [使用 CLB 为 Pod 分配公网地址映射](./docs/clb-mapping.md)
- [CRD 字段说明](docs/crd.md)
- [API 参考](docs/api.md)
- [Roadmap](docs/roadmap.md)
- [贡献指南](docs/contributing.md)
- [技术亮点](docs/inside.md)

## 项目状态与版本说明

当前项目正处于活跃开发中，请及时更新版本以获得最新能力，参考 [版本说明](CHANGELOG.md)。
