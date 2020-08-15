package routers

import (
	"github.com/astaxie/beego"
	"oms/controllers"
)

func init() {
	//resources
	beego.Router("/", &controllers.MainController{})
	beego.Router("/host", &controllers.HostController{})
	beego.Router("/group", &controllers.GroupController{})
	beego.Router("/tag", &controllers.TagController{})

	//page
	beego.Router("/groupPage", &controllers.GroupPathController{})
	//ws ssh
	beego.Router("/ws/ssh", &controllers.WebSocketController{})
}
