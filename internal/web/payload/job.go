package payload

type GetJobsParam struct {
	HostId int `form:"host_id"`
}

type GetJobParam struct {
	Id int `uri:"id" binding:"required"`
}

type PostJobForm struct {
	Name   string `form:"name" binding:"required"`
	Type   string `form:"type" binding:"required"`
	Spec   string `form:"spec" binding:"required"`
	Cmd    string `form:"cmd" binding:"required"`
	HostId int    `form:"host_id" binding:"required"`
}

type PutJobForm struct {
	Id   int    `form:"id" binding:"required"`
	Name string `form:"name"`
	Type string `form:"type"`
	Spec string `form:"spec"`
	Cmd  string `form:"cmd"`
}

type DeleteJobParam struct {
	Id int `uri:"id" binding:"required"`
}

type OptionsJobForm struct {
	Id int `form:"id" binding:"required"`
}
