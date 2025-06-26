# 安装

## 前提条件

安装 `tke-extend-network-controller` 前请确保满足以下前提条件：
1. 确保腾讯云账号是带宽上移账号，参考 [账户类型说明](https://cloud.tencent.com/document/product/1199/49090) 进行判断或升级账号类型（如果账号创建的时间很早，有可能是传统账号）。
2. 创建了 [TKE](https://cloud.tencent.com/product/tke) 集群，且集群版本大于等于 1.26。
3. 集群中安装了 [cert-manager](https://cert-manager.io/docs/installation/) (webhook 依赖证书)，可通过 [TKE 应用市场](https://console.cloud.tencent.com/tke2/helm/market) 安装。
4. 需要一个腾讯云子账号的访问密钥(SecretID、SecretKey)，参考[子账号访问密钥管理](https://cloud.tencent.com/document/product/598/37140)，要求账号至少具有以下权限：
    ```json
    {
        "version": "2.0",
        "statement": [
            {
                "effect": "allow",
                "action": [
                    "clb:CreateLoadBalancer",
                    "clb:DeleteLoadBalancer",
                    "clb:DescribeLoadBalancers",
                    "clb:CreateListener",
                    "clb:DeleteListener",
                    "clb:DeleteLoadBalancerListeners",
                    "clb:DescribeListeners",
                    "clb:RegisterTargets",
                    "clb:BatchRegisterTargets",
                    "clb:DeregisterTargets",
                    "clb:BatchDeregisterTargets",
                    "clb:DescribeTargets",
                    "clb:DescribeQuota",
                    "clb:DescribeTaskStatus",
                    "vpc:DescribeAddresses",
                    "cvm:DescribeAddresses"
                ],
                "resource": [
                    "*"
                ]
            }
        ]
    }
    ```

## 安装方法

在 [TKE 应用市场](https://console.cloud.tencent.com/tke2/helm/market) 的网络分类中找到 `tke-extend-network-controller`，编辑 `values.yaml`，根据需求进行配置，以下几个参数是必填的：

```yaml
vpcID: "" # TKE 集群所在 VPC ID (vpc-xxx)
region: "" # TKE 集群所在地域，如 ap-guangzhou
clusterID: "" # TKE 集群 ID (cls-xxx)
secretID: "" # 腾讯云子账号的 SecretID
secretKey: "" # 腾讯云子账号的 SecretKey
```

配置完成后单击【完成】即可安装到集群。

另外您也可以通过 helm 安装：

```bash
helm repo add tke-extend-network-controller https://tkestack.github.io/tke-extend-network-controller
helm upgrade --install --namespace kube-system -f values.yaml \
  tke-extend-network-controller tke-extend-network-controller/tke-extend-network-controller
```

> 升级方法：先执行 `helm repo update`，然后再重复执行上面相同的 `helm upgrade` 完整命令即可完成升级。
