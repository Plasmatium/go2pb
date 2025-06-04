package main

import (
	"go/ast"

	"github.com/samber/lo"
)

type ProtoMessage struct {
	Name   string
	Path   string
	Fields []ProtoField

	OriginalFieldsList []*ast.Field
}

func (m *ProtoMessage) ParseFields(mmap map[string]*ProtoMessage) {
	if len(m.Fields) > 0 {
		return
	}
	firstLevelTypeNames := lo.FilterMap(m.OriginalFieldsList, func(field *ast.Field, _idx int) (string, bool) {
		if len(field.Names) == 0 {
			return "", false
		}
		return field.Names[0].Name, true
	})

	fieldList := make([]ProtoField, 0)
	for _, field := range m.OriginalFieldsList {
		// handle embeded field
		if len(field.Names) == 0 {
			// find that embeded message and recursively parse it
			typeName, _, _ := getFieldProtoType(field.Type)
			embededMsg := mmap[typeName]
			embededMsg.ParseFields(mmap)

			// shadow embeded fields if it has the same name with current message
			filtered := lo.Filter(embededMsg.Fields, func(f ProtoField, _idx int) bool {
				return !lo.Contains(firstLevelTypeNames, f.Name)
			})

			fieldList = append(fieldList, filtered...)
			continue
		}
		// handle multiple names in one line
		for _, name := range field.Names {
			if rune(name.Name[0]) >= 'a' && rune(name.Name[0]) <= 'z' {
				// skip unexported field
				continue
			}
			fieldType, repeated, optional := getFieldProtoType(field.Type)
			tag := getTagFromField(field.Tag)
			if tag == "-" {
				continue
			}

			fieldType = typeAdaptor(fieldType)

			fieldList = append(fieldList, ProtoField{
				Name:     name.Name,
				Type:     fieldType,
				Repeated: repeated,
				Optional: optional,
				Tag:      tag,
			})
		}
	}
	m.Fields = fieldList
}
