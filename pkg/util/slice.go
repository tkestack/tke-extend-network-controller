package util

func ConvertPtrSlice[T any](ptrSlice []*T) []T {
	valueSlice := make([]T, len(ptrSlice))
	for i, ptr := range ptrSlice {
		if ptr != nil {
			valueSlice[i] = *ptr
		} else {
			var zero T // 自动填充类型的零值
			valueSlice[i] = zero
		}
	}
	return valueSlice
}
