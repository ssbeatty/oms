package jsonschema

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/swaggest/refl"
)

var (
	typeOfJSONRawMsg      = reflect.TypeOf(json.RawMessage{})
	typeOfTime            = reflect.TypeOf(time.Time{})
	typeOfDate            = reflect.TypeOf(Date{})
	typeOfTextUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	typeOfTextMarshaler   = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	typeOfEmptyInterface  = reflect.TypeOf((*interface{})(nil)).Elem()
	typeOfSchemaInliner   = reflect.TypeOf((*SchemaInliner)(nil)).Elem()
)

const (
	// ErrSkipProperty indicates that property should not be added to object.
	ErrSkipProperty = sentinelError("property skipped")
)

type sentinelError string

func (e sentinelError) Error() string {
	return string(e)
}

// IgnoreTypeName is a marker interface to ignore type name of mapped value and use original.
type IgnoreTypeName interface {
	IgnoreTypeName()
}

// SchemaInliner is a marker interface to inline schema without creating a definition.
type SchemaInliner interface {
	InlineJSONSchema()
}

// IgnoreTypeName instructs reflector to keep original type name during mapping.
func (s Schema) IgnoreTypeName() {}

// Described exposes description.
type Described interface {
	Description() string
}

// Titled exposes title.
type Titled interface {
	Title() string
}

// Ref is a definition reference.
type Ref struct {
	Path string
	Name string
}

// Schema creates schema instance from reference.
func (r Ref) Schema() Schema {
	s := r.Path + r.Name

	return Schema{
		Ref: &s,
	}
}

// Reflector creates JSON Schemas from Go values.
type Reflector struct {
	DefaultOptions []func(*ReflectContext)
	typesMap       map[reflect.Type]interface{}
	defNames       map[reflect.Type]string
}

// AddTypeMapping creates substitution link between types of src and dst when reflecting JSON Schema.
func (r *Reflector) AddTypeMapping(src, dst interface{}) {
	if r.typesMap == nil {
		r.typesMap = map[reflect.Type]interface{}{}
	}

	r.typesMap[refl.DeepIndirect(reflect.TypeOf(src))] = dst
}

// InterceptDefName allows modifying reflected definition names.
func (r *Reflector) InterceptDefName(f func(t reflect.Type, defaultDefName string) string) {
	r.DefaultOptions = append(r.DefaultOptions, func(rc *ReflectContext) {
		rc.DefName = f
	})
}

func checkSchemaSetup(v reflect.Value, s *Schema) (bool, error) {
	vi := v.Interface()
	if v.Kind() == reflect.Ptr && v.IsNil() {
		vi = reflect.New(v.Type().Elem()).Interface()
	}

	reflectEnum(s, "", vi)

	if exposer, ok := v.Interface().(Exposer); ok {
		schema, err := exposer.JSONSchema()
		if err != nil {
			return true, err
		}

		*s = schema

		return true, nil
	}

	if exposer, ok := v.Interface().(RawExposer); ok {
		schemaBytes, err := exposer.JSONSchemaBytes()
		if err != nil {
			return true, err
		}

		err = json.Unmarshal(schemaBytes, s)
		if err != nil {
			return true, err
		}

		return true, nil
	}

	return false, nil
}

