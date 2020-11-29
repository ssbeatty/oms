package main

import (
	"oms/routers"
)

func main() {
	//@TODO task
	//toolbox.StartTask()
	//defer toolbox.StopTask()
	//
	//getHostStatus := toolbox.NewTask("getHostStatus", "0 */5 * * * *", tasks.GetHostStatus)
	//toolbox.AddTask("getHostStatus", getHostStatus)

	routers.InitGinServer()
}
