package portpool

import (
	"context"
	"fmt"
	"iter"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"

	ctrl "sigs.k8s.io/controller-runtime"
)

var ppLog = ctrl.Log.WithName("portpool")

type LBPort struct {
	ProtocolPort
	LbId string // 负载均衡实例 ID
}

// Port 唯一标识一个分配的端口
type ProtocolPort struct {
	Port     uint16 // 端口号
	EndPort  uint16 // 结束端口号
	Protocol string // 协议 TCP/UDP/QUIC/TCP_SSL
}

func (p ProtocolPort) Key() ProtocolPort {
	l4Protocol := p.Protocol
	switch p.Protocol {
	case "TCP_SSL":
		l4Protocol = "TCP"
	case "QUIC":
		l4Protocol = "UDP"
	}
	return ProtocolPort{
		Port:     p.Port,
		EndPort:  p.EndPort,
		Protocol: l4Protocol,
	}
}

func NewProtocolPortFromBinding(status *networkingv1alpha1.PortBindingStatus) ProtocolPort {
	port := ProtocolPort{
		Port:     status.LoadbalancerPort,
		Protocol: status.Protocol,
	}
	if status.LoadbalancerEndPort != nil {
		port.EndPort = *status.LoadbalancerEndPort
	}
	return port
}

type PortAllocation struct {
	ProtocolPort
	*PortPool
	LBKey
}

func (pa PortAllocation) Release() {
	pa.ReleasePort(pa.LBKey, pa.ProtocolPort)
}

type PortAllocations []PortAllocation

func (pas PortAllocations) Pools() []string {
	if len(pas) == 0 {
		return []string{}
	}
	pools := make(map[string]struct{})
	for _, pa := range pas {
		pools[pa.PortPool.Name] = struct{}{}
	}
	ret := []string{}
	for pool := range pools {
		ret = append(ret, pool)
	}
	return ret
}

func (pas PortAllocations) String() string {
	ret := []string{}
	for _, pa := range pas {
		ret = append(ret, fmt.Sprintf("%s:%d/%s", pa.LBKey.LbId, pa.Port, pa.Protocol))
	}
	return fmt.Sprint(ret)
}

func (pas PortAllocations) Release() {
	for _, pa := range pas {
		pa.Release()
	}
}

type CreateLbResult int

const (
	CreateLbResultError CreateLbResult = iota
	CreateLbResultSuccess
	CreateLbResultForbidden
)

type CLBPortPool interface {
	GetName() string
	GetRegion() string
	GetStartPort() uint16
	GetEndPort() uint16
	GetListenerQuota() uint16
	GetSegmentLength() uint16
	TryCreateLB(ctx context.Context) (CreateLbResult, error)
}

type LBKey struct {
	LbId   string
	Region string
}

type MaxPort struct {
	Tcp uint16
	Udp uint16
}

func NewLBKey(lbId, region string) LBKey {
	return LBKey{
		LbId:   lbId,
		Region: region,
	}
}

func NewLBKeyFromBinding(binding *networkingv1alpha1.PortBindingStatus) LBKey {
	return NewLBKey(binding.LoadbalancerId, binding.Region)
}

// PortPool 管理单个端口池的状态
type PortPool struct {
	Name              string
	LbPolicy          string
	LbBlacklist       map[LBKey]struct{}
	lbBlacklist       []string
	maxPort           *MaxPort
	scaleUpRequested  atomic.Bool // 是否有扩容请求
	mu                sync.Mutex
	cache             map[LBKey]map[ProtocolPort]struct{}
	lbList            []LBKey
}

func (pp *PortPool) IsPrecreateListenerEnabled() bool {
	return pp.maxPort != nil
}

func (pp *PortPool) getCache() iter.Seq2[LBKey, map[ProtocolPort]struct{}] {
	return func(yield func(LBKey, map[ProtocolPort]struct{}) bool) {
		switch pp.LbPolicy {
		case constant.LbPolicyInOrder, constant.LbPolicyUniform: // 有序分配 或 均匀分配
			if pp.LbPolicy == constant.LbPolicyUniform { // 如果是均匀分配，则需要按已分配数量排序，找分配数量最小的 lb 分配
				slices.SortFunc(pp.lbList, func(a, b LBKey) int {
					return len(pp.cache[a]) - len(pp.cache[b])
				})
			}
			for _, lbKey := range pp.lbList {
				if _, exists := pp.LbBlacklist[lbKey]; exists { // 若 lb 在黑名单中，则跳过
					continue
				}
				if !yield(lbKey, pp.cache[lbKey]) { // 若 yield 返回 false 则中断
					return
				}
			}
		default: // 默认用 Random，按 map 的 key 顺序遍历（golang map 的 key 是无序的，每次遍历顺序随机）
			for lbKey, allocated := range pp.cache {
				if _, exists := pp.LbBlacklist[lbKey]; exists { // 若 lb 在黑名单中，则跳过
					continue
				}
				if !yield(lbKey, allocated) { // 若 yield 返回 false 则中断
					return
				}
			}
		}
	}
}

func (pp *PortPool) IsLbExists(key LBKey) bool {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	_, exists := pp.cache[key]
	return exists
}

