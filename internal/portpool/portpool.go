package portpool

import (
	"context"
	"sync"

	"github.com/imroc/tke-extend-network-controller/pkg/clb"
	"github.com/pkg/errors"
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

type PortAllocation struct {
	ProtocolPort
	*PortPool
	LbId string
}

func (pa PortAllocation) Release() {
	pa.ReleasePort(pa.LbId, pa.ProtocolPort)
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
	CreateLbResultCreating
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

// PortPool 管理单个端口池的状态
type PortPool struct {
	CLBPortPool
	mu    sync.Mutex
	cache map[string]map[ProtocolPort]struct{}
}

func (pp *PortPool) IsLbExists(lbId string) bool {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	_, exists := pp.cache[lbId]
	return exists
}

// 分配指定端口
func (pp *PortPool) AllocatePort(ctx context.Context, quota int64, ports ...ProtocolPort) ([]PortAllocation, error) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if len(pp.cache) == 0 {
		return nil, ErrNoLbReady
	}
	// 获取监听器数量配额
	if quota == 0 {
		q, err := clb.Quota.GetQuota(ctx, pp.GetRegion(), clb.TOTAL_LISTENER_QUOTA)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		quota = q
	}
	for lbId, allocated := range pp.cache { // 遍历所有 lb，尝试分配端口
		if int64(len(allocated)+len(ports)) > quota { // 监听器数量已满，换下个 lb
			continue
		}
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
					LbId:         lbId,
				}
				result = append(result, pa)
			}
			return result, nil
		}
	}
	// 所有 lb 都无法分配此端口，返回空结果
	return nil, nil
}

func (pp *PortPool) IsAllocated(lbId string, port ProtocolPort) bool {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	cache := pp.cache[lbId]
	if cache == nil {
		return false
	}
	_, ok := cache[port.Key()]
	return ok
}

// 释放已分配的端口
func (pp *PortPool) ReleasePort(lbId string, port ProtocolPort) bool {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	cache := pp.cache[lbId]
	if cache == nil {
		return false
	}
	delete(cache, port.Key())
	return true
}

func (pp *PortPool) ReleaseLb(lbId string) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	delete(pp.cache, lbId)
}

func (pp *PortPool) AddLbId(lbId string) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if _, exists := pp.cache[lbId]; exists {
		return
	}
	pp.cache[lbId] = make(map[ProtocolPort]struct{})
}

func (pp *PortPool) EnsureLbIds(lbIds []string) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	lbMap := make(map[string]struct{})
	for lbId := range pp.cache {
		lbMap[lbId] = struct{}{}
	}
	lbToAdd := []string{}
	for _, lbId := range lbIds {
		if _, exists := lbMap[lbId]; exists {
			delete(lbMap, lbId)
		} else {
			lbToAdd = append(lbToAdd, lbId)
		}
	}
	for lbId := range lbMap { // 删除多余的 lb
		ppLog.Info("remove unused lbId", "lbId", lbId)
		delete(pp.cache, lbId)
	}
	// 添加缺失的lb
	for _, lbId := range lbToAdd {
		ppLog.Info("add lbId to PortPool", "lbId", lbId, "pool", pp.GetName())
		pp.cache[lbId] = make(map[ProtocolPort]struct{})
	}
}
