package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/toolbox"
	_ "oms/routers"
	"oms/tasks"
)

func main() {
	// static files
	beego.SetStaticPath("/static", "static")

	// tasks
	toolbox.StartTask()
	defer toolbox.StopTask()

	clearCache := toolbox.NewTask("clearCache", "0 0 */2 * * *", tasks.ClearCache)
	toolbox.AddTask("clearCache", clearCache)
	getHostStatus := toolbox.NewTask("getHostStatus", "0 */5 * * * *", tasks.GetHostStatus)
	toolbox.AddTask("getHostStatus", getHostStatus)

	// main func
	beego.Run()

}