// Reflect walks Go value and builds its JSON Schema based on types and field tags.
//
// Values can be populated from field tags of original field:
//   type MyObj struct {
//      BoundedNumber int `query:"boundedNumber" minimum:"-100" maximum:"100"`
//      SpecialString string `json:"specialString" pattern:"^[a-z]{4}$" minLength:"4" maxLength:"4"`
//   }
//
// Note: field tags are only applied to inline schemas, if you use named type then referenced schema
// will be created and tags will be ignored. This happens because referenced schema can be used in
// multiple fields with conflicting tags, therefore customization of referenced schema has to done on
// the type itself via RawExposer, Exposer or Preparer.
//
// These tags can be used:
//   - `title`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.6.1
//   - `description`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.6.1
//   - `default`, can be scalar or JSON value,
//  		https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.6.2
//   - `const`, can be scalar or JSON value,
//          https://json-schema.org/draft/2020-12/json-schema-validation.html#rfc.section.6.1.3
//   - `pattern`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.2.3
//   - `format`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.7
//   - `multipleOf`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.1
//   - `maximum`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.2
//   - `minimum`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.3
//   - `maxLength`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.2.1
//   - `minLength`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.2.2
//   - `maxItems`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.3.2
//   - `minItems`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.3.3
//   - `maxProperties`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.4.1
//   - `minProperties`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.4.2
//   - `exclusiveMaximum`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.2
//   - `exclusiveMinimum`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.3
//   - `uniqueItems`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.3.4
//   - `enum`, tag value must be a JSON or comma-separated list of strings,
//  		https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.5.1
//
// Unnamed fields can be used to configure parent schema:
//   type MyObj struct {
//      BoundedNumber int `query:"boundedNumber" minimum:"-100" maximum:"100"`
//      SpecialString string `json:"specialString" pattern:"^[a-z]{4}$" minLength:"4" maxLength:"4"`
//      _             struct{} `additionalProperties:"false" description:"MyObj is my object."`
//   }
//
// In case of a structure with multiple name tags, you can enable filtering of unnamed fields with
// ReflectContext.UnnamedFieldWithTag option and add matching name tags to structure (e.g. query:"_").
//   type MyObj struct {
//      BoundedNumber int `query:"boundedNumber" minimum:"-100" maximum:"100"`
//      SpecialString string `json:"specialString" pattern:"^[a-z]{4}$" minLength:"4" maxLength:"4"`
//      // These parent schema tags would only be applied to `query` schema reflection (not for `json`).
//      _ struct{} `query:"_" additionalProperties:"false" description:"MyObj is my object."`
//   }
//
// Additionally there are structure can implement any of special interfaces for fine-grained Schema control:
// RawExposer, Exposer, Preparer.
//
// These interfaces allow exposing particular schema keywords:
// Titled, Described, Enum, NamedEnum.
func (r *Reflector) Reflect(i interface{}, options ...func(rc *ReflectContext)) (Schema, error) {
	rc := ReflectContext{}
	rc.Context = context.Background()
	rc.DefinitionsPrefix = "#/definitions/"
	rc.PropertyNameTag = "json"
	rc.Path = []string{"#"}
	rc.typeCycles = make(map[refl.TypeString]bool)

	InterceptType(checkSchemaSetup)(&rc)

	for _, option := range r.DefaultOptions {
		option(&rc)
	}

	for _, option := range options {
		option(&rc)
	}

	schema, err := r.reflect(i, &rc, false, nil)
	if err == nil && len(rc.definitions) > 0 {
		schema.Definitions = make(map[string]SchemaOrBool, len(rc.definitions))

		for typeString, def := range rc.definitions {
			def := def
			ref := rc.definitionRefs[typeString]

			if rc.CollectDefinitions != nil {
				rc.CollectDefinitions(ref.Name, def)
			} else {
				schema.Definitions[ref.Name] = def.ToSchemaOrBool()
			}
		}
	}

	return schema, err
}

func removeNull(t *Type) {
	if t.SimpleTypes != nil && *t.SimpleTypes == Null {
		t.SimpleTypes = nil
	} else if len(t.SliceOfSimpleTypeValues) > 0 {
		for i, ti := range t.SliceOfSimpleTypeValues {
			if ti == Null {
				// Remove Null from slice.
				t.SliceOfSimpleTypeValues = append(t.SliceOfSimpleTypeValues[:i],
					t.SliceOfSimpleTypeValues[i+1:]...)
			}
		}

		if len(t.SliceOfSimpleTypeValues) == 1 {
			t.SimpleTypes = &t.SliceOfSimpleTypeValues[0]
			t.SliceOfSimpleTypeValues = nil
		}
	}
}

func (r *Reflector) reflectDefer(defName string, typeString refl.TypeString, rc *ReflectContext, schema Schema, keepType bool) Schema {
	if rc.RootNullable && len(rc.Path) == 0 {
		schema.AddType(Null)
	}

	if schema.Ref != nil {
		return schema
	}

	if rc.InlineRefs {
		return schema
	}

	if !rc.RootRef && len(rc.Path) == 0 {
		return schema
	}

	if defName == "" {
		return schema
	}

	if !rc.RootRef && defName == rc.rootDefName {
		ref := Ref{Path: "#"}

		return ref.Schema()
	}

	if rc.definitions == nil {
		rc.definitions = make(map[refl.TypeString]Schema, 1)
		rc.definitionRefs = make(map[refl.TypeString]Ref, 1)
	}

	rc.definitions[typeString] = schema
	ref := Ref{Path: rc.DefinitionsPrefix, Name: defName}
	rc.definitionRefs[typeString] = ref

	s := ref.Schema()

	if keepType {
		s.Type = schema.Type
	}

	s.ReflectType = schema.ReflectType

	return s
}

