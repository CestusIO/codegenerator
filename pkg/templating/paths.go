package templating

import (
	"code.cestus.io/libs/codegenerator/pkg/locations"
)

// GetTemplatesRoots returns the list of templates roots as detected on the system.
func GetTemplatesRoots() []string {
	return locations.GetRoots("templates")
}
