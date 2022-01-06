package templating

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

func compareFolders(t *testing.T, referenceRootPath string, rootPath string) {
	t.Helper()

	filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		t.Run(path, func(t *testing.T) {
			relPath, _ := filepath.Rel(rootPath, path)
			referencePath := filepath.Join(referenceRootPath, relPath)

			refInfo, err := os.Stat(referencePath)

			if err != nil {
				t.Fatalf("%s is not supposed to exist", path)
			}

			if info.IsDir() {
				if !refInfo.IsDir() {
					t.Fatalf("%s was not supposed to be a directory", path)
				}
			} else {
				if refInfo.IsDir() {
					t.Fatalf("%s was supposed to be a directory", path)
				}

				compareFiles(t, referencePath, path)
			}
		})

		return nil
	})
}

func compareFiles(t *testing.T, referencePath string, path string) {
	t.Helper()

	reference, err := os.ReadFile(referencePath)

	if err != nil {
		t.Fatalf("unable to read reference file: %s", err)
	}

	output, err := os.ReadFile(path)

	if err != nil {
		t.Fatalf("unable to read file: %s", err)
	}

	diff := difflib.ContextDiff{
		A:        difflib.SplitLines(string(reference)),
		B:        difflib.SplitLines(string(output)),
		FromFile: referencePath,
		ToFile:   path,
		Context:  3,
		Eol:      "\n",
	}
	result, _ := difflib.GetContextDiffString(diff)

	if result != "" {
		t.Fatalf("produced files are different:\n%s", result)
	}
}

func TestRender(t *testing.T) {
	for _, packName := range []string{"foo"} {
		t.Run(packName, func(t *testing.T) {
			outputPath := "fixtures/templates/output"
			referencePath := "fixtures/templates/reference"
			ctx := "world"
			pp := NewPackProvider()
			RegisterFSPackProviders(pp, []string{"fixtures/templates"})
			pack, err := pp.Provide("", packName)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			templates, err := pack.LoadTemplates()

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if err := os.MkdirAll(outputPath, 0755); err != nil {
				t.Fatalf("expected no error but got: %s", err)
			}
			data := `Updated text.

// region CODE_REGION(Foo)
Text that will be lost
// endregion

There too.
`
			os.WriteFile(filepath.Join(outputPath, "d.txt"), []byte(data), 0755)

			data = `This is some stuff.
`
			os.WriteFile(filepath.Join(outputPath, "f.txt"), []byte(data), 0755)

			generatedFiles, _, err := Render(templates, outputPath, ctx)
			defer os.RemoveAll(outputPath)

			if err != nil {
				t.Errorf("expected no error but got: %s", err)
			}

			reference := []string{
				filepath.Clean(fmt.Sprintf("%s/a.txt", outputPath)),
				filepath.Clean(fmt.Sprintf("%s/b.txt", outputPath)),
				filepath.Clean(fmt.Sprintf("%s/c-%s.txt", outputPath, ctx)),
				filepath.Clean(fmt.Sprintf("%s/d.txt", outputPath)),
				filepath.Clean(fmt.Sprintf("%s/e.txt", outputPath)),
				filepath.Clean(fmt.Sprintf("%s/f.txt", outputPath)),
				filepath.Clean(fmt.Sprintf("%s/replacedpath/h-world.txt", outputPath)),
			}
			sort.Strings(reference)
			sort.Strings(generatedFiles)

			if !reflect.DeepEqual(reference, generatedFiles) {
				t.Errorf("expected %v, got %v", reference, generatedFiles)
			}

			compareFolders(t, referencePath, outputPath)
		})
	}
}

func TestRenderWithPatterns(t *testing.T) {
	for _, packName := range []string{"foo"} {
		t.Run(packName, func(t *testing.T) {
			outputPath := "fixtures/templates/output"
			referencePath := "fixtures/templates/reference"
			ctx := "world"
			pp := NewPackProvider()
			RegisterFSPackProviders(pp, []string{"fixtures/templates"})
			pack, err := pp.Provide("", packName)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			templates, err := pack.LoadTemplates()

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if err := os.MkdirAll(outputPath, 0755); err != nil {
				t.Fatalf("expected no error but got: %s", err)
			}

			patterns := []string{
				"a.txt",
				"c-world.txt",
			}

			generatedFiles, _, err := Render(templates, outputPath, ctx, patterns...)
			defer os.RemoveAll(outputPath)

			if err != nil {
				t.Errorf("expected no error but got: %s", err)
			}

			reference := []string{
				filepath.Clean(fmt.Sprintf("%s/a.txt", outputPath)),
				filepath.Clean(fmt.Sprintf("%s/c-%s.txt", outputPath, ctx)),
			}
			sort.Strings(reference)
			sort.Strings(generatedFiles)

			if !reflect.DeepEqual(reference, generatedFiles) {
				t.Errorf("expected %v, got %v", reference, generatedFiles)
			}

			compareFolders(t, referencePath, outputPath)
		})
	}
}

func TestRenderInvalidName(t *testing.T) {
	outputPath := "fixtures/templates/output"
	ctx := "world"
	pp := NewPackProvider()
	RegisterFSPackProviders(pp, []string{"fixtures/templates"})
	pack, err := pp.Provide("", "invalid-name")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	templates, err := pack.LoadTemplates()

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	generatedFiles, _, err := Render(templates, outputPath, ctx)
	defer os.RemoveAll(outputPath)

	if err == nil {
		t.Errorf("expected an error")
	}

	if generatedFiles != nil {
		t.Errorf("expected no generated files but got: %v", generatedFiles)
	}
}

func TestRenderInvalidContent(t *testing.T) {
	outputPath := "fixtures/templates/output"
	ctx := "world"
	pp := NewPackProvider()
	RegisterFSPackProviders(pp, []string{"fixtures/templates"})
	pack, err := pp.Provide("", "invalid-content")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	templates, err := pack.LoadTemplates()

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	generatedFiles, _, err := Render(templates, outputPath, ctx)
	defer os.RemoveAll(outputPath)

	if err == nil {
		t.Errorf("expected an error")
	}

	if generatedFiles != nil {
		t.Errorf("expected no generated files but got: %v", generatedFiles)
	}
}
