/*
task job init
*/

package task

import (
	"github.com/ssbeatty/oms/internal/config"
	"github.com/ssbeatty/oms/internal/models"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultTempDate = 14 * 24 * time.Hour
)

// CronStatusJob 获取host状态并更新到数据库
func (m *Manager) CronStatusJob() {
	hosts, err := models.GetAllHostWithOutPreload()
	if err != nil {
		m.logger.Errorf("error when GetAllHostWithOutPreload, err: %v", err)
	}
	for i := 0; i < len(hosts); i++ {
		go m.sshManager.GetStatus(hosts[i])
	}
}

// CronClearInstanceCache clear instance cache
func (m *Manager) CronClearInstanceCache() {
	tempDate := DefaultTempDate
	if int(m.config().App.TempDate) != 0 {
		tempDate = m.config().App.TempDate
	}
	err := models.ClearInstance(time.Now().Local().Add(-tempDate), 0)
	if err != nil {
		return
	}
}

// CronClearUploadFiles clear upload file
func (m *Manager) CronClearUploadFiles() {

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

	uploadPath := filepath.Join(m.config().App.DataPath, config.UploadPath)
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

}
