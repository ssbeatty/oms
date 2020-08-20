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
	beego.Router("/browse", &controllers.MainController{}, "get:FileBrowsePage")
	//ws ssh
	beego.Router("/ssh", &controllers.MainController{}, "get:SshPage")
	beego.Router("/ws/ssh/:id", &controllers.WebSocketController{})
	//tools
	beego.Router("/tools/cmd", &controllers.ToolController{}, "get:RunCmd")
	beego.Router("/tools/upload", &controllers.ToolController{}, "post:FileUpload")
	beego.Router("/tools/browse", &controllers.ToolController{}, "get:GetPathInfo")
	beego.Router("/tools/download", &controllers.ToolController{}, "get:DownLoadFile")
}
