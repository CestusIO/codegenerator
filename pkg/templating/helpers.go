package templating

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"sync"
	"text/template"
	"unicode"

	"strings"

	"gopkg.in/yaml.v3"
)

var CodeGeneratorName string

func init() {
	CodeGeneratorName = "cestus.io codegenerator"
}

// indexOfInitialisms is a thread-safe implementation of the sorted index of initialisms.
// Since go1.9, this may be implemented with sync.Map.
type indexOfInitialisms struct {
	sortMutex *sync.Mutex
	index     *sync.Map
}

func newIndexOfInitialisms() *indexOfInitialisms {
	return &indexOfInitialisms{
		sortMutex: new(sync.Mutex),
		index:     new(sync.Map),
	}
}

func (m *indexOfInitialisms) load(initial map[string]bool) *indexOfInitialisms {
	m.sortMutex.Lock()
	defer m.sortMutex.Unlock()
	for k, v := range initial {
		m.index.Store(k, v)
	}
	return m
}

func (m *indexOfInitialisms) isInitialism(key string) bool {
	_, ok := m.index.Load(key)
	return ok
}

func (m *indexOfInitialisms) sorted() (result []string) {
	m.sortMutex.Lock()
	defer m.sortMutex.Unlock()
	m.index.Range(func(key, value interface{}) bool {
		k := key.(string)
		result = append(result, k)
		return true
	})
	sort.Sort(sort.Reverse(byInitialism(result)))
	return
}

type byInitialism []string

func (s byInitialism) Len() int {
	return len(s)
}
func (s byInitialism) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byInitialism) Less(i, j int) bool {
	if len(s[i]) != len(s[j]) {
		return len(s[i]) < len(s[j])
	}

	return strings.Compare(s[i], s[j]) > 0
}

// commonInitialisms are common acronyms that are kept as whole uppercased words.
var commonInitialisms *indexOfInitialisms

// initialisms is a slice of sorted initialisms
var initialisms []string

var isInitialism func(string) bool

// GoNamePrefixFunc sets an optional rule to prefix go names
// which do not start with a letter.
//
// e.g. to help convert "123" into "{prefix}123"
//
// The default is to prefix with "X"
var GoNamePrefixFunc func(string) string

func init() {
	// Taken from https://github.com/golang/lint/blob/3390df4df2787994aea98de825b964ac7944b817/lint.go#L732-L769
	var configuredInitialisms = map[string]bool{
		"ACL":   true,
		"API":   true,
		"ASCII": true,
		"CPU":   true,
		"CSS":   true,
		"DNS":   true,
		"EOF":   true,
		"GUID":  true,
		"HTML":  true,
		"HTTPS": true,
		"HTTP":  true,
		"ID":    true,
		"IP":    true,
		"IPv4":  true,
		"IPv6":  true,
		"JSON":  true,
		"LHS":   true,
		"OAI":   true,
		"QPS":   true,
		"RAM":   true,
		"RHS":   true,
		"RPC":   true,
		"SLA":   true,
		"SMTP":  true,
		"SQL":   true,
		"SSH":   true,
		"TCP":   true,
		"TLS":   true,
		"TTL":   true,
		"UDP":   true,
		"UI":    true,
		"UID":   true,
		"UUID":  true,
		"URI":   true,
		"URL":   true,
		"UTF8":  true,
		"VM":    true,
		"XML":   true,
		"XMPP":  true,
		"XSRF":  true,
		"XSS":   true,
	}

	// a thread-safe index of initialisms
	commonInitialisms = newIndexOfInitialisms().load(configuredInitialisms)
	initialisms = commonInitialisms.sorted()

	// a test function
	isInitialism = commonInitialisms.isInitialism
}

var nameReplaceTable = map[rune]string{
	'@': "At ",
	'&': "And ",
	'|': "Pipe ",
	'$': "Dollar ",
	'!': "Bang ",
	'-': "",
	'_': "",
}

type (
	splitter struct {
		postSplitInitialismCheck bool
		initialisms              []string
	}

	splitterOption func(*splitter) *splitter
)

// split calls the splitter; splitter provides more control and post options
func split(str string) []string {
	lexems := newSplitter().split(str)
	result := make([]string, 0, len(lexems))

	for _, lexem := range lexems {
		result = append(result, lexem.GetOriginal())
	}

	return result

}

func (s *splitter) split(str string) []nameLexem {
	return s.toNameLexems(str)
}

