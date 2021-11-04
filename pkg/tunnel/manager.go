package tunnel

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"oms/models"
	"oms/pkg/transport"
	"oms/pkg/utils"
	"sync"
	"time"
)

const (
	DefaultInterval = 30 * time.Second
)

var (
	DefaultManager = NewManager()

	tunnelNum = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tunnel_register_nums",
		Help: "Current Num of Tunnel.",
	})
)

func init() {
	go DefaultManager.Start()
	prometheus.MustRegister(tunnelNum)
}

type Manager struct {
	tunnels *utils.SafeMap
	closer  chan bool
}

func NewManager() *Manager {
	manager := &Manager{
		tunnels: utils.NewSageMap(),
		closer:  make(chan bool),
	}

	return manager
}

func (m *Manager) Length() int {
	return m.tunnels.Length()
}

func (m *Manager) initTunnelFromModels(modelTunnels []*models.Tunnel) {
	for _, modelTunnel := range modelTunnels {
		err := m.addTunnel(modelTunnel, &modelTunnel.Host)
		if err != nil {
			log.Errorf("error when add tunnel, err: %v", err)
		}
		m.updateTunnelStatus()

		tunnelNum.Set(float64(DefaultManager.Length()))
	}
}

func (m *Manager) updateTunnelStatus() {
	m.tunnels.Range(func(key, value interface{}) bool {
		id := key.(int)
		realTunnel := value.(*SSHTunnel)
		_, err := models.UpdateTunnelStatus(id, realTunnel.Status(), realTunnel.GetErrorMsg())
		if err != nil {
			log.Errorf("error when update model tunnel status, err: %v", err)
			// 如果数据没有这条数据 停止隧道
			if !models.ExistedTunnel(id) {
				m.RemoveTunnel(id)
			}
		}
		return true
	})
}

func (m *Manager) Start() {
	var once sync.Once

	once.Do(func() {
		tunnels, err := models.GetAllTunnel()
		if err != nil {
			log.Errorf("error when get all tunnel, err: %v", err)
		}
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

func (m *Manager) addTunnel(modelTunnel *models.Tunnel, host *models.Host) error {
	client, err := transport.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		return err
	}
	realTunnel := NewSSHTunnel(client.GetSSHClient(), modelTunnel.Destination, modelTunnel.Source, modelTunnel.Mode)
	go realTunnel.Start()

	m.tunnels.Store(modelTunnel.Id, realTunnel)

	return nil
}

func (m *Manager) RemoveTunnel(id int) {
	val, ok := m.tunnels.Load(id)
	if !ok {
		return
	}

	realTunnel := val.(*SSHTunnel)
	defer realTunnel.Close()

	m.tunnels.Delete(id)
}

func (m *Manager) AddTunnel(hostId int, mode, src, dest string) (*models.Tunnel, error) {
	host, err := models.GetHostById(hostId)
	if err != nil {
		return nil, err
	}
	tunnel, err := models.InsertTunnel(mode, src, dest, host)
	if err != nil {
		return nil, err
	}
	err = DefaultManager.addTunnel(tunnel, host)
	if err != nil {
		return nil, err
	}
	return tunnel, nil
}
