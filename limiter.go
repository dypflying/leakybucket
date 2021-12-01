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
	"time"
)

//Resolution defines the resolution precision of a rate limiter
type Resolution = int64

type resolutionDefs struct {
	Microsecond     Resolution
	MicrosecondX10  Resolution
	MicrosecondX100 Resolution
	Millisecond     Resolution
}

//ResolutionEnum defines the 4 levels of precision that can be used for rate limiting
var ResolutionEnum = &resolutionDefs{
	Microsecond:     Resolution(time.Microsecond),
	MicrosecondX10:  Resolution(10 * time.Microsecond),
	MicrosecondX100: Resolution(100 * time.Microsecond),
	Millisecond:     Resolution(time.Millisecond), //the default
}

var (
	errorReject = errors.New("rejected")
)

//Limiter defines a default rate limiter
type Limiter interface {
	//return the delay time in micro seconds, and the error if rejected
	GetDelayInMicroseconds() (int64, error)
	//this will block the caller routine to a delay time if throtted, return error if it is rejected.
	Get() error
	SetRate(rate uint32) Limiter
	SetBurst(burst uint32) Limiter
	SetNodelay(nodelay bool) Limiter
	SetResolution(resolution Resolution) Limiter
}

//ZoneLimiter defines a rate limiter which can be used for specific keys
type ZoneLimiter interface {
	//throttle with a specific key
	//return the delay time in micro seconds, and the error if rejected
	GetDelayInMicroseconds(key interface{}) (int64, error)
	//throttle with a specific key
	//this will block the caller routine to a delay time if throtted, return error if it is rejected.
	Get(key interface{}) error
	SetRate(rate uint32) ZoneLimiter
	SetBurst(burst uint32) ZoneLimiter
	SetNodelay(nodelay bool) ZoneLimiter
	SetResolution(resolution Resolution) ZoneLimiter
	AddZoneItem(key interface{}) error
	DeleteZoneItem(key interface{}) error
	SetZoneItem(key interface{}, rate uint32, burst uint32, nodelay bool)
}

type limiterMeta struct {
	//configuration variables
	nodelay    bool
	burst      uint32
	rate       uint32
	resolution Resolution
}

type limiterRecord struct {
	//dynamic couting variables
	last   int64
	excess int64
}
