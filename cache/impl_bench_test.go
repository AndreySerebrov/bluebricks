package cache

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
)

func BenchmarkRepeatedFetch(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	fetcher := NewMockFetcher(ctrl)
	fetcher.EXPECT().Fetch(0).Return("test", nil).Times(1)
	cache := New(fetcher, time.Millisecond*100, 1000)

	for b.Loop() {
		val, err := cache.Fetch(0)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if val != "test" {
			b.Fatalf("expected 'test', got '%s'", val)
		}
	}
}

// Performance under high concurrency
func BenchmarkHighConcurrencyFetch(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	fetcher := NewMockFetcher(ctrl)
	var counter int32
	fetcher.EXPECT().Fetch(0).Do(func(x int) {
		counter++
		time.Sleep(time.Millisecond * 50) // Simulate some delay
		counter--
	}).Return("test", nil).Times(1)
	cache := New(fetcher, time.Millisecond*1000, 1000)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			val, err := cache.Fetch(0)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if val != "test" {
				b.Fatalf("expected 'test', got '%s'", val)
			}
		}
	})
}

// Cached execution (cold/warm)
func BenchmarkCachedExecution(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	fetcher := NewMockFetcher(ctrl)
	fetcher.EXPECT().Fetch(0).Return("test", nil).Times(1)
	cache := New(fetcher, time.Millisecond*100, 1000)

	// Warm up the cache
	for i := 0; i < 10; i++ {
		val, err := cache.Fetch(0)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if val != "test" {
			b.Fatalf("expected 'test', got '%s'", val)
		}
	}

	b.ResetTimer()

	for b.Loop() {
		val, err := cache.Fetch(0)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if val != "test" {
			b.Fatalf("expected 'test', got '%s'", val)
		}
	}
}
