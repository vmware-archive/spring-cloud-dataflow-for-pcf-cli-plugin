package skipper_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSkipper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Skipper Suite")
}
