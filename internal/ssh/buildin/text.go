package buildin

import (
	"encoding/json"
	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
	"github.com/pkg/errors"
	"github.com/ssbeatty/jsonschema"
	"github.com/ssbeatty/oms/pkg/transport"
	"github.com/ssbeatty/oms/pkg/types"
	"github.com/ssbeatty/oms/pkg/utils"
	"io/ioutil"
	"os"
	"strings"
)

// JsonYamlReplaceStep 上传多个文件
type JsonYamlReplaceStep struct {
	types.BaseStep
	Path   string      `json:"path" jsonschema:"required=true" jsonschema_description:"例如: $.path1.path2[0].item"`
	Value  interface{} `json:"value" jsonschema:"required=true,oneof_type=string;array" jsonschema_description:"替换的节点值, 输入字符串类型需要: \"{value}\""`
	Remote string      `json:"remote" jsonschema:"required=true" jsonschema_description:"远程Yaml/Json路径(不支持大文件)"`
}

func (bs *JsonYamlReplaceStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	err := session.Client.NewSftpClient()
	if err != nil {
		return nil, err
	}

	path, err := yaml.PathString(bs.Path)
	if err != nil {
		return nil, errors.Wrap(err, "parse json path error")
	}
	if !session.Client.PathExists(bs.Remote) {
		return nil, errors.New("remote not exist")
	}

	var (
		value string
	)
	ext := utils.GetFileExt(bs.Remote)

	switch bs.Value.(type) {
	case string:
		value = bs.Value.(string)
	case []string, []interface{}:
		switch ext {
		case "json":
			itl, _ := json.Marshal(bs.Value)
			value = string(itl)
		case "yaml", "yml":
			itl, _ := yaml.Marshal(bs.Value)
			value = string(itl)
		}
	}
	fn, err := session.Client.GetSftpClient().OpenFile(bs.Remote, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(fn)
	if err != nil {
		return nil, err
	}

	file, err := parser.ParseBytes(b, 0)
	if err != nil {
		return nil, err
	}

	if err := path.ReplaceWithReader(file, strings.NewReader(value)); err != nil {
		return nil, err
	}

	fn.Close()

	fn, err = session.Client.GetSftpClient().OpenFile(bs.Remote, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return nil, err
	}

	_, err = fn.Write([]byte(file.String()))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (bs *JsonYamlReplaceStep) GetSchema(instance types.Step) (interface{}, error) {
	schema, err := bs.BaseStep.GetSchema(instance)
	if err != nil {
		return nil, err
	}
	value, ok := schema.(*jsonschema.Schema).Properties.Get("value")
	if ok {
		for idx, _ := range value.(*jsonschema.Schema).OneOf {
			item := value.(*jsonschema.Schema).OneOf[idx]
			if item.Type == "array" {
				item.Title = "列表"
				item.Description = ""
				item.Items = &jsonschema.Schema{
					Type: "string",
				}
			} else if item.Type == "string" {
				item.Title = "文本"
				item.Description = ""
			}
		}
	}
	return schema, nil
}

func (bs *JsonYamlReplaceStep) Create() types.Step {
	return &JsonYamlReplaceStep{}
}

func (bs *JsonYamlReplaceStep) Name() string {
	return StepNameYamlJson
}

func (bs *JsonYamlReplaceStep) Desc() string {
	return "修改Json(Yaml)文件"
}
