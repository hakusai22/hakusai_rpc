package hakusai_rpc

import (
	"encoding/json"
	"fmt"
	"hakusai_rpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

// Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{}
// <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->

// 在一次连接中，Option 固定在报文的最开始，Header 和 Body 可以有多个，即报文可能是这样的。
//  Option | Header1 | Body1 | Header2 | Body2 | ...

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this's a geerpc request
	CodecType   codec.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// Server represents an RPC Server.
type Server struct{}

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer()

//ServeConn 的实现就和之前讨论的通信过程紧密相关了，首先使用 json.NewDecoder 反序列化得到 Option 实例，
//检查 MagicNumber 和 CodeType 的值是否正确。然后根据 CodeType 得到对应的消息编解码器，接下来的处理交给 serverCodec。
func (server *Server) ServeConn(conn io.ReadWriteCloser) {

	// defer方式关闭conn
	defer func() { _ = conn.Close() }()
	var opt Option
	// 首先使用 json.NewDecoder 反序列化得到 Option 实例
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	// 检查 MagicNumber 和 CodeType 的值是否正确。
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	// 然后根据 CodeType 得到对应的消息编解码器，接下来的处理交给 serverCodec
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

// 1. 读取请求 readRequest
// 2. 处理请求 handleRequest
// 3. 回复请求 sendResponse
func (server *Server) serveCodec(cc codec.Codec) {
	// 这个控制的不是并发，是服务处理过程中，防止提前退出
	sending := new(sync.Mutex) // make sure to send a complete response 确保发送完整的回复
	wg := new(sync.WaitGroup)  // wait until all request are handled 等到所有请求都处理完毕
	// 之前提到过，在一次连接中，允许接收多个请求，即多个 request header 和 request body，
	// 因此这里使用了 for 无限制地等待请求的到来，直到发生错误（例如连接被关闭，接收到的报文有问题等），这里需要注意的点有三个：
	// 3.尽力而为，只有在 header 解析失败时，才终止循环。
	for {
		// 1. 读取请求 readRequest
		req, err := server.readRequest(cc)
		if err != nil {
			// 只有在 header 解析失败时，才终止循环。
			if req == nil {
				break // it's not possible to recover, so close the connection
			}
			req.h.Error = err.Error()
			// 3. 回复请求 sendResponse
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 1.handleRequest 使用了协程并发执行请求
		// 2.处理请求是并发的，但是回复请求的报文必须是逐个发送的，并发容易导致多个回复报文交织在一起，客户端无法解析。在这里使用锁(sending)保证。
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

// request stores all information of a call
type request struct {
	h            *codec.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// TODO: now we don't know the type of request argv
	// day 1, just suppose it's string
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	// TODO, should call registered rpc methods to get the right replyv
	// day 1, just print argv and send a hello message
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}

// Accept 实现了 Accept 方式，net.Listener 作为参数，for 循环等待 socket 连接建立，
// 并开启子协程处理，处理过程交给了 ServerConn 方法。
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		// 并开启子协程处理，处理过程交给了 ServerConn 方法。
		go server.ServeConn(conn)
	}
}

// Accept DefaultServer 是一个默认的 Server 实例，主要为了用户使用方便。
func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}
