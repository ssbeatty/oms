package main

import (
	"encoding/gob"
	"github.com/astaxie/beego"
	_ "github.com/astaxie/beego/session/redis"
	"github.com/astaxie/beego/toolbox"
	"oms/models"
	_ "oms/routers"
	"oms/tasks"
)

func main() {
	// static files
	beego.SetStaticPath("/static", "static")

	// tasks
	toolbox.StartTask()
	defer toolbox.StopTask()

	getHostStatus := toolbox.NewTask("getHostStatus", "0 */5 * * * *", tasks.GetHostStatus)
	toolbox.AddTask("getHostStatus", getHostStatus)

	// Register
	gob.Register(models.User{})

	// main func
	beego.Run()

}
