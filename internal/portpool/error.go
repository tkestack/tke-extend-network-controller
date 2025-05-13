package portpool

import "errors"

var (
	ErrPoolNotFound          = errors.New("port pool not found")
	ErrNoPortAvailable       = errors.New("no available port in pool")
	ErrSegmentLengthNotEqual = errors.New("segment length is not equal across all port pools")
	ErrNewLBCreated          = errors.New("new lb created")
	ErrNewLBCreating         = errors.New("new lb is creating")
	ErrUnknown               = errors.New("unknown error")
	ErrNoFreeLb              = errors.New("no free lb available")
)
