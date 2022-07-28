package refl

import (
	"reflect"
	"strings"
)

// TypeString is a type name with import path.
type TypeString string

// GoType returns string representation of type name including import path.
func GoType(t reflect.Type) TypeString {
	s := t.Name()
	pkgPath := t.PkgPath()

	if pkgPath != "" {
		pos := strings.Index(pkgPath, "/vendor/")
		if pos != -1 {
			pkgPath = pkgPath[pos+8:]
		}

		s = pkgPath + "." + s
	}

	ts := t.String()
	typeRef := s

	pos := strings.LastIndex(typeRef, "/")
	if pos != -1 {
		typeRef = typeRef[pos+1:]
	}

	if typeRef != ts {
		s = pkgPath + "::" + t.String()
	}

	// nolint:exhaustive // This switch only looks into specific kind.
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		if pkgPath == "" {
			return "[]" + GoType(t.Elem())
		}
	case reflect.Ptr:
		if pkgPath == "" {
			return "*" + GoType(t.Elem())
		}
	case reflect.Map:
		if pkgPath == "" {
			return "map[" + GoType(t.Key()) + "]" + GoType(t.Elem())
		}
	}

	return TypeString(s)
}
