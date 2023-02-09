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

// ModifyFileParams 解析修改文件接口使用
type ModifyFileParams struct {
	Id            string `json:"id"`
	HostId        int    `json:"host_id" binding:"required"`
	ModifyContent string `json:"modify_content"`
}

type MkdirParams struct {
	OptionsFileParams
	Dir string `form:"dir" binding:"required"`
}

type ImportResponse struct {
	CreateGroup      []string `json:"create_group"`
	CreateTag        []string `json:"create_tag"`
	CreateHost       []string `json:"create_host"`
	CreatePrivateKey []string `json:"create_private_key"`
}

type FileTaskCancelForm struct {
	Id   int    `form:"id" binding:"required"`
	Type string `form:"type" binding:"required"`
	File string `form:"file" binding:"required"`
}
