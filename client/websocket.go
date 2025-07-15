package client

import (
	"TianHe-API/protocol"
	"TianHe-API/utils"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// WebSocket连接状态
const (
	StateDisconnected = iota
	StateConnecting
	StateConnected
	StateReconnecting
)

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	roomID      int
	conn        net.Conn
	state       int
	stateMutex  sync.RWMutex
	sendChan    chan []byte
	receiveChan chan []byte
	closeChan   chan struct{}
	done        chan struct{}
	wg          sync.WaitGroup
}

// NewWebSocketClient 创建新的WebSocket客户端
func NewWebSocketClient(roomID int) *WebSocketClient {
	return &WebSocketClient{
		roomID:      roomID,
		state:       StateDisconnected,
		sendChan:    make(chan []byte, 100),
		receiveChan: make(chan []byte, 100),
		closeChan:   make(chan struct{}),
		done:        make(chan struct{}),
	}
}

// Connect 连接到弹幕服务器
func (c *WebSocketClient) Connect() error {
	c.setState(StateConnecting)

	// 连接到弹幕服务器
	conn, err := net.DialTimeout("tcp", "broadcastlv.chat.bilibili.com:2243", 10*time.Second)
	if err != nil {
		c.setState(StateDisconnected)
		return fmt.Errorf("连接失败: %v", err)
	}

	c.conn = conn
	c.setState(StateConnected)

	// 启动读写协程
	c.wg.Add(2)
	go c.readLoop()
	go c.writeLoop()

	utils.Logger.Infof("房间 %d WebSocket连接成功", c.roomID)
	return nil
}

// Disconnect 断开连接
func (c *WebSocketClient) Disconnect() {
	c.setState(StateDisconnected)

	select {
	case <-c.done:
		return // 已经关闭
	default:
		close(c.done)
	}

	if c.conn != nil {
		c.conn.Close()
	}

	c.wg.Wait()
	utils.Logger.Infof("房间 %d WebSocket连接已断开", c.roomID)
}

// SendPacket 发送数据包
func (c *WebSocketClient) SendPacket(packet *protocol.Packet) error {
	if c.getState() != StateConnected {
		return fmt.Errorf("连接未建立或已断开")
	}

	data := packet.Encode()
	select {
	case c.sendChan <- data:
		return nil
	case <-c.done:
		return fmt.Errorf("连接已关闭")
	default:
		return fmt.Errorf("发送队列已满")
	}
}

// ReceivePacket 接收数据包
func (c *WebSocketClient) ReceivePacket() (*protocol.Packet, error) {
	select {
	case data := <-c.receiveChan:
		return protocol.DecodePacket(data)
	case <-c.done:
		return nil, fmt.Errorf("连接已关闭")
	}
}

// IsConnected 检查是否已连接
func (c *WebSocketClient) IsConnected() bool {
	return c.getState() == StateConnected
}

// getState 获取连接状态
func (c *WebSocketClient) getState() int {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state
}

// setState 设置连接状态
func (c *WebSocketClient) setState(state int) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	c.state = state
}

// readLoop 读取循环
func (c *WebSocketClient) readLoop() {
	defer c.wg.Done()
	defer c.setState(StateDisconnected)

	for {
		select {
		case <-c.done:
			return
		default:
			// 设置读取超时
			c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			// 读取包头
			headerBuf := make([]byte, protocol.HeaderLength)
			_, err := io.ReadFull(c.conn, headerBuf)
			if err != nil {
				if !c.isConnectionClosed(err) {
					utils.Logger.Errorf("房间 %d 读取包头失败: %v", c.roomID, err)
				}
				return
			}

			// 解析包头获取包长度
			packetLength := binary.BigEndian.Uint32(headerBuf[0:4])
			if packetLength < protocol.HeaderLength || packetLength > 32768 {
				utils.Logger.Errorf("房间 %d 数据包长度异常: %d", c.roomID, packetLength)
				continue
			}

			// 读取完整数据包
			totalData := make([]byte, packetLength)
			copy(totalData, headerBuf)

			if packetLength > protocol.HeaderLength {
				_, err = io.ReadFull(c.conn, totalData[protocol.HeaderLength:])
				if err != nil {
					utils.Logger.Errorf("房间 %d 读取包体失败: %v", c.roomID, err)
					return
				}
			}

			// 处理数据包
			c.handlePacket(totalData)
		}
	}
}

