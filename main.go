package main

import (
	log "github.com/sirupsen/logrus"
	_ "oms/conf"
	"oms/routers"
	"oms/services/tasks"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {
	// add init tasks
	taskService := tasks.NewTaskService()
	taskService.Start()

	routers.InitGinServer()
}
