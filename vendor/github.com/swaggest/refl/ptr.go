package refl

import "reflect"

// DeepIndirect returns first encountered non-pointer type.
func DeepIndirect(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t
}
