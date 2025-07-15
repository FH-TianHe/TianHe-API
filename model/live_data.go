package model

import "time"

// 弹幕消息
type DanmuMessage struct {
	Text      string    `json:"text"`
	UserName  string    `json:"user_name"`
	UserID    int64     `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
	Color     string    `json:"color"`
	FontSize  int       `json:"font_size"`
}

// 礼物消息
type GiftMessage struct {
	GiftName  string    `json:"gift_name"`
	GiftID    int       `json:"gift_id"`
	UserName  string    `json:"user_name"`
	UserID    int64     `json:"user_id"`
	Num       int       `json:"num"`
	Price     int       `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// 进房消息
type WelcomeMessage struct {
	UserName  string    `json:"user_name"`
	UserID    int64     `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
	IsVip     bool      `json:"is_vip"`
}

// 关注消息
type FollowMessage struct {
	UserName  string    `json:"user_name"`
	UserID    int64     `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// 直播间统计
type LiveStats struct {
	OnlineCount int       `json:"online_count"`
	Timestamp   time.Time `json:"timestamp"`
}
