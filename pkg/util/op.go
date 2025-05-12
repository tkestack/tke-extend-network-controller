package util

type StatusOp int

const (
	StatusOpNone StatusOp = iota
	StatusOpUpdate
	StatusOpDelete
)
