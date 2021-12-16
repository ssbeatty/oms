package payload

import "mime/multipart"

// v1

type GetPrivateKeyParam struct {
	Id int `uri:"id" binding:"required"`
}

type PostPrivateKeyForm struct {
	Name       string                `form:"name" binding:"required"`
	Passphrase string                `form:"passphrase"`
	KeyFile    *multipart.FileHeader `form:"key_file" binding:"required"`
}

type PutPrivateKeyForm struct {
	Id         int                   `form:"id" binding:"required"`
	Name       string                `form:"name"`
	Passphrase string                `form:"passphrase"`
	KeyFile    *multipart.FileHeader `form:"key_file"`
}

type DeletePrivateKeyParam struct {
	Id int `uri:"id" binding:"required"`
}