func (pp *PortPool) AllocatePortFromRange(ctx context.Context, startPort, endPort, quota, segmentLength uint16, protocol string) ([]PortAllocation, bool) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if len(pp.cache) == 0 {
		return nil, true
	}
	portNum := 1
	if protocol == constant.ProtocolTCPUDP {
		portNum = 2
	}
	quotaExceeded := true
	for lb, allocated := range pp.getCache() {
		if uint16(len(allocated)+portNum) > quota { // 监听器数量已满，换下个 lb
			continue
		}
		quotaExceeded = false
		for port := startPort; port <= endPort; port += segmentLength { // 遍历该端口池的所有端口号
			endPort := uint16(0)
			if segmentLength > 1 {
				endPort = port + segmentLength - 1
			}
			if result := pp.tryAllocateFromLb(lb, allocated, port, endPort, protocol); len(result) > 0 {
				return result, false
			}
		}
	}
	return nil, quotaExceeded
}

// RequestScaleUp 请求扩容，使用 CAS 保证并发安全，只有第一个请求成功。
// 返回 true 表示本次请求设标记成功（应通知 CLBPortPool reconcile），false 表示已有其他请求在先。
func (pp *PortPool) RequestScaleUp() bool {
	return pp.scaleUpRequested.CompareAndSwap(false, true)
}

// HasScaleUpRequest 检查是否有扩容请求（只读，不重置标记）
func (pp *PortPool) HasScaleUpRequest() bool {
	return pp.scaleUpRequested.Load()
}

// ResetScaleUpRequest 重置扩容请求标记
func (pp *PortPool) ResetScaleUpRequest() {
	pp.scaleUpRequested.Store(false)
}

// 尝试从 lb 中分配端口
func (pp *PortPool) tryAllocateFromLb(lbKey LBKey, allocated map[ProtocolPort]struct{}, port, endPort uint16, protocol string) []PortAllocation {
	ports := portsToAllocate(port, endPort, protocol)
	// 确保所有待分配的端口都未被分配，也如果是预创建监听器，确保待分配的端口都在预创建端口范围内
	for _, port := range ports {
		if _, exists := allocated[port.Key()]; exists { // 有端口已被占用，标记无法分配
			return nil
		}
		if pp.maxPort != nil { // 启用了监听器预创建，确保待分配的端口都在预创建端口范围内，否则标记无法分配
			switch port.Protocol {
			case "TCP":
				if pp.maxPort.Tcp > 0 && port.Port > pp.maxPort.Tcp {
					return nil
				}
			case "UDP":
				if pp.maxPort.Udp > 0 && port.Port > pp.maxPort.Udp {
					return nil
				}
			default: // 不可能出现的协议
				panic(fmt.Sprintf("unknown protocol: %s", port.Protocol))
			}
		}
	}
	result := []PortAllocation{}
	for _, port := range ports {
		allocated[port.Key()] = struct{}{}
		pa := PortAllocation{
			PortPool:     pp,
			ProtocolPort: port,
			LBKey:        lbKey,
		}
		result = append(result, pa)
	}
	return result
}

// 分配指定端口
func (pp *PortPool) AllocatePort(ctx context.Context, quota int64, port, endPort uint16, protocol string) ([]PortAllocation, bool) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if len(pp.cache) == 0 {
		return nil, true
	}
	quotaExceeded := true
	portNum := 1
	if protocol == constant.ProtocolTCPUDP {
		portNum = 2
	}
	for lbKey, allocated := range pp.getCache() {
		if int64(len(allocated)+portNum) > quota { // 监听器数量已满，换下个 lb
			continue
		}
		quotaExceeded = false
		if result := pp.tryAllocateFromLb(lbKey, allocated, port, endPort, protocol); len(result) > 0 {
			return result, false
		}
	}
	// 所有 lb 都无法分配此端口，返回空结果
	return nil, quotaExceeded
}

func (pp *PortPool) RemoveLB(lbKey LBKey) bool {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	_, exists := pp.cache[lbKey]
	if !exists {
		return false
	}
	pp.removeLBUnlock(lbKey)
	return true
}

func (pp *PortPool) removeLBUnlock(lbKey LBKey) {
	delete(pp.cache, lbKey)
	pp.lbList = slices.DeleteFunc(pp.lbList, func(lb LBKey) bool {
		return lb == lbKey
	})
}

// 释放已分配的端口
func (pp *PortPool) ReleasePort(lbKey LBKey, port ProtocolPort) bool {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	cache, exists := pp.cache[lbKey]
	if !exists {
		return false
	}
	delete(cache, port.Key())
	return true
}

func (pp *PortPool) AllocatedPorts(lbKey LBKey) uint16 {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if allocated, exists := pp.cache[lbKey]; exists {
		return uint16(len(allocated))
	}
	return 0
}

func (pp *PortPool) EnsureLbIds(lbKeys []LBKey) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if reflect.DeepEqual(lbKeys, pp.lbList) {
		return
	}

	lbToDelete := make(map[LBKey]struct{})
	for lbKey := range pp.cache {
		lbToDelete[lbKey] = struct{}{}
	}
	lbToAdd := []LBKey{}
	for _, lbKey := range lbKeys {
		if _, exists := lbToDelete[lbKey]; exists {
			delete(lbToDelete, lbKey)
		} else {
			lbToAdd = append(lbToAdd, lbKey)
		}
	}
	for lbKey := range lbToDelete { // 删除多余的 lb
		ppLog.Info("remove lb", "lb", lbKey, "pool", pp.Name)
		pp.removeLBUnlock(lbKey)
	}
	// 添加缺失的lb
	if len(lbToAdd) > 0 {
		for _, lbKey := range lbToAdd {
			ppLog.Info("add lb", "lb", lbKey, "pool", pp.Name)
			pp.cache[lbKey] = make(map[ProtocolPort]struct{})
		}
		pp.lbList = lbKeys
	}
}
