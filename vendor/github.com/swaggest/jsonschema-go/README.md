# JSON Schema structures for Go

<img align="right" width="100px" src="https://avatars0.githubusercontent.com/u/13019229?s=200&v=4">

[![Build Status](https://github.com/swaggest/jsonschema-go/workflows/test-unit/badge.svg)](https://github.com/swaggest/jsonschema-go/actions?query=branch%3Amaster+workflow%3Atest-unit)
[![Coverage Status](https://codecov.io/gh/swaggest/jsonschema-go/branch/master/graph/badge.svg)](https://codecov.io/gh/swaggest/jsonschema-go)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/swaggest/jsonschema-go)
[![time tracker](https://wakatime.com/badge/github/swaggest/jsonschema-go.svg)](https://wakatime.com/badge/github/swaggest/jsonschema-go)
![Code lines](https://sloc.xyz/github/swaggest/jsonschema-go/?category=code)
![Comments](https://sloc.xyz/github/swaggest/jsonschema-go/?category=comments)

This library provides Go structures to marshal/unmarshal and reflect [JSON Schema](https://json-schema.org/) documents.

## Reflector

[Documentation](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Reflector.Reflect).

```go
type MyStruct struct {
    Amount float64  `json:"amount" minimum:"10.5" example:"20.6" required:"true"`
    Abc    string   `json:"abc" pattern:"[abc]"`
    _      struct{} `additionalProperties:"false"`                   // Tags of unnamed field are applied to parent schema.
    _      struct{} `title:"My Struct" description:"Holds my data."` // Multiple unnamed fields can be used.
}

reflector := jsonschema.Reflector{}

schema, err := reflector.Reflect(MyStruct{})
if err != nil {
    log.Fatal(err)
}

j, err := json.MarshalIndent(schema, "", " ")
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(j))

// Output:
// {
//  "title": "My Struct",
//  "description": "Holds my data.",
//  "required": [
//   "amount"
//  ],
//  "additionalProperties": false,
//  "properties": {
//   "abc": {
//    "pattern": "[abc]",
//    "type": "string"
//   },
//   "amount": {
//    "examples": [
//     20.6
//    ],
//    "minimum": 10.5,
//    "type": "number"
//   }
//  },
//  "type": "object"
// }
```

## Customization

By default, JSON Schema is generated from Go struct field types and tags.
It works well for the majority of cases, but if it does not there are rich customization options.

### Implementing interfaces on a type

There are a few interfaces that can be implemented on a type to customize JSON Schema generation.

* [`Preparer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Preparer) allows to change generated JSON Schema.
* [`Exposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Exposer) overrides generated JSON Schema.
* [`RawExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go/jsonschema.RawExposer) overrides generated JSON Schema.
* [`Described`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Described) exposes description.
* [`Titled`](https://pkg.go.dev/github.com/swaggest/jsonschema-go/jsonschema.Titled) exposes title.
* [`Enum`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Enum) exposes enum values.
* [`NamedEnum`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#NamedEnum) exposes enum values with names.

And a few interfaces to expose subschemas (`anyOf`, `allOf`, `oneOf`, `not` and `if`, `then`, `else`).
* [`AnyOfExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go/jsonschema.AnyOfExposer) exposes `anyOf` subschemas.
* [`AllOfExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go/jsonschema.AllOfExposer) exposes `allOf` subschemas.
* [`OneOfExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go/jsonschema.OneOfExposer) exposes `oneOf` subschemas.
* [`NotExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go/jsonschema.NotExposer) exposes `not` subschema.
* [`IfExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go/jsonschema.IfExposer) exposes `if` subschema.
* [`ThenExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go/jsonschema.ThenExposer) exposes `then` subschema.
* [`ElseExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go/jsonschema.ElseExposer) exposes `else` subschema.

### Configuring the reflector

Additional centralized configuration is available with 
[`jsonschema.ReflectContext`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#ReflectContext) and 
[`Reflect`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Reflector.Reflect) options.

* [`CollectDefinitions`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#CollectDefinitions) disables definitions storage in schema and calls user function instead.
* [`DefinitionsPrefix`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#DefinitionsPrefix) sets path prefix for definitions.
* [`PropertyNameTag`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#PropertyNameTag) allows using field tags other than `json`.
* [`InterceptType`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#InterceptType) called for every type during schema reflection.
* [`InterceptProperty`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#InterceptProperty) called for every property during schema reflection.
* [`InlineRefs`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#InlineRefs) tries to inline all references (instead of creating definitions).
* [`RootNullable`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#RootNullable) enables nullability of root schema.
* [`RootRef`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#RootRef) converts root schema to definition reference.
* [`StripDefinitionNamePrefix`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#StripDefinitionNamePrefix) strips prefix from definition name.
* [`PropertyNameMapping`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#PropertyNameMapping) explicit name mapping instead field tags.
* [`ProcessWithoutTags`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#ProcessWithoutTags) enables processing fields without any tags specified.
