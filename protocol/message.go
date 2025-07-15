package protocol

import (
	"encoding/json"
	"errors"

	"github.com/tidwall/gjson"
)

// 消息命令常量
const (
	CmdDanmu        = "DANMU_MSG"          // 弹幕消息
	CmdGift         = "SEND_GIFT"          // 礼物消息
	CmdWelcome      = "WELCOME"            // 欢迎消息
	CmdFollow       = "NOTICE_MSG"         // 关注消息
	CmdRoomChange   = "ROOM_CHANGE"        // 房间信息变更
	CmdOnlineCount  = "ONLINE_RANK_COUNT"  // 在线人数
	CmdComboSend    = "COMBO_SEND"         // 连击
	CmdWelcomeGuard = "WELCOME_GUARD"      // 舰长进入
	CmdGuardBuy     = "GUARD_BUY"          // 购买舰长
	CmdSuperChat    = "SUPER_CHAT_MESSAGE" // SC消息
)

// ParseMessage 解析消息
func ParseMessage(data []byte) (string, map[string]interface{}, error) {
	// 解析JSON
	result := gjson.ParseBytes(data)

	cmd := result.Get("cmd").String()
	if cmd == "" {
		return "", nil, errors.New("消息格式错误：缺少cmd字段")
	}

	// 将数据转换为map
	var msgData map[string]interface{}
	err := json.Unmarshal(data, &msgData)
	if err != nil {
		return "", nil, err
	}

	return cmd, msgData, nil
}

// BuildAuthMessage 构建认证消息
func BuildAuthMessage(roomID int, token string) []byte {
	authMsg := map[string]interface{}{
		"roomid":    roomID,
		"protover":  1,
		"platform":  "web",
		"clientver": "1.4.0",
		"type":      2,
	}

	if token != "" {
		authMsg["key"] = token
	}

	data, _ := json.Marshal(authMsg)
	return data
}

// MessageInfo 消息信息结构
type MessageInfo struct {
	Cmd       string                 `json:"cmd"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Info      []interface{}          `json:"info,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
}

// IsValidMessage 检查是否为有效消息
func IsValidMessage(cmd string) bool {
	validCmds := map[string]bool{
		CmdDanmu:        true,
		CmdGift:         true,
		CmdWelcome:      true,
		CmdFollow:       true,
		CmdRoomChange:   true,
		CmdOnlineCount:  true,
		CmdComboSend:    true,
		CmdWelcomeGuard: true,
		CmdGuardBuy:     true,
		CmdSuperChat:    true,
	}

	return validCmds[cmd]
}

// GetMessagePriority 获取消息优先级
func GetMessagePriority(cmd string) int {
	priorities := map[string]int{
		CmdSuperChat:   1, // 最高优先级
		CmdGuardBuy:    2,
		CmdGift:        3,
		CmdDanmu:       4,
		CmdWelcome:     5,
		CmdFollow:      6,
		CmdOnlineCount: 7,
		CmdRoomChange:  8,
	}

	if priority, exists := priorities[cmd]; exists {
		return priority
	}

	return 9 // 默认最低优先级
}
