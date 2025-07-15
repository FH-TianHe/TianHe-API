package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// 操作类型
	OpHeartbeat      = 2 // 心跳
	OpHeartbeatReply = 3 // 心跳回应
	OpMessage        = 5 // 消息
	OpUserAuth       = 7 // 认证
	OpConnect        = 8 // 连接成功
)

const (
	// 协议版本
	ProtocolVersion = 1
	// 头部长度
	HeaderLength = 16
)

type Packet struct {
	PacketLength int32  // 包长度
	HeaderLength int16  // 头部长度
	Version      int16  // 协议版本
	Operation    int32  // 操作类型
	SequenceID   int32  // 序列号
	Body         []byte // 包体
}

// 编码数据包
func (p *Packet) Encode() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, p.PacketLength)
	binary.Write(buf, binary.BigEndian, p.HeaderLength)
	binary.Write(buf, binary.BigEndian, p.Version)
	binary.Write(buf, binary.BigEndian, p.Operation)
	binary.Write(buf, binary.BigEndian, p.SequenceID)

	if p.Body != nil {
		buf.Write(p.Body)
	}

	return buf.Bytes()
}

// 解码数据包
func DecodePacket(data []byte) (*Packet, error) {
	if len(data) < HeaderLength {
		return nil, errors.New("数据包长度不足")
	}

	buf := bytes.NewReader(data)
	packet := &Packet{}

	binary.Read(buf, binary.BigEndian, &packet.PacketLength)
	binary.Read(buf, binary.BigEndian, &packet.HeaderLength)
	binary.Read(buf, binary.BigEndian, &packet.Version)
	binary.Read(buf, binary.BigEndian, &packet.Operation)
	binary.Read(buf, binary.BigEndian, &packet.SequenceID)

	if len(data) > HeaderLength {
		packet.Body = data[HeaderLength:]
	}

	return packet, nil
}

// 创建心跳包
func NewHeartbeatPacket() *Packet {
	return &Packet{
		PacketLength: HeaderLength,
		HeaderLength: HeaderLength,
		Version:      ProtocolVersion,
		Operation:    OpHeartbeat,
		SequenceID:   1,
	}
}

// 创建认证包
func NewAuthPacket(roomID int, token string) *Packet {
	authData := fmt.Sprintf(`{"roomid":%d,"protover":1,"platform":"web","clientver":"1.4.0","type":2,"key":"%s"}`, roomID, token)
	body := []byte(authData)

	return &Packet{
		PacketLength: int32(HeaderLength + len(body)),
		HeaderLength: HeaderLength,
		Version:      ProtocolVersion,
		Operation:    OpUserAuth,
		SequenceID:   1,
		Body:         body,
	}
}
