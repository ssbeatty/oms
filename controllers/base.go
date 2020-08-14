package controllers

const (
	HttpStatusOk    = "200"
	HttpStatusError = "400"
)

type ResponseGet struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type ResponsePost struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}