func (r *Reflector) reflect(i interface{}, rc *ReflectContext, keepType bool, parent *Schema) (schema Schema, err error) {
	var (
		typeString refl.TypeString
		defName    string
		t          = reflect.TypeOf(i)
		v          = reflect.ValueOf(i)
	)

	defer func() {
		rc.Path = rc.Path[:len(rc.Path)-1]

		if t == nil {
			return
		}

		if err != nil {
			return
		}

		schema = r.reflectDefer(defName, typeString, rc, schema, keepType)
	}()

	if t == nil || t == typeOfEmptyInterface {
		return schema, nil
	}

	schema.ReflectType = t
	schema.Parent = parent

	if t.Kind() == reflect.Ptr && t.Elem() != typeOfJSONRawMsg {
		schema.AddType(Null)
	}

	t = refl.DeepIndirect(t)

	if t == nil || t == typeOfEmptyInterface {
		schema.Type = nil

		return schema, nil
	}

	typeString = refl.GoType(t)
	defName = r.defName(rc, t)

	if mappedTo, found := r.typesMap[t]; found {
		t = refl.DeepIndirect(reflect.TypeOf(mappedTo))
		v = reflect.ValueOf(mappedTo)

		if _, ok := mappedTo.(IgnoreTypeName); !ok {
			typeString = refl.GoType(t)
			defName = r.defName(rc, t)
		}
	}

	if len(rc.Path) == 1 {
		rc.rootDefName = defName
	}

	// Shortcut on embedded map or slice.
	if !rc.SkipEmbeddedMapsSlices {
		if et := refl.FindEmbeddedSliceOrMap(i); et != nil {
			t = et
		}
	}

	if r.isWellKnownType(t, &schema) {
		return schema, nil
	}

	if rc.InterceptType != nil {
		if ret, err := rc.InterceptType(v, &schema); err != nil || ret {
			return schema, err
		}
	}

	if ref, ok := rc.definitionRefs[typeString]; ok && defName != "" {
		return ref.Schema(), nil
	}

	if rc.typeCycles[typeString] {
		return schema, nil
	}

	if t.PkgPath() != "" && len(rc.Path) > 1 && defName != "" {
		rc.typeCycles[typeString] = true
	}

	if vd, ok := v.Interface().(Described); ok {
		schema.WithDescription(vd.Description())
	}

	if vt, ok := v.Interface().(Titled); ok {
		schema.WithTitle(vt.Title())
	}

	if err = r.applySubSchemas(v, rc, &schema); err != nil {
		return schema, err
	}

	if err = r.kindSwitch(t, v, &schema, rc); err != nil {
		return schema, err
	}

	if rc.InterceptType != nil {
		if ret, err := rc.InterceptType(v, &schema); err != nil || ret {
			return schema, err
		}
	}

	if preparer, ok := v.Interface().(Preparer); ok {
		err := preparer.PrepareJSONSchema(&schema)

		return schema, err
	}

	return schema, nil
}

