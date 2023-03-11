package buildin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/ssbeatty/oms/pkg/transport"
	"github.com/ssbeatty/oms/pkg/types"
	"github.com/ssbeatty/oms/pkg/utils"
	"io"
	"os"
	"path/filepath"
)

// FileUploadStep 上传文件
type FileUploadStep struct {
	types.BaseStep
	cfg *fileUploadStepConfig
}

type fileUploadStepConfig struct {
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

	switch bs.cfg.Options {
	case "upload":
		if exists, err := utils.PathExists(bs.cfg.File); !exists {
			return nil, errors.Wrap(err, "本地缓存不存在")
		}
		fName := filepath.Base(bs.cfg.File)
		if fName != "" && len(fName) > GUIDLength {
			fName = fName[GUIDLength:]
		}
		err := session.Client.UploadFile(bs.cfg.File, bs.cfg.Remote, fName)
		if err != nil {
			return nil, err
		}
		return []byte(fmt.Sprintf("上传成功, 远端路径: %s\r\n", bs.cfg.Remote)), nil

	case "remove":
		if session.Client.IsDir(bs.cfg.Remote) {
			err := session.Client.RemoveDir(bs.cfg.Remote)
			if err != nil {
				return nil, err
			}
			return []byte("删除成功!"), nil
		}
		err := session.Client.Remove(bs.cfg.Remote)
		if err != nil {
			return nil, err
		}
		return []byte("删除成功!"), nil
	default:
		return nil, errors.New("do not support options")
	}
}

func (bs *FileUploadStep) Create(conf []byte) (types.Step, error) {
	cfg := &fileUploadStepConfig{}

	err := json.Unmarshal(conf, cfg)
	if err != nil {
		return nil, err
	}
	return &FileUploadStep{
		cfg: cfg,
	}, nil
}

func (bs *FileUploadStep) Config() interface{} {
	return bs.cfg
}

func (bs *FileUploadStep) Name() string {
	return StepNameFile
}

func (bs *FileUploadStep) GetSchema() (interface{}, error) {

	return types.GetSchema(bs.cfg)
}

func (bs *FileUploadStep) Desc() string {
	return "文件操作"
}

// MultiFileUploadStep 上传多个文件
type MultiFileUploadStep struct {
	types.BaseStep
	cfg *multiFileUploadStepConfig
}

type multiFileUploadStepConfig struct {
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

	for _, f := range bs.cfg.Files {
		fName := filepath.Base(f)
		if fName != "" && len(fName) > GUIDLength {
			fPath := filepath.ToSlash(filepath.Join(bs.cfg.RemoteDir, fName[GUIDLength:]))
			err := session.Client.UploadFile(f, fPath, fName[GUIDLength:])
			if err != nil {
				return nil, err
			}
		}

		total++
	}

	return []byte(fmt.Sprintf("上传成功, 远端路径: %s, 共上传文件%d个\r\n", bs.cfg.RemoteDir, total)), nil
}

func (bs *MultiFileUploadStep) Create(conf []byte) (types.Step, error) {
	cfg := &multiFileUploadStepConfig{}

	err := json.Unmarshal(conf, cfg)
	if err != nil {
		return nil, err
	}
	return &MultiFileUploadStep{
		cfg: cfg,
	}, nil
}

func (bs *MultiFileUploadStep) Name() string {
	return StepMultiNameFile
}

func (bs *MultiFileUploadStep) Desc() string {
	return "上传多个文件"
}

func (bs *MultiFileUploadStep) Config() interface{} {
	return bs.cfg
}

func (bs *MultiFileUploadStep) GetSchema() (interface{}, error) {

	return types.GetSchema(bs.cfg)
}

// ZipFileStep 上传压缩文件
type ZipFileStep struct {
	types.BaseStep

	cfg *zipFileStepConfig
}

type zipFileStepConfig struct {
	File   string `json:"file" jsonschema:"format=data-url" jsonschema_description:"*.tar | *.tar.gz | *.zip"`
	Remote string `json:"remote" jsonschema:"required=true" jsonschema_description:"解压到远端文件夹"`
}

func (bs *ZipFileStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	err := session.Client.NewSftpClient()
	if err != nil {
		return nil, err
	}
	ext := utils.GetFileExt(bs.cfg.File)

	if exists, err := utils.PathExists(bs.cfg.File); !exists {
		return nil, err
	}

	if !session.Client.PathExists(bs.cfg.Remote) {
		err = session.Client.MkdirAll(bs.cfg.Remote)
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

	return []byte(fmt.Sprintf("解压成功, 远端路径: %s\r\n", bs.cfg.Remote)), nil
}

func (bs *ZipFileStep) unTar(session *transport.Session, _gzip bool) error {
	var (
		tr *tar.Reader
	)

	fr, err := os.Open(bs.cfg.File)
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

		dstFileDir := filepath.ToSlash(filepath.Join(bs.cfg.Remote, hdr.Name))

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
	reader, err := zip.OpenReader(bs.cfg.File)
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
		filename := filepath.ToSlash(filepath.Join(bs.cfg.Remote, file.Name))
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

func (bs *ZipFileStep) Create(conf []byte) (types.Step, error) {
	cfg := &zipFileStepConfig{}

	err := json.Unmarshal(conf, cfg)
	if err != nil {
		return nil, err
	}
	return &ZipFileStep{
		cfg: cfg,
	}, nil
}

func (bs *ZipFileStep) Name() string {
	return StepNameZipFile
}

func (bs *ZipFileStep) Desc() string {
	return "解压缩文件"
}

func (bs *ZipFileStep) Config() interface{} {
	return bs.cfg
}

func (bs *ZipFileStep) GetSchema() (interface{}, error) {

	return types.GetSchema(bs.cfg)
}
