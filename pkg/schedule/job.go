package schedule

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"oms/models"
	"oms/pkg/transport"
	"path/filepath"
	"strconv"
	"sync/atomic"
)

type JobType string
type JobStatus string

const (
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
	ID     int
	name   string
	Type   JobType
	host   *models.Host
	cmd    string
	status atomic.Value
	logger *log.Logger
	log    string // log path
	quit   chan bool
	std    *lumberjack.Logger
	spec   string
}

func NewJob(id int, name, cmd, spec string, t JobType, host *models.Host) *Job {
	logger := log.New()
	if name == "" {
		name = strconv.Itoa(id)
	}
	tmp := filepath.Join(DefaultTmpPath, fmt.Sprintf("%s.log", name))
	std := &lumberjack.Logger{
		Filename:   tmp,
		MaxSize:    500, // megabytes
		MaxBackups: 2,
		MaxAge:     30,   //days
		Compress:   true, // disabled by default
	}
	logger.SetOutput(std)

	job := &Job{
		ID:     id,
		name:   name,
		Type:   t,
		host:   host,
		cmd:    cmd,
		quit:   make(chan bool, 1),
		logger: logger,
		log:    tmp,
		std:    std,
		spec:   spec,
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
	j.logger.Infof("task name: [%s], cmd: '%s' running", j.Name(), j.cmd)
	client, err := transport.NewClient(j.host.Addr, j.host.Port, j.host.User, j.host.PassWord, []byte(j.host.KeyFile))
	if err != nil {
		j.logger.Errorf("error when new ssh client, err: %v", err)
		return
	}
	session, err := client.NewSessionWithPty(20, 20)
	if err != nil {
		j.logger.Errorf("create new session failed, err: %v", err)
	}
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
		// TODO retry
		err := session.RunTaskWithQuit(j.cmd, j.quit)
		if err != nil {
			j.logger.Errorf("error when run cmd, err: %v", err)
			j.UpdateStatus(JobStatusFatal)
			return
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
		log.Errorf("error when update job status, err: %v", err)
	}
}

func (j *Job) Close() {
	switch j.Type {
	case JobTypeCron:
		taskService.Remove(j.Name())
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

// Register 开始task & 添加到poll
func Register(id int, job *Job) error {
	if job.Status() == JobStatusRunning {
		log.Info("task is running already.")
		return errors.New("task is running already")
	}
	switch job.Type {
	case JobTypeCron:
		err := taskService.AddByJob(job.Name(), job.spec, job)
		if err != nil {
			log.Errorf("error when register job, err: %v", err)
			return err
		}
	case JobTypeTask:
		go job.Run()
	}

	TaskPoll.Store(id, job)

	return nil
}

// UnRegister 关闭task & 从poll删除
func UnRegister(id int) error {
	key, ok := TaskPoll.Load(id)
	if ok {
		job := key.(*Job)
		job.Close()
		defer TaskPoll.Delete(id)
	} else {
		return errors.New("job is stopped already")
	}
	log.Infof("un register a job: %d, success", id)

	return nil
}

func NewJobWithRegister(modelJob *models.Job, status string) error {
	realJob := NewJob(modelJob.Id, modelJob.Name, modelJob.Cmd, modelJob.Spec, JobType(modelJob.Type), &modelJob.Host)
	realJob.status.Store(JobStatus(status))
	if err := Register(modelJob.Id, realJob); err != nil {
		return err
	}

	log.Infof("register a new job, type: %s, name: %s, cmd: %s success", modelJob.Type, modelJob.Name, modelJob.Cmd)

	return nil
}

func RemoveJob(id int) error {
	log.Infof("recv singinl to remove job: %d", id)
	err := UnRegister(id)
	if err != nil {
		return err
	}
	err = models.DeleteJobById(id)
	if err != nil {
		return err
	}

	return nil
}

func StartJob(modelJob *models.Job) error {
	log.Infof("recv singinl to start job: %d", modelJob.Id)
	if key, ok := TaskPoll.Load(modelJob.Id); ok {
		job := key.(*Job)
		job.status.Store(JobStatusReady)
		err := Register(modelJob.Id, job)
		if err != nil {
			return err
		}
	} else {
		err := NewJobWithRegister(modelJob, string(JobStatusReady))
		if err != nil {
			return err
		}
	}
	return nil
}

func StopJob(id int) error {
	log.Infof("recv singinl to stop job: %d", id)
	err := UnRegister(id)
	if err != nil {
		return err
	}

	return nil
}

func initJobFromModels(modelJobs []*models.Job) {
	log.Info("init all job without stopped.")
	for _, modelJob := range modelJobs {
		err := NewJobWithRegister(modelJob, modelJob.Status)
		if err != nil {
			log.Errorf("error when register a new job, err: %v", err)
		}
	}
}
