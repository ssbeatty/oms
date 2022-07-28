package ssh

import (
	"fmt"
	"testing"
)

func TestGenJsonSchema(t *testing.T) {
	r := RunCmdStep{}
	schema, err := r.GetSchema(&r)
	if err != nil {
		return
	}

	fmt.Println(string(schema))

	s := RunShellStep{}
	schema, err = s.GetSchema(&s)
	if err != nil {
		return
	}

	fmt.Println(string(schema))

	f := FileUploadStep{}
	schema, err = f.GetSchema(&f)
	if err != nil {
		return
	}

	fmt.Println(string(schema))
}
