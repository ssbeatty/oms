package logger

import "github.com/astaxie/beego/logs"

var Logger = logs.GetLogger()

func init() {
	//logs.SetLogger("console")
	logs.EnableFuncCallDepth(true)
	logs.Async()
	logs.Async(1e3)
	logs.SetLogger(logs.AdapterFile,`{"filename":"oms.log","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":10,"color":true}`)
}
