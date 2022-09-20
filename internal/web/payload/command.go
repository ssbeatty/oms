package payload

type SearchCmdHistoryParams struct {
	KeyWord string `form:"keyword" binding:"required"`
	Limit   int    `form:"limit"`
}

type DeleteCmdHistoryParam struct {
	Id int `uri:"id" binding:"required"`
}

type GetQuicklyCommandParam struct {
	Id int `uri:"id" binding:"required"`
}

type PostQuicklyCommandForm struct {
	Name     string `form:"name" binding:"required"`
	Cmd      string `form:"cmd" binding:"required"`
	AppendCR bool   `form:"append_cr"`
}

type PutQuicklyCommandForm struct {
	Id       int    `form:"id" binding:"required"`
	Name     string `form:"name"`
	Cmd      string `form:"cmd"`
	AppendCR bool   `form:"append_cr"`
}

type DeleteQuicklyCommandParam struct {
	Id int `uri:"id" binding:"required"`
}
