// go run sleep.go
package coordination

import (
	"sync"
	"testing"
	"time"
)

var done bool

func TestPeriodic(t *testing.T) {
	go periodic()
	time.Sleep(3 * time.Second)
}

func TestPeriodicCancel(t *testing.T) {
	var mu sync.Mutex

	go periodicCancel(&mu)
	time.Sleep(3 * time.Second)
	mu.Lock()
	done = true
	mu.Unlock()
	time.Sleep(2 * time.Second)
}

func periodic() {
	for {
		println("wake up..")
		time.Sleep(time.Second)
	}
}

func periodicCancel(mu *sync.Mutex) {
	for {
		mu.Lock()
		if done {
			println("done!")
			mu.Unlock()
			return
		}
		mu.Unlock()

		println("wake up..")
		time.Sleep(time.Second)

	}
}
