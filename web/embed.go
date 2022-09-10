package web

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed omsUI/dist
var EmbededFiles embed.FS

//func init() {
//	data, _ := EmbededFiles.ReadFile("omsUI/dist/index.html")
//	fmt.Println(string(data))
//}

type ServeFileSystem struct {
	E    embed.FS
	Path string
}

type File struct {
	name string
	fs.File
}

func (f *File) Readdir(count int) ([]fs.FileInfo, error) {
	ff, ok := f.File.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: f.name, Err: errors.New("not implemented")}
	}
	fileList, err := ff.ReadDir(count)
	if err != nil {
		return nil, err
	}
	rspList := []fs.FileInfo{}
	for _, v := range fileList {
		temp, err := v.Info()
		if err != nil {
			return nil, err
		}
		rspList = append(rspList, temp)
	}
	return rspList, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	ff, ok := f.File.(io.Seeker)
	if !ok {
		return 0, &fs.PathError{Op: "Seek", Path: f.name, Err: errors.New("not implemented")}
	}
	return ff.Seek(offset, whence)
}

func (c *ServeFileSystem) Open(name string) (http.File, error) {
	name = path.Join(c.Path, name)
	f, err := c.E.Open(name)
	if err != nil {
		return nil, err
	}
	ff := File{
		name: name,
		File: f,
	}
	return &ff, nil
}

func (c *ServeFileSystem) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {

		p = path.Join(c.Path, p)
		f, err := c.E.Open(p)
		if err != nil {
			return false
		}
		err = f.Close()
		if err != nil {
			return false
		}
		return true
	}
	return false
}
