# 安装

## 前提条件

安装 `tke-extend-network-controller` 前请确保满足以下前提条件：
1. 确保腾讯云账号是带宽上移账号，参考 [账户类型说明](https://cloud.tencent.com/document/product/1199/49090) 进行判断或升级账号类型（如果账号创建的时间很早，有可能是传统账号）。
2. 创建了 [TKE](https://cloud.tencent.com/product/tke) 集群，且集群版本大于等于 1.26。
3. 集群中安装了 [cert-manager](https://cert-manager.io/docs/installation/) (webhook 依赖证书)。
4. 本地安装了 [helm](https://helm.sh) 命令，且能通过 helm 命令操作 TKE 集群（参考[本地 Helm 客户端连接集群](https://cloud.tencent.com/document/product/457/32731)）。
5. 需要一个腾讯云子账号的访问密钥(SecretID、SecretKey)，参考[子账号访问密钥管理](https://cloud.tencent.com/document/product/598/37140)，要求账号至少具有以下权限：
    ```json
    {
        "version": "2.0",
        "statement": [
            {
                "effect": "allow",
                "action": [
                    "clb:DescribeLoadBalancerBackends",
                    "clb:DescribeLoadBalancerListeners",
                    "clb:DescribeLoadBalancers",
                    "clb:CreateLoadBalancer",
                    "clb:DescribeTargets",
                    "clb:DeleteLoadBalancer",
                    "clb:DeleteLoadBalancerListeners",
                    "clb:BatchDeregisterTargets",
                    "clb:BatchRegisterTargets",
                    "clb:DeregisterTargets",
                    "clb:CreateLoadBalancerListeners",
                    "clb:CreateListener",
                    "clb:RegisterTargets",
                    "clb:DeleteLoadBalancers",
                    "clb:DescribeLoadBalancersDetail"
                ],
                "resource": [
                    "*"
                ]
            }
        ]
    }
    ```

## 使用 helm 安装

1. 添加 helm repo:

```bash
helm repo add tke-extend-network-controller https://imroc.github.io/tke-extend-network-controller
```

2. 创建 `values.yaml` 并配置:

```yaml
vpcID: "" # TKE 集群所在 VPC ID (vpc-xxx)
clusterID: "" # TKE 集群 ID (cls-xxx)
secretID: "" # 腾讯云子账号的 SecretID
secretKey: "" # 腾讯云子账号的 SecretKey
```

3. 安装到 TKE 集群：
```bash
helm upgrade --install -f values.yaml \
  --namespace tke-extend-network-controller --create-namespace \
  tke-extend-network-controller tke-extend-network-controller/tke-extend-network-controller
```

> 1. 如果要升级版本，先执行 `helm repo update`，再重复执行上面的安装命令即可。
> 2. 如果要更改配置，直接修改 `values.yaml`，再重复执行上面的安装命令即可。
