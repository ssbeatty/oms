package websocket

import (
	"oms/internal/models"
	"oms/internal/ssh"
)

type WebService interface {
	RunCmdWithContext(host *models.Host, cmd string, sudo bool, ch chan interface{})
	GetSSHManager() *ssh.Manager
}
