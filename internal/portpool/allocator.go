package portpool

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
)

// PortAllocator 管理多个端口池
type PortAllocator struct {
	mu    sync.RWMutex
	pools PortPools // 端口池名称到实例的映射
}

// NewPortAllocator 创建新的端口分配器
func NewPortAllocator() *PortAllocator {
	return &PortAllocator{
		pools: make(PortPools),
	}
}

func (pa *PortAllocator) GetPool(name string) *PortPool {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	if pool, exists := pa.pools[name]; exists {
		return pool
	}
	return nil
}

// CanAllocate 检查指定端口池是否还能分配出至少一个 TCPUDP 端口（dry run）
func (pa *PortAllocator) CanAllocate(name string, startPort, endPort, quota, segmentLength uint16) bool {
	pa.mu.RLock()
	pool, exists := pa.pools[name]
	pa.mu.RUnlock()
	if !exists {
		return false
	}
	return pool.CanAllocate(startPort, endPort, quota, segmentLength)
}

func (pa *PortAllocator) AllocatedPorts(name string, lbKey LBKey) uint16 {
	pa.mu.Lock()
	pool, exists := pa.pools[name]
	pa.mu.Unlock()
	if exists {
		return pool.AllocatedPorts(lbKey)
	}
	return 0
}

// 确保指定端口池的LbIds符合预期
func (pa *PortAllocator) EnsureLbIds(name string, lbKeys []LBKey) error {
	if len(lbKeys) == 0 {
		return nil
	}
	pa.mu.Lock()
	pool, exists := pa.pools[name]
	pa.mu.Unlock()

	if exists {
		pool.EnsureLbIds(lbKeys)
	} else {
		return fmt.Errorf("port pool %q is not exists", name)
	}
	return nil
}

// EnsurePool 添加新的端口池
func (pa *PortAllocator) EnsurePool(pool *networkingv1alpha1.CLBPortPool) (added bool) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	p, exists := pa.pools[pool.Name]

	if !exists {
		p = &PortPool{
			Name:        pool.Name,
			LbBlacklist: make(map[LBKey]struct{}),
			cache:       make(map[LBKey]map[ProtocolPort]struct{}),
		}
		pa.pools[pool.Name] = p
		added = true
	}

	lbPolicy := constant.LbPolicyRandom
	if pool.Spec.LbPolicy != nil {
		lbPolicy = *pool.Spec.LbPolicy
	}
	if p.LbPolicy != lbPolicy {
		p.LbPolicy = lbPolicy
	}
	if !reflect.DeepEqual(p.lbBlacklist, pool.Spec.LbBlacklist) {
		p.lbBlacklist = pool.Spec.LbBlacklist
		p.LbBlacklist = make(map[LBKey]struct{})
		for _, lbId := range pool.Spec.LbBlacklist {
			lbKey := LBKey{LbId: lbId, Region: pool.GetRegion()}
			if _, exists := p.LbBlacklist[lbKey]; !exists {
				p.LbBlacklist[lbKey] = struct{}{}
			}
		}
	}
	if lcp := pool.Spec.ListenerPrecreate; lcp != nil && lcp.Enabled {
		p.maxPort = &MaxPort{}
		startPort := pool.Spec.StartPort
		tcpNum := util.GetValue(lcp.TCP)
		if tcpNum > 0 {
			p.maxPort.Tcp = startPort + tcpNum - 1
		}
		udpNum := util.GetValue(lcp.UDP)
		if udpNum > 0 {
			p.maxPort.Udp = startPort + udpNum - 1
		}
	}
	return
}

// RemovePool 移除端口池
func (pa *PortAllocator) RemovePool(name string) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	delete(pa.pools, name)
}

func (pa *PortAllocator) getPortPools(pools []string) (PortPools, error) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	pp, err := pa.pools.Sub(pools...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return pp, nil
}

// Allocate 分配一个端口
func (pa *PortAllocator) Allocate(ctx context.Context, pools []string, protocol string, useSamePortAcrossPools bool) (PortAllocations, error) {
	portPools, err := pa.getPortPools(pools)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if ports, err := portPools.AllocatePort(ctx, protocol, useSamePortAcrossPools); err != nil {
		return nil, errors.WithStack(err)
	} else {
		return ports, nil
	}
}

func (pa *PortAllocator) ReleaseBinding(binding *networkingv1alpha1.PortBindingStatus) bool {
	return pa.Release(binding.Pool, NewLBKeyFromBinding(binding), NewProtocolPortFromBinding(binding))
}

// Release 释放一个端口
func (pa *PortAllocator) Release(pool string, lbKey LBKey, port ProtocolPort) bool {
	if pp := pa.GetPool(pool); pp != nil {
		return pp.ReleasePort(lbKey, port)
	}
	return false
}

func (pa *PortAllocator) IsLbExists(pool string, lbKey LBKey) bool {
	if pp := pa.GetPool(pool); pp != nil {
		return pp.IsLbExists(lbKey)
	}
	return false
}

func (pa *PortAllocator) RemoveLB(pool string, lbKey LBKey) bool {
	if pp := pa.GetPool(pool); pp != nil {
		return pp.RemoveLB(lbKey)
	}
	return false
}

var Allocator = NewPortAllocator()

func (pa *PortAllocator) MarkAllocated(poolName string, lbKey LBKey, port uint16, endPort *uint16, protocol string) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pool, ok := pa.pools[poolName]
	if !ok {
		return
	}
	finalEndPort := uint16(0)
	if endPort != nil {
		finalEndPort = *endPort
	}
	if lb := pool.cache[lbKey]; lb != nil {
		lb[ProtocolPort{Port: port, EndPort: finalEndPort, Protocol: protocol}] = struct{}{}
	}
}
