// conditional variable (semaphore)

/* common pattern
mu.Lock()
doSomething()
cond.Broadcast()
mu.Unlock()

---

mu.Lock();
while cond == false {
	cond.wait()
}
mu.Unlock
*/

package coordination

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestSemaphore(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	var count int
	var finished int
	for i := 0; i < 10; i++ {
		go func() {
			vote := requestVote()
			mu.Lock()
			defer mu.Unlock()
			if vote == 1 {
				count++
			}
			finished++
			cond.Broadcast()
		}()
	}

	mu.Lock()
	for count < 5 && finished != 10 {
		// acutally we don't need to wait 5 times
		println("waiting....")
		cond.Wait()
	}
	mu.Unlock()
	if count >= 5 {
		println("receiveed 5+ votes")
	} else {
		println("lost")
	}
}

func TestBusyWaiting(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	var mu sync.Mutex

	var count int
	var finished int
	for i := 0; i < 10; i++ {
		go func() {
			vote := requestVote()
			mu.Lock()
			defer mu.Unlock()
			if vote == 1 {
				count++
			}
			finished++
		}()
	}

	for {
		mu.Lock()
		if count >= 5 || finished == 10 {
			break
		}
		mu.Unlock()
		time.Sleep(1 * time.Microsecond)
		println("waiting....")
	}

	if count >= 5 {
		println("receiveed 5+ votes")
	} else {
		println("lost")
	}
}

func requestVote() int {
	time.Sleep(time.Millisecond * 10)
	return rand.Intn(1000) % 2
}
