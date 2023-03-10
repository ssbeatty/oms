package websocket

import (
	"github.com/ssbeatty/oms/internal/models"
	"github.com/ssbeatty/oms/internal/ssh"
)

type WebService interface {
	RunCmdWithContext(host *models.Host, cmd ssh.Command, ch chan *ssh.Result)
	GetSSHManager() *ssh.Manager
}
