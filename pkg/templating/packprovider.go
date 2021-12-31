package templating

import "errors"

// Packprovider provides a template pack
type PackProvider interface {
	Provide(templateType, templateName string) (Pack, error)
}

// PackGroupProvider is a meta pack provider
type PackGroupProvider interface {
	RegisterProvider(provider PackProvider) error
}

type packLoader struct {
	providers []PackProvider
}

func (p *packLoader) RegisterProvider(provider PackProvider) error {
	p.providers = append(p.providers, provider)
	return nil
}

func (p *packLoader) Provide(templateType, templateName string) (Pack, error) {
	for _, provider := range p.providers {
		pack, err := provider.Provide(templateType, templateName)
		if err == nil {
			return pack, err
		}
	}
	return nil, errors.New("no pack found")
}

func NewPackProvider() *packLoader {
	return &packLoader{}
}
