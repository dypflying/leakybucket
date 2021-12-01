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
	"sync/atomic"
	"time"
)

type rateLimiter struct {
	limiterMeta
	limiterRecord
}

//NewRateLimiter is the contructor for a rate limiter
func NewRateLimiter(rate uint32) Limiter {

	r := &rateLimiter{}
	r.rate = rate
	r.resolution = ResolutionEnum.Millisecond
	atomic.StoreInt64(&r.excess, 0)
	atomic.StoreInt64(&r.last, 0)
	return r
}

func (r *rateLimiter) SetRate(rate uint32) Limiter {
	if r != nil {
		r.rate = rate
	}
	return r
}

func (r *rateLimiter) SetBurst(burst uint32) Limiter {
	if r != nil {
		r.burst = burst
	}
	return r
}

func (r *rateLimiter) SetNodelay(nodelay bool) Limiter {
	if r != nil {
		r.nodelay = nodelay
	}
	return r
}

func (r *rateLimiter) SetResolution(resolution Resolution) Limiter {
	if r != nil {
		r.resolution = resolution
	}
	return r
}

func (r *rateLimiter) Get() error {

	delay, err := r.GetDelayInMicroseconds()
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
func (r *rateLimiter) GetDelayInMicroseconds() (int64, error) {
	var (
		excess, lastExcess int64
	)
	if r.rate == 0 {
		return 0, errorReject
	}
	resolutionFactor := 1e9 / r.resolution
	rate := int64(r.rate) * resolutionFactor
	burst := int64(r.burst) * resolutionFactor

	for {
		//note: after golang 1.17, it introduced UnixMilli() and UnixMicro() functions,
		//but to support the golang before 1.17, we still use UnixNano to retrieve the timestamps.
		now := time.Now().UnixNano() / r.resolution

		elapsed := now - r.last
		//Note: the elapsed value may be huge since it is retrieved from the nanoseconds from 1970.1.1 for the first call of the object
		//here we set a quota for the elapsed with a maximum of 1 hour
		if elapsed > int64(time.Hour)/r.resolution {
			elapsed = int64(time.Hour) / r.resolution
		}

		lastExcess = atomic.LoadInt64(&r.excess)
		excess = lastExcess - rate/resolutionFactor*elapsed + resolutionFactor

		if excess < 0 {
			excess = 0
		}

		if excess > burst {
			return 0, errorReject
		}
		if atomic.CompareAndSwapInt64(&r.excess, lastExcess, excess) {
			r.last = now
			break
		}
	}
	if !r.nodelay {
		delayInSecond := float64(excess) / float64(rate)
		return int64(delayInSecond * 1e6), nil
	}

	return 0, nil
}
