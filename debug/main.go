package main
/*
This example demonstrates usage of the zone rate limiter, one can create a zone rate limiter with a "rate" setting, plus a "burst" value as a supplement, which is strongly suggested setting it as a small number since this value is also important to the leaky-bucket algorithm. 
Make it as an analogy, the "rate" parameter defines the size of the leaky hole of the bucket, the "burst" parameter defines the capacity of the leaky bucket. 
With the new limiter instance, add a specific key to the zone rate limiter or set a specific key with customized rate limit settings, so the rate limiter will take effect for the key.
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
    //it is supposed to be finished with the exact time span of: (10 routines) * (1000 reqs/routine) / 1000 req/second) = 10 seconds 
}
