package cache

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// Concurrent calls with same input are deduplicated
func Test_RepeatedFetch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockFetcher(ctrl)
	fetcher.EXPECT().Fetch(0).Return("test", nil).Times(1)
	cache := New(fetcher, time.Millisecond*100, 1000)

	for range 10 {
		val, err := cache.Fetch(0)
		require.NoError(t, err)
		require.Equal(t, "test", val)
		time.Sleep(time.Millisecond * 5) // Simulate some delay
	}
}

// Return values are correctly cached
func Test_DifferentIdFetch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockFetcher(ctrl)
	for i := 0; i < 10; i++ {
		fetcher.EXPECT().Fetch(i).Return(fmt.Sprintf("test%d", i), nil).Times(1)
	}
	cache := New(fetcher, time.Millisecond*100, 1000)

	for i := range 10 {
		val, err := cache.Fetch(i)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("test%d", i), val)
		time.Sleep(time.Millisecond * 5) // Simulate some delay
	}
}

// Results expire after 5 minutes (TTL)
func Test_TTL_Check(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockFetcher(ctrl)
	fetcher.EXPECT().Fetch(0).Return("test", nil).Times(2)
	cache := New(fetcher, time.Millisecond*100, 1000)

	for range 10 {
		val, err := cache.Fetch(0)
		require.NoError(t, err)
		require.Equal(t, "test", val)
		time.Sleep(time.Millisecond * 5) // Simulate some delay
	}

	// Wait for TTL to expire
	time.Sleep(time.Millisecond * 150)

	// Fetch again, should trigger fetcher again
	val, err := cache.Fetch(0)
	require.NoError(t, err)
	require.Equal(t, "test", val)
}

// Concurrent calls with same input are deduplicated
func Test_Concurrent(t *testing.T) {
	n := 100
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var counter atomic.Int32
	fetcher := NewMockFetcher(ctrl)
	fetcher.EXPECT().Fetch(0).Do(func(x int) {
		counter.Add(1)
		//Check if there are more than 2 concurrent fetches
		require.Less(t, counter.Load(), int32(2))
		time.Sleep(time.Millisecond * 500)
		counter.Add(-1)
	}).Return("test", nil).Times(1)
	cache := New(fetcher, time.Millisecond*1000, 1000)

	done := make(chan struct{})
	for i := range n {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			val, err := cache.Fetch(0)
			require.NoError(t, err)
			require.Equal(t, "test", val)
		}(i)
	}

	for range n {
		<-done
	}
}

// Cache never exceeds 1000 entries
// Oldest entries are evicted when the cache is full
func Test_Capacity(t *testing.T) {
	cap := 1000
	doubleCall := 200
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockFetcher(ctrl)
	for i := range doubleCall {
		fetcher.EXPECT().Fetch(i).Return(fmt.Sprintf("test%d", i), nil).Times(1)
	}

	for i := doubleCall; i < cap+doubleCall; i++ {
		fetcher.EXPECT().Fetch(i).Return(fmt.Sprintf("test%d", i), nil).Times(1)
	}

	for i := range doubleCall {
		fetcher.EXPECT().Fetch(i).Return(fmt.Sprintf("test%d", i), nil).Times(1)
	}
	cache := New(fetcher, time.Millisecond*1000, cap)

	for i := 0; i < cap+doubleCall; i++ {
		val, err := cache.Fetch(i)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("test%d", i), val)
	}

	for i := range doubleCall {
		val, err := cache.Fetch(i)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("test%d", i), val)
	}
}

func Test_ErrorHandling(t *testing.T) {
	cap := 1000

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockFetcher(ctrl)
	for i := range cap {
		fetcher.EXPECT().Fetch(i).Return("", fmt.Errorf("error")).Times(1)
	}

	for i := range cap {
		fetcher.EXPECT().Fetch(i).Return(fmt.Sprintf("test%d", i), nil).Times(1)
	}

	cache := New(fetcher, time.Millisecond*1000, cap)

	for i := 0; i < cap; i++ {
		val, err := cache.Fetch(i)
		require.Error(t, err)
		require.Equal(t, "", val)
	}

	for i := 0; i < cap; i++ {
		val, err := cache.Fetch(i)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("test%d", i), val)
	}
}
