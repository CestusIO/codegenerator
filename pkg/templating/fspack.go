package templating

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type fspack struct {
	name string
	fs   fs.ReadDirFS
}

func (p fspack) GetName() string { return p.name }
func (p fspack) LoadTemplates() (templates []Template, err error) {
	err = fs.WalkDir(p.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		f, err := p.fs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		template, err := LoadTemplate(path, f)

		if err != nil {
			return err
		}

		templates = append(templates, template)
		return nil
	})
	return
}

type fsPackProvider struct {
	fs fs.FS
}

func (p *fsPackProvider) Provide(templateType, templateName string) (Pack, error) {
	name := filepath.Join(templateType, templateName)
	return p.scanPacks(name)
}

func (p *fsPackProvider) scanPacks(name string) (*fspack, error) {
	_, err := fs.ReadDir(p.fs, name)
	if err != nil {
		return nil, errors.New("pack not found")
	}
	newRoot, err := fs.Sub(p.fs, name)
	if err != nil {
		return nil, errors.New("failed cd'ing to pack root")
	}
	pack := fspack{
		name: name,
		fs:   newRoot.(fs.ReadDirFS),
	}
	return &pack, nil
}

//NewFSPackProvider creates a new file system backed pack provider
func NewFsPackProvider(root string) *fsPackProvider {
	return &fsPackProvider{
		fs: os.DirFS(root),
	}
}

//RegisterFSPackProviders registers a slice of pack providers for a list of file system roots
func RegisterFSPackProviders(p PackGroupProvider, roots []string) {
	for _, root := range roots {
		p.RegisterProvider(NewFsPackProvider(root))
	}
}
