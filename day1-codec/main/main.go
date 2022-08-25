package main

import (
	"encoding/json"
	"fmt"
	"geerpc"
	"geerpc/codec"
	"log"
	"net"
	"time"
)

/**
如果不加消息编码，本质上是两个tcp的conn直接通信：
w -> conn -> conn -> r；
如果加上消息编码，就变成
w -> bufio -> gob -> conn -> conn -> gob -> r
这个流式处理的很漂亮
*/

func startServer(addr chan string) {
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	// 失败 直接打印日志
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	// 对地址进行赋值 应该是引用传递
	addr <- l.Addr().String()
	geerpc.Accept(l)
}

func main() {
	log.SetFlags(0)
	//切片初始化地址
	addr := make(chan string)
	// 开启一个协程
	go startServer(addr)

	// in fact, following code is like a simple geerpc client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	// 客户端简单的使用time.sleep()方式隔离协议交换阶段与RPC消息阶段，减少这种问题发生的可能。
	time.Sleep(time.Second)
	// send options
	_ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	cc := codec.NewGobCodec(conn)
	// send request & receive response
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = cc.Write(h, fmt.Sprintf("geerpc req %d", h.Seq))
		// 这里就是读取服务端返回的response的header
		// 如果server端还没写入，client端这边就读不到就会被block
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
