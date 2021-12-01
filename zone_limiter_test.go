package ratelimit

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	defaultKey    = "test.com"
	customizedKey = 2022
	noExistKey    = "none exist"
)

func runZoneWithTime(limiter ZoneLimiter, routineNum, repeatNum int, millisecond int64, errRate float32, requestKey interface{}) error {

	maxTick := time.NewTicker(time.Duration(float32(millisecond)*(1.0+errRate)) * time.Millisecond)
	minDuration := time.Now().Add(time.Duration(float32(millisecond)*(1.0-errRate)) * time.Millisecond)
	finishCh := make(chan interface{}, 1)
	go func() {
		goZoneRateTest(limiter, routineNum, repeatNum, requestKey)
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

func goZoneRateTest(limiter ZoneLimiter, routineNum, repeatNum int, requestKey interface{}) uint64 {

	wg := sync.WaitGroup{}
	wg.Add(routineNum)

	var errCount uint64
	atomic.StoreUint64(&errCount, 0)
	for i := 0; i < routineNum; i++ {
		go func() {
			for i := 0; i < repeatNum; i++ {
				err := limiter.Get(requestKey)
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

//the zone's rate limit to 1000 req/s, burst is 10, nodelay to false, microsecond resolution
//10,000 reqs in 10 routines, supposed to be finished in exact 10 seconds, acceptable error ratio is 1%.
func TestZone1(t *testing.T) {
	rl := NewZoneRateLimiter(1000).SetBurst(10).SetResolution(ResolutionEnum.Microsecond)
	rl.AddZoneItem(defaultKey)

	t.Parallel()
	err := runZoneWithTime(rl, 10, 1000, 10000.0, 0.01, defaultKey)
	if err != nil {
		t.Errorf("Finished unexpectedly: %v", err)
	}
}

//the zone's rate limit to 1000 req/s, burst is 100, nodelay to false, microsecond resolution
//10,000 reqs in 100 routines, supposed to be finished in exact 10 seconds, acceptable error ratio is 1%.
func TestZone2(t *testing.T) {

	rl := NewZoneRateLimiter(1000).SetBurst(100).SetResolution(ResolutionEnum.Microsecond)
	rl.AddZoneItem(defaultKey)

	t.Parallel()
	err := runZoneWithTime(rl, 100, 100, 10000.0, 0.01, defaultKey)

	if err != nil {
		t.Errorf("Finished unexpectedly: %v", err)
	}
}

//the zone's rate limit to 100 req/s, burst is 10, nodelay to false, millisecond resolution
//1,000 reqs in 2 routines, supposed to be finished in exact 10 seconds, acceptable error ratio is 1%.
func TestZone3(t *testing.T) {

	rl := NewZoneRateLimiter(100).SetBurst(10).SetResolution(ResolutionEnum.Millisecond)
	rl.AddZoneItem(defaultKey)

	t.Parallel()
	err := runZoneWithTime(rl, 2, 500, 10000.0, 0.01, defaultKey)

	if err != nil {
		t.Errorf("Finished unexpectedly: %v", err)
	}
}

//the zone's rate limit to 10,000 req/s, burst is 100, nodelay to false, microsecond resolution
//100,000 reqs in 10 routines, supposed to be finished in exact 10 seconds, acceptable error ratio is 1%.
func TestZone4(t *testing.T) {

	rl := NewZoneRateLimiter(10000).SetBurst(100).SetResolution(ResolutionEnum.Microsecond)
	rl.AddZoneItem(defaultKey)
	t.Parallel()
	err := runZoneWithTime(rl, 10, 10000, 10000.0, 0.01, defaultKey)

	if err != nil {
		t.Errorf("Finished unexpectedly: %v", err)
	}
}

//test a none-exist key in the rate limit zone
//that no limit for the key is expected
func TestNolimit(t *testing.T) {

	rl := NewZoneRateLimiter(10000).SetBurst(100).SetResolution(ResolutionEnum.Microsecond)
	rl.AddZoneItem(defaultKey)

	t.Parallel()
	errCount := goZoneRateTest(rl, 10, 10000, noExistKey)

	if errCount != 0 {
		t.Errorf("Finished unexpectedly, there should be no limit for key: %v", noExistKey)
	}
}

//the zone's rate limit to 10,000 req/s, burst is 100, nodelay to false, microsecond resolution
//for the key customizedKey, its overwrited rate is set to 100, burst is 10, nodelay to false,
//1,000 reqs in 10 routines, supposed to be finished in exact 10 seconds, acceptable error ratio is 1%.
func TestCustomerized(t *testing.T) {

	rl := NewZoneRateLimiter(10000).SetBurst(100).SetResolution(ResolutionEnum.Microsecond)
	rl.AddZoneItem(defaultKey)
	rl.SetZoneItem(customizedKey, 100, 10, false)
	t.Parallel()

	err := runZoneWithTime(rl, 10, 100, 10000.0, 0.01, customizedKey)

	if err != nil {
		t.Errorf("Finished unexpectedly: %v", err)
	}
}
