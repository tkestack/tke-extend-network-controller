package clb

import "sync"

var (
	lock      sync.Mutex
	lbLockMap = make(map[string]*sync.Mutex)
)

func getLbLock(lbId string) *sync.Mutex {
	lock.Lock()
	defer lock.Unlock()
	mux, ok := lbLockMap[lbId]
	if !ok {
		mux = &sync.Mutex{}
		lbLockMap[lbId] = mux
	}
	return mux
}
