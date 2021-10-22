package schedule

import (
	log "github.com/sirupsen/logrus"
	"oms/models"
)

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
