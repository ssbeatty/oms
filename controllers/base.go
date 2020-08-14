package controllers

const (
	HttpStatusOk    = "200"
	HttpStatusError = "400"
)

type Response struct {
	Code string
	Msg  string
	Data interface{} `json:"data"`
}
