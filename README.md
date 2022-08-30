Go Leaky-bucket Rate Limiter
============================
This package provides an Golang implemented leaky-bucket algorithm, which is largely ported from the implementation of the Nginx's rate limiter, and introduces a few new features as enhancements. This package uses the compare-and-swap (CAS) mechanism to achieve data consistency for multi-threading programs.

Table of Contents
=================
- [Go Leaky-bucket Rate Limiter](#go-leaky-bucket-rate-limiter)
- [Table of Contents](#table-of-contents)
- [Quick Start](#quick-start)
    - [Simple rate limiter example](#simple-rate-limiter-example)
    - [Zone rate limiter example](#zone-rate-limiter-example)
- [Leaky Bucket Algorithm](#leaky-bucket-algorithm)
    - [Which algorithm?](#which-algorithm)
    - [Dig into it](#dig-into-it)
    - [Best practice](#best-practice)
- [Specification](#specification)
    - [Simple Rate Limiter](#simple-rate-limiter)
    - [Zone Rate Limiter](#zone-rate-limiter)
    - [Resolution](#resolution)
- [License](#license)
- [Report Issues](#report-issues)
- [Contact Author](#contact-author)


Quick Start
=====
### Simple rate limiter example
```go
package main
/*
This example demonstrates the simplest usage of the rate limiter, 
one can create a rate limiter with a "rate" setting, plus a "burst" 
value as a supplement, which is strongly suggested by setting it as a small number
since this value is also important to the leaky-bucket algorithm. 
Make it as an analogy, the "rate" parameter defines the size of the leaky 
hole of the bucket, the "burst" parameter defines the capacity of the leaky bucket.  
*/
import (
  "fmt"
  "sync"

  leakybucket "github.com/dypflying/leakybucket"
)

func main() {
    
  rl := leakybucket.NewRateLimiter(1000).SetBurst(10) //
  wg := sync.WaitGroup{}
  routineNum := 10
  wg.Add(routineNum)
  for i := 0; i < routineNum; i++ {
    go func() {
      for i := 0; i < 1000; i++ {
        err := rl.Get()
        if err == nil {
          fmt.Println("take successfully")
        } else {
          fmt.Printf("%v\n", err)
        }
      }
      wg.Done()
    }()
  }
  wg.Wait()

  //it is supposed to be finished with the exact time span of: 
  //(10 routines) * (1000 reqs/routine) / 1000 req/second) = 10 seconds 

}
```

### Zone rate limiter example
```go
package main
/*
This example demonstrates usage of the zone rate limiter, 
one can create a zone rate limiter with a "rate" setting, 
plus a "burst" value as a supplement, which is strongly suggested by setting 
it as a small number since this value is also important to the leaky-bucket algorithm. 
Make it as an analogy, the "rate" parameter defines the size of the 
leaky hole of the bucket, the "burst" parameter defines the capacity 
of the leaky bucket. 
With the new limiter instance, add a specific key to the zone rate limiter 
or set a specific key with customized rate limit settings, so the rate limiter 
will take effect for the key.
*/
import (
  "fmt"
  "sync"

  leakybucket "github.com/dypflying/leakybucket"
)

func main() {
  rl := leakybucket.NewZoneRateLimiter(1000).SetBurst(10).SetResolution(leakybucket.ResolutionEnum.Microsecond) // per second
  rl.AddZoneItem("test.com")
  //or customize the key related rate limit setting 
  //rl.SetZoneItem("test.com", 100, 10, false)
  wg := sync.WaitGroup{}
  routineNum := 10
  wg.Add(routineNum)
  for i := 0; i < routineNum; i++ {
    go func() {
      for i := 0; i < 1000; i++ {
        err := rl.Get("test.com")
        if err == nil {
          fmt.Println("take successfully")
        } else {
          fmt.Printf("%v\n", err)
        }
      }
      wg.Done()
    }()
  }
  wg.Wait()
  //it is supposed to be finished with the exact time span of: 
  //(10 routines) * (1000 reqs/routine) / 1000 req/second) = 10 seconds 
}
```

[Back to TOC](#table-of-contents)


Leaky Bucket Algorithm
=======================
![Leaky Bucket Algorithm](https://user-images.githubusercontent.com/7089479/187326664-cd520ad7-83ed-40e2-8120-4577f3db1d1e.jpeg)

The leaky bucket algorithm is one of the three classic traffic shaping algorithms, while the other two are the token bucket algorithm and the sliding window algorithm.  
Suppose there is a bucket with a hole at the bottom into which tasks are pouring water at all kinds of rate, but the water is leaking at a fixed rate, if the poured water is too fast for the bucket to accommodate, the overflowed water is discarded. 
The leaky bucket algorithm is designed for smoothing the bursty traffic, no matter what the input rate is, the output rate is constant. 

### Which algorithm?
- Leaky-bucket vs Token-bucket: they are both used for bursty traffic shaping, though there is no explicit cutoff line for their scenarios, the leaky-bucket algorithm is likely used to "protect other systems", such as the Nginx to protect upstreams, and the token bucket algorithm is likely used to "protect self-system". The pakcage of https://pkg.go.dev/golang.org/x/time/rate is an implementation of token-bucket algorithm.
- Leaky-bucket vs Sliding-windows: Leaky-bucket is used for not only throttling traffic but also smoothing out the bursty traffic since it uses a configurable bucket to accommodate the  bursty traffic, the sliding-windows is only used for thottling traffic within a configured rate. 

### Dig into it
In fact, there are 3 key factors that control how the algorithm work:
1. The output rate value is an analogy to the size of the hole. 
2. The burst value is an analogy to the capacity of the bucket. 
3. The resolution of the time window, which is not in the picture of the algorithm. It can be used in conjunction with the burst value to precisely control the behavior of the algorithm. For instance, if the expected rate is 1,000 reqs/s, in the algorithm, it actually means every millisecond allows 1 req, if 2 reqs arrive within 1 millisecond, the second one will be put into the bucket for a delayed output or be discarded if the bucket is overflowed. However if the rate is 1,000,000 reqs/s, that means every microsecond allows 1 req, or every millisecond allows 1,000 reqs, if the implementation of the algorithm uses millisecond as the resolution of the time window, it requires a minimum burst value of 1000-1; if the implementation of the algorithm uses microsecond as the resolution of the time window, it won't require such a big burst instead. Though the combination of the usage of the "burst" and "resolution" can work for nearly all kinds of scenarios, they are slightly different, imagine such a senario for the former usage, 1,000 reqs arrvie within a microsecond, the former configuration (with a resolution of millisecond and a burst of 1,000) will output all reqs without delays, but the latter configuration (with a resolution of microsecond and a small burst value, e.g. 10) will only output a small number of reqs and discard the majority of the reqs.

### Best practice 
In personal opinions: 
- If the limit rate is no more than 1,000, take millisecond as the resolution and a relatively small burst value. 
- If the limit rate is more than 1,000, especially more than 10,000, take microsecond or 10X/100X microsecond as resolution, and a relatively small burst value. 
- If won't drop any traffic, in other words, just want to smooth out the bursty traffic,  configure a large enough burst value, just a kindly reminder, with this, you have to make sure there are still enough available goroutines besides the algorithm-blocked goroutines. 

[Back to TOC](#table-of-contents)

Specification 
=============
### Simple Rate Limiter
The export methods: 

- NewRateLimiter(rate uint32): Create a simple rate limiter with a rate value, zero value indicates denying all requests.  
- SetRate(rate uint32): Set the rate value. 
- SetBurst(burst uint32): Set the burst value, default is 0, details refer to the above algorithm explanation.
- SetResolution(resolution Resolution): Set the time window's resolution, default is the millisecond, details refer to the above algorithm explanation. 
- SetNodelay(nodelay bool): Set the nodelay option, default is false. If it is set to true, then the requests are either output without delay or rejected, but the water level in the bucket remains the same. nodelay is likely be used for traffic throttling only, not suitable for any traffic smoothing. 

- Get(): Rate limit method for the imcoming traffic, it will block/non-block the caller routine to a delay time automatically, or return error if the traffic is rejected.
- GetDelayInMicroseconds(): Another rate limit method for the imcoming traffic, unlike the Get() method, it returns the delay time in microseconds without blocking the caller routine, or an error if the traffic is rejected, the caller can handle the delay time by itself. 


### Zone Rate Limiter
The Zone rate limiter provides ways for a set of key-specific traffic shaping.
The export methods: 

- NewZoneRateLimiter(rate uint32): Create a simple rate limiter with a default rate value, the value can be overwritten for a specific key configuration.
- SetRate(rate uint32): Set the default rate value, overwritable by a specific key configuration.
- SetBurst(burst uint32): Set the default burst value, default is 0, overwritable by a specific key configuration.
- SetNodelay(nodelay bool): Set the nodelay option, default is false, overwritable by a specific key configuration.
- SetResolution(resolution Resolution): Set the time window's resolution, default is the millisecond. 
- AddZoneItem(key interface{}): Add a key to the zone rate limiter. 
- DeleteZoneItem(key interface{}): Delete a key from the zone rate limiter. 
- SetZoneItem(key interface{}, rate uint32, burst uint32, nodelay bool): Customize the rate limit setting for a specific key. 
  
- Get(key interface{},): Rate limit method for the imcoming traffic for the specific key, it will block/non-block the caller routine to a delay time automatically, or error if the traffic is rejected.
- GetDelayInMicroseconds(key interface{},): Another rate limit method for the imcoming traffic for the specific key, unlike the Get() method, it returns the delay time in microseconds without blocking the caller routine, or an error if the traffic is rejected, the caller can handle the delay time by itself. 

### Resolution 
- ResolutionEnum.Millisecond: 0.001 second, the default option. 
- ResolutionEnum.MicrosecondX100: 0.0001 second. 
- ResolutionEnum.MicrosecondX10: 0.00001 second. 
- ResolutionEnum.Microsecond: 0.000001 second. 

[Back to TOC](#table-of-contents)

License 
=======
MIT License

Copyright (c) Yunpeng Deng(dypflying)

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

[Back to TOC](#table-of-contents)

Report Issues 
=============
Please report bugs via the [GitHub Issue Tracker](https://github.com/dypflying/leakybucket/issues) or [Contact Author](#author-and-contact) 

[Back to TOC](#table-of-contents)

Contact Author
==============
- Author: Yunpeng Deng (dypflying)
- Mailto: dypflying@sina.com

Either English or Chinese is welcome.

[Back to TOC](#table-of-contents)