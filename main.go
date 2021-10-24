package main

import (
	log "github.com/sirupsen/logrus"
	_ "oms/conf"
	"oms/pkg/schedule"
	"oms/routers"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {
	// add init tasks
	taskService := schedule.NewSchedule()
	taskService.Start()

	if err := taskService.AddByFunc("loop-status", "*/5 * * * *", schedule.GetHostStatus); err != nil {
		log.Println("init loop-status error!", err)
	}

	routers.InitGinServer()
}
