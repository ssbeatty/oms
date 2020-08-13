package controllers

import "github.com/astaxie/beego/orm"

var o = orm.NewOrm()

const (
	HttpStatusOk    = "200"
	HttpStatusError = "400"
)

type Response struct {
	Code string
	Msg  string
	Data interface{} `json:"data"`
}
