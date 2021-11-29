package tunnel

import (
	"oms/internal/models"
	"oms/internal/ssh"
	"oms/internal/utils"
	"oms/pkg/logger"
	"oms/pkg/tunnel"
	"sync"
	"time"
)

const (
	DefaultInterval = 30 * time.Second
)

type Manager struct {
	tunnels    *utils.SafeMap
	closer     chan bool
	sshManager *ssh.Manager
	logger     *logger.Logger
}

func NewManager(sshManager *ssh.Manager) *Manager {
	manager := &Manager{
		tunnels:    utils.NewSafeMap(),
		closer:     make(chan bool),
		sshManager: sshManager,
		logger:     logger.NewLogger("tunnelManager"),
	}

	return manager
}

// GetTunnelList 获取tunnels poll
func (m *Manager) GetTunnelList() *utils.SafeMap {
	return m.tunnels
}

// Init 启动并注册数据库的tunnel到tunnels poll
func (m *Manager) Init() *Manager {
	go m.Start()
	return m
}

func (m *Manager) initTunnelFromModels(modelTunnels []*models.Tunnel) {
	for _, modelTunnel := range modelTunnels {
		err := m.AddTunnel(modelTunnel, &modelTunnel.Host)
		if err != nil {
			m.logger.Errorf("error when add tunnel, err: %v", err)
		}
	}
	m.updateAllTunnelStatus()
}

func (m *Manager) updateAllTunnelStatus() {
	m.tunnels.Range(func(key, value interface{}) bool {
		id := key.(int)
		realTunnel := value.(*tunnel.SSHTunnel)
		_, err := models.UpdateTunnelStatus(id, realTunnel.Status(), realTunnel.GetErrorMsg())
		if err != nil {
			m.logger.Errorf("error when update model tunnel status, err: %v", err)
			// 如果数据没有这条数据 停止隧道
			if !models.ExistedTunnel(id) {
				m.RemoveTunnel(id)
			}
		}
		// re connect
		if !realTunnel.Status() {
			go realTunnel.Start()
		}
		return true
	})
}

func (m *Manager) Start() {
	var once sync.Once

	once.Do(func() {
		tunnels, err := models.GetAllTunnel()
		if err != nil {
			m.logger.Errorf("error when get all tunnel, err: %v", err)
		}
		m.initTunnelFromModels(tunnels)
	})

	ticker := time.NewTicker(DefaultInterval)

	for {
		select {
		case <-m.closer:
			m.Clear()
			m.logger.Debug("tunnel manager exit.")
			return
		case <-ticker.C:
			m.updateAllTunnelStatus()
		}
	}

}

func (m *Manager) Clear() {
	m.tunnels.Range(func(key, value interface{}) bool {
		realTunnel := value.(*tunnel.SSHTunnel)
		realTunnel.Close()
		return true
	})
}

func (m *Manager) Close() {
	close(m.closer)
}

// AddTunnel create new tunnel
func (m *Manager) AddTunnel(modelTunnel *models.Tunnel, host *models.Host) error {
	client, err := m.sshManager.NewClient(host)
	if err != nil {
		return err
	}
	realTunnel := tunnel.NewSSHTunnel(client.GetSSHClient(), modelTunnel.Destination, modelTunnel.Source, modelTunnel.Mode)
	go realTunnel.Start()

	m.tunnels.Store(modelTunnel.Id, realTunnel)

	return nil
}

// RemoveTunnel 从tunnels poll删除tunnel 并停止
func (m *Manager) RemoveTunnel(id int) {
	val, ok := m.tunnels.Load(id)
	if !ok {
		return
	}

	realTunnel := val.(*tunnel.SSHTunnel)
	defer realTunnel.Close()

	m.tunnels.Delete(id)
}