func (r *Reflector) applySubSchemas(v reflect.Value, rc *ReflectContext, schema *Schema) error {
	vi := v.Interface()

	if e, ok := vi.(OneOfExposer); ok {
		var schemas []SchemaOrBool

		for _, item := range e.JSONSchemaOneOf() {
			rc.Path = append(rc.Path, "oneOf")

			s, err := r.reflect(item, rc, false, schema)
			if err != nil {
				return fmt.Errorf("failed to reflect 'oneOf' values of %T: %w", vi, err)
			}

			schemas = append(schemas, s.ToSchemaOrBool())
		}

		schema.OneOf = schemas
	}

	if e, ok := vi.(AnyOfExposer); ok {
		var schemas []SchemaOrBool

		for _, item := range e.JSONSchemaAnyOf() {
			rc.Path = append(rc.Path, "anyOf")

			s, err := r.reflect(item, rc, false, schema)
			if err != nil {
				return fmt.Errorf("failed to reflect 'anyOf' values of %T: %w", vi, err)
			}

			schemas = append(schemas, s.ToSchemaOrBool())
		}

		schema.AnyOf = schemas
	}

	if e, ok := vi.(AllOfExposer); ok {
		var schemas []SchemaOrBool

		for _, item := range e.JSONSchemaAllOf() {
			rc.Path = append(rc.Path, "allOf")

			s, err := r.reflect(item, rc, false, schema)
			if err != nil {
				return fmt.Errorf("failed to reflect 'allOf' values of %T: %w", vi, err)
			}

			schemas = append(schemas, s.ToSchemaOrBool())
		}

		schema.AllOf = schemas
	}

	if e, ok := vi.(NotExposer); ok {
		rc.Path = append(rc.Path, "not")

		s, err := r.reflect(e.JSONSchemaNot(), rc, false, schema)
		if err != nil {
			return fmt.Errorf("failed to reflect 'not' value of %T: %w", vi, err)
		}

		schema.WithNot(s.ToSchemaOrBool())
	}

	if e, ok := vi.(IfExposer); ok {
		rc.Path = append(rc.Path, "if")

		s, err := r.reflect(e.JSONSchemaIf(), rc, false, schema)
		if err != nil {
			return fmt.Errorf("failed to reflect 'if' value of %T: %w", vi, err)
		}

		schema.WithIf(s.ToSchemaOrBool())
	}

	if e, ok := vi.(ThenExposer); ok {
		rc.Path = append(rc.Path, "if")

		s, err := r.reflect(e.JSONSchemaThen(), rc, false, schema)
		if err != nil {
			return fmt.Errorf("failed to reflect 'then' value of %T: %w", vi, err)
		}

		schema.WithThen(s.ToSchemaOrBool())
	}

	if e, ok := vi.(ElseExposer); ok {
		rc.Path = append(rc.Path, "if")

		s, err := r.reflect(e.JSONSchemaElse(), rc, false, schema)
		if err != nil {
			return fmt.Errorf("failed to reflect 'else' value of %T: %w", vi, err)
		}

		schema.WithElse(s.ToSchemaOrBool())
	}

	return nil
}

func (r *Reflector) isWellKnownType(t reflect.Type, schema *Schema) bool {
	if t == typeOfTime {
		schema.AddType(String)
		schema.WithFormat("date-time")

		return true
	}

	if t == typeOfDate {
		schema.AddType(String)
		schema.WithFormat("date")

		return true
	}

	if (t.Implements(typeOfTextUnmarshaler) || reflect.PtrTo(t).Implements(typeOfTextUnmarshaler)) &&
		(t.Implements(typeOfTextMarshaler) || reflect.PtrTo(t).Implements(typeOfTextMarshaler)) {
		schema.AddType(String)

		return true
	}

	return false
}

func (r *Reflector) defName(rc *ReflectContext, t reflect.Type) string {
	if t.PkgPath() == "" || t == typeOfTime || t == typeOfJSONRawMsg || t == typeOfDate {
		return ""
	}

	if t.Implements(typeOfSchemaInliner) {
		return ""
	}

	if r.defNames == nil {
		r.defNames = map[reflect.Type]string{}
	}

	defName, found := r.defNames[t]
	if found {
		return defName
	}

	try := 1

	for {
		if t.PkgPath() == "main" {
			defName = toCamel(strings.Title(t.Name()))
		} else {
			defName = toCamel(path.Base(t.PkgPath())) + strings.Title(t.Name())
		}

		if rc.DefName != nil {
			defName = rc.DefName(t, defName)
		}

		if try > 1 {
			defName = defName + "Type" + strconv.Itoa(try)
		}

		conflict := false

		for tt, dn := range r.defNames {
			if dn == defName && tt != t {
				conflict = true

				break
			}
		}

		if !conflict {
			r.defNames[t] = defName

			return defName
		}

		try++
	}
}

