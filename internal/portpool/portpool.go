package portpool

import (
	"context"
	"slices"
	"sync"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/constant"

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
	Name     string
	LbPolicy string
	mu       sync.Mutex
	cache    map[LBKey]map[ProtocolPort]struct{}
	lbList   []LBKey
}

func (pp *PortPool) IsLbExists(key LBKey) bool {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	_, exists := pp.cache[key]
	return exists
}

// 分配指定端口
func (pp *PortPool) AllocatePort(ctx context.Context, quota int64, ports ...ProtocolPort) ([]PortAllocation, bool) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if len(pp.cache) == 0 {
		return nil, true
	}
	quotaExceeded := true
	tryAllocate := func(lbKey LBKey, allocated map[ProtocolPort]struct{}) []PortAllocation {
		if int64(len(allocated)+len(ports)) > quota { // 监听器数量已满，换下个 lb
			return nil
		}
		quotaExceeded = false
		canAllocate := true
		for _, port := range ports { // 确保所有待分配的端口都未被分配
			if _, exists := allocated[port.Key()]; exists { // 有端口已被占用，标记无法分配
				canAllocate = false
				break
			}
		}
		if canAllocate { // 找到有 lb 可分配端口，分配端口并返回
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
		return nil
	}
	switch pp.LbPolicy {
	case constant.LbPolicyInOrder, constant.LbPolicyUniform: // 有序分配 或 均匀分配
		if pp.LbPolicy == constant.LbPolicyUniform { // 如果是均匀分配，则需要按已分配数量排序，找分配数量最小的 lb 分配
			slices.SortFunc(pp.lbList, func(a, b LBKey) int {
				return len(pp.cache[a]) - len(pp.cache[b])
			})
		}
		for _, lbKey := range pp.lbList {
			allocated := pp.cache[lbKey]
			result := tryAllocate(lbKey, allocated)
			if len(result) > 0 {
				return result, false
			}
		}
	default: // 默认用 Random，按 map 的 key 顺序遍历（golang map 的 key 是无序的，每次遍历顺序随机）
		for lbKey, allocated := range pp.cache {
			result := tryAllocate(lbKey, allocated)
			if len(result) > 0 {
				return result, false
			}
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
	for _, lbKey := range lbToAdd {
		ppLog.Info("add lb", "lb", lbKey, "pool", pp.Name)
		pp.cache[lbKey] = make(map[ProtocolPort]struct{})
		pp.lbList = append(pp.lbList, lbKey)
	}
}
