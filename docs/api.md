# API Reference

## Packages
- [networking.cloud.tencent.com/v1alpha1](#networkingcloudtencentcomv1alpha1)


## networking.cloud.tencent.com/v1alpha1


### Resource Types
- [CLB](#clb)
- [CLBList](#clblist)
- [CLBPodBinding](#clbpodbinding)
- [DedicatedCLBService](#dedicatedclbservice)
- [DedicatedNatgwService](#dedicatednatgwservice)



#### CLB



CLB is the Schema for the clbs API



_Appears in:_
- [CLBList](#clblist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `CLB` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CLBSpec](#clbspec)_ |  |  |  |
| `status` _[CLBStatus](#clbstatus)_ |  |  |  |


#### CLBList



CLBList contains a list of CLB





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `CLBList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[CLB](#clb) array_ |  |  |  |


#### CLBPodBinding



CLBPodBinding is the Schema for the clbpodbindings API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `networking.cloud.tencent.com/v1alpha1` | | |
| `kind` _string_ | `CLBPodBinding` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CLBPodBindingSpec](#clbpodbindingspec)_ |  |  |  |
| `status` _[CLBPodBindingStatus](#clbpodbindingstatus)_ |  |  |  |


#### CLBPodBindingSpec



CLBPodBindingSpec defines the desired state of CLBPodBinding



_Appears in:_
- [CLBPodBinding](#clbpodbinding)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podName` _string_ |  |  |  |
| `lbId` _string_ |  |  |  |
| `lbRegion` _string_ |  |  |  |
| `lbPort` _integer_ |  |  |  |
| `protocol` _string_ |  |  |  |
| `targetPort` _integer_ |  |  |  |


#### CLBPodBindingStatus



CLBPodBindingStatus defines the observed state of CLBPodBinding



_Appears in:_
- [CLBPodBinding](#clbpodbinding)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `state` _string_ | INSERT ADDITIONAL STATUS FIELD - define observed state of cluster<br />Important: Run "make" to regenerate code after modifying this file |  |  |


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


#### DedicatedCLBServiceSpec



DedicatedCLBServiceSpec defines the desired state of DedicatedCLBService



_Appears in:_
- [DedicatedCLBService](#dedicatedclbservice)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `minPort` _integer_ |  |  |  |
| `maxPort` _integer_ |  |  |  |
| `serviceName` _integer_ |  |  |  |
| `extensiveParameters` _string_ |  |  |  |
| `existedLbIds` _string array_ |  |  |  |


#### DedicatedCLBServiceStatus



DedicatedCLBServiceStatus defines the observed state of DedicatedCLBService



_Appears in:_
- [DedicatedCLBService](#dedicatedclbservice)



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



