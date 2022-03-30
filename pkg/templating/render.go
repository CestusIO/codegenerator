package templating

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"code.cestus.io/libs/codegenerator/pkg/placeholder"
)

// Render a list of templates to the specified directory.
//
// Returns the list of generated files, as absolute paths as well as
// extra-rendering commands to execute..
//
// If patterns is specified, only the files that match at least one of the specified patterns will be rendered.
func Render(templates []Template, root string, ctx interface{}, patterns ...string) (generated []string, cmds [][]string, err error) {
	for _, tmpl := range templates {
		var relPath string

		if relPath, err = tmpl.GetName().Render(ctx); err != nil {
			err = fmt.Errorf("rendering name for template `%s`: %s", tmpl.GetPath(), err)
			return
		}

		if len(patterns) > 0 {
			matched := false

			for _, pattern := range patterns {
				if ok, err := filepath.Match(pattern, relPath); err != nil {
					return nil, nil, fmt.Errorf("matching pattern `%s`: %s", pattern, err)
				} else if ok {
					matched = true
					break
				}
			}

			if !matched {
				// None of the patterns matched. Skip the file.
				continue
			}
		}

		if tmpl.GetHeader().If != nil {
			if ok, err := tmpl.GetHeader().If(ctx); err != nil {
				return nil, nil, fmt.Errorf("failed to evaluate header `if` condition in `%s`: %s", tmpl.GetPath(), err)
			} else if !ok {
				continue
			}
		}

		if tmpl.GetHeader().If != nil {
			if ok, err := tmpl.GetHeader().IfOr(ctx); err != nil {
				return nil, nil, fmt.Errorf("failed to evaluate header `ifor` condition in `%s`: %s", tmpl.GetPath(), err)
			} else if !ok {
				continue
			}
		}

		path := filepath.Join(root, relPath)
		dirPath := filepath.ToSlash(filepath.Dir(path))

		if err = os.MkdirAll(dirPath, 0755); err != nil {
			return
		}

		var placeholders []placeholder.Placeholder

		var existingData []byte

		if existingData, err = os.ReadFile(path); err != nil && !os.IsNotExist(err) {
			return
		} else if err == nil {
			// If the generated file already exists and IfNotExists is
			// specified, don't overwrite it.
			if tmpl.GetHeader().IfNotExists {
				generated = append(generated, path)
				continue
			}
		}

		output := &bytes.Buffer{}

		if err = tmpl.GetContent().Render(output, ctx); err != nil {
			err = fmt.Errorf("rendering content for template `%s`: %s", tmpl.GetPath(), err)
			return
		}

		// If the file already exists, we replace the placeholders in the
		// initial files with the generated ones and reuse that file instead.
		if existingData != nil {
			placeholders = placeholder.FindAll(output.Bytes())
			output = bytes.NewBuffer(placeholder.ReplaceAll(existingData, placeholders))
		}

		if output.Len() == 0 && tmpl.GetHeader().RemoveIfEmpty {
			os.Remove(path)
			continue
		}
		if err = os.WriteFile(path, output.Bytes(), 0666); err != nil {
			return
		}
		generated = append(generated, path)

		var generatorCmds [][]string

		generatorCmds, err = tmpl.RenderGeneratorCommands(ctx)

		if err != nil {
			err = fmt.Errorf("in template %s: %s", tmpl.GetName(), err)
			return
		}

		cmds = append(cmds, generatorCmds...)
		p, _ := filepath.Rel(root, path)
		if strings.HasSuffix(p, ".go") {
			cmds = append(cmds, []string{"goimports", "-l", "-w", "./" + p})
			if !tmpl.GetHeader().NoGoGenerate {
				cmds = append(cmds, []string{"go", "generate", "./" + p})
			}
		}
	}
	sort.Strings(generated)

	return
}
