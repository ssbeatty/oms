package schedule

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"oms/models"
	"oms/pkg/transport"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type JobType string

const (
	JobTypeCron    JobType = "cron"
	JobTypeTask    JobType = "task"
	DefaultTmpPath         = "tmp"
)

var TaskPoll *sync.Map

func init() {
	err := os.MkdirAll(DefaultTmpPath, 0644)
	if err != nil {
		log.Error("error when make all tmp path, err: %v", err)
		return
	}
}

// Job is cron task or long task
type Job struct {
	Name   string
	Type   JobType
	host   *models.Host
	cmd    string
	status bool
	logger *log.Logger
	log    string // log path
	quit   chan bool
	std    *lumberjack.Logger
	spec   string
}

func NewJob(id int, cmd, spec string, t JobType, host *models.Host) *Job {
	logger := log.New()
	name := strconv.Itoa(id)
	tmp := filepath.Join(DefaultTmpPath, fmt.Sprintf("%s.log", name))
	std := &lumberjack.Logger{
		Filename:   tmp,
		MaxSize:    500, // megabytes
		MaxBackups: 2,
		MaxAge:     30,   //days
		Compress:   true, // disabled by default
	}
	logger.SetOutput(std)
	return &Job{
		Name:   name,
		Type:   t,
		host:   host,
		cmd:    cmd,
		quit:   make(chan bool),
		logger: logger,
		log:    tmp,
		std:    std,
		spec:   spec,
	}
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
	if j.Type == JobTypeCron {
		err = session.Run(j.cmd)
		if err != nil {
			j.logger.Errorf("error when run cmd, err: %v", err)
			return
		}
	} else if j.Type == JobTypeTask {
		session.RunTaskWithQuit(j.cmd, j.quit)
	}
}

func (j *Job) Close() {
	switch j.Type {
	case JobTypeCron:
		taskService.Remove(j.Name)
	case JobTypeTask:
		close(j.quit)
	}
}

// Register 开始task & 添加到poll
func Register(id int, job *Job) {
	switch job.Type {
	case JobTypeCron:
		err := taskService.AddByJob(job.Name, job.spec, job)
		if err != nil {
			log.Errorf("error when register job, err: %v", err)
			return
		}
	case JobTypeTask:
		go job.Run()
	}

	TaskPoll.Store(id, job)
}

// UnRegister 关闭task & 从poll删除
func UnRegister(id int) {
	key, ok := TaskPoll.Load(id)
	if ok {
		job := key.(*Job)
		job.Close()
		defer TaskPoll.Delete(id)
	}
}
