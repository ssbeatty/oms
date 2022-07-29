/*
task job init
*/

package task

import (
	"oms/internal/models"
	"oms/internal/ssh"
	"os"
	"path/filepath"
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

// CronClearUploadFiles clear upload file
func (m *Manager) CronClearUploadFiles() {
	m.logger.Info("=====================Task CronClearUploadFiles start=====================")

	existStepFiles := make(map[string]struct{})
	playbooks, err := models.GetAllPlayBook()
	if err != nil {
		return
	}
	for _, playbook := range playbooks {
		for _, step := range playbook.StepsObj {
			for _, cache := range step.GetCaches() {
				existStepFiles[filepath.ToSlash(cache)] = struct{}{}
			}
		}
	}

	uploadPath := filepath.Join(m.config().App.DataPath, ssh.UploadPath)
	files, err := os.ReadDir(uploadPath)
	if err != nil {
		return
	}

	for _, file := range files {
		fPath := filepath.Join(uploadPath, file.Name())
		if _, ok := existStepFiles[filepath.ToSlash(fPath)]; !ok {
			m.logger.Infof("delete a upload cache file: %s", fPath)
			_ = os.Remove(fPath)
		}
	}

	m.logger.Info("===================== Task CronClearUploadFiles end =====================")
}
