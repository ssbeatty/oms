package main

import (
	"oms/routers"
	"oms/services/tasks"
)

func main() {
	// add init tasks
	taskService := tasks.NewTaskService()
	taskService.Start()

	routers.InitGinServer()
}
