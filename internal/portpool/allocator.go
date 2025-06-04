package portpool

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
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

func (pa *PortAllocator) AddLbId(name string, lbKey LBKey) error {
	pa.mu.Lock()
	pool, exists := pa.pools[name]
	pa.mu.Unlock()
	if exists {
		pool.AddLbId(lbKey)
	} else {
		return fmt.Errorf("port pool %q is not exists", name)
	}
	return nil
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

// AddPoolIfNotExists 添加新的端口池
func (pa *PortAllocator) AddPoolIfNotExists(name string) bool {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	if _, exists := pa.pools[name]; exists {
		return false
	}

	pool := &PortPool{
		Name:  name,
		cache: make(map[LBKey]map[ProtocolPort]struct{}),
	}
	pa.pools[name] = pool
	return true
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

// Release 释放一个端口
func (pa *PortAllocator) Release(pool string, lbKey LBKey, port ProtocolPort) bool {
	if pp := pa.GetPool(pool); pp != nil {
		return pp.ReleasePort(lbKey, port)
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
