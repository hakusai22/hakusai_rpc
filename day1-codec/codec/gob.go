package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// GobCodec 首先定义 GobCodec 结构体，
//这个结构体由四部分构成，conn 是由构建函数传入，
//通常是通过 TCP 或者 Unix 建立 socket 时得到的链接实例，
//dec 和 enc 对应 gob 的 Decoder 和 Encoder，
//buf 是为了防止阻塞而创建的带缓冲的 Writer，一般这么做能提升性能。
type GobCodec struct {
	conn io.ReadWriteCloser //连接
	buf  *bufio.Writer      //io的buffer
	dec  *gob.Decoder       //解密
	enc  *gob.Encoder       //加密
}

// 将空值转换为 *GobCodec 类型
// 再转换为 Codec 接口
// 如果转换失败，说明 GobCodec 并没有实现 Codec 接口的所有方法
var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

// 接着实现 ReadHeader、ReadBody、Write 和 Close 方法。
func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err = c.enc.Encode(h); err != nil {
		log.Println("rpc: gob error encoding header:", err)
		return
	}
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc: gob error encoding body:", err)
		return
	}
	return
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}
