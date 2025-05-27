package portpool

import (
	"errors"
	"fmt"
)

type ErrPoolNotFound struct {
	Pool string
}

func (e *ErrPoolNotFound) Error() string {
	return fmt.Sprintf("port pool %q not found", e.Pool)
}

var (
	ErrNoPortAvailable       = errors.New("no available port in pool")
	ErrSegmentLengthNotEqual = errors.New("segment length is not equal across all port pools")
	ErrNewLBCreated          = errors.New("new lb created")
	ErrNewLBCreating         = errors.New("new lb is creating")
	ErrUnknown               = errors.New("unknown error")
	ErrNoFreeLb              = errors.New("no free lb available")
	ErrNoLbReady             = errors.New("no lb ready")
)