func newSplitter(options ...splitterOption) *splitter {
	splitter := &splitter{
		postSplitInitialismCheck: false,
		initialisms:              initialisms,
	}

	for _, option := range options {
		splitter = option(splitter)
	}

	return splitter
}

// withPostSplitInitialismCheck allows to catch initialisms after main split process
func withPostSplitInitialismCheck(s *splitter) *splitter {
	s.postSplitInitialismCheck = true
	return s
}

type (
	initialismMatch struct {
		start, end int
		body       []rune
		complete   bool
	}
	initialismMatches []*initialismMatch
)

func (s *splitter) toNameLexems(name string) []nameLexem {
	nameRunes := []rune(name)
	matches := s.gatherInitialismMatches(nameRunes)
	return s.mapMatchesToNameLexems(nameRunes, matches)
}

func (s *splitter) gatherInitialismMatches(nameRunes []rune) initialismMatches {
	matches := make(initialismMatches, 0)

	for currentRunePosition, currentRune := range nameRunes {
		newMatches := make(initialismMatches, 0, len(matches))

		// check current initialism matches
		for _, match := range matches {
			if keepCompleteMatch := match.complete; keepCompleteMatch {
				newMatches = append(newMatches, match)
				continue
			}

			// drop failed match
			currentMatchRune := match.body[currentRunePosition-match.start]
			if !s.initialismRuneEqual(currentMatchRune, currentRune) {
				continue
			}

			// try to complete ongoing match
			if currentRunePosition-match.start == len(match.body)-1 {
				// we are close; the next step is to check the symbol ahead
				// if it is a small letter, then it is not the end of match
				// but beginning of the next word

				if currentRunePosition < len(nameRunes)-1 {
					nextRune := nameRunes[currentRunePosition+1]
					if newWord := unicode.IsLower(nextRune); newWord {
						// oh ok, it was the start of a new word
						continue
					}
				}

				match.complete = true
				match.end = currentRunePosition
			}

			newMatches = append(newMatches, match)
		}

		// check for new initialism matches
		for _, initialism := range s.initialisms {
			initialismRunes := []rune(initialism)
			if s.initialismRuneEqual(initialismRunes[0], currentRune) {
				newMatches = append(newMatches, &initialismMatch{
					start:    currentRunePosition,
					body:     initialismRunes,
					complete: false,
				})
			}
		}

		matches = newMatches
	}

	return matches
}

func (s *splitter) mapMatchesToNameLexems(nameRunes []rune, matches initialismMatches) []nameLexem {
	nameLexems := make([]nameLexem, 0)

	var lastAcceptedMatch *initialismMatch
	for _, match := range matches {
		if !match.complete {
			continue
		}

		if firstMatch := lastAcceptedMatch == nil; firstMatch {
			nameLexems = append(nameLexems, s.breakCasualString(nameRunes[:match.start])...)
			nameLexems = append(nameLexems, s.breakInitialism(string(match.body)))

			lastAcceptedMatch = match

			continue
		}

		if overlappedMatch := match.start <= lastAcceptedMatch.end; overlappedMatch {
			continue
		}

		middle := nameRunes[lastAcceptedMatch.end+1 : match.start]
		nameLexems = append(nameLexems, s.breakCasualString(middle)...)
		nameLexems = append(nameLexems, s.breakInitialism(string(match.body)))

		lastAcceptedMatch = match
	}

	// we have not found any accepted matches
	if lastAcceptedMatch == nil {
		return s.breakCasualString(nameRunes)
	}

	if lastAcceptedMatch.end+1 != len(nameRunes) {
		rest := nameRunes[lastAcceptedMatch.end+1:]
		nameLexems = append(nameLexems, s.breakCasualString(rest)...)
	}

	return nameLexems
}

func (s *splitter) initialismRuneEqual(a, b rune) bool {
	return a == b
}

func (s *splitter) breakInitialism(original string) nameLexem {
	return newInitialismNameLexem(original, original)
}

