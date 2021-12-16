package server

import (
	"oms/internal/config"
	"oms/internal/ssh"
	"oms/internal/task"
	"oms/internal/tunnel"
	"oms/internal/web"
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
	sshManager := ssh.NewManager().Init()
	taskManager := task.NewManager(sshManager).Init()
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
