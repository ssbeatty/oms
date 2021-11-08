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
	// all services
	webService *web.Service
	// all managers
	taskManager   *task.Manager
	tunnelManager *tunnel.Manager
	sshManager    *ssh.Manager
}

func NewServer(cfg *config.Conf) *Server {
	// do init all manager
	sshManager := ssh.NewManager()
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
	s.webService = web.NewService(s.cfg.App, s.sshManager, s.taskManager, s.tunnelManager)
	s.webService.SetRelease()
	// block
	s.webService.InitRouter().Run()
}