func (r *Reflector) kindSwitch(t reflect.Type, v reflect.Value, schema *Schema, rc *ReflectContext) error {
	//nolint:exhaustive // Covered with default case.
	switch t.Kind() {
	case reflect.Struct:
		switch {
		case reflect.PtrTo(t).Implements(typeOfTextUnmarshaler):
			schema.AddType(String)
		default:
			schema.AddType(Object)
			removeNull(schema.Type)

			err := r.walkProperties(v, schema, rc)
			if err != nil {
				return err
			}
		}

	case reflect.Slice, reflect.Array:
		if t == typeOfJSONRawMsg {
			break
		}

		elemType := t.Elem()

		rc.Path = append(rc.Path, "[]")
		itemValue := reflect.Zero(elemType).Interface()

		if itemValue == nil && elemType != typeOfEmptyInterface {
			itemValue = reflect.New(elemType).Interface()
		}

		itemsSchema, err := r.reflect(itemValue, rc, false, schema)
		if err != nil {
			return err
		}

		schema.AddType(Array)
		schema.WithItems(*(&Items{}).WithSchemaOrBool(itemsSchema.ToSchemaOrBool()))

	case reflect.Map:
		elemType := t.Elem()

		rc.Path = append(rc.Path, "{}")
		itemValue := reflect.Zero(elemType).Interface()

		if itemValue == nil && elemType != typeOfEmptyInterface {
			itemValue = reflect.New(elemType).Interface()
		}

		additionalPropertiesSchema, err := r.reflect(itemValue, rc, false, schema)
		if err != nil {
			return err
		}

		schema.AddType(Object)
		schema.WithAdditionalProperties(additionalPropertiesSchema.ToSchemaOrBool())

	case reflect.Bool:
		schema.AddType(Boolean)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.AddType(Integer)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.AddType(Integer)
		schema.WithMinimum(0)
	case reflect.Float32, reflect.Float64:
		schema.AddType(Number)
	case reflect.String:
		schema.AddType(String)
	case reflect.Interface:
		schema.Type = nil
	default:
		return fmt.Errorf("%s: type is not supported: %s", strings.Join(rc.Path[1:], "."), t.String())
	}

	return nil
}

// MakePropertyNameMapping makes property name mapping from struct value suitable for jsonschema.PropertyNameMapping.
func MakePropertyNameMapping(v interface{}, tagName string) map[string]string {
	res := make(map[string]string)

	refl.WalkTaggedFields(reflect.ValueOf(v), func(v reflect.Value, sf reflect.StructField, tag string) {
		res[sf.Name] = tag
	}, tagName)

	return res
}

func (r *Reflector) fieldVal(fv reflect.Value, ft reflect.Type) interface{} {
	fieldVal := fv.Interface()

	if ft != typeOfEmptyInterface {
		if ft.Kind() == reflect.Ptr && fv.IsNil() {
			fieldVal = reflect.New(ft.Elem()).Interface()
		} else if ft.Kind() == reflect.Interface && fv.IsNil() {
			fieldVal = reflect.New(ft).Interface()
		}
	}

	return fieldVal
}

