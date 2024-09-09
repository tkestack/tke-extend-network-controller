# API Reference

## Packages
- [networking.cloud.tencent.com/v1alpha1](#networkingcloudtencentcomv1alpha1)


## networking.cloud.tencent.com/v1alpha1


### Resource Types
- [DedicatedCLBListener](#dedicatedclblistener)
- [DedicatedCLBService](#dedicatedclbservice)



#### AllocatableCLBInfo







_Appears in:_
- [DedicatedCLBServiceStatus](#dedicatedclbservicestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbId` _string_ | CLB 实例的 ID。 |  |  |
| `currentPort` _integer_ | CLB 当前已被分配的端口。 |  |  |
| `autoCreate` _boolean_ | 是否是自动创建的 CLB。如果是，删除 DedicatedCLBService 时，CLB 也会被清理。 |  |  |


#### AllocatedCLBInfo







_Appears in:_
- [DedicatedCLBServiceStatus](#dedicatedclbservicestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbId` _string_ | CLB 实例的 ID。 |  |  |
| `autoCreate` _boolean_ | 是否是自动创建的 CLB。如果是，删除 DedicatedCLBService 时，CLB 也会被清理。 |  |  |


#### DedicatedCLBListener



DedicatedCLBListener is the Schema for the dedicatedclblisteners API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `DedicatedCLBListener` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[DedicatedCLBListenerSpec](#dedicatedclblistenerspec)_ |  |  |  |
| `status` _[DedicatedCLBListenerStatus](#dedicatedclblistenerstatus)_ |  |  |  |


#### DedicatedCLBListenerSpec



DedicatedCLBListenerSpec defines the desired state of DedicatedCLBListener



_Appears in:_
- [DedicatedCLBListener](#dedicatedclblistener)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbId` _string_ | CLB 实例的 ID。 |  |  |
| `lbRegion` _string_ | CLB 所在地域，不填则使用 TKE 集群所在的地域。 |  |  |
| `lbPort` _integer_ | CLB 监听器的端口号。 |  |  |
| `protocol` _string_ | CLB 监听器的协议。 |  | Enum: [TCP UDP] <br /> |
| `extensiveParameters` _string_ | 创建监听器的参数，JSON 格式，详细参数请参考 CreateListener 接口：https://cloud.tencent.com/document/api/214/30693 |  |  |
| `targetPod` _[TargetPod](#targetpod)_ | CLB 监听器绑定的目标 Pod。 |  |  |


#### DedicatedCLBListenerStatus



DedicatedCLBListenerStatus defines the observed state of DedicatedCLBListener



_Appears in:_
- [DedicatedCLBListener](#dedicatedclblistener)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `listenerId` _string_ | CLB 监听器的 ID。 |  |  |
| `state` _string_ | CLB 监听器的状态。 |  | Enum: [Bound Available Pending Failed Deleting] <br /> |
| `message` _string_ | 记录 CLB 监听器的失败信息。 |  |  |
| `address` _string_ | CLB 监听器的外部地址。 |  |  |


#### DedicatedCLBService



DedicatedCLBService is the Schema for the dedicatedclbservices API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `DedicatedCLBService` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[DedicatedCLBServiceSpec](#dedicatedclbservicespec)_ |  |  |  |
| `status` _[DedicatedCLBServiceStatus](#dedicatedclbservicestatus)_ |  |  |  |


#### DedicatedCLBServicePort







_Appears in:_
- [DedicatedCLBServiceSpec](#dedicatedclbservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `protocol` _string_ | 端口协议，支持 TCP、UDP。 |  |  |
| `targetPort` _integer_ | 目标端口。 |  |  |
| `addressPodAnnotation` _string_ | Pod 外部地址的注解，如果设置，Pod 被映射的外部 CLB 地址将会被自动写到 Pod 的该注解中，Pod 内部可通过 Downward API 感知到自身的外部地址。 |  |  |


#### DedicatedCLBServiceSpec



DedicatedCLBServiceSpec defines the desired state of DedicatedCLBService



_Appears in:_
- [DedicatedCLBService](#dedicatedclbservice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbRegion` _string_ | CLB 所在地域，不填则使用 TKE 集群所在的地域。 |  |  |
| `vpcId` _string_ | CLB 所在 VPC ID，不填则使用 TKE 集群所在的 VPC 的 ID。 |  |  |
| `minPort` _integer_ | CLB 端口范围的最小端口号。 | 500 |  |
| `maxPort` _integer_ | CLB 端口范围的最大端口号。 | 50000 |  |
| `maxPod` _integer_ | 限制单个 CLB 的 Pod/监听器 的最大数量。 |  |  |
| `selector` _object (keys:string, values:string)_ | Pod 的标签选择器，被选中的 Pod 会被绑定到 CLB 监听器下。 |  |  |
| `ports` _[DedicatedCLBServicePort](#dedicatedclbserviceport) array_ | Pod 监听的端口。 |  |  |
| `listenerExtensiveParameters` _string_ | 创建监听器的参数，JSON 格式，详细参数请参考 CreateListener 接口：https://cloud.tencent.com/document/api/214/30693 |  |  |
| `existedLbIds` _string array_ | 复用的已有的 CLB ID，可动态追加。 |  |  |
| `lbAutoCreate` _[LbAutoCreate](#lbautocreate)_ | 启用自动创建 CLB 的功能。 |  |  |


#### DedicatedCLBServiceStatus



DedicatedCLBServiceStatus defines the observed state of DedicatedCLBService



_Appears in:_
- [DedicatedCLBService](#dedicatedclbservice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `allocatableLb` _[AllocatableCLBInfo](#allocatableclbinfo) array_ | 可分配端口的 CLB 列表 |  |  |
| `allocatedLb` _[AllocatedCLBInfo](#allocatedclbinfo) array_ | 已分配完端口的 CLB 列表 |  |  |


#### LbAutoCreate







_Appears in:_
- [DedicatedCLBServiceSpec](#dedicatedclbservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enable` _boolean_ | 是否启用自动创建 CLB 的功能，如果启用，当 CLB 不足时，会自动创建新的 CLB。 |  |  |
| `extensiveParameters` _string_ | 创建 CLB 时的参数，JSON 格式，详细参数请参考 CreateLoadBalancer 接口：https://cloud.tencent.com/document/api/214/30692 |  |  |


#### TargetPod







_Appears in:_
- [DedicatedCLBListenerSpec](#dedicatedclblistenerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podName` _string_ | Pod 的名称。 |  |  |
| `targetPort` _integer_ | Pod 监听的端口。 |  |  |


