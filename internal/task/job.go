package task

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"oms/internal/config"
	"oms/internal/models"
	"oms/internal/ssh"
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

	JobStatusReady   JobStatus = "ready"
	JobStatusRunning JobStatus = "running"
	JobStatusStop    JobStatus = "stop"
	JobStatusDone    JobStatus = "done"

	MarkText     = "###mark###"
	ErrorText    = "[error]"
	DoneMartText = "###done###"
)

// Job is cron task or long task
type Job struct {
	ID      int
	name    string
	Type    JobType
	hosts   []*models.Host
	cmd     string
	cmdType string
	status  atomic.Value
	log     string // log path
	spec    string
	engine  *Manager
	cmdId   int
}

func NewSyncBuffer(fd *os.File) *syncBuffer {

	buf := &syncBuffer{
		fd:   fd,
		quit: make(chan struct{}, 1),
	}

	go buf.flushComboOutput()

	return buf
}

type syncBuffer struct {
	bytes.Buffer
	fd   *os.File
	quit chan struct{}
	mu   sync.Mutex
}

func (w *syncBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.Buffer.Write(p)
}

func (w *syncBuffer) WriteWithMsg(output []byte, msg string) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, _ = fmt.Fprintf(&w.Buffer, msg)

	return w.Buffer.Write(output)
}

func (w *syncBuffer) flush() {
	if w.Buffer.Len() != 0 {
		_, err := w.fd.Write(w.Buffer.Bytes())
		if err != nil {
			return
		}
		w.Buffer.Reset()
	}
}

func (w *syncBuffer) Close() {
	w.flush()

	w.quit <- struct{}{}
	defer w.fd.Close()
}

func (w *syncBuffer) flushComboOutput() {
	tick := time.NewTicker(time.Millisecond * time.Duration(120))

	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			w.flush()
		case <-w.quit:
			return
		}
	}
}

func (m *Manager) NewJob(id int, name, cmd, spec, cmdType string, cmdId int, t JobType, host []*models.Host) *Job {

	if name == "" {
		name = strconv.Itoa(id)
	}
	log := filepath.Join(path.Join(m.config().App.DataPath, config.DefaultTaskTmpPath), fmt.Sprintf("%d-%s", id, name))

	job := &Job{
		ID:      id,
		name:    name,
		Type:    t,
		hosts:   host,
		cmd:     cmd,
		cmdType: cmdType,
		spec:    spec,
		engine:  m,
		log:     log,
		cmdId:   cmdId,
	}
	job.UpdateStatus(JobStatusReady)

	return job
}

func (j *Job) runPlayer(client *transport.Client) ([]byte, error) {
	modPlayer, err := models.GetPlayBookById(j.cmdId)
	if err != nil {
		return nil, err
	}
	steps, err := j.engine.sshManager.ParseSteps(modPlayer.Steps)
	if err != nil {
		return nil, err
	}
	player := ssh.NewPlayer(client, steps, true)

	return player.Run(context.Background())

}

func (j *Job) runCmd(client *transport.Client) ([]byte, error) {
	session, err := client.NewPty()
	if err != nil {
		j.engine.logger.Errorf("create new session failed, host name: %s, err: %v", err)
	}
	defer session.Close()

	return session.Sudo(j.cmd, client.Conf.Password)
}

func (j *Job) run(client *transport.Client, host *models.Host, wg *sync.WaitGroup, std *syncBuffer) {
	defer wg.Done()

	var (
		err    error
		output []byte
	)

	switch j.cmdType {
	case ssh.CMDTypePlayer:
		output, err = j.runPlayer(client)
	default:
		output, err = j.runCmd(client)
	}

	if err != nil {
		j.engine.logger.Errorf("error when run cmd: %v, host name: %s, msg: %s", err, host.Name, output)
		_, err = std.WriteWithMsg(output, fmt.Sprintf("%s[host_id:%d]%s\n", MarkText, host.Id, ErrorText))
		return
	}

	_, err = std.WriteWithMsg(output, fmt.Sprintf("%s[host_id:%d]\n", MarkText, host.Id))
	if err != nil {
		j.engine.logger.Debugf("error write outputs, err: %v", err)
	}

}

func (j *Job) Run() {
	if j.Status() == JobStatusStop {
		return
	}
	j.engine.logger.Debugf("job, name: %s, cmd: %s, running.", j.name, j.cmd)
	defer func() {
		if j.Status() == JobStatusRunning {
			j.UpdateStatus(JobStatusDone)
		}
		j.engine.logger.Debugf("job, name: %s, cmd: %s, done.", j.name, j.cmd)
	}()

	j.UpdateStatus(JobStatusRunning)

	var wg sync.WaitGroup
	wg.Add(len(j.hosts))

	instance, err := j.createInstance()
	if err != nil {
		j.engine.logger.Errorf("error when create instance, err: %v", err)
		return
	}

	fd, err := os.OpenFile(instance.LogPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, fs.ModePerm)
	if err != nil {
		j.engine.logger.Errorf("error when open tmp file, err: %v", err)
		return
	}

	std := NewSyncBuffer(fd)

	defer std.Close()

	_ = instance.UpdateStatus(models.InstanceStatusRunning)
	for _, host := range j.hosts {
		client, err := j.engine.sshManager.NewClientWithSftp(host)

		if err != nil {
			j.engine.logger.Errorf("error when new ssh client, host name: %s, err: %v", err, host.Name)

			_, _ = fmt.Fprintf(std, "%s[host_id:%d]%s\n[FATIL ERROR]: %s: \n", MarkText, host.Id, ErrorText, err.Error())
			continue
		}

		go j.run(client, host, &wg, std)
	}

	wg.Wait()

	_, _ = fmt.Fprintf(std, "%s\n", DoneMartText)

	_ = instance.Done()
}

func (j *Job) createInstance() (*models.TaskInstance, error) {
	now := time.Now().Local()
	instance, err := models.InsertTaskInstance(j.ID, now)
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
	j.UpdateStatus(JobStatusStop)
}

func (j *Job) Name() string {
	return j.name
}

func (j *Job) Cmd() string {
	return j.cmd
}
