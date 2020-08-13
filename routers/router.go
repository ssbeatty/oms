package routers

import (
	"github.com/astaxie/beego"
	"oms/controllers"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/host", &controllers.HostController{})

	//ws ssh
	beego.Router("/ws/ssh", &controllers.WebSocketController{})
}
