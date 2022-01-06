package placeholder

import (
	"os"
	"strings"
	"testing"

	"fmt"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/require"
)

func TestFindAndReplaceAll(t *testing.T) {
	data, _ := os.ReadFile("fixtures/before.txt")
	templateData, _ := os.ReadFile("fixtures/template.txt")

	actual := FindAndReplaceAll(data, templateData)
	expected, _ := os.ReadFile("fixtures/after.txt")

	diff := difflib.ContextDiff{
		A:        difflib.SplitLines(string(expected)),
		B:        difflib.SplitLines(string(actual)),
		FromFile: "fixtures/after.txt",
		ToFile:   "fixtures/actual.txt",
		Context:  3,
		Eol:      "\n",
	}
	result, _ := difflib.GetContextDiffString(diff)

	require.Empty(t, result, "produced files are different")
}

func TestFindAndReplaceAllCrLf(t *testing.T) {
	data, _ := os.ReadFile("fixtures/before.txt")
	templateData, _ := os.ReadFile("fixtures/template.txt")

	// Convert input file into old DOS line ending format. Emulate the autocrlf git option.
	data = []byte(strings.Replace(string(data), "\n", "\r\n", -1))

	actual := FindAndReplaceAll(data, templateData)
	expected, _ := os.ReadFile("fixtures/after.txt")

	diff := difflib.ContextDiff{
		A:        difflib.SplitLines(string(expected)),
		B:        difflib.SplitLines(string(actual)),
		FromFile: "fixtures/after.txt",
		ToFile:   "fixtures/actual.txt",
		Context:  3,
		Eol:      "\n",
	}
	result, _ := difflib.GetContextDiffString(diff)

	require.Empty(t, result, "produced files are different")
}

func TestIdentifierParsing(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Foo", "Foo"},
		{"Foo_Bar", "Foo_Bar"},
		{"0", "0"},
		{"____", "____"},
		{"00_aa_bb_cc", "00_aa_bb_cc"},
		{"\t    Foo Bar\t\t\t\t   \t\t\t", "Foo Bar"},
		{"Foo Bar", "Foo Bar"},
		{"Foo-Bar", "Foo-Bar"},
		{"Foo/Bar", "Foo/Bar"},
		{"  \tFoo.Bar   \t", "Foo.Bar"},
	}

	for _, tc := range testCases {
		t.Run("TestIdentifierParsing", func(t *testing.T) {
			data := []byte(fmt.Sprintf("// region CODE_REGION(%s)\nSome data here\n// endregion", tc.input))

			placeholders := FindAll(data)
			require.Len(t, placeholders, 1)
			require.Equal(t, tc.expected, placeholders[0].Identifier)
		})
	}
}
