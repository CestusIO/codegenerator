package templating

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

// A TemplateName represents the name of a template.
type TemplateName interface {
	// Render renders the name of the file.
	Render(ctx interface{}) (string, error)
}

type rawTemplateName struct {
	RelPath string
}

func (n rawTemplateName) Render(ctx interface{}) (string, error) { return n.RelPath, nil }

type templatedTemplateName struct {
	RelPath     string
	Source      string
	PathReplace []pathReplace
}

func (n templatedTemplateName) Render(ctx interface{}) (string, error) {
	relDir, filename := filepath.Split(n.RelPath)
	if len(n.Source) > 0 {
		tmpl, err := template.New("").Funcs(templatesFuncMap).Funcs(sprig.TxtFuncMap()).Parse(n.Source)
		if err != nil {
			return "", err
		}

		name := &bytes.Buffer{}
		err = tmpl.Execute(name, ctx)
		if err != nil {
			return "", err
		}
		filename = name.String()
	}
	// template path
	for _, r := range n.PathReplace {
		relDir = strings.ReplaceAll(relDir, r.old, r.new)
	}
	tmpl, err := template.New("").Funcs(templatesFuncMap).Funcs(sprig.TxtFuncMap()).Parse(relDir)

	if err != nil {
		return "", err
	}
	newPath := &bytes.Buffer{}
	err = tmpl.Execute(newPath, ctx)
	if err != nil {
		return "", err
	}
	relDir = newPath.String()
	//

	return filepath.Join(relDir, filename), nil
}

// A TemplateContent represents a file content.
type TemplateContent interface {
	// Render the template to the specified writer.
	Render(w io.Writer, ctx interface{}) error
}

type rawTemplateContent struct {
	Source io.Reader
}

func (c rawTemplateContent) Render(w io.Writer, ctx interface{}) error {
	_, err := io.Copy(w, c.Source)
	return err
}

type templatedTemplateContent struct {
	TemplateContent
	LeftDelimiter  string
	RightDelimiter string
}

func (c templatedTemplateContent) Render(w io.Writer, ctx interface{}) error {
	source := &bytes.Buffer{}

	if err := c.TemplateContent.Render(source, ctx); err != nil {
		return err
	}

	tmpl, err := template.New("").Funcs(templatesFuncMap).Funcs(sprig.TxtFuncMap()).Delims(c.LeftDelimiter, c.RightDelimiter).Parse(source.String())

	if err != nil {
		return err
	}

	return tmpl.Execute(w, ctx)
}

// A Template represents a file or directory to render.
type Template interface {
	GetPath() string
	GetName() TemplateName
	GetContent() TemplateContent
	GetHeader() Header
	RenderGeneratorCommands(ctx interface{}) ([][]string, error)
}

type templateImpl struct {
	Path    string
	Name    TemplateName
	Content TemplateContent
	Header  Header
}

func (t templateImpl) GetPath() string             { return t.Path }
func (t templateImpl) GetName() TemplateName       { return t.Name }
func (t templateImpl) GetContent() TemplateContent { return t.Content }
func (t templateImpl) GetHeader() Header           { return t.Header }
func (t templateImpl) RenderGeneratorCommands(ctx interface{}) (commands [][]string, err error) {
	for _, cmd := range t.Header.GeneratorCommands {
		tmpl, err := template.New("").Funcs(templatesFuncMap).Funcs(sprig.TxtFuncMap()).Parse(cmd)

		if err != nil {
			return nil, fmt.Errorf("failed to initialize rendering of generator command (%s): %s", cmd, err)
		}

		cmdline := &bytes.Buffer{}
		err = tmpl.Execute(cmdline, ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to render generator command (%s): %s", cmd, err)
		}
	}

	return commands, nil
}

// LoadTemplate loads a template from a reader
func LoadTemplate(path string, r io.Reader) (template Template, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("loading template from %s: %s", path, err)
		}
	}()

	var data []byte

	if data, err = io.ReadAll(r); err != nil {
		return
	}

	r = bytes.NewBuffer(data)

	var templateName TemplateName
	var templateContent TemplateContent
	var header Header

	switch filepath.Ext(path) {
	case ".template":
		path := path[:len(path)-9]

		var reader io.Reader
		reader, err = ParseHeaders(r, &header)

		if err != nil {
			return
		}

		if header.Filename != "" || len(header.PathReplace) > 0 {
			templateName = templatedTemplateName{
				RelPath:     path,
				Source:      header.Filename,
				PathReplace: header.PathReplace,
			}
		} else {
			templateName = rawTemplateName{
				RelPath: path,
			}
		}
		templateContent = templatedTemplateContent{
			TemplateContent: rawTemplateContent{
				Source: reader,
			},
			LeftDelimiter:  header.Delimiters[0],
			RightDelimiter: header.Delimiters[1],
		}
	default:
		templateName = rawTemplateName{
			RelPath: path,
		}
		templateContent = rawTemplateContent{
			Source: r,
		}
	}

	template = templateImpl{
		Path:    path,
		Name:    templateName,
		Content: templateContent,
		Header:  header,
	}

	return
}
