// Пакет osexit содержит анализатор, запрещающий прямой вызов os.Exit
// в функции main пакета main. Вызовы os.Exit во вспомогательных функциях не проверяются.
package osexit

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer — анализатор, проверяющий отсутствие прямых вызовов os.Exit в main.
var Analyzer = &analysis.Analyzer{
	Name:     "osexit",
	Doc:      "запрещает прямые вызовы os.Exit в функции main пакета main",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Анализатор актуален только для пакета main.
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Обходим только объявления функций верхнего уровня.
	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" || fn.Body == nil {
			return
		}

		// Пропускаем сгенерированные тест-main файлы, которые инструментарий Go
		// создаёт для каждого тест-бинарника (они намеренно вызывают os.Exit(m.Run())).
		fileName := pass.Fset.File(fn.Pos()).Name()
		if strings.HasSuffix(fileName, "_testmain.go") ||
			strings.Contains(fileName, "go-build") {
			return
		}

		// Обходим тело main() в поисках вызовов os.Exit.
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			if pkgIdent.Name == "os" && sel.Sel.Name == "Exit" {
				pass.Reportf(call.Pos(), "прямой вызов os.Exit в функции main пакета main")
			}

			return true
		})
	})

	return nil, nil
}
