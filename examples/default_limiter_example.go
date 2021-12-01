package examples

import (
	"fmt"
	"sync"
	"time"

	leakybucket "github.com/dypflying/leakybucket"
)

//SingleRoutineExample is to set a rate of 100 reqs/s with a bucket capacity of 10
//this case is supposed to be completed with exact 10 seconds
func SingleRoutineExample() {

	rl := leakybucket.NewRateLimiter(1000).SetBurst(10) // per second
	for i := 0; i < 10000; i++ {
		err := rl.Get()
		if err == nil {
			fmt.Println("take successfully")
		} else {
			fmt.Printf("%v\n", err)
		}
	}
}

//MultiRoutinesExample is to set a rate of 100 reqs/s with a bucket capacity of 10
//this case is supposed to be completed with exact 10 seconds
func MultiRoutinesExample() {

	rl := leakybucket.NewRateLimiter(1000).SetBurst(10).SetResolution(leakybucket.ResolutionEnum.Millisecond).SetNodelay(false) // per second
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

}

//MultiRoutinesNodelayExample is to set a rate of 1000 reqs/s with a bucket capacity of 10 and with nodelay option
//in this case, most of the request will be rejected
//nodelay is only use for traffic precisely throttling, in this case, it allows only 1 request per 1/1000 second (1 req/millesecond)
func MultiRoutinesNodelayExample() {

	rl := leakybucket.NewRateLimiter(1000).SetBurst(10).SetNodelay(true).SetResolution(leakybucket.ResolutionEnum.Microsecond)
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

}

//MultiRoutinesDelayedExample is similar to MultiRoutinesExample()
//in this case, use GetDelayInMicroseconds instead of Get() for manually controling the delay time
//if one sleep the delay time output by the func, it will have the same effect with Get()
func MultiRoutinesDelayedExample() {
	rl := leakybucket.NewRateLimiter(10000).SetBurst(10) // per second
	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			for i := 0; i < 10000; i++ {
				delay, err := rl.GetDelayInMicroseconds()
				if err == nil {
					if delay == 0 {
						fmt.Println("take successfully")
					} else {
						time.Sleep(time.Duration(delay) * time.Microsecond)
						fmt.Printf("delay taking in %d Microsecond\n", delay)
					}
				} else {
					fmt.Printf("%v\n", err)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
