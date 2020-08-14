package main

import (
	"github.com/astaxie/beego"
	_ "oms/routers"
)

func main() {
	// static files
	beego.SetStaticPath("/static", "static")

	// main func
	beego.Run()
}
