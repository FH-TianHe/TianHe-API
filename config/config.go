package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	RoomIDs     []int  `json:"room_ids"`
	DanmuServer string `json:"danmu_server"`
	DanmuPort   int    `json:"danmu_port"`
	LogLevel    string `json:"log_level"`
	CookiePath  string `json:"cookie_path"`
	MaxRetries  int    `json:"max_retries"`
	RetryDelay  int    `json:"retry_delay"`
}

func NewConfig() *Config {
	cfg := &Config{
		RoomIDs:     []int{21452505}, // 默认房间号
		DanmuServer: "broadcastlv.chat.bilibili.com",
		DanmuPort:   2243,
		LogLevel:    "info",
		CookiePath:  "config/cookie.json",
		MaxRetries:  3,
		RetryDelay:  5,
	}

	// 从环境变量读取房间号
	if envRoomIDs := os.Getenv("ROOM_IDS"); envRoomIDs != "" {
		roomStrs := strings.Split(envRoomIDs, ",")
		var roomIDs []int
		for _, roomStr := range roomStrs {
			if roomID, err := strconv.Atoi(strings.TrimSpace(roomStr)); err == nil {
				roomIDs = append(roomIDs, roomID)
			}
		}
		if len(roomIDs) > 0 {
			cfg.RoomIDs = roomIDs
		}
	}

	return cfg
}

// Cookie 结构
type Cookie struct {
	SESSDATA          string `json:"SESSDATA"`
	BiliJct           string `json:"bili_jct"`
	DedeUserID        string `json:"DedeUserID"`
	DedeUserID__ckMd5 string `json:"DedeUserID__ckMd5"`
	Sid               string `json:"sid"`
	ExpireTime        int64  `json:"expire_time"`
}

// 保存Cookie
func (c *Cookie) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// 加载Cookie
func LoadCookie(path string) (*Cookie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cookie Cookie
	err = json.Unmarshal(data, &cookie)
	if err != nil {
		return nil, err
	}

	return &cookie, nil
}
