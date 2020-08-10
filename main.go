package main

import (
	"github.com/astaxie/beego"
	_ "oms/routers"
)

func main() {
	beego.SetStaticPath("/images", "static/images")
	beego.SetStaticPath("/css", "static/css")
	beego.SetStaticPath("/js", "static/js")
	beego.Run()
}
