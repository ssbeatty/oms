package controllers

type PolicyType string

const (
	HttpStatusOk = "200"
)

type Response struct {
	Code string
	Msg  string
	Data interface{} `json:"data"`
}
