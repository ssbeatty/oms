package payload

type GetJobsParam struct {
	ExecuteId int `form:"execute_id"`
}

type GetJobParam struct {
	Id int `uri:"id" binding:"required"`
}

type PostJobForm struct {
	Name        string `form:"name" binding:"required"`
	Type        string `form:"type" binding:"required"`
	Spec        string `form:"spec"`
	Cmd         string `form:"cmd"`
	CmdId       int    `form:"cmd_id"`
	CmdType     string `form:"cmd_type" binding:"required"`
	ExecuteID   int    `form:"execute_id" binding:"required"`
	ExecuteType string `form:"execute_type" binding:"required"`
}

type PutJobForm struct {
	Id      int    `form:"id" binding:"required"`
	Name    string `form:"name"`
	Type    string `form:"type"`
	Spec    string `form:"spec"`
	Cmd     string `form:"cmd"`
	CmdId   int    `form:"cmd_id"`
	CmdType string `form:"cmd_type"`
}

type DeleteJobParam struct {
	Id int `uri:"id" binding:"required"`
}

type OptionsJobForm struct {
	Id int `form:"id" binding:"required"`
}

type GetTaskInstanceParam struct {
	Page
	JobId int `form:"job_id"`
}

type GetTaskInstanceLogParam struct {
	Id int `form:"id" binding:"required"`
}

type DeleteTaskInstanceFrom struct {
	JobId     int   `form:"job_id"`
	TimeStamp int64 `form:"time_stamp"`
}