func (s *splitter) breakCasualString(str []rune) []nameLexem {
	segments := make([]nameLexem, 0)
	currentSegment := ""

	addCasualNameLexem := func(original string) {
		segments = append(segments, newCasualNameLexem(original))
	}

	addInitialismNameLexem := func(original, match string) {
		segments = append(segments, newInitialismNameLexem(original, match))
	}

	addNameLexem := func(original string) {
		if s.postSplitInitialismCheck {
			for _, initialism := range s.initialisms {
				if upper(initialism) == upper(original) {
					addInitialismNameLexem(original, initialism)
					return
				}
			}
		}

		addCasualNameLexem(original)
	}

	for _, rn := range string(str) {
		if replace, found := nameReplaceTable[rn]; found {
			if currentSegment != "" {
				addNameLexem(currentSegment)
				currentSegment = ""
			}

			if replace != "" {
				addNameLexem(replace)
			}

			continue
		}

		if !unicode.In(rn, unicode.L, unicode.M, unicode.N, unicode.Pc) {
			if currentSegment != "" {
				addNameLexem(currentSegment)
				currentSegment = ""
			}

			continue
		}

		if unicode.IsUpper(rn) {
			if currentSegment != "" {
				addNameLexem(currentSegment)
			}
			currentSegment = ""
		}

		currentSegment += string(rn)
	}

	if currentSegment != "" {
		addNameLexem(currentSegment)
	}

	return segments
}

type (
	nameLexem interface {
		GetUnsafeGoName() string
		GetOriginal() string
		IsInitialism() bool
	}

	initialismNameLexem struct {
		original          string
		matchedInitialism string
	}

	casualNameLexem struct {
		original string
	}
)

func newInitialismNameLexem(original, matchedInitialism string) *initialismNameLexem {
	return &initialismNameLexem{
		original:          original,
		matchedInitialism: matchedInitialism,
	}
}

func newCasualNameLexem(original string) *casualNameLexem {
	return &casualNameLexem{
		original: original,
	}
}

func (l *initialismNameLexem) GetUnsafeGoName() string {
	return l.matchedInitialism
}

func (l *casualNameLexem) GetUnsafeGoName() string {
	var first rune
	var rest string
	for i, orig := range l.original {
		if i == 0 {
			first = orig
			continue
		}
		if i > 0 {
			rest = l.original[i:]
			break
		}
	}
	if len(l.original) > 1 {
		return string(unicode.ToUpper(first)) + lower(rest)
	}

	return l.original
}

func (l *initialismNameLexem) GetOriginal() string {
	return l.original
}

func (l *casualNameLexem) GetOriginal() string {
	return l.original
}

func (l *initialismNameLexem) IsInitialism() bool {
	return true
}

func (l *casualNameLexem) IsInitialism() bool {
	return false
}

// Removes leading whitespaces
func trim(str string) string {
	return strings.Trim(str, " ")
}

// Shortcut to strings.ToUpper()
func upper(str string) string {
	return strings.ToUpper(trim(str))
}

// Shortcut to strings.ToLower()
func lower(str string) string {
	return strings.ToLower(trim(str))
}

// Camelize an uppercased word
func Camelize(word string) (camelized string) {
	for pos, ru := range []rune(word) {
		if pos > 0 {
			camelized += string(unicode.ToLower(ru))
		} else {
			camelized += string(unicode.ToUpper(ru))
		}
	}
	return
}

// ToGoPackageName returns lowercase string without separator.
var ToGoPackageName = func(name string) string {
	return strings.ToLower(ToGoName(name))
}

// ToGoName sanitizes a name for a public Go variable.
func ToGoName(name string) string {
	lexems := newSplitter(withPostSplitInitialismCheck).split(name)

	result := ""
	for _, lexem := range lexems {
		goName := lexem.GetUnsafeGoName()

		// to support old behavior
		if lexem.IsInitialism() {
			goName = upper(goName)
		}
		result += goName
	}

	if len(result) > 0 {
		// Only prefix with X when the first character isn't an ascii letter
		first := []rune(result)[0]
		if !unicode.IsLetter(first) || (first > unicode.MaxASCII && !unicode.IsUpper(first)) {
			if GoNamePrefixFunc == nil {
				return "X" + result
			}
			result = GoNamePrefixFunc(name) + result
		}
		first = []rune(result)[0]
		if unicode.IsLetter(first) && !unicode.IsUpper(first) {
			result = string(append([]rune{unicode.ToUpper(first)}, []rune(result)[1:]...))
		}
	}

	return result
}

// ToVarName sanitizes a name for a Go variable.
func ToVarName(name string) string {
	res := ToGoName(name)
	if isInitialism(res) {
		return lower(res)
	}
	if len(res) <= 1 {
		return lower(res)
	}
	return lower(res[:1]) + res[1:]
}

// ToFileName sanitizes a name for a filename.
func ToFileName(name string) string {
	in := split(name)
	out := make([]string, 0, len(in))

	for _, w := range in {
		out = append(out, lower(w))
	}

	return strings.Join(out, "_")
}

// ToCommandName lowercases and underscores a go type name
func ToCommandName(name string) string {
	in := split(name)
	out := make([]string, 0, len(in))

	for _, w := range in {
		out = append(out, lower(w))
	}
	return strings.Join(out, "-")
}

