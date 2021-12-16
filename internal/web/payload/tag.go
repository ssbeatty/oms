package payload

type GetTagParam struct {
	Id int `uri:"id" binding:"required"`
}

type PostTagForm struct {
	Name string `form:"name" binding:"required"`
}

type PutTagForm struct {
	Id   int    `form:"id" binding:"required"`
	Name string `form:"name"`
}

type DeleteTagParam struct {
	Id int `uri:"id" binding:"required"`
}
