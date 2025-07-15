package main

import (
	"TianHe-API/auth"
	"TianHe-API/client"
	"TianHe-API/config"
	"TianHe-API/utils"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 初始化日志
	utils.InitLogger()

	// 读取配置
	cfg := config.NewConfig()

	// 检查登录状态
	if !auth.IsLoggedIn() {
		utils.Logger.Info("未检测到有效登录状态，开始扫码登录...")
		err := auth.QRCodeLogin()
		if err != nil {
			utils.Logger.Fatalf("登录失败: %v", err)
		}
		utils.Logger.Info("登录成功！")
	} else {
		utils.Logger.Info("检测到有效登录状态")
	}

	// 创建客户端管理器
	manager := client.NewManager(cfg)

	// 添加要监听的房间
	for _, roomID := range cfg.RoomIDs {
		err := manager.AddRoom(roomID)
		if err != nil {
			utils.Logger.Errorf("添加房间 %d 失败: %v", roomID, err)
		} else {
			utils.Logger.Infof("开始监听房间 %d", roomID)
		}
	}

	// 启动监听
	manager.Start()

	fmt.Printf("开始监听 %d 个直播间...\n", len(cfg.RoomIDs))

	// 定期检查登录状态
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if !auth.IsLoggedIn() {
				utils.Logger.Warn("登录状态失效，请重新登录")
				manager.Stop()
				return
			}
		}
	}()

	// 等待退出信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	fmt.Println("正在关闭...")
	manager.Stop()
}
