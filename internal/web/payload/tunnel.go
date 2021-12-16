package payload

type GetTunnelsParam struct {
	HostId int `form:"host_id"`
}

type GetTunnelParam struct {
	Id int `uri:"id" binding:"required"`
}

type PostTunnelForm struct {
	Mode        string `form:"mode" binding:"required"`
	Source      string `form:"source" binding:"required,hostname_port"`
	Destination string `form:"destination" binding:"required,hostname_port"`
	HostId      int    `form:"host_id" binding:"required"`
}

type PutTunnelForm struct {
	Id          int    `form:"id" binding:"required"`
	Mode        string `form:"mode"`
	Source      string `form:"source" binding:"len=0|hostname_port"`
	Destination string `form:"destination" binding:"len=0|hostname_port"`
	HostId      int    `form:"host_id" binding:"required"`
}

type DeleteTunnelParam struct {
	Id int `uri:"id" binding:"required"`
}
