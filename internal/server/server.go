package server

import (
	"oms/internal/config"
	"oms/internal/web"
)

type Service struct {
	cfg        *config.Conf
	webService *web.Service
}

func NewService(cfg *config.Conf) *Service {
	service := &Service{
		cfg: cfg,
	}

	return service
}

func (s *Service) Run() {
	s.webService = web.NewService(s.cfg.App.Addr, s.cfg.App.Port)
	s.webService.SetRelease()
	// block
	s.webService.InitRouter().Run()
}
