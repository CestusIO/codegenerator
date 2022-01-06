package templating

import (
	"errors"
	"io/fs"
	"path"
)

type embededPackProvider struct {
	fs fs.ReadDirFS
}

func (p *embededPackProvider) Provide(templateType, templateName string) (Pack, error) {
	// embedded filesystems use / regardless of the platform. So we cannot use filepath.Join which is os aware
	name := path.Join(templateType, templateName)
	return p.scanPacks(name)
}

func (p *embededPackProvider) scanPacks(name string) (*embededPack, error) {
	_, err := fs.ReadDir(p.fs, name)
	if err != nil {
		return nil, errors.New("pack not found")
	}
	newRoot, err := fs.Sub(p.fs, name)
	if err != nil {
		return nil, errors.New("failed cd'ing to pack root")
	}
	pack := embededPack{
		name: name,
		fs:   newRoot.(fs.ReadDirFS),
	}
	return &pack, nil
}
func NewEmbededPackProvider(fs fs.ReadDirFS) *embededPackProvider {
	return &embededPackProvider{
		fs: fs,
	}
}

var _ Pack = (*embededPack)(nil)

type embededPack struct {
	name string
	fs   fs.ReadDirFS
}

func (p *embededPack) GetName() string {
	return p.name
}
func (p *embededPack) GetPath() string {
	return ""
}

func (p *embededPack) LoadTemplates() (templates []Template, err error) {
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
