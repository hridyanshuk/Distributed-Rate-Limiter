package limiter

import (
	"strconv"
	"sync"
	"testing"
)

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewStore()
	var wg sync.WaitGroup

	// 100 goroutines requesting buckets
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				key := "client_" + strconv.Itoa(j)
				b := store.GetOrCreate(key, 10, 10)
				if b == nil {
					t.Errorf("expected bucket for key %s", key)
				}
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkStore_GetOrCreate(b *testing.B) {
	store := NewStore()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			store.GetOrCreate("test_client", 100, 100)
		}
	})
}
