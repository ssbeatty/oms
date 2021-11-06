/*
task job init
*/

package task

import "oms/internal/models"

// CronStatusJob 检查连接的Job
type CronStatusJob struct {
	engine *Manager
}

func (m *Manager) NewCronStatusJob() *CronStatusJob {
	return &CronStatusJob{
		engine: m,
	}
}

func (c *CronStatusJob) Run() {
	c.engine.logger.Info("======================Task CronStatusJob start======================")
	hosts, err := models.GetAllHostWithOutPreload()
	if err != nil {
		c.engine.logger.Errorf("error when GetAllHostWithOutPreload, err: %v", err)
	}
	for i := 0; i < len(hosts); i++ {
		go c.engine.sshManager.GetStatus(hosts[i])
	}
	c.engine.logger.Info("======================Task CronStatusJob end ======================")
}
