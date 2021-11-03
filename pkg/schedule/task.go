package schedule

import (
	log "github.com/sirupsen/logrus"
	"oms/models"
	"os"
	"sync"
)

var taskService *Schedule
var TaskPoll *sync.Map
var onceInitJob sync.Once

func init() {
	TaskPoll = &sync.Map{}

	// add init tasks
	taskService = NewSchedule()
	taskService.Start()

	if err := taskService.AddByFunc("loop-status", "*/5 * * * *", GetHostStatus); err != nil {
		log.Println("init loop-status error!", err)
	}

	err := os.MkdirAll(DefaultTmpPath, 0644)
	if err != nil {
		log.Errorf("error when make all tmp path, err: %v", err)
		return
	}

	onceInitJob.Do(func() {
		jobs, err := models.GetAllJob()
		if err != nil {
			log.Errorf("error when get all job, err: %v", err)
		}
		initJobFromModels(jobs)
	})

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
