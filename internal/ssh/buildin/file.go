package buildin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"io"
	"oms/internal/utils"
	"oms/pkg/transport"
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
			return nil, err
		}
		fName := filepath.Base(bs.File)
		if fName != "" && len(fName) > GUIDLength {
			fName = fName[GUIDLength:]
		}
		return nil, session.Client.UploadFile(bs.File, bs.Remote, fName)
	case "remove":
		if session.Client.IsDir(bs.Remote) {
			return nil, session.Client.RemoveDir(bs.Remote)
		}
		return nil, session.Client.Remove(bs.Remote)
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

	for _, f := range bs.Files {
		fName := filepath.Base(f)
		if fName != "" && len(fName) > GUIDLength {
			fPath := filepath.ToSlash(filepath.Join(bs.RemoteDir, fName[GUIDLength:]))
			err := session.Client.UploadFile(f, fPath, fName[GUIDLength:])
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}

func (bs *MultiFileUploadStep) Create() Step {
	return &MultiFileUploadStep{}
}

func (bs *MultiFileUploadStep) Name() string {
	return StepMultiNameFile
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

	return nil, nil
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
