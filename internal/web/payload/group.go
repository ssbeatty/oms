package payload

// v1

type GetGroupParam struct {
	Id int `uri:"id" binding:"required"`
}

type PostGroupForm struct {
	Name   string `form:"name" binding:"required"`
	Params string `form:"params"`
	Mode   int    `form:"mode"`
}

type PutGroupForm struct {
	Id     int    `form:"id" binding:"required"`
	Name   string `form:"name"`
	Params string `form:"params"`
	Mode   int    `form:"mode"`
}

type DeleteGroupParam struct {
	Id int `uri:"id" binding:"required"`
}
