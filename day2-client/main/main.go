package main

import (
	"fmt"
	"hakusai_rpc"
	"log"
	"net"
	"sync"
	"time"
)

func startServer(addr chan string) {
	// pick a free port
	l, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	hakusai_rpc.Accept(l)
}

func main() {
	log.SetFlags(1)
	addr := make(chan string)
	go startServer(addr)
	client, _ := hakusai_rpc.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("hakusai_rpc req %d", i)
			var reply string
			// 在 main 函数中使用了 client.Call 并发了 5 个 RPC 同步调用，参数和返回值的类型均为 string。
			if err := client.Call("WWWWWWWWWW.Sum", args, &reply); err != nil {
				log.Fatal("call WWWWWWWWWW.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}
