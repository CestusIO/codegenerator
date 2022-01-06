package locations

import (
	"os"
	"path/filepath"
	"strings"
)

// EnvCestusTemplateRoot points to a root folder to consider for resources loading.
const EnvCestusTemplateRoot = `CESTUS_CODEGENERATOR_TEMPLATE_ROOT`

func getExtraRoots() []string {
	if root, ok := os.LookupEnv(EnvCestusTemplateRoot); ok {
		return strings.Split(root, string(os.PathListSeparator))
	}

	return nil
}

func getBinaryRoots() []string {
	if exe, err := os.Executable(); err == nil {
		return []string{
			filepath.Clean(filepath.Join(filepath.Dir(exe), "..")),
		}
	}

	return nil
}

var roots []string

func init() {
	roots = append(roots, getExtraRoots()...)
	roots = append(roots, getBinaryRoots()...)
}

// GetRoots returns the roots, in order of preferences.
//
// If provided, all the suffixes are added to each of the returned roots with
// filepath.Join.
func GetRoots(suffixes ...string) []string {
	if len(suffixes) == 0 {
		return roots
	}

	result := make([]string, len(roots))

	for i, root := range roots {
		result[i] = filepath.Join(root, filepath.Join(suffixes...))
	}

	return result
}
