package main

import (
	"bytes"
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
	AnnotationsTable string
}

type Table struct {
	table map[string]*TableEntry
}

type TableEntry struct {
	Comment string
	Default *string
}

func NewTable() *Table {
	return &Table{
		table: make(map[string]*TableEntry),
	}
}

func (t *Table) setDefault(annotation, value string) {
	if _, ok := t.table[annotation]; !ok {
		t.table[annotation] = &TableEntry{Default: &value}
	} else {
		t.table[annotation].Default = &value
	}
}

func (t *Table) appendComment(annotation, value string) {
	if value == "" {
		return
	}

	// trim comment artifacts
	comment := strings.Trim(value, "/ \n")
	if comment == "" {
		return
	}

	if current, ok := t.table[annotation]; !ok {
		t.table[annotation] = &TableEntry{Comment: comment}
	} else {
		t.table[annotation].Comment = fmt.Sprintf("%s %s", current.Comment, comment)
	}
}

func (t *Table) String() string {
	tableStr := strings.Builder{}

	tableStr.WriteString("| Annotation | Default | Description |\n")
	tableStr.WriteString("| --- | --- | --- |\n")

	// sorted keys (annotations)
	annotations := make([]string, 0, len(t.table))
	for key := range t.table {
		annotations = append(annotations, key)
	}
	sort.Strings(annotations)

	for _, annotation := range annotations {
		var defaultVal string
		if t.table[annotation].Default != nil {
			defaultVal = *t.table[annotation].Default
		} else {
			defaultVal = "-"
		}

		tableStr.WriteString(
			fmt.Sprintf(
				"| %s | `%s` | %s |\n",
				annotation,
				defaultVal,
				t.table[annotation].Comment,
			),
		)
	}

	return tableStr.String()
}

func walk(table *Table) func(n ast.Node) bool {
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

			if valueSpec.Doc == nil {
				continue
			}

			for _, comment := range valueSpec.Doc.List {
				switch {
				case strings.Contains(comment.Text, "Default: "):
					if parts := strings.Split(comment.Text, "Default: "); len(parts) == 2 {
						defaultValue := strings.Trim(parts[1], " \n.")
						table.setDefault(annotation, defaultValue)
					}
				default:
					// replace constant name with annotation name
					commentText := strings.ReplaceAll(comment.Text, constName, annotation)
					table.appendComment(annotation, commentText)
				}
			}
		}

		return true
	}
}

//go:generate go run $GOFILE
func main() {
	node, err := parser.ParseFile(&token.FileSet{}, annotationsFilePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Println("error parsing file:", err)
		os.Exit(1)
	}

	table := NewTable()
	ast.Inspect(node, walk(table))

	tmpl, err := template.New("annotations").Parse(TemplateStr)
	if err != nil {
		fmt.Println("error parsing template:", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, Template{AnnotationsTable: table.String()}); err != nil {
		fmt.Println("error executing template:", err)
		os.Exit(1)
	}

	// end-of-file fix
	result := strings.TrimRight(buf.String(), "\n") + "\n"

	file, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("error creating file:", err)
		os.Exit(1)
	}
	defer file.Close()

	if _, err := file.WriteString(result); err != nil {
		fmt.Println("error writing to file:", err)
	}
}
