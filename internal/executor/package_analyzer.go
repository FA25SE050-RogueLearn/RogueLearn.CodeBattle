package executor

import (
	"fmt"
	"go/parser"
	"go/token"
)

// PackageAnalyzer helps finding which packages are being imported in a piece of code
type PackageAnalyzer interface {
	Analyze(code string) ([]string, error)
}

type GoPackageAnalyzer struct {
}

func NewGoPackageAnalyzer() *GoPackageAnalyzer {
	return &GoPackageAnalyzer{}
}

func (p *GoPackageAnalyzer) Analyze(code string) ([]string, error) {
	var pkgs []string
	fset := token.NewFileSet()

	// 0 means parse everything
	node, err := parser.ParseFile(fset, "code.go", code, 0)
	if err != nil {
		fmt.Println("error")
		return pkgs, err
	}

	// this take all the packages that are explicitly imported (import statement)
	for _, i := range node.Imports {
		// strip the double quote ("fmt", "strings")
		p := i.Path.Value[1 : len(i.Path.Value)-1]
		pkgs = append(pkgs, p)
	}

	return pkgs, nil
}
