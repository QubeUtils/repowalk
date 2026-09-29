package aggregator

import (
	"bytes"
	"encoding/json"
	"strings"
	"text/template"

	"github.com/QubeUtils/repowalk/internal/walker"
	"gopkg.in/yaml.v3"
)

// DumpOptions holds configuration for generating the LLM dump.
type DumpOptions struct {
	TemplateString string
	TreeString     string
	Persona        string
}

var personas = map[string]string{
	"security": "You are an expert security researcher. Conduct a thorough security audit of the following codebase, looking for vulnerabilities, logical flaws, and insecure practices. Focus on OWASP top 10 and language-specific pitfalls.",
	"refactor": "You are a senior software architect. Analyze the following codebase and suggest architectural improvements, design pattern applications, and code refactorings to enhance maintainability, performance, and readability.",
	"review":   "You are a strict and meticulous code reviewer. Perform a comprehensive code review of the provided codebase. Point out bugs, edge cases, style violations, and provide concrete examples of how to improve the code.",
}

const defaultTemplate = `The following is a representation of the repository context.
{{ if .Tree }}
# Repository Structure
` + "```\n" + `{{ .Tree }}
` + "```\n" + `{{ end }}
# Files
{{ range .Files }}
## {{ .Path }}
` + "```\n" + `{{ .ContentString }}
` + "```\n" + `{{ end }}`

type fileTemplateData struct {
	Path          string
	ContentString string
}

type DataExport struct {
	Tree  string             `json:"tree,omitempty" yaml:"tree,omitempty"`
	Files []fileTemplateData `json:"files" yaml:"files"`
}

func buildDataExport(nodes []walker.FileNode, opts DumpOptions) DataExport {
	var files []fileTemplateData
	for _, n := range nodes {
		if n.IsBinary || len(n.Content) == 0 {
			continue
		}
		files = append(files, fileTemplateData{
			Path:          n.Path,
			ContentString: string(n.Content),
		})
	}
	return DataExport{
		Tree:  opts.TreeString,
		Files: files,
	}
}

// GenerateMarkdown creates the final Markdown text suitable for LLMs.
func GenerateMarkdown(nodes []walker.FileNode, opts DumpOptions) (string, error) {
	tmplStr := opts.TemplateString
	if tmplStr == "" {
		tmplStr = defaultTemplate
	}

	tmpl, err := template.New("dump").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	data := buildDataExport(nodes, opts)

	var buf bytes.Buffer

	if opts.Persona != "" {
		if prompt, ok := personas[strings.ToLower(opts.Persona)]; ok {
			buf.WriteString(prompt + "\n\n")
		} else {
			// Fallback if persona not found, maybe just use it as a custom prompt
			buf.WriteString(opts.Persona + "\n\n")
		}
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

// GenerateJSON creates a JSON representation of the context.
func GenerateJSON(nodes []walker.FileNode, opts DumpOptions) (string, error) {
	data := buildDataExport(nodes, opts)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GenerateYAML creates a YAML representation of the context.
func GenerateYAML(nodes []walker.FileNode, opts DumpOptions) (string, error) {
	data := buildDataExport(nodes, opts)
	b, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
