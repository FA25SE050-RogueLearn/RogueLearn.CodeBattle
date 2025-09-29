package executor

import (
	"fmt"
	"strings"
)

const (
	DefaultMaxCodeLength = 1000

	goCodePlaceHolder   = "// USER_CODE_HERE"
	goImportPlaceHolder = "// IMPORTS_HERE"

	pythonCodePlaceHolder = "# USER_CODE_HERE"

	jsCodePlaceHolder = "// USER_CODE_HERE"
)

type LanguagePlaceHolder struct {
	codePlaceHolder   string
	importPlaceHolder string
}

func getLanguagePlaceHolder(lang string) LanguagePlaceHolder {
	var result LanguagePlaceHolder
	switch lang {
	case "Golang":
		result = LanguagePlaceHolder{
			codePlaceHolder:   goCodePlaceHolder,
			importPlaceHolder: goImportPlaceHolder,
		}
	case "Python":
		result = LanguagePlaceHolder{
			codePlaceHolder:   pythonCodePlaceHolder,
			importPlaceHolder: "",
		}
	case "Javascript":
		result = LanguagePlaceHolder{
			codePlaceHolder:   jsCodePlaceHolder,
			importPlaceHolder: "",
		}
	}

	return result
}

type CodeBuilder interface {
	Build(lang, driverCode, userCode string) (string, error)
}

// ConcreteCodeBuilder is a struct for building complete and functionable code.
type ConcreteCodeBuilder struct {
	lang string

	// store a map of package analyzers for future compiled languages integration
	pkgAnalyzers map[string]PackageAnalyzer
}

func NewCodeBuilder(pkgAnalyzer PackageAnalyzer) CodeBuilder {
	return &ConcreteCodeBuilder{
		pkgAnalyzers: make(map[string]PackageAnalyzer),
	}
}

// Build will generate the code after sanitizing, dynamically imports, and combining driverCode, userCode.
func (c *ConcreteCodeBuilder) Build(lang, driverCode, userCode string) (string, error) {
	// Sanitize code
	err := Sanitize(userCode, lang, DefaultMaxCodeLength)
	if err != nil {
		return "", err
	}

	// only generate imports for compiled languages
	var imports string
	if c.pkgAnalyzers != nil {
		pkgs, err := c.pkgAnalyzers[lang].Analyze(userCode)
		if err != nil {
			return "", err
		}

		imports = c.generateImports(pkgs)
	}

	// combining altogether (imports + driverCode + userCode)
	placeholder := getLanguagePlaceHolder(lang)
	importedCode := strings.Replace(userCode, placeholder.importPlaceHolder, imports, 1)
	finalCode := strings.Replace(driverCode, placeholder.codePlaceHolder, importedCode, 1)

	return finalCode, nil
}

func (c *ConcreteCodeBuilder) generateImports(pkgs []string) string {
	imports := `
	import (
	)
	`
	for _, pkg := range pkgs {
		imports += fmt.Sprintf("\t%s\n", pkg)
	}

	return imports
}
