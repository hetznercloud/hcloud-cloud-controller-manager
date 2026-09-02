package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"text/template"
)

type TemplateData struct {
	ConstTable string
}

// docTypes maps the type an annotation is declared with to the type shown in
// the reference documentation.
var docTypes = map[string]string{
	"String":        "string",
	"Bool":          "bool",
	"Int":           "int",
	"Duration":      "duration",
	"Strings":       "string",
	"IP":            "string",
	"Protocol":      "tcp | http | https",
	"AlgorithmType": "round_robin | least_connections",
	"Certificates":  "string",
}

type ConstantDocTable struct {
	entries map[string]*DocEntry
}

type DocEntry struct {
	rawCommentLines []string
	constName       string
	declaredType    string
	pos             int

	Description string
	Type        string
	Default     string
	ReadOnly    bool
	Internal    bool
}

func NewDocTable() *ConstantDocTable {
	return &ConstantDocTable{
		entries: make(map[string]*DocEntry),
	}
}

func (t *ConstantDocTable) AddEntry(constValue, constName, declaredType string, pos int) {
	t.entries[constValue] = &DocEntry{
		rawCommentLines: make([]string, 0),
		constName:       constName,
		declaredType:    declaredType,
		pos:             pos,
	}
}

func (t *ConstantDocTable) AppendComment(constValue, value string) {
	// trim comment artifacts
	comment := strings.Trim(value, "/ \n")
	if comment == "" {
		return
	}

	t.entries[constValue].rawCommentLines = append(t.entries[constValue].rawCommentLines, comment)
}

func (t *ConstantDocTable) FromAST(node ast.Node) (*ConstantDocTable, error) {
	ast.Inspect(node, t.visitFunc())

	for constValue, entry := range t.entries {
		commentBuilder := strings.Builder{}

		if entry.declaredType != "" {
			docType, ok := docTypes[entry.declaredType]
			if !ok {
				return nil, fmt.Errorf("unknown type %s for %s", entry.declaredType, constValue)
			}
			entry.Type = docType
		}

		for i, line := range entry.rawCommentLines {
			if val := parseMetadataValue(line, "Type: "); val != "" {
				entry.Type = val
				continue
			}

			if val := parseMetadataValue(line, "Default: "); val != "" {
				entry.Default = val
				continue
			}

			if val := parseMetadataValue(line, "Read-only: "); val != "" {
				if readOnly, err := strconv.ParseBool(val); err == nil {
					entry.ReadOnly = readOnly
				}
				continue
			}

			if val := parseMetadataValue(line, "Internal: "); val != "" {
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
			return nil, fmt.Errorf("missing Type for %s", constValue)
		}

		comment := strings.Trim(commentBuilder.String(), " ")
		for constValueInner, entryInner := range t.entries {
			comment = strings.ReplaceAll(
				comment,
				fmt.Sprintf("[%s]", entryInner.constName),
				fmt.Sprintf("`%s`", constValueInner),
			)
		}
		entry.Description = comment
	}

	return t, nil
}

func (t *ConstantDocTable) hasReadOnly() bool {
	for _, entry := range t.entries {
		if entry.Internal {
			continue
		}

		if entry.ReadOnly {
			return true
		}
	}

	return false
}

func (t *ConstantDocTable) String() string {
	tableStr := strings.Builder{}

	hasReadOnlyColumn := t.hasReadOnly()

	if hasReadOnlyColumn {
		tableStr.WriteString("| Name | Type | Default | Read-only | Description |\n")
		tableStr.WriteString("| --- | --- | --- | --- | --- |\n")
	} else {
		tableStr.WriteString("| Name | Type | Default | Description |\n")
		tableStr.WriteString("| --- | --- | --- | --- |\n")
	}

	constValues := make([]string, len(t.entries))
	for constValue, entry := range t.entries {
		constValues[entry.pos] = constValue
	}

	for _, constValue := range constValues {
		// Skip internal const values
		if t.entries[constValue].Internal {
			continue
		}

		// Escape pipe characters in Type to avoid breaking the table
		typeVal := strings.ReplaceAll(t.entries[constValue].Type, "|", "\\|")

		defaultVal := "-"
		if t.entries[constValue].Default != "" {
			defaultVal = t.entries[constValue].Default
		}

		fmt.Fprintf(&tableStr, "| `%s` ", constValue)
		fmt.Fprintf(&tableStr, "| `%s` ", typeVal)
		fmt.Fprintf(&tableStr, "| `%s` ", defaultVal)

		if hasReadOnlyColumn {
			readOnlyVal := "No"
			if t.entries[constValue].ReadOnly {
				readOnlyVal = "Yes"
			}
			fmt.Fprintf(&tableStr, "| `%s` ", readOnlyVal)
		}

		fmt.Fprintf(&tableStr, "| %s |\n", t.entries[constValue].Description)
	}

	return tableStr.String()
}

func (t *ConstantDocTable) visitFunc() func(n ast.Node) bool {
	return func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		if genDecl.Tok != token.CONST {
			return true
		}

		for pos, spec := range genDecl.Specs {
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
			value := strings.ReplaceAll(literal.Value, "\"", "")

			var declaredType string
			if ident, ok := valueSpec.Type.(*ast.Ident); ok {
				declaredType = ident.Name
			}

			t.AddEntry(value, constName, declaredType, pos)

			for _, comment := range valueSpec.Doc.List {
				t.AppendComment(value, comment.Text)
			}
		}

		return true
	}
}

func parseMetadataValue(line, key string) string {
	parts := strings.Split(line, key)
	if len(parts) > 1 {
		return strings.Trim(parts[1], " \n.")
	}
	return ""
}

func generateDocs(templatePath string, constFilePath string, outputPath string) error {
	// Read template file
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}

	// Parse AST
	astNode, err := parser.ParseFile(&token.FileSet{}, constFilePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	// Create table from AST
	docTable, err := NewDocTable().FromAST(astNode)
	if err != nil {
		return err
	}

	// Create Markdown file from template
	tmpl, err := template.New("consts").Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	tmplData := TemplateData{ConstTable: docTable.String()}
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
	docs := []struct {
		TemplatePath string
		SourcePath   string
		TargetPath   string
	}{
		{
			// Generate Load Balancer annotations documentation
			TemplatePath: "./load_balancer_annotations.md.tmpl",
			SourcePath:   "../internal/annotation/load_balancer.go",
			TargetPath:   "../docs/reference/load_balancer_annotations.md",
		},
		{
			// Generate Load Balancer env documentation
			TemplatePath: "./load_balancer_envs.md.tmpl",
			SourcePath:   "../internal/config/load_balancer_envs.go",
			TargetPath:   "../docs/reference/load_balancer_envs.md",
		},
	}

	for _, doc := range docs {
		if err := generateDocs(doc.TemplatePath, doc.SourcePath, doc.TargetPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}
