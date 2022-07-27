/*
task job init
*/

package task

import (
	"oms/internal/models"
	"time"
)

// CronStatusJob 获取host状态并更新到数据库
func (m *Manager) CronStatusJob() {
	m.logger.Info("======================Task CronStatusJob start======================")
	hosts, err := models.GetAllHostWithOutPreload()
	if err != nil {
		m.logger.Errorf("error when GetAllHostWithOutPreload, err: %v", err)
	}
	for i := 0; i < len(hosts); i++ {
		go m.sshManager.GetStatus(hosts[i])
	}
	m.logger.Info("======================Task CronStatusJob end ======================")
}

// CronClearInstanceCache clear instance cache
func (m *Manager) CronClearInstanceCache() {
	m.logger.Info("====================Task CronClearInstanceCache start====================")
	err := models.ClearInstance(time.Now().Local().Add(-m.config().App.TempDate), 0)
	if err != nil {
		return
	}
	m.logger.Info("==================== Task CronClearInstanceCache end ====================")
}