// ToHumanNameTitle represents a code name as a human series of words with the first letters titleized
func ToHumanNameTitle(name string) string {
	in := newSplitter(withPostSplitInitialismCheck).split(name)

	out := make([]string, 0, len(in))
	for _, w := range in {
		original := w.GetOriginal()
		if !w.IsInitialism() {
			out = append(out, Camelize(original))
		} else {
			out = append(out, original)
		}
	}
	return strings.Join(out, " ")
}

// ToYAML returns the yaml version of the input.
func ToYAML(s interface{}) string {
	result, err := yaml.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(result)
}

func getHeaderMessage(message string) string {
	return fmt.Sprintf(
		"Code generated by %s\n\n%s",
		CodeGeneratorName,
		message,
	)
}

// CodeSectionFileHeader return a header for file using code sections.
func CodeSectionFileHeader() string {
	return getHeaderMessage("Modifications in code regions will be lost during regeneration!")
}

// FullyEditableFileHeader return a header for fully editable files.
func FullyEditableFileHeader() string { return getHeaderMessage("You CAN edit this file !") }

// NonEditableFileHeader return a header non editable files.
func NonEditableFileHeader() string { return getHeaderMessage("DO NOT EDIT.") }

func commentLines(input, commentChars string) string {
	result := strings.Replace(commentChars+" "+input, "\n", "\n"+commentChars+" ", -1)
	return strings.Replace(result, " \n", "\n", -1)
}

// ToGoComment comments the input using go comments `//`.
func ToGoComment(input string) string { return commentLines(input, "//") }

// ToYAMLComment comments the input using yaml comments `#`.
func ToYAMLComment(input string) string { return commentLines(input, "#") }

// ToMarkdownQuote sets the input using markdown quote.
func ToMarkdownQuote(input string) string { return commentLines(input, ">") }

// ToMap takes a list of strings as key-value pairs and returns a map.
func ToMap(input ...string) map[string]string {
	if len(input)%2 != 0 {
		panic(errors.New("odd number of values in ToMap call, expected a list of key-value pairs"))
	}

	results := make(map[string]string)

	for i := 0; i < len(input); i += 2 {
		key := input[i]
		value := input[i+1]
		results[key] = value
	}

	return results
}

// ReplacePathVar replaces a variable in a path, if it exists.
//
// If it doesn't, the path is returned unchanged.
func ReplacePathVar(varName string, value interface{}, path string) string {
	re := regexp.MustCompile(fmt.Sprintf(`^(.*)\{[ ]*%s[ ]*\}(.*)$`, regexp.QuoteMeta(varName)))
	return re.ReplaceAllString(path, fmt.Sprintf(`${1}%v${2}`, value))
}

// ReplacePathVars replaces all variables in a path.
//
// If no variable is present, the path is returned unchanged.
func ReplacePathVars(value interface{}, path string) string {
	re := regexp.MustCompile(`\{[ ]*[\w\-]+[ ]*\}`)
	return re.ReplaceAllString(path, fmt.Sprintf(`${1}%v${2}`, value))
}

// IsLast returns true if index is last element of slice
func IsLast(index int, slice interface{}) bool {
	return index == reflect.ValueOf(slice).Len()-1
}

// Capitalize returns a copy of the string with its first character capitalized and the rest lowercased
func Capitalize(s string) string {
	return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
}

var templatesFuncMap = template.FuncMap{
	"ToGoName":                ToGoName,
	"ToVarName":               ToVarName,
	"ToFileName":              ToFileName,
	"ToCommandName":           ToCommandName,
	"ToGoPackageName":         ToGoPackageName,
	"ToYAML":                  ToYAML,
	"ToHumanNameTitle":        ToHumanNameTitle,
	"CodeSectionFileHeader":   CodeSectionFileHeader,
	"NonEditableFileHeader":   NonEditableFileHeader,
	"FullyEditableFileHeader": FullyEditableFileHeader,
	"ToYAMLComment":           ToYAMLComment,
	"ToGoComment":             ToGoComment,
	"ToMarkdownQuote":         ToMarkdownQuote,
	"ToMap":                   ToMap,
	"ToSlash":                 filepath.ToSlash,
	"ReplacePathVar":          ReplacePathVar,
	"ReplacePathVars":         ReplacePathVars,
	"Rel":                     filepath.Rel,
	"IsLast":                  IsLast,
	"Capitalize":              Capitalize,
}
