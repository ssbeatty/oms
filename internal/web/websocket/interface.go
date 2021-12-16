package websocket

import (
	"oms/internal/models"
	"oms/internal/ssh"
)

type WebService interface {
	RunCmdWithContext(host *models.Host, cmd string, sudo bool, ch chan interface{})
	ParseHostList(pType string, id int) []*models.Host
	GetSSHManager() *ssh.Manager
}
