package dataflow_test

import (
	. "github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/dataflow"

	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DataflowShellCommand", func() {
	const (
		fileName = "filename"
		url      = "https://some.host/path"
	)

	var (
		skipSslValidation bool
		cmd               *exec.Cmd
	)

	JustBeforeEach(func() {
		cmd = DataflowShellCommand(fileName, url, skipSslValidation)
	})

	Context("when SSL validation is to be performed", func() {
		BeforeEach(func() {
			skipSslValidation = false
		})

		It("should produce the correct command", func() {
			Expect(cmd.Args).To(Equal([]string{"java", "-jar", fileName, "--dataflow.uri=" + url, "--dataflow.credentials-provider-command=cf oauth-token"}))
		})
	})

	Context("when SSL validation is to be skipped", func() {
		BeforeEach(func() {
			skipSslValidation = true
		})

		It("should produce the correct command", func() {
			Expect(cmd.Args).To(Equal([]string{"java", "-jar", fileName, "--dataflow.uri=" + url, "--dataflow.credentials-provider-command=cf oauth-token", "--dataflow.skip-ssl-validation=true"}))
		})
	})

})
