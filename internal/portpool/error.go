package portpool

import "errors"

var (
	ErrPoolNotFound          = errors.New("port pool not found")
	ErrNoPortAvailable       = errors.New("no available port in pool")
	ErrPortNotAllocated      = errors.New("port not allocated")
	ErrPortAllocated         = errors.New("port allocated")
	ErrSegmentLengthNotEqual = errors.New("segment length is not equal across all port pools")
	ErrLBCreated             = errors.New("new clb created")
	ErrWaitLBScale           = errors.New("waiting for clb scale")
	ErrUnknown               = errors.New("unknown error")
	ErrListenerQuotaExceeded = errors.New("listener quota exceeded")
)
