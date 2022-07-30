/*
Manager of cron | task , base on ssh Manager
*/

package task

import (
	"errors"
	"io/fs"
	"oms/internal/config"
	"oms/internal/models"
	"oms/internal/ssh"
	"oms/internal/utils"
	"oms/pkg/logger"
	"oms/pkg/schedule"
	"os"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
)

type Manager struct {
	// cron schedule engine
	taskService *schedule.Schedule
	// all cron & task in map
	taskPoll *utils.SafeMap
	onceJob  sync.Once
	logger   *logger.Logger
	cfg      atomic.Value

	// base
	sshManager *ssh.Manager
}

func NewManager(sshManager *ssh.Manager, cfg *config.Conf) *Manager {
	manager := &Manager{
		taskService: schedule.NewSchedule(),
		taskPoll:    utils.NewSafeMap(),
		onceJob:     sync.Once{},
		sshManager:  sshManager,
		logger:      logger.NewLogger("taskManager"),
	}

	manager.cfg.Store(cfg)

	return manager
}

func (m *Manager) config() *config.Conf {
	return m.cfg.Load().(*config.Conf)
}

// Init 启动crontab daemon, 注册全局任务, 创建日志文件夹并初始化job
func (m *Manager) Init() *Manager {
	// init schedule engine
	m.taskService.Start()

	// init all build in cron
	if err := m.taskService.AddByFunc("build-in-loop-status", "0 * * * * *", m.CronStatusJob, true); err != nil {
		m.logger.Errorf("init build-in-loop-status error: %v", err)
	}
	if err := m.taskService.AddByFunc("build-in-loop-clear-instance", "0 0 0 * * *", m.CronClearInstanceCache, true); err != nil {
		m.logger.Errorf("init build-in-loop-clear-instance: %v", err)
	}
	if err := m.taskService.AddByFunc("build-in-loop-clear-upload", "0 0 0 * * *", m.CronClearUploadFiles, true); err != nil {
		m.logger.Errorf("init build-in-loop-clear-upload: %v", err)
	}

	// path for job log
	err := os.MkdirAll(path.Join(m.config().App.DataPath, config.DefaultTmpPath), fs.ModePerm)
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

// ClearLogs 删除job的日志
func (m *Manager) ClearLogs(job *Job) error {
	err := os.RemoveAll(job.log)
	if err != nil {
		return err
	}
	return nil
}

// UnRegister 关闭task & 从poll删除
func (m *Manager) UnRegister(id int, clear bool) error {
	job, ok := m.GetJob(id)
	if ok {
		job.Close()
		if clear {
			err := m.ClearLogs(job)
			if err != nil {
				m.logger.Errorf("error when clear job logs, err: %v", err)
			}
		}
		defer m.taskPoll.Delete(id)
	} else {
		return errors.New("job is remove already")
	}
	m.logger.Infof("un register a job: %d, success", id)

	return nil
}

// NewJobWithRegister 新建一个job并注册到task poll
func (m *Manager) NewJobWithRegister(modelJob *models.Job, status string) error {
	hosts, err := models.ParseHostList(modelJob.ExecuteType, modelJob.ExecuteID)
	if err != nil {
		return err
	}

	realJob := m.NewJob(modelJob.Id, modelJob.Name, modelJob.Cmd, modelJob.Spec, modelJob.CmdType, JobType(modelJob.Type), hosts)
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
	err := m.UnRegister(id, true)
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
		jId := strconv.Itoa(job.ID)
		existed := m.taskService.IsExists(jId)
		if existed {
			return nil
		}
		err := m.taskService.AddByJob(jId, job.spec, job)
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
	} else {
		err := m.NewJobWithRegister(modelJob, string(JobStatusReady))
		if err != nil {
			return err
		}
	}
	return nil
}

// StopJob 从task poll删除job
func (m *Manager) StopJob(id int) error {
	m.logger.Infof("received signal to stop job: %d", id)
	job, ok := m.GetJob(id)
	if ok {
		job.Close()
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
		} else if JobStatus(modelJob.Status) == JobStatusDone && JobType(modelJob.Type) == JobTypeTask {
			// 任务执行结束了 不能每次启动都执行 仅当为task时 cron正常来说还是应该执行
			status = string(JobStatusStop)
		}
		err := m.NewJobWithRegister(modelJob, status)
		if err != nil {
			m.logger.Errorf("error when register a new job, err: %v", err)
		}
	}
}
