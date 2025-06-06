package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/pflag"
)

type ProtoField struct {
	Name     string
	Type     string
	Repeated bool
	Optional bool
	Tag      string
}

var (
	inGlob  = pflag.StringP("in", "i", "", "input go file, support glob pattern")
	outDir  = pflag.StringP("out", "o", "", "output directory")
	baseDir = pflag.StringP("base", "b", "", "base directory for relative import path")
)

func loadFlags() {
	pflag.Parse()
	if *inGlob == "" || *outDir == "" || *baseDir == "" {
		fmt.Println("missing input or output directory or base dir")
		pflag.Usage()
		os.Exit(1)
	}
	if !strings.HasSuffix(*inGlob, "*.go") {
		*inGlob = filepath.Join(*inGlob, "*.go")
	}

	fmt.Println("base dir:", *baseDir)
	fmt.Println("input glob pattern:", *inGlob)
	fmt.Println("output directory:", *outDir)
}

func main() {
	loadFlags()

	files, err := filepath.Glob(*inGlob)
	outputDir := *outDir
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	messagesMap := make(map[string]*ProtoMessage)
	// message name -> import path
	importsMap := make(map[string]string)
	// path -> message name list
	importsMapReversed := make(map[string][]string)

	for _, file := range files {
		messages, imports, err := ParseGoFile(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for _, msg := range messages {
			messagesMap[msg.Name] = msg
		}
		for name, path := range imports {
			importsMap[name] = path
			importsMapReversed[path] = append(importsMapReversed[path], name)
		}
	}

	for _, msg := range messagesMap {
		msg.ParseFields(messagesMap)
	}

	for protoFileName, msgList := range importsMapReversed {
		outputPath := filepath.Join(outputDir, protoFileName)

		var imports []string
		for _, msg := range msgList {
			msg := messagesMap[msg]
			currImports := SearchImports(msg, importsMap)
			imports = append(imports, currImports...)
		}
		imports = lo.Uniq(imports)
		messages := lo.Map(msgList, func(msg string, _idx int) *ProtoMessage {
			return messagesMap[msg]
		})

		packageName := protoFileName[:strings.LastIndex(protoFileName, ".")]
		protoContent := GenerateProto(messages, packageName, imports)
		if err := os.WriteFile(outputPath, []byte(protoContent), 0644); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	fmt.Println("done")
}

func SearchImports(msg *ProtoMessage, importsMap map[string]string) (imports []string) {
	for _, field := range msg.Fields {
		// skip if type is primitive
		if lo.Contains(primitives, field.Type) {
			continue
		}
		// skip if type is defined in the same file
		if path, ok := importsMap[field.Type]; ok {
			if path == msg.Path {
				continue
			}
			imports = append(imports, path)
		}
	}
	return
}

// protobuf primitive types
var primitives = []string{
	"bool",
	"int32",
	"int64",
	"uint32",
	"uint64",
	"float",
	"double",
	"string",
	"bytes",
}

var aliasMap = make(map[string]string)

func ParseGoFile(filename string) ([]*ProtoMessage, map[string]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	messages := make([]*ProtoMessage, 0)
	imports := make(map[string]string)
	fileParts := strings.Split(filename, "/")
	fileName := fileParts[len(fileParts)-1]
	if !strings.HasSuffix(fileName, ".go") {
		return nil, nil, fmt.Errorf("invalid file name: %s", filename)
	}
	protoFileName := strings.TrimSuffix(fileName, ".go") + ".proto"

	ast.Inspect(node, func(n ast.Node) bool {
		if decl, ok := n.(*ast.GenDecl); ok && decl.Tok == token.TYPE {
			for _, spec := range decl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					msgName := typeSpec.Name.Name
					if rune(msgName[0]) >= 'a' && rune(msgName[0]) <= 'z' {
						// skip unexported message
						continue
					}
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						msg := &ProtoMessage{
							Name:   msgName,
							Fields: make([]ProtoField, 0),
						}
						imports[msg.Name] = protoFileName
						msg.Path = protoFileName
						msg.OriginalFieldsList = structType.Fields.List
						messages = append(messages, msg)
					} else if aliasType, ok := typeSpec.Type.(*ast.Ident); ok {
						// handle alias type
						aliasMap[msgName] = typeAdaptor(aliasType.Name)
					} else if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						// handle interface type as alias of google.protobuf.Any
						aliasMap[msgName] = "google.protobuf.Any"
					}
				}
			}
		}

		return true
	})

	return messages, imports, nil
}

