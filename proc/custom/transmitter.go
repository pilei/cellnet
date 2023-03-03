package custom

import (
	"io"
	"net"

	"github.com/davyxu/cellnet"
	"github.com/davyxu/cellnet/util"
)

type mbMessageTransmitter struct {
}

type socketOpt interface {
	MaxPacketSize() int
	ApplySocketReadTimeout(conn net.Conn, callback func())
	ApplySocketWriteTimeout(conn net.Conn, callback func())
}

func (mbMessageTransmitter) OnRecvMessage(ses cellnet.Session) (msg interface{}, err error) {
	reader, ok := ses.Raw().(io.Reader)
	// 转换错误，或者连接已经关闭时退出
	if !ok || reader == nil {
		return nil, nil
	}
	opt := ses.Peer().(socketOpt)
	if conn, ok := reader.(net.Conn); ok {
		// 有读超时时，设置超时
		opt.ApplySocketReadTimeout(conn, func() {
			msg, err = util.RecvMbPacket(reader, opt.MaxPacketSize())
		})
	}
	return
}

func (mbMessageTransmitter) OnSendMessage(ses cellnet.Session, msg interface{}) (err error) {

	writer, ok := ses.Raw().(io.Writer)

	// 转换错误，或者连接已经关闭时退出
	if !ok || writer == nil {
		return nil
	}

	opt := ses.Peer().(socketOpt)

	// 有写超时时，设置超时
	opt.ApplySocketWriteTimeout(writer.(net.Conn), func() {

		err = util.SendMBPacket(writer, ses.(cellnet.ContextSet), msg)

	})

	return
}

type proxyMessageTransmitter struct {
}

func (proxyMessageTransmitter) OnRecvMessage(ses cellnet.Session) (msg interface{}, err error) {

	reader, ok := ses.Raw().(io.Reader)

	// 转换错误，或者连接已经关闭时退出
	if !ok || reader == nil {
		return nil, nil
	}

	opt := ses.Peer().(socketOpt)

	if conn, ok := reader.(net.Conn); ok {

		// 有读超时时，设置超时
		opt.ApplySocketReadTimeout(conn, func() {

			msg, err = util.RecvTkProxyPacket(reader, opt.MaxPacketSize())

		})
	}

	return
}

func (proxyMessageTransmitter) OnSendMessage(ses cellnet.Session, msg interface{}) (err error) {

	writer, ok := ses.Raw().(io.Writer)

	// 转换错误，或者连接已经关闭时退出
	if !ok || writer == nil {
		return nil
	}

	opt := ses.Peer().(socketOpt)

	// 有写超时时，设置超时
	opt.ApplySocketWriteTimeout(writer.(net.Conn), func() {

		err = util.SendTkProxyPacket(writer, ses.(cellnet.ContextSet), msg)

	})

	return
}