// writeLoop 发送循环
func (c *WebSocketClient) writeLoop() {
	defer c.wg.Done()

	for {
		select {
		case data := <-c.sendChan:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			_, err := c.conn.Write(data)
			if err != nil {
				if !c.isConnectionClosed(err) {
					utils.Logger.Errorf("房间 %d 发送数据失败: %v", c.roomID, err)
				}
				return
			}
		case <-c.done:
			return
		}
	}
}

// handlePacket 处理数据包
func (c *WebSocketClient) handlePacket(data []byte) {
	packet, err := protocol.DecodePacket(data)
	if err != nil {
		utils.Logger.Errorf("房间 %d 解析数据包失败: %v", c.roomID, err)
		return
	}

	switch packet.Operation {
	case protocol.OpMessage:
		// 检查是否需要解压
		if packet.Version == 2 {
			// zlib压缩
			decompressed, err := c.decompress(packet.Body)
			if err != nil {
				utils.Logger.Errorf("房间 %d 解压缩失败: %v", c.roomID, err)
				return
			}

			// 递归处理解压后的多个数据包
			c.handleMultiPackets(decompressed)
		} else {
			// 普通消息包
			select {
			case c.receiveChan <- data:
			case <-c.done:
				return
			default:
				utils.Logger.Warnf("房间 %d 接收队列已满，丢弃消息", c.roomID)
			}
		}
	case protocol.OpHeartbeatReply:
		// 心跳回应
		select {
		case c.receiveChan <- data:
		case <-c.done:
			return
		default:
		}
	case protocol.OpConnect:
		// 连接成功确认
		utils.Logger.Infof("房间 %d 认证成功", c.roomID)
	}
}

// handleMultiPackets 处理多个数据包
func (c *WebSocketClient) handleMultiPackets(data []byte) {
	offset := 0
	for offset < len(data) {
		if offset+protocol.HeaderLength > len(data) {
			break
		}

		// 读取包长度
		packetLength := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		if offset+packetLength > len(data) {
			break
		}

		// 提取单个数据包
		packetData := data[offset : offset+packetLength]

		// 递归处理
		c.handlePacket(packetData)

		offset += packetLength
	}
}

// decompress zlib解压缩
func (c *WebSocketClient) decompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// isConnectionClosed 检查是否为连接关闭错误
func (c *WebSocketClient) isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}

	// 检查常见的连接关闭错误
	errStr := err.Error()
	return contains(errStr, "connection reset by peer") ||
		contains(errStr, "broken pipe") ||
		contains(errStr, "connection refused") ||
		contains(errStr, "use of closed network connection")
}

// contains 字符串包含检查
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[0:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				indexOf(s, substr) >= 0))
}

// indexOf 查找子字符串位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetStateName 获取状态名称
func (c *WebSocketClient) GetStateName() string {
	switch c.getState() {
	case StateDisconnected:
		return "已断开"
	case StateConnecting:
		return "连接中"
	case StateConnected:
		return "已连接"
	case StateReconnecting:
		return "重连中"
	default:
		return "未知"
	}
}

// Ping 发送ping测试
func (c *WebSocketClient) Ping() error {
	if !c.IsConnected() {
		return fmt.Errorf("连接未建立")
	}

	heartbeat := protocol.NewHeartbeatPacket()
	return c.SendPacket(heartbeat)
}

// GetConnectionInfo 获取连接信息
func (c *WebSocketClient) GetConnectionInfo() map[string]interface{} {
	info := make(map[string]interface{})
	info["room_id"] = c.roomID
	info["state"] = c.GetStateName()
	info["connected"] = c.IsConnected()

	if c.conn != nil {
		if addr := c.conn.RemoteAddr(); addr != nil {
			info["remote_addr"] = addr.String()
		}
		if addr := c.conn.LocalAddr(); addr != nil {
			info["local_addr"] = addr.String()
		}
	}

	return info
}
