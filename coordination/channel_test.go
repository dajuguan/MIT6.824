package coordination

import "testing"

func TestChannelSync(t *testing.T) {
	c := make(chan bool)
	go func() {
		c <- true
	}()
	for val := range c {
		println(val)
		break
	}
}
