package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const genHeader = "// Код сгенерирован автоматически. НЕ РЕДАКТИРОВАТЬ.\n\npackage %s\n\n"

type fieldDef struct {
	name     string
	typeExpr ast.Expr
}

type structInfo struct {
	name   string
	fields []fieldDef
}

type pkgData struct {
	pkgName string
	structs []*structInfo
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка получения рабочей директории: %v\n", err)
		os.Exit(1)
	}

	pkgMap := make(map[string]*pkgData)

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || strings.HasPrefix(name, ".") || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") ||
			strings.HasSuffix(path, "reset.gen.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		dir := filepath.Dir(path)
		if _, exists := pkgMap[dir]; !exists {
			pkgMap[dir] = &pkgData{pkgName: f.Name.Name}
		}

		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			if !isGenerateReset(genDecl.Doc) {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				si := &structInfo{name: typeSpec.Name.Name}
				for _, field := range structType.Fields.List {
					for _, fieldName := range field.Names {
						si.fields = append(si.fields, fieldDef{
							name:     fieldName.Name,
							typeExpr: field.Type,
						})
					}
				}
				pkgMap[dir].structs = append(pkgMap[dir].structs, si)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка обхода директорий: %v\n", err)
		os.Exit(1)
	}

	for dir, data := range pkgMap {
		if len(data.structs) == 0 {
			continue
		}

		var buf bytes.Buffer
		fmt.Fprintf(&buf, genHeader, data.pkgName)

		for _, s := range data.structs {
			writeReset(&buf, s)
		}

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка форматирования в %s: %v\nисходный код:\n%s\n", dir, err, buf.String())
			continue
		}

		outPath := filepath.Join(dir, "reset.gen.go")
		if err := os.WriteFile(outPath, formatted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ошибка записи файла %s: %v\n", outPath, err)
			os.Exit(1)
		}
		fmt.Printf("сгенерирован: %s\n", outPath)
	}
}

// isGenerateReset проверяет наличие директивы // generate:reset в группе комментариев.
func isGenerateReset(cg *ast.CommentGroup) bool {
	if cg == nil {
		return false
	}
	for _, c := range cg.List {
		if strings.TrimSpace(c.Text) == "// generate:reset" {
			return true
		}
	}
	return false
}

// receiverName возвращает имя получателя метода — первая буква имени типа в нижнем регистре.
func receiverName(typeName string) string {
	for _, r := range typeName {
		return string(unicode.ToLower(r))
	}
	return "v"
}

// writeReset генерирует метод Reset() для структуры.
func writeReset(buf *bytes.Buffer, s *structInfo) {
	recv := receiverName(s.name)
	fmt.Fprintf(buf, "func (%s *%s) Reset() {\n", recv, s.name)
	fmt.Fprintf(buf, "\tif %s == nil {\n\t\treturn\n\t}\n\n", recv)
	for _, f := range s.fields {
		writeFieldReset(buf, recv, f.name, f.typeExpr)
	}
	fmt.Fprintf(buf, "}\n\n")
}

// zeroValue возвращает нулевое значение для примитивного типа.
// Возвращает ("", false) если тип не является известным примитивом.
func zeroValue(typeName string) (string, bool) {
	switch typeName {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune", "float32", "float64", "complex64", "complex128":
		return "0", true
	case "bool":
		return "false", true
	case "string":
		return `""`, true
	}
	return "", false
}

// writeFieldReset генерирует код сброса для одного поля структуры.
func writeFieldReset(buf *bytes.Buffer, recv, name string, typ ast.Expr) {
	switch t := typ.(type) {
	case *ast.Ident:
		if zero, ok := zeroValue(t.Name); ok {
			// Примитив — присваиваем нулевое значение
			fmt.Fprintf(buf, "\t%s.%s = %s\n", recv, name, zero)
		} else {
			// Именованный тип из текущего пакета — пробуем вызвать Reset() через интерфейс
			fmt.Fprintf(buf, "\tif resetter, ok := any(&%s.%s).(interface{ Reset() }); ok {\n", recv, name)
			fmt.Fprintf(buf, "\t\tresetter.Reset()\n")
			fmt.Fprintf(buf, "\t}\n")
		}

	case *ast.ArrayType:
		if t.Len == nil {
			// Слайс — обрезаем до нуля, не зануляем
			fmt.Fprintf(buf, "\t%s.%s = %s.%s[:0]\n", recv, name, recv, name)
		}
		// Массивы фиксированного размера задание не упоминает — пропускаем

	case *ast.MapType:
		// Мапа — очищаем встроенным clear
		fmt.Fprintf(buf, "\tclear(%s.%s)\n", recv, name)

	case *ast.StarExpr:
		switch inner := t.X.(type) {
		case *ast.Ident:
			if zero, ok := zeroValue(inner.Name); ok {
				// Указатель на примитив — зануляем значение по указателю
				fmt.Fprintf(buf, "\tif %s.%s != nil {\n\t\t*%s.%s = %s\n\t}\n",
					recv, name, recv, name, zero)
			} else {
				// Указатель на структуру — пробуем вызвать Reset() через интерфейс
				fmt.Fprintf(buf, "\tif resetter, ok := any(%s.%s).(interface{ Reset() }); ok && %s.%s != nil {\n",
					recv, name, recv, name)
				fmt.Fprintf(buf, "\t\tresetter.Reset()\n")
				fmt.Fprintf(buf, "\t}\n")
			}
		default:
			// Указатель на импортированный или составной тип — пробуем Reset() через интерфейс
			fmt.Fprintf(buf, "\tif resetter, ok := any(%s.%s).(interface{ Reset() }); ok && %s.%s != nil {\n",
				recv, name, recv, name)
			fmt.Fprintf(buf, "\t\tresetter.Reset()\n")
			fmt.Fprintf(buf, "\t}\n")
		}

	case *ast.SelectorExpr:
		// Импортированный тип (pkg.Type) — пробуем Reset() через интерфейс на указателе
		fmt.Fprintf(buf, "\tif resetter, ok := any(&%s.%s).(interface{ Reset() }); ok {\n", recv, name)
		fmt.Fprintf(buf, "\t\tresetter.Reset()\n")
		fmt.Fprintf(buf, "\t}\n")
	}
}
