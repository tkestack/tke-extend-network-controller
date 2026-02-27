package portpool

import (
	"sync"
	"testing"
)

func TestRequestScaleUp(t *testing.T) {
	t.Run("首次请求成功", func(t *testing.T) {
		pp := &PortPool{Name: "test-pool"}
		if !pp.RequestScaleUp() {
			t.Error("首次 RequestScaleUp() 应返回 true")
		}
	})

	t.Run("重复请求失败", func(t *testing.T) {
		pp := &PortPool{Name: "test-pool"}
		pp.RequestScaleUp()
		if pp.RequestScaleUp() {
			t.Error("重复 RequestScaleUp() 应返回 false")
		}
	})

	t.Run("重置后可以再次请求", func(t *testing.T) {
		pp := &PortPool{Name: "test-pool"}
		pp.RequestScaleUp()
		if !pp.ResetScaleUpRequest() {
			t.Error("ResetScaleUpRequest() 应返回 true（之前有请求）")
		}
		if pp.ResetScaleUpRequest() {
			t.Error("再次 ResetScaleUpRequest() 应返回 false（已重置）")
		}
		if !pp.RequestScaleUp() {
			t.Error("重置后 RequestScaleUp() 应返回 true")
		}
	})

	t.Run("并发请求只有一个成功", func(t *testing.T) {
		pp := &PortPool{Name: "test-pool"}
		goroutines := 1000
		successCount := 0
		var mu sync.Mutex
		var wg sync.WaitGroup
		wg.Add(goroutines)
		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				if pp.RequestScaleUp() {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if successCount != 1 {
			t.Errorf("并发 %d 个请求，应只有 1 个成功，实际 %d 个成功", goroutines, successCount)
		}
	})
}
