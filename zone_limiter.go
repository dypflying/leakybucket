package ratelimit

/*
Copyright (c) Yunpeng Deng(dypflying)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type zoneItem struct {
	limiterMeta
	limiterRecord
}

type zoneRateLimiter struct {
	limiterMeta
	zoneMap sync.Map
}

//NewZoneRateLimiter is the contructor for a zone rate limiter
func NewZoneRateLimiter(rate uint32) ZoneLimiter {
	z := &zoneRateLimiter{}
	z.resolution = ResolutionEnum.Millisecond
	z.rate = rate
	return z
}

func (z *zoneRateLimiter) SetRate(rate uint32) ZoneLimiter {
	if z != nil {
		z.rate = rate
	}
	return z
}

func (z *zoneRateLimiter) SetBurst(burst uint32) ZoneLimiter {
	if z != nil {
		z.burst = burst
	}
	return z
}

func (z *zoneRateLimiter) SetNodelay(nodelay bool) ZoneLimiter {
	if z != nil {
		z.nodelay = nodelay
	}
	return z
}

func (z *zoneRateLimiter) SetResolution(resolution Resolution) ZoneLimiter {
	if z != nil {
		z.resolution = resolution
	}
	return z
}

func (z *zoneRateLimiter) AddZoneItem(key interface{}) error {
	if z != nil && key != nil {
		if _, ok := z.zoneMap.Load(key); ok {
			return errors.New("key exists")
		}
		item := &zoneItem{}
		item.nodelay = z.nodelay
		item.rate = z.rate
		item.burst = z.burst
		atomic.StoreInt64(&item.excess, 0)
		atomic.StoreInt64(&item.last, 0)
		z.zoneMap.Store(key, item)
	}
	return nil
}

func (z *zoneRateLimiter) DeleteZoneItem(key interface{}) error {
	if z != nil && key != nil {
		if _, ok := z.zoneMap.Load(key); !ok {
			return errors.New("key not exists")
		}
		z.zoneMap.Delete(key)
	}
	return nil
}

func (z *zoneRateLimiter) SetZoneItem(key interface{}, rate uint32, burst uint32, nodelay bool) {
	if z != nil && key != nil {
		if v, ok := z.zoneMap.Load(key); ok {
			item := v.(*limiterMeta)
			item.burst = burst
			item.rate = rate
			item.nodelay = nodelay
		} else {
			item := &zoneItem{}
			item.nodelay = nodelay
			item.rate = rate
			item.burst = burst
			atomic.StoreInt64(&item.excess, 0)
			atomic.StoreInt64(&item.last, 0)
			z.zoneMap.Store(key, item)
		}
	}
}

func (z *zoneRateLimiter) Get(key interface{}) error {

	delay, err := z.GetDelayInMicroseconds(key)
	if err != nil {
		return err
	} else if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Microsecond)
	}
	return nil
}

//return:
//	#1. the delay time in microseconds
//	#2. error if rejected
func (z *zoneRateLimiter) GetDelayInMicroseconds(key interface{}) (int64, error) {

	if key == nil {
		//do nothing
		return 0, nil
	}

	var (
		excess, lastExcess int64
		item               *zoneItem
	)

	if v, ok := z.zoneMap.Load(key); ok {
		item = v.(*zoneItem)
	} else {
		return 0, nil
	}

	if item.rate <= 0 {
		return 0, errorReject
	}

	resolutionFactor := 1e9 / z.resolution
	rate := int64(item.rate) * resolutionFactor
	burst := int64(item.burst) * resolutionFactor

	for {
		//note: after golang 1.17, it introduced UnixMilli() and UnixMicro() functions,
		//but to support the golang before 1.17, we still use UnixNano to retrieve the timestamps.
		now := time.Now().UnixNano() / z.resolution

		elapsed := now - item.last
		//Note: the elapsed value may be huge since it is retrieved from the nanoseconds from 1970.1.1 for the first call of the object
		//here we set a quota for the elapsed with a maximum of 1 hour
		if elapsed > int64(time.Hour)/z.resolution {
			elapsed = int64(time.Hour) / z.resolution
		}

		lastExcess = atomic.LoadInt64(&item.excess)
		excess = lastExcess - rate/resolutionFactor*elapsed + resolutionFactor

		if excess < 0 {
			excess = 0
		}

		if excess > burst {
			return 0, errorReject
		}
		if atomic.CompareAndSwapInt64(&item.excess, lastExcess, excess) {
			item.last = now
			break
		}
	}
	if !item.nodelay {
		delayInSecond := float64(excess) / float64(rate)
		return int64(delayInSecond * 1e6), nil
	}
	return 0, nil
}
