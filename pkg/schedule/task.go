package schedule

import (
	log "github.com/sirupsen/logrus"
	"oms/models"
)

var taskService *Schedule

func init() {
	// add init tasks
	taskService = NewSchedule()
	taskService.Start()

	if err := taskService.AddByFunc("loop-status", "*/5 * * * *", GetHostStatus); err != nil {
		log.Println("init loop-status error!", err)
	}
}

func GetHostStatus() {
	log.Println("======================Task GetHostStatus start======================")
	hosts, err := models.GetAllHost()
	if err != nil {
		log.Errorf("GetHostStatus error when GetAllHost, err: %v", err)
	}
	for i := 0; i < len(hosts); i++ {
		go models.GetStatus(hosts[i])
	}
	log.Println("======================Task GetHostStatus end ======================")
}
