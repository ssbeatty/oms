package controllers

import (
	"github.com/astaxie/beego/logs"
)

var Logger = logs.GetLogger()

func init() {
	logs.SetLogger("console")
}
