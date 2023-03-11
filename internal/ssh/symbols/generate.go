package symbols

import "reflect"

var (
	Symbols = make(map[string]map[string]reflect.Value)
)

//go:generate yaegi extract github.com/ssbeatty/oms/pkg/transport
//go:generate yaegi extract github.com/ssbeatty/oms/pkg/types
//go:generate yaegi extract github.com/ssbeatty/jsonschema
