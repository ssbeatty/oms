package buildin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"oms/pkg/transport"
	"oms/pkg/utils"
	"os"
	"path/filepath"
)

// FileUploadStep 上传文件
type FileUploadStep struct {
	BaseStep
	File    string `json:"file" jsonschema:"format=data-url"`
	Options string `json:"options" jsonschema:"enum=upload,enum=remove,required=true" jsonschema_description:"upload: 上传到远端 remove: 删除远端文件或者目录"`
	Remote  string `json:"remote" jsonschema:"required=true" jsonschema_description:"远程文件路径"`
}

func (bs *FileUploadStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	err := session.Client.NewSftpClient()
	if err != nil {
		return nil, err
	}

	if sudo {
		// todo change fs perm
	}

	switch bs.Options {
	case "upload":
		if exists, err := utils.PathExists(bs.File); !exists {
			return nil, errors.Wrap(err, "本地缓存不存在")
		}
		fName := filepath.Base(bs.File)
		if fName != "" && len(fName) > GUIDLength {
			fName = fName[GUIDLength:]
		}
		err := session.Client.UploadFile(bs.File, bs.Remote, fName)
		if err != nil {
			return nil, err
		}
		return []byte(fmt.Sprintf("上传成功, 远端路径: %s\r\n", bs.Remote)), nil

	case "remove":
		if session.Client.IsDir(bs.Remote) {
			err := session.Client.RemoveDir(bs.Remote)
			if err != nil {
				return nil, err
			}
			return []byte("删除成功!"), nil
		}
		err := session.Client.Remove(bs.Remote)
		if err != nil {
			return nil, err
		}
		return []byte("删除成功!"), nil
	default:
		return nil, errors.New("do not support options")
	}
}

func (bs *FileUploadStep) Create() Step {
	return &FileUploadStep{}
}

func (bs *FileUploadStep) Name() string {
	return StepNameFile
}

func (bs *FileUploadStep) Desc() string {
	return "文件操作"
}

// MultiFileUploadStep 上传多个文件
type MultiFileUploadStep struct {
	BaseStep
	Files     []string `json:"files" jsonschema:"format=data-url,required=true"`
	RemoteDir string   `json:"remote_dir" jsonschema:"required=true" jsonschema_description:"远程文件夹路径"`
}

func (bs *MultiFileUploadStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	err := session.Client.NewSftpClient()
	if err != nil {
		return nil, err
	}

	var (
		total int
	)

	for _, f := range bs.Files {
		fName := filepath.Base(f)
		if fName != "" && len(fName) > GUIDLength {
			fPath := filepath.ToSlash(filepath.Join(bs.RemoteDir, fName[GUIDLength:]))
			err := session.Client.UploadFile(f, fPath, fName[GUIDLength:])
			if err != nil {
				return nil, err
			}
		}

		total++
	}

	return []byte(fmt.Sprintf("上传成功, 远端路径: %s, 共上传文件%d个\r\n", bs.RemoteDir, total)), nil
}

func (bs *MultiFileUploadStep) Create() Step {
	return &MultiFileUploadStep{}
}

func (bs *MultiFileUploadStep) Name() string {
	return StepMultiNameFile
}

func (bs *MultiFileUploadStep) Desc() string {
	return "上传多个文件"
}

// ZipFileStep 上传多个文件
type ZipFileStep struct {
	BaseStep
	File   string `json:"file" jsonschema:"format=data-url" jsonschema_description:"*.tar | *.tar.gz | *.zip"`
	Remote string `json:"remote" jsonschema:"required=true" jsonschema_description:"解压到远端文件夹"`
}

func (bs *ZipFileStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	err := session.Client.NewSftpClient()
	if err != nil {
		return nil, err
	}
	ext := utils.GetFileExt(bs.File)

	if exists, err := utils.PathExists(bs.File); !exists {
		return nil, err
	}

	if !session.Client.PathExists(bs.Remote) {
		err = session.Client.MkdirAll(bs.Remote)
		if err != nil {
			return nil, err
		}
	}

	switch ext {
	case "tar":
		err = bs.unTar(session, false)
		if err != nil {
			return nil, err
		}
	case "tar.gz":
		err = bs.unTar(session, true)
		if err != nil {
			return nil, err
		}
	case "zip":
		err = bs.unZip(session)
		if err != nil {
			return nil, err
		}
	}

	return []byte(fmt.Sprintf("解压成功, 远端路径: %s\r\n", bs.Remote)), nil
}

func (bs *ZipFileStep) unTar(session *transport.Session, _gzip bool) error {
	var (
		tr *tar.Reader
	)

	fr, err := os.Open(bs.File)
	if err != nil {
		return err
	}
	defer fr.Close()

	if _gzip {
		gr, err := gzip.NewReader(fr)
		if err != nil {
			return err
		}

		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(fr)
	}

	for {
		hdr, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case hdr == nil:
			continue
		}

		dstFileDir := filepath.ToSlash(filepath.Join(bs.Remote, hdr.Name))

		switch hdr.Typeflag {
		case tar.TypeDir:
			if b := session.Client.PathExists(dstFileDir); !b {
				if err := session.Client.MkdirAll(dstFileDir); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			if b := session.Client.PathExists(filepath.Dir(dstFileDir)); !b {
				if err := session.Client.MkdirAll(dstFileDir); err != nil {
					return err
				}
			}
			file, err := session.Client.GetSftpClient().OpenFile(dstFileDir, os.O_CREATE|os.O_RDWR)
			if err != nil {
				return err
			}
			_, err = io.Copy(file, tr)
			if err != nil {
				return err
			}
			file.Close()
		}
	}

}

func (bs *ZipFileStep) unZip(session *transport.Session) error {
	reader, err := zip.OpenReader(bs.File)
	if err != nil {
		return err
	}

	defer reader.Close()

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			if err = session.Client.MkdirAll(file.Name); err != nil {
				return err
			}
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		filename := filepath.ToSlash(filepath.Join(bs.Remote, file.Name))
		err = session.Client.MkdirAll(filepath.Dir(filename))
		if err != nil {
			return err
		}
		w, err := session.Client.GetSftpClient().OpenFile(filename, os.O_CREATE|os.O_RDWR)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, rc)
		if err != nil {
			return err
		}
		w.Close()
		rc.Close()
	}
	return nil
}

func (bs *ZipFileStep) Create() Step {
	return &ZipFileStep{}
}

func (bs *ZipFileStep) Name() string {
	return StepNameZipFile
}

func (bs *ZipFileStep) Desc() string {
	return "解压缩文件"
}
