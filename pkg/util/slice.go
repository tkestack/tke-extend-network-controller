package util

func ConvertStringPointSlice(s []*string) []string {
	ss := make([]string, len(s))
	for i := range s {
		ss[i] = *s[i]
	}
	return ss
}
