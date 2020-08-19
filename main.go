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

	test1 := toolbox.NewTask("test1", "0 0 * * * *", tasks.Task1)
	toolbox.AddTask("test1", test1)

	// main func
	beego.Run()

}
