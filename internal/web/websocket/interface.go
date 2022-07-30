package websocket

import (
	"oms/internal/models"
	"oms/internal/ssh"
)

type WebService interface {
	RunCmdWithContext(host *models.Host, cmd ssh.Command, sudo bool, ch chan interface{})
	GetSSHManager() *ssh.Manager
}
