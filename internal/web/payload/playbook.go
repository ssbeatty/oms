package payload

type GetPlayBookParam struct {
	Id int `uri:"id" binding:"required"`
}

type PostPlayBookForm struct {
	Name  string `form:"name" binding:"required"`
	Steps string `form:"steps" binding:"required"`
}

type PutPlayBookForm struct {
	Id    int    `form:"id" binding:"required"`
	Name  string `form:"name"`
	Steps string `form:"steps"`
}

type DeletePlayBookParam struct {
	Id int `uri:"id" binding:"required"`
}

type UploadResponse struct {
	Files []File `json:"files"`
}

type File struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CachePath string `json:"cache_path"`
	Status    bool   `json:"status"`
}
