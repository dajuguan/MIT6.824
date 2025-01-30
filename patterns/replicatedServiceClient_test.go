// call on many servers
// - return if got response from any server
// - retry to call on many servers when timeout
package patterns

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

type Reply int
type Resp struct {
	server int
	val    Reply
}
type ReplicatedClient interface {
	Init(servers []string, callOnce func(server string, arg int) Reply)
	Call(arg string) Reply
}

type Client struct {
	servers  []string
	callOnce func(string, int) Reply

	mu             sync.Mutex
	preferedServer int
}

func (c *Client) Init(servers []string, callOnce func(string, int) Reply) {
	c.servers = servers
	c.callOnce = callOnce
}

func (c *Client) Call(arg int) Reply {
	timeout := time.Second
	t := time.NewTimer(timeout)
	defer t.Stop()

	result := make(chan Resp, 1)
	retry := func() {
		for i := range c.servers {
			i := i
			go func() {
				println("fetching ", i)
				result <- Resp{i, c.callOnce(c.servers[i], arg)}
			}()
		}
	}

	retry()
	for { // busy waiting
		select {
		case reply := <-result:
			return reply.val
		case <-t.C:
			t.Reset(timeout)
			println("retrying")
			retry()
		}
	}
}

func TestRSC(t *testing.T) {
	c := Client{
		servers: []string{"s1", "s2", "s3"},
		callOnce: func(server string, arg int) Reply {
			r := time.Duration(rand.Intn(1000))
			time.Sleep(time.Millisecond*r + time.Second*1)
			return Reply(arg)
		},
		preferedServer: -1,
	}

	res := c.Call(3)
	println("got res:", res)
}
