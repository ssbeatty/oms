package schedule

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"oms/models"
	"oms/pkg/utils"
	"os"
	"sync"
	"time"
)

var (
	taskService *Schedule
	TaskPoll    *utils.SafeMap
	onceInitJob sync.Once

	taskNum = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "task_register_nums",
		Help: "Current Num of Task.",
	})
)

func init() {
	TaskPoll = utils.NewSageMap()

	// add init tasks
	taskService = NewSchedule()
	taskService.Start()

	if err := taskService.AddByFunc("loop-status", "*/5 * * * *", GetHostStatus); err != nil {
		log.Println("init loop-status error!", err)
	}

	err := os.MkdirAll(DefaultTmpPath, fs.ModePerm)
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

	prometheus.MustRegister(taskNum)

	go func() {
		for {
			<-time.After(time.Second * 5)
			taskNum.Set(float64(TaskPoll.Length()))
		}
	}()
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
