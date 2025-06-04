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
)
