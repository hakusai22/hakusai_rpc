package codec

import (
	"io"
)

type Header struct {
	ServiceMethod string // ServiceMethod 是服务名和方法名，通常与 Go 语言中的结构体和方法相映射
	Seq           uint64 // Seq 是请求的序号，也可以认为是某个请求的 ID，用来区分不同的请求。
	Error         string // Error 是错误信息，客户端置为空，服务端如果如果发生错误，将错误信息置于 Error 中。
}

// Codec 抽象出对消息体进行编解码的接口 Codec，抽象出接口是为了实现不同的 Codec 实例
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// NewCodecFunc 抽象出 Codec 的构造函数，客户端和服务端可以通过 Codec 的 Type 得到构造函数，从而创建 Codec 实例
// 返回的是构造函数，而非实例。
type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

// 我们定义了 2 种 Codec，Gob 和 Json，但是实际代码中只实现了 Gob 一种，事实上，2 者的实现非常接近，甚至只需要把 gob 换成 json 即可
const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // not implemented
)

// NewCodecFuncMap 创建一个map key是类型 value是NewCodecFunc 抽象出 Codec 的构造函数
var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
