package portpool

import "errors"

var (
	ErrPoolNotFound          = errors.New("port pool not found")
	ErrNoPortAvailable       = errors.New("no available port in pool")
	ErrSegmentLengthNotEqual = errors.New("segment length is not equal across all port pools")
	ErrWaitLBScale           = errors.New("waiting for clb scale")
	ErrUnknown               = errors.New("unknown error")
	ErrNoFreeLb              = errors.New("no free lb available")
)