func typeAdaptor(typeName string) string {
	switch typeName {
	case "int":
		typeName = "int64"
	case "float32":
		fallthrough
	case "float64":
		fallthrough
	case "float":
		typeName = "double"
	case "uint":
		fallthrough
	case "uint8":
		fallthrough
	case "uint16":
		typeName = "uint32"
	case "uint64":
		typeName = "uint64"
	case "time.Duration":
		typeName = "google.protobuf.Duration"
	case "time.Time":
		typeName = "google.protobuf.Timestamp"
	}
	// else remain unchanged
	return typeName
}

// recursively find alias type, if not found, return original type name
func tryGetRootAliasType(typeName string, aliasMap map[string]string) string {
	if lo.Contains(primitives, typeName) {
		return typeName
	}
	if strings.HasPrefix(typeName, "google.protobuf.") {
		return typeName
	}
	if alias, found := aliasMap[typeName]; found {
		return tryGetRootAliasType(alias, aliasMap)
	}
	return typeName
}

func getFieldProtoType(expr ast.Expr) (string, bool, bool) {
	var fieldType string
	var isRepeated, isOptional bool

	switch t := expr.(type) {
	case *ast.Ident:
		fieldType = t.Name
	case *ast.StarExpr:
		isOptional = true
		fieldType, _, _ = getFieldProtoType(t.X)
	case *ast.ArrayType:
		isRepeated = true
		fieldType, _, _ = getFieldProtoType(t.Elt)
	case *ast.SelectorExpr:
		// handle imported type
		fieldType = fmt.Sprintf("%s.%s", t.X.(*ast.Ident).Name, t.Sel.Name)
	case *ast.MapType:
		// handle map type
		fieldType = "map<"
		keyType, _, _ := getFieldProtoType(t.Key)
		fieldType += keyType + ", "
		valueType, _, _ := getFieldProtoType(t.Value)
		fieldType += valueType + ">"
	default:
		fieldType = "google.protobuf.Any"
	}

	return fieldType, isRepeated, isOptional
}

var tagRegexSplitter = regexp.MustCompile(`[\w_][\w\d_]*:`)

// extract tag from field
// e.g. `json:"name,omitempty"` -> "name"
func getTagFromField(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}

	rawTag := strings.Trim(tag.Value, "`")

	parts := tagRegexSplitter.Split(rawTag, 2)
	if len(parts) < 2 {
		return ""
	}

	variableName := strings.Trim(parts[1], "\"")
	if idx := strings.Index(variableName, ","); idx != -1 {
		variableName = variableName[:idx] // ignore "omitempty"
	}

	return variableName
}

func GenerateProto(messages []*ProtoMessage, packageName string, imports []string) string {
	var sb strings.Builder
	sb.WriteString("/* Code generated by go2pb. DO NOT EDIT. */\n")
	sb.WriteString("/* source: https://github.com/Plasmatium/go2pb */\n\n")
	sb.WriteString("syntax = \"proto3\";\n\n")
	sb.WriteString("package ")
	protoPackage := filepath.Base(*baseDir)
	sb.WriteString(protoPackage)
	sb.WriteString(";\n\n")

	for _, imp := range imports {
		if *baseDir != "" {
			imp = strings.Join([]string{protoPackage, imp}, "/")
		}
		sb.WriteString(fmt.Sprintf("import \"%s\";\n", imp))
	}
	sb.WriteString(`option go_package = "`)
	sb.WriteString(*baseDir)
	sb.WriteString(`";

import "google/protobuf/any.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/timestamp.proto";


`)
	if len(imports) > 0 {
		sb.WriteString("\n")
	}

	for _, msg := range messages {
		GenerateMessage(msg, &sb)
	}

	return sb.String()
}

func GenerateMessage(msg *ProtoMessage, sb *strings.Builder) {
	sb.WriteString("message " + msg.Name + " {\n")
	fieldNumber := 1
	for _, field := range msg.Fields {
		sb.WriteString("  ")
		if field.Repeated {
			sb.WriteString("repeated ")
		} else if field.Optional {
			sb.WriteString("optional ")
		}

		typ := tryGetRootAliasType(field.Type, aliasMap)
		typ = typeAdaptor(typ)

		variableName := field.Tag
		if variableName == "" {
			variableName = ToSnakeCase(field.Name)
		}

		sb.WriteString(typ + " " + variableName)
		sb.WriteString(" = " + fmt.Sprintf("%d", fieldNumber) + ";\n")
		fieldNumber++
	}
	sb.WriteString("}\n\n")
}
