package util

import "strings"

type StatusOp int

const (
	StatusOpNone StatusOp = iota
	StatusOpUpdate
	StatusOpDelete
)

// IsIPv6LB 判断 CLB 是否为 IPv6 类型
// AddressIPVersion 可选值：IPV4、IPV6（NAT64）、IPv6FullChain（纯 IPv6）
// 当 AddressIPVersion 为 IPV6 或 IPv6FullChain 时，CLB 是 IPv6 类型
func IsIPv6LB(addressIPVersion *string) bool {
	if addressIPVersion == nil {
		return false
	}
	v := strings.ToLower(*addressIPVersion)
	return v == "ipv6" || v == "ipv6fullchain"
}
