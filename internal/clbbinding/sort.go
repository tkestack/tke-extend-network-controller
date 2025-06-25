package clbbinding

import (
	"slices"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
)

func SortPortBindings(bindings []networkingv1alpha1.PortBindingStatus) {
	slices.SortFunc(bindings, func(a, b networkingv1alpha1.PortBindingStatus) int {
		// 端口池
		if a.Pool > b.Pool {
			return 1
		} else if a.Pool < b.Pool {
			return -1
		}

		// lbId 排序
		if a.LoadbalancerId > b.LoadbalancerId {
			return 1
		} else if a.LoadbalancerId < b.LoadbalancerId {
			return -1
		}

		// lb 端口
		if a.LoadbalancerPort > b.LoadbalancerPort {
			return 1
		} else if a.LoadbalancerPort < b.LoadbalancerPort {
			return -1
		}

		// 容器端口
		if a.Port > b.Port {
			return 1
		} else if a.Port < b.Port {
			return -1
		}

		// 协议
		if a.Protocol > b.Protocol {
			return 1
		} else if a.Protocol < b.Protocol {
			return -1
		}

		// 监听器
		if a.ListenerId > b.ListenerId {
			return 1
		} else if a.ListenerId < b.ListenerId {
			return -1
		}

		// 证书
		if a.CertId != nil && b.CertId != nil {
			if *a.CertId > *b.CertId {
				return 1
			} else if *a.CertId < *b.CertId {
				return -1
			}
		}

		// endPort
		if a.LoadbalancerEndPort != nil && b.LoadbalancerEndPort != nil {
			if *a.LoadbalancerEndPort > *b.LoadbalancerEndPort {
				return 1
			} else if *a.LoadbalancerEndPort < *b.LoadbalancerEndPort {
				return -1
			}
		}

		// region
		if a.Region > b.Region {
			return 1
		} else if a.Region < b.Region {
			return -1
		}
		return 0
	})
}
