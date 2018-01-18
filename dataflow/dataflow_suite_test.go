package dataflow_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDataflow(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dataflow Suite")
}
