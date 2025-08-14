package clb

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

var cacheLock sync.Mutex

var cache map[LBKey]*ListenerCache = make(map[LBKey]*ListenerCache)

func GetListenerCache(lbKey LBKey) *ListenerCache {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	lisCache, ok := cache[lbKey]
	if !ok {
		lisCache = NewListenerCache(lbKey.LbId, lbKey.Region)
		cache[lbKey] = lisCache
	}
	return lisCache
}

func GetListener(ctx context.Context, lbId, region string, port uint16, protocol string) (*Listener, error) {
	lbKey := LBKey{
		LbId:   lbId,
		Region: region,
	}
	lisCache := GetListenerCache(lbKey)
	lis, err := lisCache.Get(ctx, port, protocol)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return lis, nil
}

type ListenerCache struct {
	mux         sync.Mutex
	initialized bool
	LbId        string
	Region      string
	Listeners   map[PortKey]*Listener
}

type PortKey struct {
	Port     uint16
	Protocol string
}

type LBKey struct {
	LbId   string
	Region string
}

func NewListenerCache(lbId, region string) *ListenerCache {
	return &ListenerCache{
		LbId:      lbId,
		Region:    region,
		Listeners: make(map[PortKey]*Listener),
	}
}

func (c *ListenerCache) Set(lis *Listener) {
	portKey := PortKey{
		Port:     uint16(lis.Port),
		Protocol: lis.Protocol,
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	c.Listeners[portKey] = lis
}

func (c *ListenerCache) Get(ctx context.Context, port uint16, protocol string) (*Listener, error) {
	portKey := PortKey{
		Port:     port,
		Protocol: protocol,
	}
	c.mux.Lock()
	lis, ok := c.Listeners[portKey]
	c.mux.Unlock()
	if ok {
		return lis, nil
	}
	// 本地缓存中没有，尝试调 API 获取
	lis, err := GetListenerByPort(ctx, c.Region, c.LbId, int64(port), protocol)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	c.mux.Lock()
	c.Listeners[portKey] = lis
	c.mux.Unlock()
	return lis, nil
}

func (c *ListenerCache) EnsureInit(ctx context.Context) error {
	if c.initialized {
		return nil
	}
	allLis, err := GetAllListeners(ctx, c.Region, c.LbId)
	if err != nil {
		return errors.WithStack(err)
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.initialized {
		return nil
	}
	for _, lis := range allLis {
		c.Listeners[PortKey{Port: uint16(lis.Port), Protocol: lis.Protocol}] = lis
	}
	c.initialized = true
	return nil
}
