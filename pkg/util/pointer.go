package util

import "reflect"

func GetPtr[T any](val T) *T {
	return &val
}

func IsValueEqual[T comparable](ptr *T, val T) bool {
	if reflect.ValueOf(ptr).IsNil() {
		return false
	}
	return *ptr == val
}

func GetValue[T any](ptr *T) T {
	if reflect.ValueOf(ptr).IsNil() {
		return *new(T)
	}
	return *ptr
}

// 如果指针为 nil，或者值为空，则设置指定值
func SetIfEmpty[T any](ptr *T, val T) {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Pointer {
		return
	}
	if v.IsNil() || v.Elem().IsZero() {
		return
	}
	*ptr = val
}

func IsZero(v any) bool {
	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Pointer {
		if vv.IsNil() {
			return true
		}
		vv = vv.Elem()
	}
	return vv.IsZero()
}
