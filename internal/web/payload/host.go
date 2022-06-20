package payload

// v1

type GetHostParam struct {
	Id int `uri:"id" binding:"required"`
}

type PostHostForm struct {
	HostName     string `form:"hostname" binding:"required"`
	User         string `form:"user" binding:"required"`
	Addr         string `form:"addr" binding:"required,ip_addr"`
	Port         int    `form:"port" binding:"required,min=0,max=65535"`
	PassWord     string `form:"password"`
	Group        int    `form:"group"`
	PrivateKeyId int    `form:"private_key_id" binding:"required_without=PassWord"`
	Tags         string `form:"tags"`
	VNCPort      int    `form:"vnc_port"`
}

type PutHostForm struct {
	Id           int    `form:"id" binding:"required"`
	HostName     string `form:"hostname"`
	User         string `form:"user"`
	Addr         string `form:"addr" binding:"len=0|ip_addr"`
	Port         int    `form:"port" binding:"min=0,max=65535"`
	PassWord     string `form:"password"`
	Group        int    `form:"group"`
	PrivateKeyId int    `form:"private_key_id"`
	Tags         string `form:"tags"`
	VNCPort      int    `form:"vnc_port"`
}

type DeleteHostParam struct {
	Id int `uri:"id" binding:"required"`
}