func (r *Reflector) walkProperties(v reflect.Value, parent *Schema, rc *ReflectContext) error {
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()

		if refl.IsZero(v) {
			v = reflect.Zero(t)
		} else {
			v = v.Elem()
		}
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		var (
			tag      string
			tagFound bool
		)

		if rc.PropertyNameMapping != nil {
			tag, tagFound = rc.PropertyNameMapping[field.Name]
		} else {
			tag, tagFound = field.Tag.Lookup(rc.PropertyNameTag)
		}

		// Skip explicitly discarded field.
		if tag == "-" {
			continue
		}

		if tag == "" && field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := r.walkProperties(v.Field(i), parent, rc); err != nil {
				return err
			}

			continue
		}

		// Use unnamed fields to configure parent schema.
		if field.Name == "_" && (!rc.UnnamedFieldWithTag || tagFound) {
			if err := refl.PopulateFieldsFromTags(parent, field.Tag); err != nil {
				return err
			}

			var additionalProperties *bool
			if err := refl.ReadBoolPtrTag(field.Tag, "additionalProperties", &additionalProperties); err != nil {
				return err
			}

			if additionalProperties != nil {
				parent.AdditionalProperties = &SchemaOrBool{TypeBoolean: additionalProperties}
			}

			continue
		}

		// Skip the field if tag is not set.
		if !rc.ProcessWithoutTags && !tagFound {
			continue
		}

		// Skip the field if it's non-exported.  There is field.IsExported() method, but it was introduced in go 1.17
		// and will break backward compatibility.
		if field.PkgPath != "" {
			continue
		}

		propName := strings.Split(tag, ",")[0]
		omitEmpty := strings.Contains(tag, ",omitempty")
		required := false

		if propName == "" {
			propName = field.Name
		}

		if err := refl.ReadBoolTag(field.Tag, "required", &required); err != nil {
			return err
		}

		if required {
			parent.Required = append(parent.Required, propName)
		}

		ft := t.Field(i).Type
		fieldVal := r.fieldVal(v.Field(i), ft)

		rc.Path = append(rc.Path, propName)

		propertySchema, err := r.reflect(fieldVal, rc, true, parent)
		if err != nil {
			if errors.Is(err, ErrSkipProperty) {
				continue
			}

			return err
		}

		if !omitEmpty {
			checkNullability(&propertySchema, rc, ft)
		}

		if propertySchema.Type != nil && propertySchema.Type.SimpleTypes != nil {
			if !rc.SkipNonConstraints {
				err = checkInlineValue(&propertySchema, field, "default", propertySchema.WithDefault)
				if err != nil {
					return fmt.Errorf("%s: %w", strings.Join(append(rc.Path[1:], field.Name), "."), err)
				}
			}

			err = checkInlineValue(&propertySchema, field, "const", propertySchema.WithConst)
			if err != nil {
				return err
			}
		}

		if err := refl.PopulateFieldsFromTags(&propertySchema, field.Tag); err != nil {
			return err
		}

		deprecated := false
		if err := refl.ReadBoolTag(field.Tag, "deprecated", &deprecated); err != nil {
			return err
		} else if deprecated {
			propertySchema.WithExtraPropertiesItem("deprecated", true)
		}

		if !rc.SkipNonConstraints {
			if err := reflectExample(&propertySchema, field); err != nil {
				return err
			}
		}

		reflectEnum(&propertySchema, field.Tag, nil)

		// Remove temporary kept type from referenced schema.
		if propertySchema.Ref != nil {
			propertySchema.Type = nil
		}

		if rc.InterceptProperty != nil {
			if err := rc.InterceptProperty(propName, field, &propertySchema); err != nil {
				if errors.Is(err, ErrSkipProperty) {
					continue
				}

				return err
			}
		}

		if parent.Properties == nil {
			parent.Properties = make(map[string]SchemaOrBool, 1)
		}

		parent.Properties[propName] = SchemaOrBool{
			TypeObject: &propertySchema,
		}
	}

	return nil
}

func checkInlineValue(propertySchema *Schema, field reflect.StructField, tag string, setter func(interface{}) *Schema) error {
	var val interface{}

	t := *propertySchema.Type.SimpleTypes

	switch t {
	case Integer:
		var v *int64

		if err := refl.ReadIntPtrTag(field.Tag, tag, &v); err != nil {
			return err
		}

		if v != nil {
			val = *v
		}
	case Number:
		var v *float64

		if err := refl.ReadFloatPtrTag(field.Tag, tag, &v); err != nil {
			return err
		}

		if v != nil {
			val = *v
		}

	case String:
		var v *string

		refl.ReadStringPtrTag(field.Tag, tag, &v)

		if v != nil {
			val = *v
		}

	case Boolean:
		var v *bool

		if err := refl.ReadBoolPtrTag(field.Tag, tag, &v); err != nil {
			return err
		}

		if v != nil {
			val = *v
		}

	case Array, Null, Object:
	}

	if val != nil {
		setter(val)
	}

	return nil
}

func checkNullability(propertySchema *Schema, rc *ReflectContext, ft reflect.Type) {
	if propertySchema.HasType(Array) ||
		(propertySchema.HasType(Object) && len(propertySchema.Properties) == 0 && propertySchema.Ref == nil) {
		propertySchema.AddType(Null)
	}

	if propertySchema.Ref != nil && ft.Kind() != reflect.Struct {
		def := rc.getDefinition(*propertySchema.Ref)

		if (def.HasType(Array) || def.HasType(Object)) && !def.HasType(Null) {
			if rc.EnvelopNullability {
				refSchema := *propertySchema
				propertySchema.Ref = nil
				propertySchema.AnyOf = []SchemaOrBool{
					Null.ToSchemaOrBool(),
					refSchema.ToSchemaOrBool(),
				}
			} else {
				def.AddType(Null)
			}
		}
	}
}

