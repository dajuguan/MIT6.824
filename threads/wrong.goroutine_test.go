package threads

import (
	"fmt"
	"sync"
	"testing"
)

// expected print: 0,1,2,3,4
// actually we'll get 5,5,5,5,5
func TestFail(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			Print(i)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestSucceed(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			Print(i)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func Print(i int) {
	fmt.Println(i)
}
