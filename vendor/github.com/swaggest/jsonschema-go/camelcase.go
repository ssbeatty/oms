package jsonschema

import (
	"regexp"
	"strings"
)

var (
	numberSequence    = regexp.MustCompile(`([a-zA-Z])(\d+)([a-zA-Z]?)`)
	numberReplacement = []byte(`$1 $2 $3`)
)

// toCamel converts a string to CamelCase.
func toCamel(s string) string {
	b := numberSequence.ReplaceAll([]byte(s), numberReplacement)
	s = string(b)
	s = strings.Trim(s, " ")
	n := ""
	capNext := true

	for _, v := range s {
		if v >= 'A' && v <= 'Z' {
			n += string(v)
		}

		if v >= '0' && v <= '9' {
			n += string(v)
		}

		if v >= 'a' && v <= 'z' {
			if capNext {
				n += strings.ToUpper(string(v))
			} else {
				n += string(v)
			}
		}

		if v == '_' || v == ' ' || v == '-' || v == '.' {
			capNext = true
		} else {
			capNext = false
		}
	}

	return n
}
