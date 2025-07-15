package handler

import (
	"TianHe-API/model"
	"TianHe-API/utils"
	"fmt"
	"time"
)

type GiftHandler struct {
	roomID int
}

func NewGiftHandler(roomID int) *GiftHandler {
	return &GiftHandler{roomID: roomID}
}

func (h *GiftHandler) Handle(data map[string]interface{}) {
	giftData, ok := data["data"].(map[string]interface{})
	if !ok {
		return
	}

	gift := &model.GiftMessage{
		GiftName:  giftData["giftName"].(string),
		GiftID:    int(giftData["giftId"].(float64)),
		UserName:  giftData["uname"].(string),
		UserID:    int64(giftData["uid"].(float64)),
		Num:       int(giftData["num"].(float64)),
		Price:     int(giftData["price"].(float64)),
		Timestamp: time.Now(),
	}

	// 输出礼物信息
	fmt.Printf("[房间%d-礼物] %s 送出了 %d 个 %s (价值: %d)\n",
		h.roomID, gift.UserName, gift.Num, gift.GiftName, gift.Price)
	utils.Logger.Infof("房间%d 礼物 - %s: %d个%s", h.roomID, gift.UserName, gift.Num, gift.GiftName)
}

type WelcomeHandler struct {
	roomID int
}

func NewWelcomeHandler(roomID int) *WelcomeHandler {
	return &WelcomeHandler{roomID: roomID}
}

func (h *WelcomeHandler) Handle(data map[string]interface{}) {
	welcomeData, ok := data["data"].(map[string]interface{})
	if !ok {
		return
	}

	welcome := &model.WelcomeMessage{
		UserName:  welcomeData["uname"].(string),
		UserID:    int64(welcomeData["uid"].(float64)),
		Timestamp: time.Now(),
		IsVip:     welcomeData["vip"].(float64) > 0,
	}

	fmt.Printf("[房间%d-进房] %s 进入了直播间\n", h.roomID, welcome.UserName)
	utils.Logger.Infof("房间%d 进房 - %s", h.roomID, welcome.UserName)
}

type FollowHandler struct {
	roomID int
}

func NewFollowHandler(roomID int) *FollowHandler {
	return &FollowHandler{roomID: roomID}
}

func (h *FollowHandler) Handle(data map[string]interface{}) {
	msgType, ok := data["msg_type"].(float64)
	if !ok || msgType != 2 {
		return
	}

	fmt.Printf("[房间%d-关注] 有用户关注了主播\n", h.roomID)
	utils.Logger.Infof("房间%d 关注事件", h.roomID)
}
