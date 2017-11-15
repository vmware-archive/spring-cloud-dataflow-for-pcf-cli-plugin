/*
 * Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under
 * the terms of the under the Apache License, Version 2.0 (the "License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package serviceutil_test

import (
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/serviceutil"

	"errors"

	"net/http"

	"code.cloudfoundry.org/cli/plugin/models"
	"code.cloudfoundry.org/cli/plugin/pluginfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/httpclient/httpclientfakes"
)

var _ = Describe("ServiceInstanceURL", func() {

	const errMessage = "some error"

	var (
		fakeCliConnection   *pluginfakes.FakeCliConnection
		serviceInstanceName string
		accessToken         string
		authClient          *httpclientfakes.FakeAuthenticatedClient
		serviceInstanceURL  string
		err                 error
		testError           error
	)

	BeforeEach(func() {
		fakeCliConnection = &pluginfakes.FakeCliConnection{}
		serviceInstanceName = "service-instance-name"
		authClient = &httpclientfakes.FakeAuthenticatedClient{}
		testError = errors.New(errMessage)
	})

	JustBeforeEach(func() {
		serviceInstanceURL, err = serviceutil.ServiceInstanceURL(fakeCliConnection, serviceInstanceName, accessToken, authClient)
	})

	It("should get the service", func() {
		Expect(fakeCliConnection.GetServiceCallCount()).To(Equal(1))
		Expect(fakeCliConnection.GetServiceArgsForCall(0)).To(Equal(serviceInstanceName))
	})

	Context("when the service instance is not found", func() {
		BeforeEach(func() {
			fakeCliConnection.GetServiceReturns(plugin_models.GetService_Model{}, testError)
		})

		It("should propagate the error", func() {
			Expect(err).To(MatchError("Service instance not found: " + errMessage))
		})
	})

	Context("when the dashboard URL is not in the correct format", func() {
		Context("because it is malformed", func() {
			BeforeEach(func() {
				fakeCliConnection.GetServiceReturns(plugin_models.GetService_Model{
					DashboardUrl: "://",
				}, nil)
			})

			It("should return a suitable error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("parse ://: missing protocol scheme"))
			})
		})

		Context("because its path format is invalid", func() {
			BeforeEach(func() {
				fakeCliConnection.GetServiceReturns(plugin_models.GetService_Model{
					DashboardUrl: "https://spring-cloud-broker.some.host.name",
				}, nil)
			})

			It("should return a suitable error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("path of https://spring-cloud-broker.some.host.name has no segments"))
			})
		})
	})

	Context("when the dashboard URL is in the correct format", func() {
		BeforeEach(func() {
			fakeCliConnection.GetServiceReturns(plugin_models.GetService_Model{
				DashboardUrl: "https://spring-cloud-broker.some.host.name/instances/guid/dashboard",
			}, nil)
		})

		It("should issue a get to the dataflow broker", func() {
			Expect(authClient.DoAuthenticatedGetCallCount()).To(Equal(1))
			url, token := authClient.DoAuthenticatedGetArgsForCall(0)
			Expect(url).To(Equal("https://spring-cloud-broker.some.host.name/instances/guid"))
			Expect(token).To(Equal(accessToken))
		})

		Context("when the dataflow broker cannot be contacted", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(nil, http.StatusBadGateway, http.Header{}, testError)
			})

			It("should return a suitable error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("dataflow service broker failed: " + errMessage))
			})
		})

		Context("when the dataflow broker returns status ok", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(nil, http.StatusOK, http.Header{}, nil)
			})

			It("should return a suitable error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("dataflow service broker did not return expected response (302): 200"))
			})
		})

		Context("when the dataflow broker returns a redirect without a location header", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(nil, http.StatusFound, http.Header{}, nil)
			})

			It("should return a suitable error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("dataflow service broker did not return a location header"))
			})
		})

		Context("when the dataflow broker returns a redirect with a location header with the wrong number of items", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(nil, http.StatusFound, http.Header{"Location": []string{}}, nil)
			})

			It("should return a suitable error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("dataflow service broker returned a location header of the wrong length (0)"))
			})
		})

		Context("when the dataflow broker returns a redirect with a location header with one item", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(nil, http.StatusFound, http.Header{"Location": []string{"https://dataflow-server-url"}}, nil)
			})

			It("should return a suitable error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(serviceInstanceURL).To(Equal("https://dataflow-server-url"))
			})
		})
	})
})

type badReader struct {
	readErr error
}

func (br *badReader) Read(p []byte) (n int, err error) {
	return 0, br.readErr
}

func (*badReader) Close() error {
	return nil
}
