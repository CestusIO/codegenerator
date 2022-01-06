package templating

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"regexp"
	"strings"
)

//pathReplace holds replacement information for path's
type pathReplace struct {
	old string
	new string
}

// A Header contains meta-information about a template.
type Header struct {
	Filename          string
	PathReplace       []pathReplace
	Delimiters        [2]string
	IfNotExists       bool
	If                func(ctx interface{}) (bool, error)
	GeneratorCommands []string
	RemoveIfEmpty     bool
	NoGoGenerate      bool
}

var headerRegexp = regexp.MustCompile(`^!!([a-z-_]+)(?:(?:[ \t]+)(.*))?$`)

// ParseHeaders parses the headers of a template file.
func ParseHeaders(r io.Reader, header *Header) (body io.Reader, err error) {
	reader := bufio.NewReader(r)
	var line string
	lineNumber := 0

	for {
		if line, err = reader.ReadString('\n'); err != nil {
			// If we fail to read a complete line, let's return it without error.
			return bytes.NewBufferString(line), nil
		}

		lineNumber++

		if match := headerRegexp.FindStringSubmatch(strings.TrimSpace(line)); match == nil {
			break
		} else {
			keyword := match[1]
			value := match[2]
			switch keyword {
			case "filename":
				header.Filename = value
			case "pathreplace":
				parts := strings.SplitN(value, " ", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("failed to parse `pathreplace` header: it requires 2 valuers")
				}
				header.PathReplace = append(header.PathReplace, pathReplace{
					old: parts[0],
					new: parts[1],
				})
			case "delimiters":
				parts := strings.SplitN(value, " ", 2)
				copy(header.Delimiters[:], parts)
			case "if":
				condition := fmt.Sprintf("{{ if %s }}X{{ end }}", value)

				tmpl, err := template.New("").Parse(condition)

				if err != nil {
					return nil, fmt.Errorf("failed to parse conditional `if` header: %s", err)
				}

				header.If = func(ctx interface{}) (bool, error) {
					buf := &bytes.Buffer{}
					err := tmpl.Execute(buf, ctx)

					return buf.Len() > 0, err
				}
			case "if-not-exists":
				header.IfNotExists = true
			case "generator-command":
				header.GeneratorCommands = append(header.GeneratorCommands, value)
			case "remove-if-empty":
				header.RemoveIfEmpty = true
			case "no-go-generate":
				header.NoGoGenerate = true
			default:
				return nil, fmt.Errorf("unknown template meta-header `%s` on line %d", keyword, lineNumber)
			}
		}
	}

	return io.MultiReader(bytes.NewBufferString(line), reader), nil
}
