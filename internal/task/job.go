package task

import (
	"errors"
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
	job.status.Store(JobStatusReady)

	return job
}

func (j *Job) Run() {
	if j.Status() == JobStatusStop || j.Status() == JobStatusFatal {
		return
	}
	defer func() {
		if j.Status() == JobStatusRunning {
			j.UpdateStatus(JobStatusDone)
		}
	}()
	//j.logger.Infoln(fmt.Sprintf("task name: [%s], cmd: '%s' running", j.Name(), j.cmd))
	client, err := j.engine.sshManager.NewClient(j.host.Addr, j.host.Port, j.host.User, j.host.PassWord, []byte(j.host.KeyFile))
	if err != nil {
		j.logger.Errorf("error when new ssh client, err: %v", err)
		return
	}
	session, err := client.NewSessionWithPty(20, 20)
	if err != nil {
		j.logger.Errorf("create new session failed, err: %v", err)
	}
	defer session.Close()

	session.SetStdout(j.std)
	j.UpdateStatus(JobStatusRunning)
	if j.Type == JobTypeCron {
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
			session, err := client.NewSessionWithPty(20, 20)
			if err != nil {
				j.logger.Errorf("create new session failed, err: %v", err)
			}
			defer session.Close()

			session.SetStdout(j.std)
			atomic.AddInt32(&j.maxRetry, 1)
			err = ssh.RunTaskWithQuit(client, j.cmd, j.quit)
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
			j.engine.logger.Debug(fmt.Sprintf("Job cmd: %s Failed: %s, retry after %v", j.Cmd(), err.Error(), duration))
		})
		if err != nil {
			j.UpdateStatus(JobStatusFatal)
			j.engine.logger.Error("RetryNotify command error", err)
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

// Register 开始task & 添加到poll
func (m *Manager) Register(id int, job *Job) error {
	if job.Status() == JobStatusRunning {
		m.logger.Info("task is running already.")
		return errors.New("task is running already")
	}
	switch job.Type {
	case JobTypeCron:
		err := m.taskService.AddByJob(job.Name(), job.spec, job)
		if err != nil {
			m.logger.Errorf("error when register job, err: %v", err)
			return err
		}
	case JobTypeTask:
		go job.Run()
	}

	m.taskPoll.Store(id, job)

	return nil
}

// UnRegister 关闭task & 从poll删除
func (m *Manager) UnRegister(id int) error {
	key, ok := m.taskPoll.Load(id)
	if ok {
		job := key.(*Job)
		job.Close()
		defer m.taskPoll.Delete(id)
	} else {
		return errors.New("job is stopped already")
	}
	m.logger.Infof("un register a job: %d, success", id)

	return nil
}

func (m *Manager) NewJobWithRegister(modelJob *models.Job, status string) error {
	realJob := m.NewJob(modelJob.Id, modelJob.Name, modelJob.Cmd, modelJob.Spec, JobType(modelJob.Type), &modelJob.Host)
	realJob.status.Store(JobStatus(status))
	if err := m.Register(modelJob.Id, realJob); err != nil {
		return err
	}

	m.logger.Infof("register a new job, type: %s, name: %s, cmd: %s success", modelJob.Type, modelJob.Name, modelJob.Cmd)

	return nil
}

func (m *Manager) RemoveJob(id int) error {
	m.logger.Infof("recv singinl to remove job: %d", id)
	err := m.UnRegister(id)
	if err != nil {
		return err
	}
	err = models.DeleteJobById(id)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) StartJob(modelJob *models.Job) error {
	m.logger.Infof("recv singinl to start job: %d", modelJob.Id)
	if key, ok := m.taskPoll.Load(modelJob.Id); ok {
		job := key.(*Job)
		job.status.Store(JobStatusReady)
		err := m.Register(modelJob.Id, job)
		if err != nil {
			return err
		}
	} else {
		err := m.NewJobWithRegister(modelJob, string(JobStatusReady))
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) StopJob(id int) error {
	m.logger.Infof("recv singinl to stop job: %d", id)
	err := m.UnRegister(id)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) GetJob(id int) (*Job, bool) {
	if key, ok := m.taskPoll.Load(id); ok {
		return key.(*Job), true
	}
	return nil, false
}

func (m *Manager) initJobFromModels(modelJobs []*models.Job) {
	m.logger.Info("init all job without stopped.")
	for _, modelJob := range modelJobs {
		err := m.NewJobWithRegister(modelJob, modelJob.Status)
		if err != nil {
			m.logger.Errorf("error when register a new job, err: %v", err)
		}
	}
}
