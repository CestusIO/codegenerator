package placeholder_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlaceholder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Placeholder Suite")
}
