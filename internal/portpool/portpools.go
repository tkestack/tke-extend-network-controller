package portpool

import (
	"context"
	"fmt"
	"strings"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"

	"github.com/imroc/tke-extend-network-controller/internal/constant"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PortPools map[string]*PortPool

func (pp PortPools) Sub(poolNames ...string) (PortPools, error) {
	sub := make(PortPools)
	for _, poolName := range poolNames {
		if pool, exists := pp[poolName]; exists {
			sub[poolName] = pool
		} else {
			return nil, &ErrPoolNotFound{Pool: poolName}
		}
	}
	return sub, nil
}

func (pp PortPools) Names() string {
	names := []string{}
	for name := range pp {
		names = append(names, name)
	}
	return strings.Join(names, ",")
}

// 从所有端口池中都分配出指定端口，不同端口池可分配不同端口
func (pp PortPools) allocatePortAcrossPools(
	ctx context.Context,
	startPort, endPort, quota, segmentLength uint16,
	getPortsToAllocate func(port, endPort uint16) (ports []ProtocolPort),
) (PortAllocations, error) {
	log.FromContext(ctx).V(10).Info("allocatePortAcrossPools", "pools", pp.Names(), "startPort", startPort, "endPort", endPort, "segmentLength", segmentLength)
	var allocatedPorts PortAllocations
LOOP_POOL:
	for _, pool := range pp { // 遍历所有端口池（由于不需要保证所有端口池的端口号相同，因此外层循环直接遍历端口池）
		for port := startPort; port <= endPort; port += segmentLength { // 遍历该端口池的所有端口号
			select {
			case <-ctx.Done():
				allocatedPorts.Release()
				if err := ctx.Err(); err != nil {
					return nil, errors.WithStack(err)
				}
				return nil, nil
			default:
			}
			endPort := uint16(0)
			if segmentLength > 1 {
				endPort = port + segmentLength - 1
			}
			portsToAllocate := getPortsToAllocate(port, endPort)
			// 尝试分配端口
			result, quotaExceeded := pool.AllocatePort(ctx, int64(quota), portsToAllocate...)
			if quotaExceeded { // 超配额，不可能分配成功，不再继续尝试
				allocatedPorts.Release()
				return nil, nil
			}
			if len(result) > 0 { // 该端口池分配到了端口，追加到结果中
				allocatedPorts = append(allocatedPorts, result...)
				log.FromContext(ctx).V(10).Info("allocated port", "pool", pool.Name, "port", port)
				continue LOOP_POOL
			} else {
				// 该端口池中无法分配此端口，尝试下一个端口
				log.FromContext(ctx).V(10).Info("no available port can be allocated, try next port", "pool", pool.Name, "port", port)
				continue
			}
		}
		// 该端口池所有端口都无法分配，不再尝试
		allocatedPorts.Release()
		return nil, nil
	}
	// 所有端口池都分配成功，返回结果
	return allocatedPorts, nil
}

// 从所有端口池中都分配出指定端口，不同端口池必须分配相同端口
func (pp PortPools) allocateSamePortAcrossPools(
	ctx context.Context,
	startPort, endPort, quota, segmentLength uint16,
	getPortsToAllocate func(port, endPort uint16) (ports []ProtocolPort),
) (PortAllocations, error) {
	log.FromContext(ctx).Info("allocateSamePortAcrossPools", "pools", pp.Names(), "startPort", startPort, "endPort", endPort, "segmentLength", segmentLength)
LOOP_PORT:
	for port := startPort; port <= endPort; port += segmentLength { // 遍历所有端口号，确保所有端口池都能分配到相同端口号
		endPort := uint16(0)
		if segmentLength > 1 {
			endPort = port + segmentLength - 1
		}
		portsToAllocate := getPortsToAllocate(port, endPort)
		var allocatedPorts PortAllocations
		for _, pool := range pp { // 在所有端口池中查找可用端口，TCP 和 UDP 端口号相同且都未被分配，则分配此端口号
			select {
			case <-ctx.Done():
				allocatedPorts.Release()
				if err := ctx.Err(); err != nil {
					return nil, errors.WithStack(err)
				}
				return nil, nil
			default:
			}

			results, quotaExceeded := pool.AllocatePort(ctx, int64(quota), portsToAllocate...)
			if quotaExceeded {
				allocatedPorts.Release()
				return nil, nil
			}
			if len(results) > 0 {
				// 此端口池分配到了此端口，追加到结果中
				allocatedPorts = append(allocatedPorts, results...)
			} else { // 有端口池无法分配此端口号，释放已分配端口，换下一个端口
				allocatedPorts.Release()
				continue LOOP_PORT
			}
		}
		// 分配结束，返回结果可能为空）
		return allocatedPorts, nil
	}
	// 所有端口池都无法分配，返回空结果
	return nil, nil
}

var (
	ErrQuotaNotEqual          = errors.New("quota not equal")
	ErrQuotaNotFound          = errors.New("quota not found")
	ErrPortPoolNotAllocatable = errors.New("port pool not allocatable")
)

// 从一个或多个端口池中分配一个指定协议的端口，分配成功返回端口号，失败返回错误
func (pp PortPools) AllocatePort(ctx context.Context, protocol string, useSamePortAcrossPools bool) (ports PortAllocations, err error) {
	startPort := uint16(0)
	endPort := uint16(65535)
	segmentLength := uint16(0)
	quota := uint16(0)
	for _, portPool := range pp {
		cpp, err := kube.GetCLBPortPool(ctx, portPool.Name)
		if cpp.Status.State != networkingv1alpha1.CLBPortPoolStateActive && cpp.Status.State != networkingv1alpha1.CLBPortPoolStateScaling {
			return nil, ErrPortPoolNotAllocatable
		}
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if cpp.Spec.StartPort > startPort {
			startPort = cpp.Spec.StartPort
		}
		if cpp.Spec.EndPort != nil && *cpp.Spec.EndPort < endPort {
			endPort = *cpp.Spec.EndPort
		}

		if segmentLength == 0 {
			if cpp.Spec.SegmentLength != nil {
				segmentLength = *cpp.Spec.SegmentLength
			}
		} else {
			if cpp.Spec.SegmentLength != nil && *cpp.Spec.SegmentLength != segmentLength {
				return nil, ErrSegmentLengthNotEqual
			}
		}
		if quota == 0 {
			if cpp.Status.Quota != quota {
				quota = cpp.Status.Quota
			}
		} else {
			if cpp.Status.Quota != quota {
				return nil, ErrQuotaNotEqual
			}
		}
	}

	if startPort > endPort {
		return nil, fmt.Errorf("there is no intersection between port ranges of port pools: %s", pp.Names())
	}
	if quota == 0 {
		return nil, ErrQuotaNotFound
	}
	if segmentLength == 0 {
		segmentLength = 1
	}

	getPortsToAllocate := func(port, endPort uint16) (ports []ProtocolPort) {
		if protocol == constant.ProtocolTCPUDP {
			ports = append(ports, ProtocolPort{
				Port:     port,
				Protocol: constant.ProtocolTCP,
				EndPort:  endPort,
			})
			ports = append(ports, ProtocolPort{
				Port:     port,
				Protocol: constant.ProtocolUDP,
				EndPort:  endPort,
			})
		} else {
			ports = append(ports, ProtocolPort{
				Port:     port,
				Protocol: protocol,
				EndPort:  endPort,
			})
		}
		return
	}

	if useSamePortAcrossPools {
		ports, err = pp.allocateSamePortAcrossPools(ctx, startPort, endPort, quota, segmentLength, getPortsToAllocate)
	} else {
		ports, err = pp.allocatePortAcrossPools(ctx, startPort, endPort, quota, segmentLength, getPortsToAllocate)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ports, nil
}

// func (pp PortPools) ReleasePort(lbId string, port uint16, protocol string) {
// 	if protocol == "TCPUDP" {
// 		tcpPort := ProtocolPort{
// 			Port:     port,
// 			Protocol: "TCP",
// 		}
// 		udpPort := ProtocolPort{
// 			Port:     port,
// 			Protocol: "UDP",
// 		}
// 		for _, portPool := range pp {
// 			if portCache := portPool.cache[lbId]; portCache != nil {
// 				delete(portCache, tcpPort)
// 				delete(portCache, udpPort)
// 				break
// 			}
// 		}
// 	} else {
// 		port := ProtocolPort{
// 			Port:     port,
// 			Protocol: protocol,
// 		}
// 		for _, portPool := range pp {
// 			if portCache := portPool.cache[lbId]; portCache != nil {
// 				delete(portCache, port)
// 				break
// 			}
// 		}
// 	}
// }
