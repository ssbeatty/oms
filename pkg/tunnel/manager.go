package tunnel

import (
	log "github.com/sirupsen/logrus"
	"oms/models"
	"oms/pkg/transport"
	"sync"
	"time"
)

const (
	DefaultInterval = 30 * time.Second
)

var (
	DefaultManager = NewManager()
)

func init() {
	go DefaultManager.Start()
}

type Manager struct {
	tunnels sync.Map
	closer  chan bool
}

func NewManager() *Manager {
	manager := &Manager{
		tunnels: sync.Map{},
		closer:  make(chan bool),
	}

	return manager
}

func (m *Manager) initTunnelFromModels(modelTunnels []*models.Tunnel) {
	for _, modelTunnel := range modelTunnels {
		err := m.AddTunnel(modelTunnel)
		if err != nil {
			log.Errorf("error when add tunnel, err: %v", err)
		}
		m.updateTunnelStatus()
	}
}

func (m *Manager) updateTunnelStatus() {
	m.tunnels.Range(func(key, value interface{}) bool {
		id := key.(int)
		realTunnel := value.(*SSHTunnel)
		_, err := models.UpdateTunnelStatus(id, realTunnel.Status(), realTunnel.GetErrorMsg())
		if err != nil {
			log.Errorf("error when update model tunnel status")
			return false
		}
		return true
	})
}

func (m *Manager) Start() {
	var once sync.Once
	tunnels, err := models.GetAllTunnel()
	if err != nil {
		log.Errorf("error when get all tunnel, err: %v", err)
	}
	once.Do(func() {
		m.initTunnelFromModels(tunnels)
	})

	ticker := time.NewTicker(DefaultInterval)

	for {
		select {
		case <-m.closer:
			log.Debug("tunnel manager exit.")
		case <-ticker.C:
			m.updateTunnelStatus()
		}
	}

}

func (m *Manager) Close() {
	close(m.closer)
}

func (m *Manager) AddTunnel(modelTunnel *models.Tunnel) error {
	host := modelTunnel.Host
	client, err := transport.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		return err
	}
	realTunnel := NewSSHTunnel(client.GetSSHClient(), modelTunnel.Destination, modelTunnel.Source, modelTunnel.Mode)
	go realTunnel.Start()

	m.tunnels.Store(modelTunnel.Id, realTunnel)

	return nil
}

func (m *Manager) RemoveTunnel(modelTunnel *models.Tunnel) {
	val, ok := m.tunnels.Load(modelTunnel.Id)
	if !ok {
		return
	}

	realTunnel := val.(*SSHTunnel)
	defer realTunnel.Close()

	m.tunnels.Delete(modelTunnel.Id)

}
