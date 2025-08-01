package main

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"text/template"
)

const annotationsFilePath = "../internal/annotation/load_balancer.go"
const outputPath = "../docs/reference/load_balancer_annotations.md"

//go:embed annotations.md.tmpl
var TemplateStr string

type Template struct {
	GoConstTable string
}

type TableEntry struct {
	Comment string
	Default *string
}

//go:generate go run $GOFILE

func main() {
	node, err := parser.ParseFile(&token.FileSet{}, annotationsFilePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Println("error parsing file:", err)
		os.Exit(1)
	}

	table := make(map[string]TableEntry)
	ast.Inspect(node, walk(table))

	tmpl, err := template.New("annotations").Parse(TemplateStr)
	if err != nil {
		fmt.Println("error parsing template:", err)
		os.Exit(1)
	}

	tableStr := strings.Builder{}
	tableStr.WriteString("| Annotation | Default | Description |\n")
	tableStr.WriteString("| --- | --- | --- |\n")

	annotations := make([]string, 0, len(table))
	for key := range table {
		annotations = append(annotations, key)
	}

	sort.Strings(annotations)

	for _, annotation := range annotations {
		if table[annotation].Default != nil {
			tableStr.WriteString(fmt.Sprintf("| %s | `%s` | %s |\n", annotation, *table[annotation].Default, table[annotation].Comment))
		} else {
			tableStr.WriteString(fmt.Sprintf("| %s | `-` | %s |\n", annotation, table[annotation].Comment))
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("error creating file:", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := tmpl.Execute(file, Template{GoConstTable: tableStr.String()}); err != nil {
		fmt.Println("error executing template:", err)
	}
}

func walk(table map[string]TableEntry) func(n ast.Node) bool {
	return func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		if genDecl.Tok != token.CONST {
			return true
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			if len(valueSpec.Values) != 1 {
				continue
			}

			if len(valueSpec.Names) != 1 {
				continue
			}

			literal, ok := valueSpec.Values[0].(*ast.BasicLit)
			if !ok {
				continue
			}

			constName := valueSpec.Names[0].Name
			// Put in backquotes for Markdown formatting
			annotation := strings.ReplaceAll(literal.Value, "\"", "`")

			if valueSpec.Doc != nil {
				for _, comment := range valueSpec.Doc.List {
					if !strings.Contains(comment.Text, "Default: ") {
						commentText := strings.ReplaceAll(comment.Text, constName, annotation)
						addTableComment(annotation, commentText, table)
						continue
					}

					parts := strings.Split(comment.Text, "Default: ")
					if len(parts) < 2 {
						continue
					}
					def := strings.Trim(parts[1], " \n.")
					setTableDefault(annotation, def, table)
				}
			}
		}

		return true
	}
}

func setTableDefault(key, value string, table map[string]TableEntry) {
	if value == "" {
		return
	}

	if current, ok := table[key]; !ok {
		table[key] = TableEntry{Default: &value}
	} else {
		table[key] = TableEntry{
			Comment: current.Comment,
			Default: &value,
		}
	}
}

func addTableComment(key, value string, table map[string]TableEntry) {
	if value == "" {
		return
	}

	// trim comment artifacts
	comment := strings.Trim(value, "/ \n")
	if comment == "" {
		return
	}

	if current, ok := table[key]; !ok {
		table[key] = TableEntry{Comment: comment}
	} else {
		table[key] = TableEntry{
			Comment: fmt.Sprintf("%s %s", current.Comment, comment),
			Default: current.Default,
		}
	}
}
