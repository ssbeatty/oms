package payload

type SearchCmdHistoryParams struct {
	KeyWord string `form:"keyword" binding:"required"`
	Limit   int    `form:"limit"`
}

type DeleteCmdHistoryParam struct {
	Id int `uri:"id" binding:"required"`
}
