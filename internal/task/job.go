package task

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
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

	MarkText  = "###mark###"
	ErrorText = "[error]"
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
	steps   []ssh.Step
}

func (m *Manager) NewJob(id int, name, cmd, spec, cmdType string, t JobType, host []*models.Host) *Job {
	var (
		steps []ssh.Step
	)

	if name == "" {
		name = strconv.Itoa(id)
	}
	log := filepath.Join(path.Join(m.config().App.DataPath, config.DefaultTmpPath), fmt.Sprintf("%d-%s", id, name))

	if cmdType == ssh.CMDTypePlayer {
		cid, err := strconv.Atoi(cmd)
		if err != nil {
			return nil
		}
		modPlayer, err := models.GetPlayBookById(cid)
		if err != nil {
			return nil
		}
		steps, err = m.sshManager.ParseSteps(modPlayer.Steps)
		if err != nil {
			return nil
		}
	}

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
		steps:   steps,
	}
	job.UpdateStatus(JobStatusReady)

	return job
}

func (j *Job) runPlayer(client *transport.Client) ([]byte, error) {
	player := ssh.NewPlayer(client, j.steps, true)

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

func (j *Job) run(client *transport.Client, host *models.Host, wg *sync.WaitGroup, std io.Writer) {
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
		_, _ = fmt.Fprintf(std, "%s[host_id:%d]%s\n", MarkText, host.Id, ErrorText)
		_, err = std.Write(output)
		return
	}

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

	j.UpdateStatus(JobStatusRunning)

	var wg sync.WaitGroup
	wg.Add(len(j.hosts))

	instance, err := j.createInstance()
	if err != nil {
		j.engine.logger.Errorf("error when create instance, err: %v", err)
		return
	}
	std := &bytes.Buffer{}

	_ = instance.UpdateStatus(models.InstanceStatusRunning)
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

	_ = instance.Done()
	_ = ioutil.WriteFile(instance.LogPath, std.Bytes(), fs.ModePerm)
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
	j.UpdateStatus(JobStatusDone)
}

func (j *Job) Name() string {
	return j.name
}

func (j *Job) Cmd() string {
	return j.cmd
}
