package patterns

import (
	"testing"
	"time"
)

func TestTimeOut(tt *testing.T) {
	timeout := time.Second
	t := time.NewTimer(timeout)
	defer t.Stop()

	result := make(chan int, 1)
	for i := 0; i < 3; i++ {
		i := i
		go func() {
			time.Sleep(time.Second * 5)
			result <- i
		}()
	}
	for {
		select {
		case r := <-result:
			println("res:", r)
			return
		case <-t.C:
			println("retrying")
			t.Reset(timeout)
		}
	}

}
