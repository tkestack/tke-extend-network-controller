package clb

import "sync"

var (
	lock      sync.Mutex
	lbLockMap = make(map[string]*sync.Mutex)
)

func getLbLock(lbId string) *sync.Mutex {
	lock.Lock()
	defer lock.Unlock()
	mu, ok := lbLockMap[lbId]
	if !ok {
		mu = &sync.Mutex{}
		lbLockMap[lbId] = mu
	}
	return mu
}
