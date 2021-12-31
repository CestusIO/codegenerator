package placeholder

import (
	"bytes"
	"fmt"
	"regexp"

	"golang.org/x/text/runes"
)

// CodeSectionMark represents a pair of code section markers.
type CodeSectionMark string

func isCR(r rune) bool {
	return r == '\r'
}

var (
	crRemover = runes.Remove(runes.Predicate(isCR))
)

func (m CodeSectionMark) findRegexp() *regexp.Regexp {
	q := regexp.QuoteMeta(string(m))
	exp := fmt.Sprintf(`(?ms)^([\t ]*%s[\t ]*region CODE_REGION\([\t ]*([^\n\)]+?)[\t ]*\)[\t ]*\n)(.*?\n?)([\t ]*%s[\t ]*endregion[\t ]*)$`, q, q)
	return regexp.MustCompile(exp)
}

func (m CodeSectionMark) parsePlaceholders(data []byte) (placeholders []Placeholder) {
	// Convert old DOS line ending format (CRLF) to linux format (LF).
	data = crRemover.Bytes(data)

	submatches := m.findRegexp().FindAllSubmatch(data, -1)

	for _, submatch := range submatches {
		placeholders = append(placeholders, Placeholder{
			Raw:        submatch[0],
			Begin:      submatch[1],
			Identifier: string(submatch[2]),
			Content:    submatch[3],
			End:        submatch[4],
			Mark:       m,
		})
	}

	return
}

// DefaultCodeSectionMarks is the list of default code section marks.
var DefaultCodeSectionMarks = []CodeSectionMark{"//", "#", "#pragma "}

// Placeholder represents a placeholder in a file.
type Placeholder struct {
	Raw        []byte
	Begin      []byte
	Identifier string
	Content    []byte
	End        []byte
	Mark       CodeSectionMark
}

// WithContent returns a copy of the placeholder with its content replaced.
func (p Placeholder) WithContent(content []byte) Placeholder {
	newRaw := append([]byte{}, p.Begin...)
	newRaw = append(newRaw, content...)
	newRaw = append(newRaw, p.End...)

	return Placeholder{
		Raw:        newRaw,
		Begin:      p.Begin,
		Identifier: p.Identifier,
		Content:    content,
		End:        p.End,
		Mark:       p.Mark,
	}
}

// FindAll finds all placeholders in data and returns them.
func FindAll(data []byte) []Placeholder {
	placeholders := make([]Placeholder, 0)

	for _, mark := range DefaultCodeSectionMarks {
		placeholders = append(placeholders, mark.parsePlaceholders(data)...)
	}

	return placeholders
}

// ReplaceAll replaces all placeholders in the specified input data and
// produces the specified output data.
func ReplaceAll(data []byte, placeholders []Placeholder) []byte {
	// Convert old DOS line ending format (CRLF) to linux format (LF).
	data = crRemover.Bytes(data)
	targetPlaceholders := FindAll(data)

	for _, placeholder := range placeholders {
		for _, targetPlaceholder := range targetPlaceholders {
			if targetPlaceholder.Identifier == placeholder.Identifier {
				newPlaceholder := targetPlaceholder.WithContent(placeholder.Content)
				data = bytes.Replace(data, targetPlaceholder.Raw, newPlaceholder.Raw, 1)
				break
			}
		}
	}

	return data
}

// FindAndReplaceAll finds all placeholders from the specified `src` and
// replace it in the specified `dest`.
func FindAndReplaceAll(src []byte, dest []byte) []byte {
	placeholders := FindAll(src)
	return ReplaceAll(dest, placeholders)
}
