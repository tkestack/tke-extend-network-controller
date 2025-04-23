package util

import (
	"os"
	"strconv"
)

func GetWorkerCount(name string) int {
	v := os.Getenv(name)
	if v == "" {
		return 1
	}
	count, err := strconv.Atoi(v)
	if err != nil {
		panic(err)
	}
	return count
}
