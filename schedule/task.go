package schedule

import (
	log "github.com/sirupsen/logrus"
	"oms/models"
)

func GetHostStatus() {
	log.Println("======================Task GetHostStatus start======================")
	hosts := models.GetAllHost()
	for i, _ := range hosts {
		go models.GetStatus(hosts[i])
	}
	log.Println("======================Task GetHostStatus end ======================")
	return
}
