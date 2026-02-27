package portpool

import (
	"sync"
	"testing"
	"time"
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
		if !pp.HasScaleUpRequest() {
			t.Error("HasScaleUpRequest() 应返回 true（有请求）")
		}
		pp.ResetScaleUpRequest()
		if pp.HasScaleUpRequest() {
			t.Error("重置后 HasScaleUpRequest() 应返回 false")
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

	t.Run("冷却期内请求被拒绝", func(t *testing.T) {
		pp := &PortPool{Name: "test-pool"}
		pp.SetScaleUpCooldown(1 * time.Second)
		if pp.RequestScaleUp() {
			t.Error("冷却期内 RequestScaleUp() 应返回 false")
		}
		time.Sleep(1100 * time.Millisecond) // 等待冷却期过
		if !pp.RequestScaleUp() {
			t.Error("冷却期过后 RequestScaleUp() 应返回 true")
		}
	})

	t.Run("冷却期内HasScaleUpRequest也返回false", func(t *testing.T) {
		pp := &PortPool{Name: "test-pool"}
		pp.scaleUpRequested.Store(true) // 直接设置标记
		pp.SetScaleUpCooldown(1 * time.Second)
		if pp.HasScaleUpRequest() {
			t.Error("冷却期内 HasScaleUpRequest() 应返回 false")
		}
		time.Sleep(1100 * time.Millisecond) // 等待冷却期过
		if !pp.HasScaleUpRequest() {
			t.Error("冷却期过后 HasScaleUpRequest() 应返回 true")
		}
	})
}
