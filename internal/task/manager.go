/*
Manager of cron | task , base on ssh Manager
*/

package task

import (
	"errors"
	"github.com/ssbeatty/oms/internal/config"
	"github.com/ssbeatty/oms/internal/models"
	"github.com/ssbeatty/oms/internal/ssh"
	"github.com/ssbeatty/oms/pkg/logger"
	"github.com/ssbeatty/oms/pkg/schedule"
	"github.com/ssbeatty/oms/pkg/utils"
	"io/fs"
	"io/ioutil"
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

// 删除tmp的文件 可能是上传时阻塞没有删除
func (m *Manager) clearTmpPath() error {
	tmpPath := path.Join(m.config().App.DataPath, config.DefaultTmpPath)

	dir, err := ioutil.ReadDir(tmpPath)
	if err != nil {
		return err
	}
	for _, d := range dir {
		_ = os.RemoveAll(path.Join(tmpPath, d.Name()))
	}

	return nil
}

// Init 启动crontab daemon, 注册全局任务, 创建日志文件夹并初始化job
func (m *Manager) Init() *Manager {
	// init schedule engine
	m.taskService.Start()

	// init all build in cron
	if err := m.taskService.AddByFunc("build-in-loop-status", "0 */2 * * * *", m.CronStatusJob, true); err != nil {
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
		m.logger.Errorf("error when make tmp path, err: %v", err)
	}

	err = os.MkdirAll(path.Join(m.config().App.DataPath, config.DefaultTaskTmpPath), fs.ModePerm)
	if err != nil {
		m.logger.Errorf("error when make task tmp path, err: %v", err)
	}

	err = os.MkdirAll(path.Join(m.config().App.DataPath, config.UploadPath), fs.ModePerm)
	if err != nil {
		m.logger.Errorf("error when make task tmp path, err: %v", err)
	}

	err = m.clearTmpPath()
	if err != nil {
		m.logger.Errorf("error when clear tmp path, err: %v", err)
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

// NewRealJob 新建一个job
func (m *Manager) NewRealJob(modelJob *models.Job) (*Job, error) {
	hosts, err := models.ParseHostList(modelJob.ExecuteType, modelJob.ExecuteID)
	if err != nil {
		return nil, err
	}

	realJob := m.NewJob(
		modelJob.Id, modelJob.Name, modelJob.Cmd, modelJob.Spec, modelJob.CmdType, modelJob.CmdId, hosts)

	return realJob, nil
}

// NewRealJobWithRegister 新建一个job并注册到task poll
func (m *Manager) NewRealJobWithRegister(modelJob *models.Job, status string) (*Job, error) {
	realJob, err := m.NewRealJob(modelJob)
	if err != nil {
		return nil, err
	}

	m.taskPoll.Store(modelJob.Id, realJob)
	realJob.UpdateStatus(JobStatus(status))

	m.logger.Infof("register a new job, type: %s, name: %s, cmd: %s success", modelJob.Type, modelJob.Name, modelJob.Cmd)

	return realJob, nil
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

// ScheduleJob 开始任务的调度
func (m *Manager) ScheduleJob(job *Job) error {
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

	return nil
}

// ExecJob 执行一次任务
func (m *Manager) ExecJob(modelJob *models.Job) error {
	var (
		err error
	)

	m.logger.Infof("received signal to exec job once: %s", modelJob.Name)
	realJob, ok := m.GetJob(modelJob.Id)
	if !ok {
		realJob, err = m.NewRealJob(modelJob)
		if err != nil {
			return err
		}
	}

	realJob.exec()

	return nil

}

// StartJob 从models注册并启动调度
func (m *Manager) StartJob(modelJob *models.Job) error {
	var (
		err error
	)

	m.logger.Infof("received signal to start job: %s", modelJob.Name)
	realJob, ok := m.GetJob(modelJob.Id)
	if ok {
		realJob.UpdateStatus(JobStatusSchedule)
	} else {
		realJob, err = m.NewRealJobWithRegister(modelJob, string(JobStatusSchedule))
		if err != nil {
			return err
		}
	}

	err = m.ScheduleJob(realJob)
	if err != nil {
		return err
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
		realJob, err := m.NewRealJobWithRegister(modelJob, status)
		if err != nil {
			m.logger.Errorf("error when register a new job, err: %v", err)
		}
		// 如果是停止的任务开机时不启动
		if JobStatus(status) == JobStatusStopped {
			continue
		}
		err = m.ScheduleJob(realJob)
		if err != nil {
			m.logger.Errorf("error when start job: %s, err: %v", realJob.Name(), err)
		}
	}
}
