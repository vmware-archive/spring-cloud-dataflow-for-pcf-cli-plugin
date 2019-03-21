/*
 * Copyright 2017-Present the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package pluginutil_test

import (
	"fmt"

	"code.cloudfoundry.org/cli/plugin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/pluginutil"
)

var _ = Describe("ParsePluginVersion", func() {

	var (
		pluginVersion string
		fail          func(format string, inserts ...interface{})
		failed        bool
		firstFailure  string
		parsedVersion plugin.VersionType
	)

	BeforeEach(func() {
		failed = false
		firstFailure = ""
		fail = func(format string, inserts ...interface{}) {
			// Capture just the first failure because ParsePluginVersion is designed to take a function that does not return normally
			if !failed {
				firstFailure = fmt.Sprintf(format, inserts...)
			}
			failed = true

		}
	})

	JustBeforeEach(func() {
		parsedVersion = pluginutil.ParsePluginVersion(pluginVersion, fail)
	})

	Context("when the input version has three integer components", func() {
		BeforeEach(func() {
			pluginVersion = "5.4.3"
		})

		It("should not fail", func() {
			Expect(failed).To(BeFalse())
		})

		It("should parse the version into its components", func() {
			Expect(parsedVersion).To(Equal(plugin.VersionType{
				Major: 5,
				Minor: 4,
				Build: 3,
			}))
		})
	})

	Context("when the input version has the wrong number of components", func() {
		BeforeEach(func() {
			pluginVersion = "2.0"
		})

		It("should fail", func() {
			Expect(failed).To(BeTrue())
		})

		It("should provide a suitable message", func() {
			Expect(firstFailure).To(Equal(`pluginVersion "2.0" has invalid format. Expected 3 dot-separated integer components.`))
		})
	})

	Context("when the input version has a non-integer component", func() {
		BeforeEach(func() {
			pluginVersion = "2.0."
		})

		It("should fail", func() {
			Expect(failed).To(BeTrue())
		})

		It("should provide a suitable message", func() {
			Expect(firstFailure).To(Equal(`pluginVersion "2.0." has invalid format. Expected integer components.`))
		})
	})

})
