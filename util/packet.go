package util

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"

	"github.com/davyxu/cellnet"
	"github.com/davyxu/cellnet/codec"
)

var (
	// ErrMaxPacket 数据包过大
	ErrMaxPacket = errors.New("packet over size")
	// ErrMinPacket 数据包过小
	ErrMinPacket = errors.New("packet short size")
	// ErrShortMsgID ID 错误
	ErrShortMsgID = errors.New("short msgid")
)

const (
	bodySize  = 2 // 包体大小字段
	msgIDSize = 2 // 消息ID字段
)

// RecvLTVPacket 接收Length-Type-Value格式的封包流程
func RecvLTVPacket(reader io.Reader, maxPacketSize int) (msg interface{}, err error) {

	// Size为uint16，占2字节
	var sizeBuffer = make([]byte, bodySize)

	// 持续读取Size直到读到为止
	_, err = io.ReadFull(reader, sizeBuffer)

	// 发生错误时返回
	if err != nil {
		return
	}

	if len(sizeBuffer) < bodySize {
		return nil, ErrMinPacket
	}

	// 用小端格式读取Size
	size := binary.LittleEndian.Uint16(sizeBuffer)

	if maxPacketSize > 0 && size >= uint16(maxPacketSize) {
		return nil, ErrMaxPacket
	}

	// 分配包体大小
	body := make([]byte, size)

	// 读取包体数据
	_, err = io.ReadFull(reader, body)

	// 发生错误时返回
	if err != nil {
		return
	}

	if len(body) < msgIDSize {
		return nil, ErrShortMsgID
	}

	msgid := binary.LittleEndian.Uint16(body)

	msgData := body[msgIDSize:]

	// 将字节数组和消息ID用户解出消息
	msg, _, err = codec.DecodeMessage(int(msgid), msgData)
	if err != nil {
		// TODO 接收错误时，返回消息
		return nil, err
	}

	return
}

// SendLTVPacket 发送Length-Type-Value格式的封包流程
func SendLTVPacket(writer io.Writer, ctx cellnet.ContextSet, data interface{}) error {

	var (
		msgData []byte
		msgID   int
		meta    *cellnet.MessageMeta
	)

	switch m := data.(type) {
	case *cellnet.RawPacket: // 发裸包
		msgData = m.MsgData
		msgID = m.MsgID
	default: // 发普通编码包
		var err error

		// 将用户数据转换为字节数组和消息ID
		msgData, meta, err = codec.EncodeMessage(data, ctx)

		if err != nil {
			return err
		}

		msgID = meta.ID
	}

	pkt := make([]byte, bodySize+msgIDSize+len(msgData))

	// Length
	binary.LittleEndian.PutUint16(pkt, uint16(msgIDSize+len(msgData)))

	// Type
	binary.LittleEndian.PutUint16(pkt[bodySize:], uint16(msgID))

	// Value
	copy(pkt[bodySize+msgIDSize:], msgData)

	// 将数据写入Socket
	err := WriteFull(writer, pkt)

	// Codec中使用内存池时的释放位置
	if meta != nil {
		codec.FreeCodecResource(meta.Codec, msgData, ctx)
	}

	return err
}

const (
	mbHeaderSize       = 8 // 客户端消息包头大小
	mbHeaderBodyLenPos = 4

	tkProxyHeaderSize       = 24 // 包体大小字段
	tkPorxyHeaderBodyLenPos = 20

	serverMsgErrorTimeOut = 1

	// KcpPacketMinSize kcp 包最短长度
	KcpPacketMinSize = 4
)

// MBMsgRecv 保存数据，做为 RecvMsgEvent 的Msg字段 pb消息包
/*
MBMsgRecv中 Header 对应下面的 Header
		  	Data 	 对应下面的 Data
type XXX struct {
	Header
	Data //主体实际数据
}

type Header struct {
	Magic 	uint32
	Length  uint32 // 主题实际数据的长度 len(Data)
}
*/

type MBMsgRecv struct {
	Header []byte
	Data   []byte
}

// ConServerMsgRecv 保存数据，做为 RecvMsgEvent 的Msg字段 C++消息包
/*
ConServerMsgRecv 的Data 包含下面的 Header + Data
type XXX struct {
	Header
	Data //主体实际数据
}

type Header struct {
	Magic   uint32 //消息魔数
	Serial  uint32 //序列号
	Origine uint16 //消息来源
	Reserve uint16 //保留
	Type    uint32 //消息类型
	Param   uint32 //消息参数（消息版本，返回值，标志位等）
	Length  uint32 //实际数据长度，不包括消息头
}
*/
type ConServerMsgRecv struct {
	Data  []byte
	ErrNo int
}

