/*
task job init
*/

package task

import "oms/internal/models"

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
