package task

import (
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
	taskService *schedule.Schedule
	taskPoll    *utils.SafeMap
	onceJob     sync.Once
	logger      *logger.Logger

	sshManager *ssh.Manager
}

type CronStatusJob struct {
	engine *Manager
}

func (m *Manager) NewCronJob() *CronStatusJob {
	return &CronStatusJob{
		engine: m,
	}
}

func (c *CronStatusJob) Run() {
	c.engine.logger.Info("======================Task GetHostStatus start======================")
	hosts, err := models.GetAllHost()
	if err != nil {
		c.engine.logger.Errorf("error when GetAllHost, err: %v", err)
	}
	for i := 0; i < len(hosts); i++ {
		go c.engine.sshManager.GetStatus(hosts[i])
	}
	c.engine.logger.Info("======================Task GetHostStatus end ======================")
}

func NewManager(sshManager *ssh.Manager) *Manager {
	return &Manager{
		taskService: schedule.NewSchedule(),
		taskPoll:    utils.NewSageMap(),
		onceJob:     sync.Once{},
		sshManager:  sshManager,
		logger:      logger.NewLogger("taskManager"),
	}
}

func (m *Manager) Init() *Manager {
	m.taskService.Start()

	statusJob := m.NewCronJob()
	if err := m.taskService.AddByJob("loop-status", "*/5 * * * *", statusJob); err != nil {
		m.logger.Errorf("init loop-status error!", err)
	}

	err := os.MkdirAll(DefaultTmpPath, fs.ModePerm)
	if err != nil {
		m.logger.Errorf("error when make all tmp path, err: %v", err)
	}

	m.onceJob.Do(func() {
		jobs, err := models.GetAllJob()
		if err != nil {
			m.logger.Errorf("error when get all job, err: %v", err)
		}
		m.initJobFromModels(jobs)
	})

	return m
}
