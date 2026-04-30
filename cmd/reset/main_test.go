package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReceiverName(t *testing.T) {
	assert.Equal(t, "u", receiverName("URLService"))
	assert.Equal(t, "h", receiverName("Handler"))
	assert.Equal(t, "v", receiverName(""))
}

func TestZeroValue(t *testing.T) {
	cases := map[string]string{
		"int":     "0",
		"int64":   "0",
		"uint":    "0",
		"float32": "0",
		"bool":    "false",
		"string":  `""`,
	}
	for typ, want := range cases {
		got, ok := zeroValue(typ)
		assert.True(t, ok, typ)
		assert.Equal(t, want, got, typ)
	}
	_, ok := zeroValue("MyStruct")
	assert.False(t, ok)
}

func TestIsGenerateReset(t *testing.T) {
	assert.False(t, isGenerateReset(nil))

	cg := &ast.CommentGroup{List: []*ast.Comment{{Text: "// other comment"}}}
	assert.False(t, isGenerateReset(cg))

	cg = &ast.CommentGroup{List: []*ast.Comment{{Text: "// generate:reset"}}}
	assert.True(t, isGenerateReset(cg))
}

func parseFieldType(t *testing.T, src string) ast.Expr {
	t.Helper()
	expr, err := parser.ParseExpr(src)
	require.NoError(t, err)
	return expr
}

func TestWriteFieldReset(t *testing.T) {
	cases := []struct {
		name    string
		typeSrc string
		want    string
	}{
		{"int", "int", "v.X = 0"},
		{"string", "string", `v.X = ""`},
		{"bool", "bool", "v.X = false"},
		{"slice", "[]int", "v.X = v.X[:0]"},
		{"map", "map[string]int", "clear(v.X)"},
		{"named", "MyType", "Reset()"},
		{"pointer to primitive", "*int", "*v.X = 0"},
		{"pointer to struct", "*MyType", "Reset()"},
		{"selector", "pkg.Type", "Reset()"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeFieldReset(&buf, "v", "X", parseFieldType(t, tt.typeSrc))
			assert.Contains(t, buf.String(), tt.want)
		})
	}
}

func TestWriteReset(t *testing.T) {
	src := `package p; type Foo struct { A int; B string }`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	require.NoError(t, err)

	gd := f.Decls[0].(*ast.GenDecl)
	ts := gd.Specs[0].(*ast.TypeSpec)
	st := ts.Type.(*ast.StructType)

	si := &structInfo{name: ts.Name.Name}
	for _, field := range st.Fields.List {
		for _, fname := range field.Names {
			si.fields = append(si.fields, fieldDef{name: fname.Name, typeExpr: field.Type})
		}
	}

	var buf bytes.Buffer
	writeReset(&buf, si)
	out := buf.String()
	assert.Contains(t, out, "func (f *Foo) Reset()")
	assert.Contains(t, out, "if f == nil")
	assert.Contains(t, out, "f.A = 0")
	assert.Contains(t, out, `f.B = ""`)
}
