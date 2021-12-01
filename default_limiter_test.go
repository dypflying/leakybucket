package ratelimit

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func runWithTime(meta limiterMeta, routineNum, repeatNum int, millisecond int64, errRate float32) error {

	maxTick := time.NewTicker(time.Duration(float32(millisecond)*(1.0+errRate)) * time.Millisecond)
	minDuration := time.Now().Add(time.Duration(float32(millisecond)*(1.0-errRate)) * time.Millisecond)
	finishCh := make(chan interface{}, 1)
	go func() {
		goRateTest(meta, routineNum, repeatNum)
		finishCh <- struct{}{}
	}()

	select {
	case <-maxTick.C:
		return errors.New("Time exceeded")
	case <-finishCh: //which means it return
		if time.Now().Before(minDuration) {
			return errors.New("Early finished")
		}
		return nil
	}
}

func goRateTest(meta limiterMeta, routineNum, repeatNum int) uint64 {

	rl := NewRateLimiter(meta.rate).SetBurst(meta.burst).SetNodelay(meta.nodelay).SetResolution(meta.resolution)
	wg := sync.WaitGroup{}
	wg.Add(routineNum)

	var errCount uint64
	atomic.StoreUint64(&errCount, 0)
	for i := 0; i < routineNum; i++ {
		go func() {
			for i := 0; i < repeatNum; i++ {
				err := rl.Get()
				if err != nil {
					atomic.AddUint64(&errCount, 1)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return errCount
}

//rate limit to 1000 req/s, burst is 10, nodelay to false, microsecond resolution
//10,000 reqs in 10 routines, supposed to be finished in exact 10 seconds, acceptable error ratio is 1%.
func TestNormal1(t *testing.T) {
	t.Parallel()
	err := runWithTime(limiterMeta{
		rate:       1000,
		burst:      10,
		nodelay:    false,
		resolution: ResolutionEnum.Microsecond,
	}, 10, 1000, 10000.0, 0.01)

	if err != nil {
		t.Errorf("Finished unexpectedly: %v", err)
	}
}

//rate limit to 1000 req/s, burst is 100, nodelay to false, microsecond resolution
//10,000 reqs in 100 routines, supposed to be finished in exact 10 seconds, acceptable error ratio is 1%.
func TestNormal2(t *testing.T) {
	t.Parallel()
	err := runWithTime(limiterMeta{
		rate:       1000,
		burst:      100,
		nodelay:    false,
		resolution: ResolutionEnum.Microsecond,
	}, 100, 100, 10000.0, 0.01)

	if err != nil {
		t.Errorf("Finished unexpectedly: %v", err)
	}
}

//rate limit to 100 req/s, burst is 10, nodelay to false, millisecond resolution
//1,000 reqs in 2 routines, supposed to be finished in exact 10 seconds, acceptable error ratio is 1%.
func TestNormal3(t *testing.T) {
	t.Parallel()
	err := runWithTime(limiterMeta{
		rate:       100,
		burst:      10,
		nodelay:    false,
		resolution: ResolutionEnum.Millisecond,
	}, 2, 500, 10000.0, 0.01)

	if err != nil {
		t.Errorf("Finished unexpectedly: %v", err)
	}
}

//rate limit to 10,000 req/s, burst is 100, nodelay to false, microsecond resolution
//100,000 reqs in 10 routines, supposed to be finished in exact 10 seconds, acceptable error ratio is 1%.
func TestNormal4(t *testing.T) {
	t.Parallel()
	err := runWithTime(limiterMeta{
		rate:       10000,
		burst:      100,
		nodelay:    false,
		resolution: ResolutionEnum.Microsecond,
	}, 10, 10000, 10000.0, 0.01)

	if err != nil {
		t.Errorf("Finished unexpectedly: %v", err)
	}
}

//if set the rate to 0
//it is expected that all request will be rejected
func TestZeroRate(t *testing.T) {
	t.Parallel()
	errCount := goRateTest(limiterMeta{
		rate:       0,
		burst:      10,
		nodelay:    false,
		resolution: ResolutionEnum.Microsecond,
	}, 10, 1000)
	if errCount != 10*1000 {
		t.Errorf("Error count is not expected")
	}
}
