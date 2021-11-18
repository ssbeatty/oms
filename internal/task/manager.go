/*
Manager of cron | task , base on ssh Manager
*/

package task

import (
	"errors"
	"io/fs"
	"oms/internal/models"
	"oms/internal/ssh"
	"oms/internal/utils"
	"oms/pkg/logger"
	"oms/pkg/schedule"
	"os"
	"sync"
)

type Manager struct {
	// cron schedule engine
	taskService *schedule.Schedule
	// all cron & task in map
	taskPoll *utils.SafeMap
	onceJob  sync.Once
	logger   *logger.Logger

	// base
	sshManager *ssh.Manager
}

func NewManager(sshManager *ssh.Manager) *Manager {
	return &Manager{
		taskService: schedule.NewSchedule(),
		taskPoll:    utils.NewSafeMap(),
		onceJob:     sync.Once{},
		sshManager:  sshManager,
		logger:      logger.NewLogger("taskManager"),
	}
}

// Init 启动crontab daemon, 注册全局任务, 创建日志文件夹并初始化job
func (m *Manager) Init() *Manager {
	// init schedule engine
	m.taskService.Start()

	// init all build in cron
	if err := m.taskService.AddByFunc("build-in-loop-status", "*/5 * * * *", m.CronStatusJob); err != nil {
		m.logger.Errorf("init build-in-loop-status error: %v", err)
	}

	// path for job log
	err := os.MkdirAll(DefaultTmpPath, fs.ModePerm)
	if err != nil {
		m.logger.Errorf("error when make all tmp path, err: %v", err)
	}

	// once do init all job in database
	m.onceJob.Do(func() {
		jobs, err := models.GetAllJob()
		if err != nil {
			m.logger.Errorf("error when get all job, err: %v", err)
		}
		m.initJobFromModels(jobs)
	})

	return m
}

// GetJobList 获取task poll对象
func (m *Manager) GetJobList() *utils.SafeMap {
	return m.taskPoll
}

// Register 开始task & 添加到poll
func (m *Manager) Register(id int, job *Job) error {
	if job.Status() == JobStatusRunning {
		m.logger.Info("task is running already.")
		return errors.New("task is running already")
	}
	err := m.startJob(job)
	if err != nil {
		return err
	}

	m.taskPoll.Store(id, job)

	return nil
}

// UnRegister 关闭task & 从poll删除
func (m *Manager) UnRegister(id int) error {
	job, ok := m.GetJob(id)
	if ok {
		job.Close()
		defer m.taskPoll.Delete(id)
	} else {
		return errors.New("job is stopped already")
	}
	m.logger.Infof("un register a job: %d, success", id)

	return nil
}

// NewJobWithRegister 新建一个job并注册到task poll
func (m *Manager) NewJobWithRegister(modelJob *models.Job, status string) error {
	realJob := m.NewJob(modelJob.Id, modelJob.Name, modelJob.Cmd, modelJob.Spec, JobType(modelJob.Type), &modelJob.Host)
	realJob.UpdateStatus(JobStatus(status))

	if err := m.Register(modelJob.Id, realJob); err != nil {
		return err
	}

	m.logger.Infof("register a new job, type: %s, name: %s, cmd: %s success", modelJob.Type, modelJob.Name, modelJob.Cmd)

	return nil
}

// RemoveJob 从task poll删除，并删除model
func (m *Manager) RemoveJob(id int) error {
	m.logger.Infof("received signal to remove job: %d", id)
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

func (m *Manager) startJob(job *Job) error {
	switch job.Type {
	case JobTypeCron:
		existed := m.taskService.IsExists(job.Name())
		if existed {
			return nil
		}
		err := m.taskService.AddByJob(job.Name(), job.spec, job)
		if err != nil {
			m.logger.Errorf("error when register job, err: %v", err)
			return err
		}
	case JobTypeTask:
		switch job.Status() { //nolint:exhaustive
		case JobStatusReady, JobStatusDone:
			go job.Run()
		default:
			return nil
		}
	}
	return nil
}

// StartJob 从models注册并启动job
func (m *Manager) StartJob(modelJob *models.Job) error {
	m.logger.Infof("received signal to start job: %d", modelJob.Id)
	job, ok := m.GetJob(modelJob.Id)
	if ok {
		job.UpdateStatus(JobStatusReady)
		err := m.startJob(job)
		if err != nil {
			return err
		}
		return nil
	}
	err := m.NewJobWithRegister(modelJob, string(JobStatusReady))
	if err != nil {
		return err
	}
	return nil
}

// StopJob 从task poll删除job
func (m *Manager) StopJob(id int) error {
	m.logger.Infof("received signal to stop job: %d", id)
	err := m.UnRegister(id)
	if err != nil {
		return err
	}

	return nil
}

// GetJob 从task poll获取job
func (m *Manager) GetJob(id int) (*Job, bool) {
	if key, ok := m.taskPoll.Load(id); ok {
		return key.(*Job), true
	}
	return nil, false
}

// initJobFromModels 从数据库加载所有的job并继承状态
func (m *Manager) initJobFromModels(modelJobs []*models.Job) {
	m.logger.Info("init all job without stopped or fatal.")
	for _, modelJob := range modelJobs {
		var status = modelJob.Status
		// running 一般是没有正常退出 每次启动除了stop和fatal都要ready
		if JobStatus(modelJob.Status) == JobStatusRunning {
			status = string(JobStatusReady)
		}
		err := m.NewJobWithRegister(modelJob, status)
		if err != nil {
			m.logger.Errorf("error when register a new job, err: %v", err)
		}
	}
}