func reflectExample(propertySchema *Schema, field reflect.StructField) error {
	var val interface{}

	if propertySchema.Type == nil || propertySchema.Type.SimpleTypes == nil {
		return nil
	}

	t := *propertySchema.Type.SimpleTypes
	switch t {
	case String:
		var example *string

		refl.ReadStringPtrTag(field.Tag, "example", &example)

		if example != nil {
			val = *example
		}
	case Integer:
		var example *int64

		if err := refl.ReadIntPtrTag(field.Tag, "example", &example); err != nil {
			return err
		}

		if example != nil {
			val = *example
		}
	case Number:
		var example *float64

		if err := refl.ReadFloatPtrTag(field.Tag, "example", &example); err != nil {
			return err
		}

		if example != nil {
			val = *example
		}
	case Boolean:
		var example *bool

		if err := refl.ReadBoolPtrTag(field.Tag, "example", &example); err != nil {
			return err
		}

		if example != nil {
			val = *example
		}
	case Array, Null, Object:
		return nil
	}

	if val != nil {
		propertySchema.WithExamples(val)
	}

	return nil
}

func reflectEnum(schema *Schema, fieldTag reflect.StructTag, fieldVal interface{}) {
	enum := enum{}
	enum.loadFromField(fieldTag, fieldVal)

	if len(enum.items) > 0 {
		schema.Enum = enum.items
		if len(enum.names) > 0 {
			if schema.ExtraProperties == nil {
				schema.ExtraProperties = make(map[string]interface{}, 1)
			}

			schema.ExtraProperties[XEnumNames] = enum.names
		}
	}
}

// enum can be use for sending enum data that need validate.
type enum struct {
	items []interface{}
	names []string
}

// loadFromField loads enum from field tag: json array or comma-separated string.
func (enum *enum) loadFromField(fieldTag reflect.StructTag, fieldVal interface{}) {
	if e, isEnumer := fieldVal.(NamedEnum); isEnumer {
		enum.items, enum.names = e.NamedEnum()
	}

	if e, isEnumer := fieldVal.(Enum); isEnumer {
		enum.items = e.Enum()
	}

	if enumTag := fieldTag.Get("enum"); enumTag != "" {
		var e []interface{}

		err := json.Unmarshal([]byte(enumTag), &e)
		if err != nil {
			es := strings.Split(enumTag, ",")
			e = make([]interface{}, len(es))

			for i, s := range es {
				e[i] = s
			}
		}

		enum.items = e
	}
}

type (
	oneOf []interface{}
	allOf []interface{}
	anyOf []interface{}
)

var (
	_ Preparer = oneOf{}
	_ Preparer = anyOf{}
	_ Preparer = allOf{}
)

// OneOf exposes list of values as JSON "oneOf" schema.
func OneOf(v ...interface{}) OneOfExposer {
	return oneOf(v)
}

// PrepareJSONSchema removes unnecessary constraints.
func (oneOf) PrepareJSONSchema(schema *Schema) error {
	schema.Type = nil
	schema.Items = nil

	return nil
}

// JSONSchemaOneOf implements OneOfExposer.
func (o oneOf) JSONSchemaOneOf() []interface{} {
	return o
}

// InlineJSONSchema implements SchemaInliner.
func (o oneOf) InlineJSONSchema() {}

// AnyOf exposes list of values as JSON "anyOf" schema.
func AnyOf(v ...interface{}) AnyOfExposer {
	return anyOf(v)
}

// PrepareJSONSchema removes unnecessary constraints.
func (anyOf) PrepareJSONSchema(schema *Schema) error {
	schema.Type = nil
	schema.Items = nil

	return nil
}

// JSONSchemaAnyOf implements AnyOfExposer.
func (o anyOf) JSONSchemaAnyOf() []interface{} {
	return o
}

// InlineJSONSchema implements SchemaInliner.
func (o anyOf) InlineJSONSchema() {}

// AllOf exposes list of values as JSON "allOf" schema.
func AllOf(v ...interface{}) AllOfExposer {
	return allOf(v)
}

// PrepareJSONSchema removes unnecessary constraints.
func (allOf) PrepareJSONSchema(schema *Schema) error {
	schema.Type = nil
	schema.Items = nil

	return nil
}

// JSONSchemaAllOf implements AllOfExposer.
func (o allOf) JSONSchemaAllOf() []interface{} {
	return o
}

// InlineJSONSchema implements SchemaInliner.
func (o allOf) InlineJSONSchema() {}
