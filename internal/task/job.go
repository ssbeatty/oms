package task

import (
	"bytes"
	"fmt"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"oms/internal/models"
	"oms/pkg/transport"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type JobType string
type JobStatus string

const (
	JobTypeCron     JobType = "cron"
	JobTypeTask     JobType = "task"      // run once
	JobTypeLongTask JobType = "long_task" // run daemon
	DefaultTmpPath          = "tmp"

	JobStatusReady   JobStatus = "ready"
	JobStatusRunning JobStatus = "running"
	JobStatusStop    JobStatus = "stop"
	JobStatusDone    JobStatus = "done"
)

type BufferCloser struct {
	bytes.Buffer
}

func (bc *BufferCloser) Close() error {
	bc.Reset()

	return nil
}

// Job is cron task or long task
type Job struct {
	ID        int
	name      string
	Type      JobType
	hosts     []*models.Host
	cmd       string
	status    atomic.Value
	log       string // log path
	startTime time.Time
	std       io.WriteCloser
	spec      string
	engine    *Manager
}

func (m *Manager) NewJob(id int, name, cmd, spec string, t JobType, host []*models.Host) *Job {
	if name == "" {
		name = strconv.Itoa(id)
	}
	tmp := filepath.Join(path.Join(m.config().App.DataPath, DefaultTmpPath), name, fmt.Sprintf("%s.log", name))
	std := &lumberjack.Logger{
		Filename:   tmp,
		MaxSize:    20, // megabytes
		MaxBackups: 3,
		MaxAge:     20,   //days
		Compress:   true, // disabled by default
	}

	job := &Job{
		ID:     id,
		name:   name,
		Type:   t,
		hosts:  host,
		cmd:    cmd,
		log:    tmp,
		std:    std,
		spec:   spec,
		engine: m,
	}
	job.UpdateStatus(JobStatusReady)

	return job
}

func (j *Job) _log(level, format string, elems ...interface{}) {
	_, _ = fmt.Fprintln(
		j.std, fmt.Sprintf(fmt.Sprintf("%s [%s] %s", time.Now().Format(time.RFC3339), level, format), elems...))
}

func (j *Job) info(format string, elems ...interface{}) {
	j._log("info", format, elems...)
}

func (j *Job) error(format string, elems ...interface{}) {
	j._log("error", format, elems...)
}

func (j *Job) run(client *transport.Client, host *models.Host, wg *sync.WaitGroup) {
	defer wg.Done()

	session, err := client.NewPty()
	if err != nil {
		j.error("create new session failed, host name: %s, err: %v", err)
	}

	output, err := session.Sudo(j.cmd, host.PassWord)
	if err != nil {
		j.error("error when run cmd: %v, host name: %s, msg: %s", err, output)
		return
	}
	defer session.Close()

	_, _ = fmt.Fprintf(j.std, "[host name: %s, addr: %s]: \n", host.Name, host.Addr)
	_, err = j.std.Write(output)
	if err != nil {
		j.engine.logger.Debugf("error write outputs, err: %v", err)
	}

}

func (j *Job) Run() {
	if j.Status() == JobStatusStop {
		return
	}
	j.engine.logger.Infof("job, name: %s, cmd: %s, running.", j.name, j.cmd)
	j.info("job, name: %s, cmd: %s, running.", j.name, j.cmd)
	defer func() {
		if j.Status() == JobStatusRunning {
			j.UpdateStatus(JobStatusDone)
		}
		j.engine.logger.Debugf("job, name: %s, cmd: %s, done.", j.name, j.cmd)
		j.info("job, name: %s, cmd: %s, done.", j.name, j.cmd)
	}()

	j.startTime = time.Now().Local()
	j.UpdateStatus(JobStatusRunning)

	var wg sync.WaitGroup

	wg.Add(len(j.hosts))
	for _, host := range j.hosts {
		client, err := j.engine.sshManager.NewClient(host)
		if err != nil {
			j.error("error when new ssh client, host name: %s, err: %v", err, host.Name)
			continue
		}

		go j.run(client, host, &wg)
	}

	wg.Wait()
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
	// close logs
	_ = j.std.Close()

	switch j.Type {
	case JobTypeCron:
		j.engine.taskService.Remove(j.Name())
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
