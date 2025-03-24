# CRD 设计

## CLBPortPool

CLB 端口池，池中所有 CLB 在相同地域，pod 的端口可被多个 CLB 端口池分配 CLB 端口映射。

```yaml
annotations:
  networking.cloud.tencent.com/clb-port-mapping: |-
    8000 TCPUDP pool-ctcc,pool-cucc useSamePortAcrossPools
    8001 UDP pool-cucc
    8001 TCP pool-cucc
```
