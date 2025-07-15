package client

import (
	"TianHe-API/auth"
	"TianHe-API/handler"
	"TianHe-API/protocol"
	"TianHe-API/utils"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type DanmuClient struct {
	roomID    int
	conn      *websocket.Conn
	done      chan struct{}
	handlers  map[string]handler.MessageHandler
	connected bool
	mutex     sync.RWMutex
}

func NewDanmuClient(roomID int) *DanmuClient {
	client := &DanmuClient{
		roomID:   roomID,
		done:     make(chan struct{}),
		handlers: make(map[string]handler.MessageHandler),
	}

	// 注册消息处理器
	client.registerHandlers()

	return client
}

func (c *DanmuClient) registerHandlers() {
	c.handlers[protocol.CmdDanmu] = handler.NewDanmuHandler(c.roomID)
	c.handlers[protocol.CmdGift] = handler.NewGiftHandler(c.roomID)
	c.handlers[protocol.CmdWelcome] = handler.NewWelcomeHandler(c.roomID)
	c.handlers[protocol.CmdFollow] = handler.NewFollowHandler(c.roomID)
}

func (c *DanmuClient) Connect() error {
	// 构建WebSocket连接URL
	u := url.URL{
		Scheme: "wss",
		Host:   "broadcastlv.chat.bilibili.com:443",
		Path:   "/sub",
	}

	// 设置请求头
	headers := make(map[string][]string)
	headers["User-Agent"] = []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"}
	headers["Origin"] = []string{"https://live.bilibili.com"}

	// 添加Cookie
	if cookieStr := auth.GetCookieString(); cookieStr != "" {
		headers["Cookie"] = []string{cookieStr}
	}

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		return err
	}

	c.mutex.Lock()
	c.conn = conn
	c.connected = true
	c.done = make(chan struct{})
	c.mutex.Unlock()

	// 发送认证包
	token := auth.GenerateToken(c.roomID)
	authPacket := protocol.NewAuthPacket(c.roomID, token)
	err = c.conn.WriteMessage(websocket.BinaryMessage, authPacket.Encode())
	if err != nil {
		c.Close()
		return err
	}

	// 启动心跳
	go c.heartbeat()

	// 启动消息接收
	go c.readMessages()

	return nil
}

func (c *DanmuClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

func (c *DanmuClient) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !c.IsConnected() {
				return
			}

			heartbeatPacket := protocol.NewHeartbeatPacket()
			err := c.conn.WriteMessage(websocket.BinaryMessage, heartbeatPacket.Encode())
			if err != nil {
				utils.Logger.Errorf("房间 %d 发送心跳失败: %v", c.roomID, err)
				c.Close()
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *DanmuClient) readMessages() {
	defer c.Close()

	for {
		select {
		case <-c.done:
			return
		default:
			messageType, data, err := c.conn.ReadMessage()
			if err != nil {
				utils.Logger.Errorf("房间 %d 读取消息失败: %v", c.roomID, err)
				return
			}

			if messageType == websocket.BinaryMessage {
				c.handleBinaryMessage(data)
			}
		}
	}
}

func (c *DanmuClient) handleBinaryMessage(data []byte) {
	packet, err := protocol.DecodePacket(data)
	if err != nil {
		utils.Logger.Errorf("房间 %d 解析数据包失败: %v", c.roomID, err)
		return
	}

	switch packet.Operation {
	case protocol.OpHeartbeatReply:
		// 心跳回应，包含在线人数
		if len(packet.Body) >= 4 {
			onlineCount := int32(packet.Body[0])<<24 | int32(packet.Body[1])<<16 |
				int32(packet.Body[2])<<8 | int32(packet.Body[3])
			utils.Logger.Debugf("房间 %d 在线人数: %d", c.roomID, onlineCount)
		}
	case protocol.OpMessage:
		// 普通消息
		c.handleMessage(packet.Body)
	case protocol.OpConnect:
		utils.Logger.Infof("房间 %d 连接成功", c.roomID)
	}
}

func (c *DanmuClient) handleMessage(data []byte) {
	cmd, msgData, err := protocol.ParseMessage(data)
	if err != nil {
		utils.Logger.Errorf("房间 %d 解析消息失败: %v", c.roomID, err)
		return
	}

	if handler, exists := c.handlers[cmd]; exists {
		handler.Handle(msgData)
	}
}

func (c *DanmuClient) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected {
		return
	}

	c.connected = false
	close(c.done)

	if c.conn != nil {
		c.conn.Close()
	}
}
