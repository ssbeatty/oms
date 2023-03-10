package server

import (
	"github.com/ssbeatty/oms/internal/config"
	"github.com/ssbeatty/oms/internal/ssh"
	"github.com/ssbeatty/oms/internal/task"
	"github.com/ssbeatty/oms/internal/tunnel"
	"github.com/ssbeatty/oms/internal/web"
)

type Server struct {
	cfg *config.Conf
	// all managers
	taskManager   *task.Manager
	tunnelManager *tunnel.Manager
	sshManager    *ssh.Manager
}

func NewServer(cfg *config.Conf) *Server {
	// do init all manager
	sshManager := ssh.NewManager(cfg).Init()
	taskManager := task.NewManager(sshManager, cfg).Init()
	tunnelManager := tunnel.NewManager(sshManager).Init()

	service := &Server{
		cfg:           cfg,
		sshManager:    sshManager,
		taskManager:   taskManager,
		tunnelManager: tunnelManager,
	}

	return service
}

func (s *Server) Run() {
	web.Serve(s.cfg.App, s.sshManager, s.taskManager, s.tunnelManager)
}
