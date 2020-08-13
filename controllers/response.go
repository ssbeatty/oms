package controllers

type Response struct {
	Code string
	Msg  string
	Data []interface{} `json:"data"`
}
