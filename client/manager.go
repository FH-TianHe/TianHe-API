package client

import (
	"TianHe-API/config"
	"TianHe-API/utils"
	"fmt"
	"sync"
	"time"
)

type Manager struct {
	clients map[int]*DanmuClient
	config  *config.Config
	mutex   sync.RWMutex
	running bool
	wg      sync.WaitGroup
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		clients: make(map[int]*DanmuClient),
		config:  cfg,
	}
}

// 添加房间
func (m *Manager) AddRoom(roomID int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.clients[roomID]; exists {
		return fmt.Errorf("房间 %d 已存在", roomID)
	}

	client := NewDanmuClient(roomID)
	m.clients[roomID] = client

	return nil
}

// 移除房间
func (m *Manager) RemoveRoom(roomID int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if client, exists := m.clients[roomID]; exists {
		client.Close()
		delete(m.clients, roomID)
		utils.Logger.Infof("移除房间 %d", roomID)
	}
}

// 启动所有客户端
func (m *Manager) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.running = true

	for roomID, client := range m.clients {
		m.wg.Add(1)
		go m.startClient(roomID, client)
	}
}

// 启动单个客户端
func (m *Manager) startClient(roomID int, client *DanmuClient) {
	defer m.wg.Done()

	retries := 0
	for m.running && retries < m.config.MaxRetries {
		err := client.Connect()
		if err != nil {
			utils.Logger.Errorf("房间 %d 连接失败: %v", roomID, err)
			retries++
			if retries < m.config.MaxRetries {
				utils.Logger.Infof("房间 %d 将在 %d 秒后重试 (%d/%d)",
					roomID, m.config.RetryDelay, retries, m.config.MaxRetries)
				time.Sleep(time.Duration(m.config.RetryDelay) * time.Second)
			}
			continue
		}

		utils.Logger.Infof("房间 %d 连接成功", roomID)
		retries = 0

		// 等待连接断开
		<-client.done

		if m.running {
			utils.Logger.Warnf("房间 %d 连接断开，准备重连", roomID)
			time.Sleep(time.Duration(m.config.RetryDelay) * time.Second)
		}
	}

	if retries >= m.config.MaxRetries {
		utils.Logger.Errorf("房间 %d 重试次数已达上限，停止连接", roomID)
	}
}

// 停止所有客户端
func (m *Manager) Stop() {
	m.mutex.Lock()
	m.running = false

	for roomID, client := range m.clients {
		client.Close()
		utils.Logger.Infof("关闭房间 %d", roomID)
	}
	m.mutex.Unlock()

	m.wg.Wait()
	utils.Logger.Info("所有客户端已关闭")
}

// 获取运行状态
func (m *Manager) GetStatus() map[int]bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	status := make(map[int]bool)
	for roomID, client := range m.clients {
		status[roomID] = client.IsConnected()
	}

	return status
}

// 获取房间列表
func (m *Manager) GetRooms() []int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	rooms := make([]int, 0, len(m.clients))
	for roomID := range m.clients {
		rooms = append(rooms, roomID)
	}

	return rooms
}