// UDPMsgRecv UDP 包封装
type UDPMsgRecv struct {
	Data []byte
}

// RecvMbPacket 接收客户端 8 字节包头
func RecvMbPacket(reader io.Reader, maxPacketSize int) (msg interface{}, err error) {
	var header = make([]byte, mbHeaderSize)
	_, err = io.ReadFull(reader, header)
	if err != nil {
		return
	}
	if len(header) < mbHeaderSize {
		return nil, ErrMinPacket
	}

	buf := bytes.NewReader(header)
	buf.Seek(mbHeaderBodyLenPos, io.SeekStart)
	var bodyLen uint32
	err = binary.Read(buf, binary.LittleEndian, &bodyLen)
	if err != nil {
		return nil, errors.New("RecvMbPacket read header failed")
	}
	if maxPacketSize > 0 && bodyLen >= uint32(maxPacketSize) {
		return nil, ErrMaxPacket
	}

	// 特殊情况: body 长度为 0 时, 仅返回头部信息
	if bodyLen == 0 {
		msg = MBMsgRecv{Header: header}
		return
	}

	msg = MBMsgRecv{
		Header: header,
		Data:   make([]byte, bodyLen)}
	body := msg.(MBMsgRecv).Data

	_, err = io.ReadFull(reader, body)
	if err != nil {
		return nil, errors.New("RecvMbPacket read body failed")
	}
	if len(body) < msgIDSize {
		return nil, ErrShortMsgID
	}
	return
}

// SendMBPacket send mobile pbc Packet data
func SendMBPacket(writer io.Writer, ctx cellnet.ContextSet, data interface{}) error {
	// buf1 := &bytes.Buffer{}
	// err := binary.Write(buf1, binary.LittleEndian, data)
	// if err != nil {
	// 	panic(err)
	// }
	// WriteFull(writer, buf1.Bytes())
	// return err
	// 业务层保证下发的都是 []byte, 不需要再做转换
	return WriteFull(writer, data.([]byte))
}

// RecvTkProxyPacket 接收tkProxy 24 字节包头
func RecvTkProxyPacket(reader io.Reader, maxPacketSize int) (msg interface{}, err error) {
	var header = make([]byte, tkProxyHeaderSize)
	_, err = io.ReadFull(reader, header)
	if err != nil {
		if errT, ok := err.(net.Error); ok {
			if errT.Timeout() {
				return ConServerMsgRecv{ErrNo: serverMsgErrorTimeOut}, nil
			}
		}
		return
	}
	if len(header) < tkProxyHeaderSize {
		return nil, ErrMinPacket
	}

	buf := bytes.NewReader(header)
	buf.Seek(tkPorxyHeaderBodyLenPos, io.SeekStart)
	var bodyLen uint32
	err = binary.Read(buf, binary.LittleEndian, &bodyLen)
	if err != nil {
		return nil, errors.New("RecvTkProxyPacket read header failed")
	}
	if maxPacketSize > 0 && bodyLen >= uint32(maxPacketSize) {
		return nil, ErrMaxPacket
	}

	// 特殊情况: body 长度为 0 时, 仅返回头部信息
	if bodyLen == 0 {
		msg = ConServerMsgRecv{Data: header}
		return
	}

	// 一次分配到位, 避免拷贝带来的自动增长
	var data = make([]byte, tkProxyHeaderSize+bodyLen)
	_, err = io.ReadFull(reader, data[tkProxyHeaderSize:])
	if err != nil {
		return nil, errors.New("RecvTkProxyPacket, read body failed")
	}
	copy(data, header)
	msg = ConServerMsgRecv{Data: data}
	return
}

// SendTkProxyPacket send tkProxy Packet data
func SendTkProxyPacket(writer io.Writer, ctx cellnet.ContextSet, data interface{}) error {
	// buf1 := &bytes.Buffer{}
	// err := binary.Write(buf1, binary.LittleEndian, data)
	// if err != nil {
	// 	panic(err)
	// }
	// WriteFull(writer, buf1.Bytes())
	// return err
	// 业务层保证下发的都是 []byte, 不需要再做转换
	return WriteFull(writer, data.([]byte))
}

// RecvUDPPacket 接收 UDP 内容, 封装成接口
func RecvUDPPacket(data []byte) interface{} {
	msg := UDPMsgRecv{Data: data}
	return msg
}
