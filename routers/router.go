package routers

import (
	"github.com/astaxie/beego"
	"oms/controllers"
)

func init() {
	//resources
	beego.Router("/", &controllers.MainController{})
	beego.Router("/host", &controllers.HostController{})
	beego.Router("/host/:id", &controllers.HostController{}, "get:GetOneHost")
	beego.Router("/group", &controllers.GroupController{})
	beego.Router("/group/:id", &controllers.GroupController{}, "get:GetOneGroup")
	beego.Router("/tag", &controllers.TagController{})
	beego.Router("/tag/:id", &controllers.TagController{}, "get:GetOneTag")

	//page
	beego.Router("/groupPage", &controllers.MainController{}, "get:GroupPage")
	beego.Router("/about", &controllers.MainController{}, "get:AboutPage")
	beego.Router("/shell", &controllers.MainController{}, "get:ShellPage")
	beego.Router("/file", &controllers.MainController{}, "get:FilePage")
	//ws ssh
	beego.Router("/ssh/:id", &controllers.MainController{}, "get:SshPage")
	beego.Router("/ws/ssh/:id", &controllers.WebSocketController{})
}
