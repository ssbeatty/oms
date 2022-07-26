package task

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"oms/internal/models"
	"oms/internal/utils"
	"oms/pkg/transport"
	"os"
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

	InstanceStatusRunning = "running"
	InstanceStatusDone    = "done"

	MarkText  = "###mark###"
	ErrorText = "[error]"
)

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
	spec      string
	engine    *Manager
}

func (m *Manager) NewJob(id int, name, cmd, spec string, t JobType, host []*models.Host) *Job {
	if name == "" {
		name = strconv.Itoa(id)
	}
	log := filepath.Join(path.Join(m.config().App.DataPath, DefaultTmpPath), fmt.Sprintf("%d-%s", id, name))

	job := &Job{
		ID:     id,
		name:   name,
		Type:   t,
		hosts:  host,
		cmd:    cmd,
		spec:   spec,
		engine: m,
		log:    log,
	}
	job.UpdateStatus(JobStatusReady)

	return job
}

func (j *Job) run(client *transport.Client, host *models.Host, wg *sync.WaitGroup, std io.Writer) {
	defer wg.Done()

	session, err := client.NewPty()
	if err != nil {
		j.engine.logger.Errorf("create new session failed, host name: %s, err: %v", err)
	}

	output, err := session.Sudo(j.cmd, host.PassWord)
	if err != nil {
		j.engine.logger.Errorf("error when run cmd: %v, host name: %s, msg: %s", err, host.Name, output)
		_, _ = fmt.Fprintf(std, "%s[host_id:%d]%s\n", MarkText, host.Id, ErrorText)
		_, err = std.Write(output)
		return
	}
	defer session.Close()

	_, _ = fmt.Fprintf(std, "%s[host_id:%d]\n", MarkText, host.Id)
	_, err = std.Write(output)
	if err != nil {
		j.engine.logger.Debugf("error write outputs, err: %v", err)
	}

}

func (j *Job) Run() {
	if j.Status() == JobStatusStop {
		return
	}
	j.engine.logger.Infof("job, name: %s, cmd: %s, running.", j.name, j.cmd)
	defer func() {
		if j.Status() == JobStatusRunning {
			j.UpdateStatus(JobStatusDone)
		}
		j.engine.logger.Debugf("job, name: %s, cmd: %s, done.", j.name, j.cmd)
	}()

	j.startTime = time.Now().Local()
	j.UpdateStatus(JobStatusRunning)

	var wg sync.WaitGroup
	wg.Add(len(j.hosts))

	instance, err := j.createInstance()
	if err != nil {
		j.engine.logger.Errorf("error when create instance, err: %v", err)
		return
	}
	std := &bytes.Buffer{}

	_ = instance.UpdateStatus(InstanceStatusRunning)
	for _, host := range j.hosts {
		client, err := j.engine.sshManager.NewClient(host)

		if err != nil {
			j.engine.logger.Errorf("error when new ssh client, host name: %s, err: %v", err, host.Name)

			_, _ = fmt.Fprintf(std, "%s[host_id:%d]%s\n", MarkText, host.Id, ErrorText)
			_, _ = fmt.Fprintf(std, "[FATIL ERROR]: %s: \n", err.Error())
			continue
		}

		go j.run(client, host, &wg, std)
	}

	wg.Wait()

	_ = instance.UpdateStatus(InstanceStatusDone)
	_ = ioutil.WriteFile(instance.LogPath, std.Bytes(), fs.ModePerm)
}

func (j *Job) createInstance() (*models.TaskInstance, error) {
	now := time.Now().Local()
	instance, err := models.InsertTaskInstance(j.ID, j.startTime, now, "")
	if err != nil {
		return nil, err
	}
	logPath := instance.GenerateLogPath(j.log)
	if exist, _ := utils.PathExists(path.Dir(logPath)); !exist {
		_ = os.MkdirAll(path.Dir(logPath), fs.ModePerm)
	}

	err = models.UpdateTaskInstanceLogTrace(instance, logPath)
	if err != nil {
		return nil, err
	}

	return instance, nil
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
		j.engine.taskService.Remove(strconv.Itoa(j.ID))
	}
	j.UpdateStatus(JobStatusDone)
}

func (j *Job) Name() string {
	return j.name
}

func (j *Job) Cmd() string {
	return j.cmd
}
