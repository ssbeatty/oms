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

	clearCache := toolbox.NewTask("clearCache", "0 0 * * * *", tasks.ClearCache)
	toolbox.AddTask("clearCache", clearCache)

	// main func
	beego.Run()

}
