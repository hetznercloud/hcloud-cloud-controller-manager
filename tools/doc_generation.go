package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

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
	Type        string
	Default     string
	ReadOnly    bool
	Internal    bool
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
	// trim comment artifacts
	comment := strings.Trim(value, "/ \n")
	if comment == "" {
		return
	}

	t.table[annotation].commentLines = append(t.table[annotation].commentLines, comment)
}

func (t *Table) FromAST(node ast.Node) (*Table, error) {
	ast.Inspect(node, t.visitFunc())

	for annotation, entry := range t.table {
		commentBuilder := strings.Builder{}

		for i, line := range entry.commentLines {
			if val := getValueFromLine(line, "Type: "); val != "" {
				entry.Type = val
				continue
			}

			if val := getValueFromLine(line, "Default: "); val != "" {
				entry.Default = val
				continue
			}

			if val := getValueFromLine(line, "Read-only: "); val != "" {
				if readOnly, err := strconv.ParseBool(val); err == nil {
					entry.ReadOnly = readOnly
				}
				continue
			}

			if val := getValueFromLine(line, "Internal: "); val != "" {
				if internal, err := strconv.ParseBool(val); err == nil {
					entry.Internal = internal
				}
				continue
			}

			if i == 0 {
				// Remove constant name from beginning of the line and uppercase first letter
				line = strings.ReplaceAll(line, fmt.Sprintf("%s ", entry.constName), "")
				line = strings.ToUpper(line[:1]) + line[1:]
			}

			commentBuilder.WriteString(" ")
			commentBuilder.WriteString(line)
		}

		if entry.Type == "" {
			return nil, fmt.Errorf("missing Type for annotation %s", annotation)
		}

		comment := strings.Trim(commentBuilder.String(), " ")
		for _, entryInner := range t.table {
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

	annotations := slices.Sorted(maps.Keys(t.table))

	for _, annotation := range annotations {
		// Skip internal annotations
		if t.table[annotation].Internal {
			continue
		}

		// Escape pipe characters in Type to avoid breaking the table
		typeVal := strings.ReplaceAll(t.table[annotation].Type, "|", "\\|")

		defaultVal := "-"
		if t.table[annotation].Default != "" {
			defaultVal = t.table[annotation].Default
		}

		readOnlyVal := "No"
		if t.table[annotation].ReadOnly {
			readOnlyVal = "Yes"
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

func getValueFromLine(line, key string) string {
	parts := strings.Split(line, key)
	if len(parts) > 1 {
		return strings.Trim(parts[1], " \n.")
	}
	return ""
}

func run(templatePath string, annotationsPath string, outputPath string) error {
	// Read template file
	tmplStr, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}

	// Parse AST
	node, err := parser.ParseFile(&token.FileSet{}, annotationsPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	// Create table from AST
	table, err := NewTable().FromAST(node)
	if err != nil {
		return err
	}

	// Create Markdown file from template
	tmpl, err := template.New("annotations").Parse(string(tmplStr))
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
	// Generate Load Balancer annotations documentation
	lbTemplatePath := "./load_balancer_annotations.md.tmpl"
	lbAnnotationsPath := "../internal/annotation/load_balancer.go"
	lbOutputPath := "../docs/reference/load_balancer_annotations.md"

	if err := run(lbTemplatePath, lbAnnotationsPath, lbOutputPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
