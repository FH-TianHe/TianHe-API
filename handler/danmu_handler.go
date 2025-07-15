package handler

import (
	"TianHe-API/model"
	"TianHe-API/utils"
	"fmt"
	"time"
)

type MessageHandler interface {
	Handle(data map[string]interface{})
}

type DanmuHandler struct {
	roomID int
}

func NewDanmuHandler(roomID int) *DanmuHandler {
	return &DanmuHandler{roomID: roomID}
}

func (h *DanmuHandler) Handle(data map[string]interface{}) {
	info, ok := data["info"].([]interface{})
	if !ok || len(info) < 3 {
		return
	}

	// 解析弹幕信息
	danmuInfo := info[0].([]interface{})
	text := info[1].(string)
	userInfo := info[2].([]interface{})

	danmu := &model.DanmuMessage{
		Text:      text,
		UserName:  userInfo[1].(string),
		UserID:    int64(userInfo[0].(float64)),
		Timestamp: time.Now(),
		Color:     fmt.Sprintf("#%06x", int(danmuInfo[3].(float64))),
		FontSize:  int(danmuInfo[2].(float64)),
	}

	// 输出弹幕
	fmt.Printf("[房间%d-弹幕] %s: %s\n", h.roomID, danmu.UserName, danmu.Text)
	utils.Logger.Infof("房间%d 弹幕 - %s: %s", h.roomID, danmu.UserName, danmu.Text)
}
