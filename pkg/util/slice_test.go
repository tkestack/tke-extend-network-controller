package util

import (
	"reflect"
	"testing"
)

func TestConvertPtrSlice(t *testing.T) {
	a := "123"
	b := "456"
	ptrSlice := []*string{&a, &b}
	got := ConvertPtrSlice(ptrSlice)
	expected := []string{"123", "456"}
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("ConvertPtrSlice failed, expected:%v, got:%v", expected, got)
	}
}
