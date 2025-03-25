package portpool

import (
	"context"
	"fmt"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (pa *PortAllocator) IsPoolExists(name string) bool {
	pa.mu.RLock()
	_, exists := pa.pools[name]
	pa.mu.RUnlock()
	return exists
}

// 确保指定端口池的LbIds符合预期
func (pa *PortAllocator) EnsureLbIds(name string, lbIds []string) error {
	pa.mu.Lock()
	pool, exists := pa.pools[name]
	pa.mu.Unlock()

	if exists {
		pool.EnsureLbIds(lbIds)
	} else {
		return fmt.Errorf("port pool %q is not exists", name)
	}
	return nil
}

// AddPool 添加新的端口池
func (pa *PortAllocator) AddPool(pp CLBPortPool) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	if _, exists := pa.pools[pp.GetName()]; exists {
		return nil
	}

	pool := &PortPool{
		CLBPortPool: pp,
		cache:       make(map[string]map[ProtocolPort]struct{}),
	}

	pa.pools[pp.GetName()] = pool
	return nil
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
	return pa.pools.Sub(pools...)
}

// Allocate 分配一个端口
func (pa *PortAllocator) Allocate(ctx context.Context, pools []string, protocol string, useSamePortAcrossPools bool) (PortAllocations, error) {
	portPools, err := pa.getPortPools(pools)
	if err != nil {
		return nil, err
	}
	return portPools.AllocatePort(ctx, protocol, useSamePortAcrossPools)
}

// Release 释放一个端口
func (pa *PortAllocator) Release(pool, lbId string, port ProtocolPort) {
	if pp := pa.GetPool(pool); pp != nil {
		pp.ReleasePort(lbId, port)
	}
}

var Allocator = NewPortAllocator()

var (
	allocator    *PortAllocator
	allocatorMux sync.Mutex
	apiClient    client.Client
)

func Init(client client.Client) {
	apiClient = client
}

func (pa *PortAllocator) MarkAllocated(poolName string, lbId string, port uint16, endPort *uint16, protocol string) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pool, ok := pa.pools[poolName]
	if !ok {
		return fmt.Errorf("pool %s not found", poolName)
	}
	finalEndPort := uint16(0)
	if endPort != nil {
		finalEndPort = *endPort
	}
	pool.cache[lbId][ProtocolPort{Port: port, EndPort: finalEndPort, Protocol: protocol}] = struct{}{}
	return nil
}
