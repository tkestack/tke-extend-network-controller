package portpool

import (
	"context"
	"fmt"
	"strings"

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
			return nil, errors.Wrapf(ErrPoolNotFound, "pool %q is not exists", poolName)
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
	startPort, endPort, segmentLength uint16,
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
			result, err := pool.AllocatePort(ctx, portsToAllocate...)
			if err != nil { // 有分配错误，释放已分配的端口
				if err == ErrNoFreeLb { // 超配额，跳出端口循环，尝试创建 CLB
					log.FromContext(ctx).V(10).Info("no free lb available when allocate port", "pool", pool.GetName(), "tryPort", port)
					break
				}
				allocatedPorts.Release()
				return nil, err
			}
			if len(result) > 0 { // 该端口池分配到了端口，追加到结果中
				allocatedPorts = append(allocatedPorts, result...)
				log.FromContext(ctx).V(10).Info("allocated port", "pool", pool.GetName(), "port", port)
				continue LOOP_POOL
			}
			// 该端口池中无法分配此端口，尝试下一个端口
			log.FromContext(ctx).V(10).Info("no available port can be allocated, try next port", "pool", pool.GetName(), "port", port)
		}
		// 该端口池所有端口都无法分配，或者监听器数量超配额，为保证事务性，释放已分配的端口，并尝试通知端口池扩容 CLB 来补充端口池
		allocatedPorts.Release()
		// 检查端口池是否可以创建 CLB
		result, err := pool.TryCreateLB(ctx)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		switch result {
		case CreateLbResultForbidden: // 不能自动创建，返回端口不足的错误
			return nil, ErrNoPortAvailable
		case CreateLbResultCreating: // 正在创建 CLB，创建完后会自动触发对账
			return nil, ErrNewLBCreating
		case CreateLbResultSuccess: // 已经通知过或通知成功，重新入队
			return nil, ErrNewLBCreated
		default: // 不可能的状态
			return nil, ErrUnknown
		}
	}
	// 所有端口池都分配成功，返回结果
	return allocatedPorts, nil
}

// 从所有端口池中都分配出指定端口，不同端口池必须分配相同端口
func (pp PortPools) allocateSamePortAcrossPools(
	ctx context.Context,
	startPort, endPort, segmentLength uint16,
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
			results, err := pool.AllocatePort(ctx, portsToAllocate...)
			if err != nil || len(results) == 0 { // 有端口池无法分配或出错，为保证事务性，释放已分配的端口
				allocatedPorts.Release()
				if err != nil { // 分配出错，返回错误
					return nil, errors.Wrap(err, "allocate port failed")
				} else { // 有端口池无法分配此端口号，换下一个端口号
					allocatedPorts = nil
					continue LOOP_PORT
				}
			}
			// 此端口池分配到了此端口，追加到结果中
			allocatedPorts = append(allocatedPorts, results...)
		}
		// 分配结束，返回结果（可能为空）
		return allocatedPorts, nil
	}
	// 所有端口池都无法分配，返回错误告知分配失败
	return nil, fmt.Errorf("no available port can be allocated across port pools %q", pp.Names())
}

// 从一个或多个端口池中分配一个指定协议的端口，分配成功返回端口号，失败返回错误
func (pp PortPools) AllocatePort(ctx context.Context, protocol string, useSamePortAcrossPools bool) (ports PortAllocations, err error) {
	startPort := uint16(0)
	endPort := uint16(65535)
	segmentLength := uint16(0)
	for _, portPool := range pp {
		if portPool.GetStartPort() > startPort {
			startPort = portPool.GetStartPort()
		}
		if portPool.GetEndPort() < endPort {
			endPort = portPool.GetEndPort()
		}
		if segmentLength == 0 {
			segmentLength = portPool.GetSegmentLength()
		} else if segmentLength != portPool.GetSegmentLength() {
			err = ErrSegmentLengthNotEqual
			return
		}
	}
	if startPort > endPort {
		err = fmt.Errorf("there is no intersection between port ranges of port pools: %s", pp.Names())
		return
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
		ports, err = pp.allocateSamePortAcrossPools(ctx, startPort, endPort, segmentLength, getPortsToAllocate)
	} else {
		ports, err = pp.allocatePortAcrossPools(ctx, startPort, endPort, segmentLength, getPortsToAllocate)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ports, nil
}

func (pp PortPools) ReleasePort(lbId string, port uint16, protocol string) {
	if protocol == "TCPUDP" {
		tcpPort := ProtocolPort{
			Port:     port,
			Protocol: "TCP",
		}
		udpPort := ProtocolPort{
			Port:     port,
			Protocol: "UDP",
		}
		for _, portPool := range pp {
			if portCache := portPool.cache[lbId]; portCache != nil {
				delete(portCache, tcpPort)
				delete(portCache, udpPort)
				break
			}
		}
	} else {
		port := ProtocolPort{
			Port:     port,
			Protocol: protocol,
		}
		for _, portPool := range pp {
			if portCache := portPool.cache[lbId]; portCache != nil {
				delete(portCache, port)
				break
			}
		}
	}
}
