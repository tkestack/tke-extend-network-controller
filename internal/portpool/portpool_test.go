package portpool

import (
	"testing"

	"github.com/tkestack/tke-extend-network-controller/internal/constant"
)

func TestCanAllocate(t *testing.T) {
	lb1 := LBKey{LbId: "lb-1", Region: "ap-guangzhou"}
	lb2 := LBKey{LbId: "lb-2", Region: "ap-guangzhou"}

	tests := []struct {
		name           string
		lbKeys         []LBKey
		allocated      map[LBKey]map[ProtocolPort]struct{} // 预设已分配的端口
		startPort      uint16
		endPort        uint16
		quota          uint16
		segmentLength  uint16
		expectedResult bool
	}{
		{
			name:           "空端口池，无法分配",
			lbKeys:         []LBKey{},
			allocated:      map[LBKey]map[ProtocolPort]struct{}{},
			startPort:      500,
			endPort:        510,
			quota:          50,
			segmentLength:  1,
			expectedResult: false,
		},
		{
			name:   "有空闲端口，可以分配",
			lbKeys: []LBKey{lb1},
			allocated: map[LBKey]map[ProtocolPort]struct{}{
				lb1: {},
			},
			startPort:      500,
			endPort:        510,
			quota:          50,
			segmentLength:  1,
			expectedResult: true,
		},
		{
			name:   "端口范围小于 quota，TCP 全部用完但 UDP 空闲，TCPUDP 无法分配（需要同端口号 TCP+UDP 都空闲）",
			lbKeys: []LBKey{lb1},
			allocated: map[LBKey]map[ProtocolPort]struct{}{
				lb1: allocateTCPPorts(500, 502),
			},
			startPort:      500,
			endPort:        502,
			quota:          50,
			segmentLength:  1,
			expectedResult: false,
		},
		{
			name:   "端口范围小于 quota，部分 TCP 被占用，剩余端口号 TCPUDP 仍可分配",
			lbKeys: []LBKey{lb1},
			allocated: map[LBKey]map[ProtocolPort]struct{}{
				lb1: allocateTCPPorts(500, 501), // 只占了 500、501 的 TCP
			},
			startPort:      500,
			endPort:        502,
			quota:          50,
			segmentLength:  1,
			expectedResult: true, // 502 的 TCP 和 UDP 都空闲
		},
		{
			name:   "端口范围小于 quota，TCP 和 UDP 全部用完，无法分配",
			lbKeys: []LBKey{lb1},
			allocated: map[LBKey]map[ProtocolPort]struct{}{
				lb1: allocateTCPUDPPorts(500, 502),
			},
			startPort:      500,
			endPort:        502,
			quota:          50,
			segmentLength:  1,
			expectedResult: false, // 所有端口号的 TCP+UDP 都被占满
		},
		{
			name:   "quota 已满，无法分配",
			lbKeys: []LBKey{lb1},
			allocated: map[LBKey]map[ProtocolPort]struct{}{
				lb1: allocateTCPUDPPorts(500, 524), // 25 个端口 × 2 协议 = 50 个监听器
			},
			startPort:      500,
			endPort:        600,
			quota:          50,
			segmentLength:  1,
			expectedResult: false,
		},
		{
			name:   "多个 LB，第一个满了但第二个有空闲",
			lbKeys: []LBKey{lb1, lb2},
			allocated: map[LBKey]map[ProtocolPort]struct{}{
				lb1: allocateTCPUDPPorts(500, 502),
				lb2: {},
			},
			startPort:      500,
			endPort:        502,
			quota:          50,
			segmentLength:  1,
			expectedResult: true,
		},
		{
			name:   "多个 LB 全部端口用完，无法分配",
			lbKeys: []LBKey{lb1, lb2},
			allocated: map[LBKey]map[ProtocolPort]struct{}{
				lb1: allocateTCPUDPPorts(500, 502),
				lb2: allocateTCPUDPPorts(500, 502),
			},
			startPort:      500,
			endPort:        502,
			quota:          50,
			segmentLength:  1,
			expectedResult: false,
		},
		{
			name:   "端口段模式，有空闲段",
			lbKeys: []LBKey{lb1},
			allocated: map[LBKey]map[ProtocolPort]struct{}{
				lb1: allocateTCPUDPPortSegments(500, 504), // 占了 500-504
			},
			startPort:      500,
			endPort:        509,
			quota:          50,
			segmentLength:  5,
			expectedResult: true, // 505-509 还空闲
		},
		{
			name:   "端口段模式，所有段都已用完",
			lbKeys: []LBKey{lb1},
			allocated: map[LBKey]map[ProtocolPort]struct{}{
				lb1: mergeMaps(
					allocateTCPUDPPortSegments(500, 504),
					allocateTCPUDPPortSegments(505, 509),
				),
			},
			startPort:      500,
			endPort:        509,
			quota:          50,
			segmentLength:  5,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := &PortPool{
				Name:        "test-pool",
				LbPolicy:    constant.LbPolicyInOrder,
				LbBlacklist: make(map[LBKey]struct{}),
				cache:       tt.allocated,
				lbList:      tt.lbKeys,
			}
			result := pp.CanAllocate(tt.startPort, tt.endPort, tt.quota, tt.segmentLength)
			if result != tt.expectedResult {
				t.Errorf("CanAllocate() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

// allocateTCPPorts 创建指定范围内的 TCP 已分配端口
func allocateTCPPorts(start, end uint16) map[ProtocolPort]struct{} {
	m := make(map[ProtocolPort]struct{})
	for p := start; p <= end; p++ {
		m[ProtocolPort{Port: p, Protocol: "TCP"}] = struct{}{}
	}
	return m
}

// allocateTCPUDPPorts 创建指定范围内的 TCP+UDP 已分配端口
func allocateTCPUDPPorts(start, end uint16) map[ProtocolPort]struct{} {
	m := make(map[ProtocolPort]struct{})
	for p := start; p <= end; p++ {
		m[ProtocolPort{Port: p, Protocol: "TCP"}] = struct{}{}
		m[ProtocolPort{Port: p, Protocol: "UDP"}] = struct{}{}
	}
	return m
}

// allocateTCPUDPPortSegments 创建端口段的 TCP+UDP 已分配端口
func allocateTCPUDPPortSegments(start, end uint16) map[ProtocolPort]struct{} {
	m := make(map[ProtocolPort]struct{})
	m[ProtocolPort{Port: start, EndPort: end, Protocol: "TCP"}] = struct{}{}
	m[ProtocolPort{Port: start, EndPort: end, Protocol: "UDP"}] = struct{}{}
	return m
}

// mergeMaps 合并多个 map
func mergeMaps(maps ...map[ProtocolPort]struct{}) map[ProtocolPort]struct{} {
	result := make(map[ProtocolPort]struct{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
