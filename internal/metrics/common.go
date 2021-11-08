package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"oms/internal/ssh"
	"oms/internal/task"
	"oms/internal/tunnel"
	"oms/pkg/transport"
	"time"
)

const (
	DefaultTimeTicker = 5 * time.Second
)

type Manager struct {
	sshManager    *ssh.Manager
	taskManager   *task.Manager
	tunnelManager *tunnel.Manager

	// 当前上传的文件数
	currentFiles prometheus.Gauge
	// 缓存的ssh client数
	currentSSHClient prometheus.Gauge
	// 内存中的job数
	currentJobs prometheus.Gauge
	// 内存中的隧道数
	currentTunnels prometheus.Gauge
	// 当前的ssh session数
	currentSessions prometheus.Gauge
}

func NewManager(ssh *ssh.Manager, task *task.Manager, tunnel *tunnel.Manager) *Manager {
	manager := &Manager{
		sshManager:    ssh,
		taskManager:   task,
		tunnelManager: tunnel,
	}
	manager.currentSSHClient = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ssh_client_cache_nums",
		Help: "SSH Client Num in Lru Cache.",
	})
	manager.currentFiles = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sftp_tran_file_nums",
		Help: "Current Files of Transport.",
	})
	manager.currentTunnels = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tunnel_register_nums",
		Help: "Current Num of Tunnel.",
	})
	manager.currentJobs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "task_register_nums",
		Help: "Current Num of Task.",
	})
	manager.currentSessions = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "current_session_nums",
		Help: "Current Nums of SSH Session.",
	})

	return manager
}

func (m *Manager) Init() *Manager {
	// register in prometheus
	prometheus.MustRegister(m.currentFiles)
	prometheus.MustRegister(m.currentSSHClient)
	prometheus.MustRegister(m.currentTunnels)
	prometheus.MustRegister(m.currentJobs)
	prometheus.MustRegister(m.currentSessions)

	transport.RegisterSessionGauge(m.currentSessions)

	go func() {
		for {
			<-time.After(DefaultTimeTicker)
			m.currentJobs.Set(float64(m.taskManager.GetJobList().Length()))
			m.currentTunnels.Set(float64(m.tunnelManager.GetTunnelList().Length()))
			m.currentSSHClient.Set(float64(m.sshManager.GetSSHList().Length()))
			m.currentFiles.Set(float64(m.sshManager.GetFileList().Length()))
		}
	}()
	return m
}
