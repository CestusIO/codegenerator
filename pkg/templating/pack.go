package templating

import (
	"fmt"
	"path/filepath"
	"strings"

	"code.cestus.io/libs/codegenerator/pkg/locations"
)

// Pack represents a template source.
type Pack interface {
	GetName() string
	LoadTemplates() ([]Template, error)
}

// LoadPack returns a template pack for the specified template type and name.
func LoadPack(templateType, templateName string) (Pack, error) {
	name := filepath.Join(templateType, templateName)
	templatesRoots := GetTemplatesRoots()
	pp := NewPackProvider()
	RegisterFSPackProviders(pp, templatesRoots)
	pack, err := pp.Provide("", name)

	if err != nil {
		return nil, fmt.Errorf("no such pack \"%s\" found in: %s, use `%s` environement variable to set a custom root location", name, strings.Join(templatesRoots, ", "), locations.EnvCestusTemplateRoot)
	}

	return pack, nil
}
