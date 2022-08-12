package websocket

import (
	"oms/internal/models"
	"oms/internal/ssh"
)

type WebService interface {
	RunCmdWithContext(host *models.Host, cmd ssh.Command, ch chan *ssh.Result)
	GetSSHManager() *ssh.Manager
}
