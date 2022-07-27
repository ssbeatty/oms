package payload

const (
	ErrHostParseEmpty = "parse host array empty"
)

type Response struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func GenerateResponsePayload(code string, msg string, data interface{}) Response {
	return Response{code, msg, data}
}

type Page struct {
	PageNum  int `form:"page_num"`
	PageSize int `form:"page_size"`
}
