package main

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	var flags flag.FlagSet

	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		// this enables optional fields to be supported.
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

		// for all files, generate setters for all fields for all messages.
		for _, file := range gen.Files {
			if !file.Generate || len(file.Messages) == 0 {
				continue
			}

			newFile := gen.NewGeneratedFile(file.GeneratedFilenamePrefix+".pb.setters.go", ".")
			newFile.P("// Code generated by protoc-gen-go-setters. DO NOT EDIT.")
			newFile.P("// source: ", file.GeneratedFilenamePrefix, ".proto")
			newFile.P("package " + file.GoPackageName)

			for _, message := range file.Messages {
				generateMessageSetters(gen, newFile, message)
			}
		}

		return nil
	})
}

func generateMessageSetters(gen *protogen.Plugin, newFile *protogen.GeneratedFile, message *protogen.Message) {
	messageName := message.GoIdent.GoName

	// generate any messages declared in a nested manner.
	// May introduce potential endless loop if messages contain circular refrences.
	// May duplicate code if Message refrences already existing message
	for _, v := range message.Messages {
		if v.Desc.IsMapEntry() {
			continue
		}
		generateMessageSetters(gen, newFile, v)
	}

	// generate oneof setters.
	for _, oneof := range message.Oneofs {
		for _, field := range oneof.Fields {
			// this distinguishes between a oneof and an optional field.
			if field.Desc.HasOptionalKeyword() {
				continue
			}

			goType, _ := fieldGoType(newFile, field)

			inputWrapperName := "&" + field.GoIdent.GoName

			info := OneOfSetterTemplate{
				MessageName:      field.Parent.GoIdent.GoName,
				StructFieldName:  field.Oneof.GoName,
				InputWrapperName: inputWrapperName,
				FieldName:        field.GoName,
				FieldType:        goType,
			}
			content := ExecuteTemplate(oneofTpl, info)
			newFile.P(content)
		}
	}

	for _, field := range message.Fields {
		if field.Oneof != nil && !field.Desc.HasOptionalKeyword() {
			continue
		}

		goType, pointer := fieldGoType(newFile, field)
		fieldType := goType
		if pointer {
			fieldType = "*" + goType
		}

		fieldName := field.GoName
		info := SetterTemplate{
			MessageName: messageName,
			FieldName:   field.GoName,
			FieldType:   fieldType,
		}

		content := ExecuteTemplate(tpl, info)
		newFile.P(content)

		// will generate an append func.
		if field.Desc.IsList() {
			info.FieldType = strings.Replace(info.FieldType, "[]", "...", 1) // only replace one to prevent [][]byte becoming ......byte
			arrayAddition := ExecuteTemplate(appendArrayTpl, info)
			newFile.P(arrayAddition)
		}

		// map set func
		if field.Desc.IsMap() {
			keyType, _ := fieldGoType(newFile, field.Message.Fields[0])
			valType, _ := fieldGoType(newFile, field.Message.Fields[1])

			ms := MapSetterTemplate{
				MessageName: messageName,
				FieldName:   fieldName,
				KeyType:     keyType,
				ValueType:   valType,
			}
			mapSetKey := ExecuteTemplate(mapSetTpl, ms)
			newFile.P(mapSetKey)
		}
	}
}

// Might be worthwhile looking at how go plugin generates proto messages.
// see https://github.com/protocolbuffers/protobuf-go/blob/master/cmd/protoc-gen-go/internal_gengo/init.go
// go types https://github.com/protocolbuffers/protobuf-go/blob/master/cmd/protoc-gen-go/internal_gengo/main.go#L646

// from https://github.com/protocolbuffers/protobuf-go/blob/master/cmd/protoc-gen-go/internal_gengo/main.go#L646
func fieldGoType(g *protogen.GeneratedFile, field *protogen.Field) (goType string, pointer bool) {
	if field.Desc.IsWeak() {
		return "struct{}", false
	}

	pointer = field.Desc.HasPresence()
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		goType = "bool"
	case protoreflect.EnumKind:
		goType = field.Enum.GoIdent.GoName
		// only import go type if it is not in the same package.
		if field.Enum.GoIdent.GoImportPath != field.Parent.GoIdent.GoImportPath {
			goType = g.QualifiedGoIdent(field.Enum.GoIdent)
		}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		goType = "int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		goType = "uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		goType = "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		goType = "uint64"
	case protoreflect.FloatKind:
		goType = "float32"
	case protoreflect.DoubleKind:
		goType = "float64"
	case protoreflect.StringKind:
		goType = "string"
	case protoreflect.BytesKind:
		goType = "[]byte"
		pointer = false // rely on nullability of slices for presence
	case protoreflect.MessageKind, protoreflect.GroupKind:
		pointer = false // pointer captured as part of the type
		goType = "*" + field.Message.GoIdent.GoName
		// only import go type if it is not in the same package.
		if field.Message.GoIdent.GoImportPath != field.Parent.GoIdent.GoImportPath {
			goType = "*" + g.QualifiedGoIdent(field.Message.GoIdent)
		}
	}

	switch {
	case field.Desc.IsList():
		return "[]" + goType, false
	case field.Desc.IsMap():
		keyType, _ := fieldGoType(g, field.Message.Fields[0])
		valType, _ := fieldGoType(g, field.Message.Fields[1])
		return fmt.Sprintf("map[%v]%v", keyType, valType), false
	}
	return goType, pointer
}

type SetterTemplate struct {
	MessageName string
	FieldName   string
	FieldType   string
}

const tpl = `

func (x *{{.MessageName}} ) Set{{.FieldName}}(in {{.FieldType}} ){
	x.{{.FieldName}} = in
}

`

const appendArrayTpl = `

func (x *{{.MessageName}} ) Append{{.FieldName}}(in {{.FieldType}} ) {
	x.{{.FieldName}} = append(x.{{.FieldName}}, in...)
}

`

type OneOfSetterTemplate struct {
	MessageName string
	// maybe parentName
	StructFieldName  string
	InputWrapperName string
	FieldName        string
	FieldType        string
}

// this is not necessarily correct
const oneofTpl = `

func (x *{{.MessageName}} ) Set{{.FieldName}}(in {{.FieldType}} ) {
	x.{{.StructFieldName}} = {{.InputWrapperName}}{ {{.FieldName}}:in }
}

`

type MapSetterTemplate struct {
	MessageName string
	FieldName   string
	KeyType     string
	ValueType   string
}

const mapSetTpl = `

func (x *{{.MessageName}} ) Set{{.FieldName}}Key(key {{.KeyType}}, val {{.ValueType}} ){
	x.{{.FieldName}}[key] = val
}

`

func ExecuteTemplate(tplate string, data any) string {
	templ, err := template.New("").Parse(tplate)
	if err != nil {
		panic(err)
	}
	buffy := bytes.NewBuffer([]byte{})
	if err := templ.Execute(buffy, data); err != nil {
		panic(err)
	}
	return buffy.String()
}
