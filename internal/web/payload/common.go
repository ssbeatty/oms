package payload

const (
	ErrHostParseEmpty = "parse host array empty"
	RespTypeMsg       = "msg"
	RespTypeError     = "error"
	RespTypeData      = "data"
)

type Response struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
	Type string      `json:"type"` // data msg error
}

func GenerateDataResponse(code string, msg string, data interface{}) Response {
	return Response{code, msg, data, RespTypeData}
}

func GenerateMsgResponse(code string, msg string) Response {
	return Response{code, msg, nil, RespTypeMsg}
}

func GenerateErrorResponse(code string, msg string) Response {
	return Response{code, msg, nil, RespTypeError}
}

type Page struct {
	PageNum  int `form:"page_num"`
	PageSize int `form:"page_size"`
}

type PageData struct {
	Total   int64       `json:"total"`
	PageNum int         `json:"page_num"`
	Data    interface{} `json:"data"`
}
