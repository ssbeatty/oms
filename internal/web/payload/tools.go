package payload

type CmdParams struct {
	Id   int    `form:"id" binding:"required"`
	Sudo bool   `form:"sudo"`
	Type string `form:"type" binding:"required"`
	Cmd  string `form:"cmd" binding:"required"`
}

type OptionsFileParams struct {
	Id     string `form:"id"`
	HostId int    `form:"host_id" binding:"required"`
}

type MkdirParams struct {
	OptionsFileParams
	Dir string `form:"dir" binding:"required"`
}
