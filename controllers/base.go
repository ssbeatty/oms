package controllers

import (
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
)

var o = orm.NewOrm()

var Logger = logs.GetLogger()

func init() {
	logs.SetLogger("console")
}

const (
	HttpStatusOk    = "200"
	HttpStatusError = "400"
)

type Response struct {
	Code string
	Msg  string
	Data interface{} `json:"data"`
}
