# 开发手册

## 使用 kubebuilder 生成

本项目的控制器代码使用 `kubebuilder` 生成，基于 [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) 框架，具体请参考 [kubebuilder 官方文档](https://book.kubebuilder.io/quick-start.html)。

## Pod 缓存

控制器除了 list watch 自身 CRD 相关资源外，还 list watch 了 Pod 资源，而大规模场景下，全量 Pod 缓存占用内存较大，因为在 Pod 入缓存前，只保留本控制器关注的字段，其他字段不缓存，这样可大大降低内存占用。当要更新 Pod 时（比如注解和finalizer），就不直接拿缓存里的 Pod 进行 Update（因为缓存字段不全，无法直接 Update），而是先从 API Server 获取 Pod，再更新。
