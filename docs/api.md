# API Reference

## Packages
- [networking.cloud.tencent.com/v1alpha1](#networkingcloudtencentcomv1alpha1)


## networking.cloud.tencent.com/v1alpha1


### Resource Types
- [CLB](#clb)
- [CLBListenerConfig](#clblistenerconfig)
- [DedicatedCLBListener](#dedicatedclblistener)
- [DedicatedCLBService](#dedicatedclbservice)
- [DedicatedNatgwService](#dedicatednatgwservice)



#### CLB



CLB is the Schema for the clbs API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `CLB` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CLBSpec](#clbspec)_ |  |  |  |
| `status` _[CLBStatus](#clbstatus)_ |  |  |  |


#### CLBHealthcheck







_Appears in:_
- [CLBListenerConfigSpec](#clblistenerconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `healthSwitch` _integer_ | whether to enable the health check, 1(enable), 0(disable) |  |  |
| `timeOut` _integer_ | health check timeout, unit: second, range 2~60, default 2 |  |  |
| `intervalTime` _integer_ | health check interval, unit: second |  |  |
| `healthNum` _integer_ | health check threshold, range 2~10, default 3 |  |  |
| `unHealthNum` _integer_ | unhealthy threshold, range 2~10, default 3 |  |  |
| `httpCode` _integer_ | health check http code, range 1~31, default 31 |  |  |
| `httpCheckPath` _string_ | health check http path |  |  |
| `httpCheckDomain` _string_ | http check domain |  |  |
| `httpCheckMethod` _string_ | http check method |  |  |
| `checkPort` _integer_ | customized check port |  |  |
| `contextType` _string_ |  |  |  |
| `sendContext` _string_ |  |  |  |
| `recvContext` _string_ |  |  |  |
| `checkType` _string_ |  |  |  |
| `httpVersion` _string_ |  |  |  |
| `sourceIpType` _integer_ |  |  |  |
| `extendedCode` _string_ |  |  |  |


#### CLBListenerConfig



CLBListenerConfig is the Schema for the clblistenerconfigs API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `CLBListenerConfig` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CLBListenerConfigSpec](#clblistenerconfigspec)_ |  |  |  |
| `status` _[CLBListenerConfigStatus](#clblistenerconfigstatus)_ |  |  |  |


#### CLBListenerConfigSpec



CLBListenerConfigSpec defines the desired state of CLBListenerConfig



_Appears in:_
- [CLBListenerConfig](#clblistenerconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `healthcheck` _[CLBHealthcheck](#clbhealthcheck)_ |  |  |  |
| `certificate` _[Certificate](#certificate)_ |  |  |  |
| `sessionExpireTime` _integer_ |  |  |  |
| `scheduler` _string_ |  |  |  |
| `sniSwitch` _integer_ |  |  |  |
| `targetType` _string_ |  |  |  |
| `sessionType` _string_ |  |  |  |
| `keepaliveEnable` _integer_ |  |  |  |
| `endPort` _integer_ |  |  |  |
| `deregisterTargetRst` _boolean_ |  |  |  |
| `multiCertInfo` _[MultiCertInfo](#multicertinfo)_ |  |  |  |
| `maxConn` _integer_ |  |  |  |
| `maxCps` _integer_ |  |  |  |
| `idleConnectTimeout` _integer_ |  |  |  |
| `snatEnable` _boolean_ |  |  |  |


#### CLBListenerConfigStatus



CLBListenerConfigStatus defines the observed state of CLBListenerConfig



_Appears in:_
- [CLBListenerConfig](#clblistenerconfig)



#### CLBSpec



CLBSpec defines the desired state of CLB



_Appears in:_
- [CLB](#clb)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `region` _string_ |  |  |  |
| `autoCreated` _boolean_ |  |  |  |


#### CLBStatus



CLBStatus defines the observed state of CLB



_Appears in:_
- [CLB](#clb)



#### CertInfo







_Appears in:_
- [MultiCertInfo](#multicertinfo)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certId` _string_ |  |  |  |


#### Certificate







_Appears in:_
- [CLBListenerConfigSpec](#clblistenerconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `sslMode` _string_ |  |  |  |
| `certId` _string_ |  |  |  |
| `certCaId` _string_ |  |  |  |


#### DedicatedCLBInfo







_Appears in:_
- [DedicatedCLBServiceStatus](#dedicatedclbservicestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbId` _string_ |  |  |  |
| `maxPort` _integer_ |  |  |  |
| `autoCreate` _boolean_ |  |  |  |


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
| `lbId` _string_ |  |  |  |
| `lbRegion` _string_ |  |  |  |
| `lbPort` _integer_ |  |  |  |
| `protocol` _string_ |  |  | Enum: [TCP UDP] <br /> |
| `listenerConfig` _string_ |  |  |  |
| `targetPod` _[TargetPod](#targetpod)_ |  |  |  |


#### DedicatedCLBListenerStatus



DedicatedCLBListenerStatus defines the observed state of DedicatedCLBListener



_Appears in:_
- [DedicatedCLBListener](#dedicatedclblistener)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `listenerId` _string_ |  |  |  |
| `state` _string_ |  |  |  |
| `message` _string_ |  |  |  |
| `address` _string_ |  |  |  |


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
| `protocol` _string_ |  |  |  |
| `targetPort` _integer_ |  |  |  |
| `addressPodAnnotation` _string_ |  |  |  |


#### DedicatedCLBServiceSpec



DedicatedCLBServiceSpec defines the desired state of DedicatedCLBService



_Appears in:_
- [DedicatedCLBService](#dedicatedclbservice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbRegion` _string_ |  |  |  |
| `vpcId` _string_ |  |  |  |
| `minPort` _integer_ |  |  |  |
| `maxPort` _integer_ |  |  |  |
| `selector` _object (keys:string, values:string)_ |  |  |  |
| `ports` _[DedicatedCLBServicePort](#dedicatedclbserviceport) array_ |  |  |  |
| `listenerConfig` _string_ |  |  |  |
| `existedLbIds` _string array_ |  |  |  |
| `lbAutoCreate` _[LbAutoCreate](#lbautocreate)_ |  |  |  |


#### DedicatedCLBServiceStatus



DedicatedCLBServiceStatus defines the observed state of DedicatedCLBService



_Appears in:_
- [DedicatedCLBService](#dedicatedclbservice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lbList` _[DedicatedCLBInfo](#dedicatedclbinfo) array_ |  |  |  |


#### DedicatedNatgwService



DedicatedNatgwService is the Schema for the dedicatednatgwservices API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `DedicatedNatgwService` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[DedicatedNatgwServiceSpec](#dedicatednatgwservicespec)_ |  |  |  |
| `status` _[DedicatedNatgwServiceStatus](#dedicatednatgwservicestatus)_ |  |  |  |


#### DedicatedNatgwServiceSpec



DedicatedNatgwServiceSpec defines the desired state of DedicatedNatgwService



_Appears in:_
- [DedicatedNatgwService](#dedicatednatgwservice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `foo` _string_ | Foo is an example field of DedicatedNatgwService. Edit dedicatednatgwservice_types.go to remove/update |  |  |


#### DedicatedNatgwServiceStatus



DedicatedNatgwServiceStatus defines the observed state of DedicatedNatgwService



_Appears in:_
- [DedicatedNatgwService](#dedicatednatgwservice)



#### LbAutoCreate







_Appears in:_
- [DedicatedCLBServiceSpec](#dedicatedclbservicespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enable` _boolean_ |  |  |  |
| `extensiveParameters` _string_ |  |  |  |


#### MultiCertInfo







_Appears in:_
- [CLBListenerConfigSpec](#clblistenerconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `sslMode` _string_ |  |  |  |
| `certList` _[CertInfo](#certinfo) array_ |  |  |  |


#### TargetPod







_Appears in:_
- [DedicatedCLBListenerSpec](#dedicatedclblistenerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podName` _string_ |  |  |  |
| `targetPort` _integer_ |  |  |  |


