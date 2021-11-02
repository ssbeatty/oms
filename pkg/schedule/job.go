package schedule

import (
	log "github.com/sirupsen/logrus"
	"io"
	"oms/models"
	"oms/pkg/transport"
)

const (
	JobTypeCron JobType = "cron"
	JobTypeTask JobType = "task"
)

type JobType string

// Job is cron task or long task
type Job struct {
	Id     string
	Type   JobType
	host   *models.Host
	cmd    string
	status bool
	logger *log.Logger
	log    string // log path
	quit   chan bool
	std    io.Writer // lumberjack.Logger
}

func (j *Job) Run() {
	client, err := transport.NewClient(j.host.Addr, j.host.Port, j.host.User, j.host.PassWord, []byte(j.host.KeyFile))
	if err != nil {
		j.logger.Errorf("error when new ssh client, err: %v", err)
		return
	}
	session, err := client.NewSession()
	if err != nil {
		j.logger.Errorf("create new session failed, err: %v", err)
	}
	session.SetStdout(j.std)
	err = session.Run(j.cmd)
	if err != nil {
		j.logger.Errorf("error when run cmd, err: %v", err)
		return
	}
}
