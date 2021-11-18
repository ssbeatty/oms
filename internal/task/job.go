package task

import (
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"gopkg.in/natefinch/lumberjack.v2"
	"oms/internal/models"
	"oms/internal/ssh"
	"oms/pkg/logger"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"
)

type JobType string
type JobStatus string

const (
	MaxRetryTimes = 10

	JobTypeCron    JobType = "cron"
	JobTypeTask    JobType = "task"
	DefaultTmpPath         = "tmp"

	JobStatusReady   JobStatus = "ready"
	JobStatusRunning JobStatus = "running"
	JobStatusStop    JobStatus = "stop"
	JobStatusDone    JobStatus = "done"
	JobStatusFatal   JobStatus = "fatal"
	JobStatusBackoff JobStatus = "backoff"
)

// Job is cron task or long task
type Job struct {
	ID       int
	name     string
	Type     JobType
	host     *models.Host
	cmd      string
	status   atomic.Value
	logger   *logger.Logger
	log      string // log path
	quit     chan bool
	std      *lumberjack.Logger
	spec     string
	maxRetry int32
	engine   *Manager
}

func (m *Manager) NewJob(id int, name, cmd, spec string, t JobType, host *models.Host) *Job {
	l := logger.NewLogger("job")
	if name == "" {
		name = strconv.Itoa(id)
	}
	tmp := filepath.Join(DefaultTmpPath, fmt.Sprintf("%s.log", name))
	std := &lumberjack.Logger{
		Filename:   tmp,
		MaxSize:    50, // megabytes
		MaxBackups: 3,
		MaxAge:     20,   //days
		Compress:   true, // disabled by default
	}
	l.SetOutput(std)

	job := &Job{
		ID:     id,
		name:   name,
		Type:   t,
		host:   host,
		cmd:    cmd,
		quit:   make(chan bool, 1),
		logger: l,
		log:    tmp,
		std:    std,
		spec:   spec,
		engine: m,
	}
	job.UpdateStatus(JobStatusReady)

	return job
}

func (j *Job) Run() {
	j.engine.logger.Infof("job, name: %s, cmd: %s, start running.", j.name, j.cmd)
	j.logger.Info("job start running.")
	if j.Status() == JobStatusStop || j.Status() == JobStatusFatal {
		return
	}
	defer func() {
		if j.Status() == JobStatusRunning {
			j.UpdateStatus(JobStatusDone)
		}
		j.engine.logger.Debugf("job, name: %s, cmd: %s, exit.", j.name, j.cmd)
	}()
	client, err := j.engine.sshManager.NewClient(j.host.Addr, j.host.Port, j.host.User, j.host.PassWord, []byte(j.host.KeyFile))
	if err != nil {
		j.logger.Errorf("error when new ssh client, err: %v", err)
		return
	}

	j.UpdateStatus(JobStatusRunning)
	if j.Type == JobTypeCron {
		session, err := client.NewPty()
		if err != nil {
			j.logger.Errorf("create new session failed, err: %v", err)
		}
		defer session.Close()

		session.SetStdout(j.std)

		err = session.Run(j.cmd)
		if err != nil {
			j.logger.Errorf("error when run cmd, err: %v", err)
			j.UpdateStatus(JobStatusBackoff)
			return
		}
	} else if j.Type == JobTypeTask {
		exp := backoff.NewExponentialBackOff()
		//exp.RandomizationFactor = 1

		operation := func() error {
			atomic.AddInt32(&j.maxRetry, 1)

			err = ssh.RunTaskWithQuit(client, j.cmd, j.quit, j.std)
			if err != nil {
				if j.maxRetry < MaxRetryTimes {
					j.UpdateStatus(JobStatusBackoff)
					return err
				} else {
					j.UpdateStatus(JobStatusFatal)
					j.engine.logger.Infof("Job command: %s max retry times", j.cmd)
					return nil
				}
			}
			return nil
		}
		err = backoff.RetryNotify(operation, exp, func(err error, duration time.Duration) {
			j.engine.logger.Debug(fmt.Sprintf("Job cmd: %s Failed, retry after %v, detail in log.", j.cmd, duration))
		})
		if err != nil {
			j.UpdateStatus(JobStatusFatal)
			j.engine.logger.Error("retry notify command error", err)
		}
	}
}

func (j *Job) Status() JobStatus {
	return j.status.Load().(JobStatus)
}

func (j *Job) UpdateStatus(status JobStatus) {
	j.status.Store(status)

	_, err := models.UpdateJobStatus(j.ID, string(status))
	if err != nil {
		j.engine.logger.Errorf("error when update job status, err: %v", err)
	}
}

func (j *Job) Close() {
	switch j.Type {
	case JobTypeCron:
		j.engine.taskService.Remove(j.Name())
	case JobTypeTask:
		j.quit <- true
	}
	j.UpdateStatus(JobStatusStop)
}

func (j *Job) Name() string {
	if j.name != "" {
		return j.name
	}
	return strconv.Itoa(j.ID)
}

func (j *Job) Log() string {
	return j.log
}

func (j *Job) Cmd() string {
	return j.cmd
}
