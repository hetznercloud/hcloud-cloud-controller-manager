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
	"strconv"
	"strings"
	"text/template"
)

const annotationsFilePath = "../internal/annotation/load_balancer.go"
const outputPath = "../docs/reference/load_balancer_annotations.md"

//go:embed load_balancer_annotations.md.tmpl
var TemplateStr string

type Template struct {
	AnnotationsTable string
}

type Table struct {
	table map[string]*TableEntry
}

type TableEntry struct {
	commentLines []string
	constName    string

	Description string
	Default     *string
	Type        *string
	ReadOnly    *bool
}

func NewTable() *Table {
	return &Table{
		table: make(map[string]*TableEntry),
	}
}

func (t *Table) AddEntry(annotation, constName string) {
	t.table[annotation] = &TableEntry{
		commentLines: make([]string, 0),
		constName:    constName,
	}
}

func (t *Table) AppendComment(annotation, value string) {
	if value == "" {
		return
	}

	// trim comment artifacts
	comment := strings.Trim(value, "/ \n")
	if comment == "" {
		return
	}

	t.table[annotation].commentLines = append(t.table[annotation].commentLines, comment)
}

func getValueFromLine(line, key string) string {
	parts := strings.Split(line, key)
	if len(parts) > 1 {
		return strings.Trim(parts[1], " \n.")
	}
	return ""
}

func (t *Table) FromAST(node ast.Node) (*Table, error) {
	ast.Inspect(node, t.visitFunc())

	for annotation, entry := range t.table {
		commentBuilder := strings.Builder{}

		for i, line := range entry.commentLines {
			switch {
			case strings.Contains(line, "Default: "):
				defaultVal := getValueFromLine(line, "Default: ")
				entry.Default = &defaultVal
			case strings.Contains(line, "Type: "):
				typeVal := getValueFromLine(line, "Type: ")
				entry.Type = &typeVal
			case strings.Contains(line, "Read-only: "):
				readOnlyVal := getValueFromLine(line, "Read-only: ")
				if readOnly, err := strconv.ParseBool(readOnlyVal); err == nil {
					entry.ReadOnly = &readOnly
				}
			default:
				if i == 0 {
					line = strings.ReplaceAll(line, fmt.Sprintf("%s ", entry.constName), "")
					line = strings.ToUpper(line[:1]) + line[1:]
				}

				commentBuilder.WriteString(" ")
				commentBuilder.WriteString(line)
			}
		}

		if entry.Type == nil || *entry.Type == "" {
			return nil, fmt.Errorf("missing Type for annotation %s", annotation)
		}

		comment := strings.Trim(commentBuilder.String(), " ")
		for annotationInner, entryInner := range t.table {
			if annotationInner == annotation {
				comment = strings.ReplaceAll(comment, fmt.Sprintf("%s ", entryInner.constName), "")
				if len(comment) > 0 {
					comment = strings.ToUpper(comment[:1]) + comment[1:]
				}
				continue
			}
			comment = strings.ReplaceAll(
				comment,
				fmt.Sprintf("%s ", entryInner.constName),
				fmt.Sprintf("`%s` ", annotation),
			)
		}
		entry.Description = comment
	}

	return t, nil
}

func (t *Table) String() string {
	tableStr := strings.Builder{}

	tableStr.WriteString("| Annotation | Type | Default | Read-only | Description |\n")
	tableStr.WriteString("| --- | --- | --- | --- | --- |\n")

	// sort annotations
	annotations := make([]string, 0, len(t.table))
	for key := range t.table {
		annotations = append(annotations, key)
	}
	sort.Strings(annotations)

	for _, annotation := range annotations {
		typeVal := "-"
		defaultVal := "-"
		readOnlyVal := "No"

		if t.table[annotation].Default != nil {
			defaultVal = *t.table[annotation].Default
		}

		if t.table[annotation].Type != nil {
			typeVal = *t.table[annotation].Type
			if strings.Contains(typeVal, "|") {
				typeVal = strings.ReplaceAll(typeVal, "|", "\\|")
			}
		}

		if t.table[annotation].ReadOnly != nil {
			if ro := *t.table[annotation].ReadOnly; ro {
				readOnlyVal = "Yes"
			}
		}

		tableStr.WriteString(
			fmt.Sprintf(
				"| `%s` | `%s` | `%s` | `%s` | %s |\n",
				annotation,
				typeVal,
				defaultVal,
				readOnlyVal,
				t.table[annotation].Description,
			),
		)
	}

	return tableStr.String()
}

func (t *Table) visitFunc() func(n ast.Node) bool {
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

			hasSingleName := len(valueSpec.Names) != 1
			hasSingleValue := len(valueSpec.Values) != 1
			// If hasNoDocs, then there is at least one comment.
			// len(valueSpec.Doc.List) > 0
			hasNoDocs := valueSpec.Doc == nil

			if hasSingleName || hasSingleValue || hasNoDocs {
				continue
			}

			literal, ok := valueSpec.Values[0].(*ast.BasicLit)
			if !ok {
				continue
			}

			constName := valueSpec.Names[0].Name
			annotation := strings.ReplaceAll(literal.Value, "\"", "")

			t.AddEntry(annotation, constName)

			for _, comment := range valueSpec.Doc.List {
				t.AppendComment(annotation, comment.Text)
			}
		}

		return true
	}
}

func run() error {
	// Parse AST
	node, err := parser.ParseFile(&token.FileSet{}, annotationsFilePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	// Create table from AST
	table, err := NewTable().FromAST(node)
	if err != nil {
		return err
	}

	// Create Markdown file from template
	tmpl, err := template.New("annotations").Parse(TemplateStr)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	tmplData := Template{AnnotationsTable: table.String()}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplData); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	result := strings.TrimRight(buf.String(), "\n") + "\n" // end-of-file fix

	// Write file to output path
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(result); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

//go:generate go run $GOFILE
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
