package util

import "testing"

func TestZero(t *testing.T) {
	var vpc *string
	if !IsZero(vpc) {
		t.Error("vpc should be zero")
	}
}
