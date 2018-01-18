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
package skipper_test

import (
	. "github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/skipper"

	"bytes"
	"io/ioutil"

	"net/http"

	"hash"

	"errors"

	"fmt"

	"crypto/sha256"

	"crypto/sha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/httpclient/httpclientfakes"
)

var _ = Describe("SkipperShellDownloadUrl", func() {
	const (
		skipperServerUrl   = "https://skipper.server"
		skipperShellUrl    = "https://skipper.shell"
		testAccessToken    = "someaccesstoken"
		errMessage         = "It's just fake. It's fake. It's made-up stuff."
		testSha1Checksum   = "cf23df2207d99a74fbe169e3eba035e633b65d94"
		testSha256Checksum = "9dec3eab5740cb087d7842bcb6bf924f9e008638dedeca16c5336bbc3c0e4453"
	)

	var (
		fakeAuthClient *httpclientfakes.FakeAuthenticatedClient
		payload        string
		testError      error
		getErr         error
		getStatus      int
		downloadUrl    string
		checksum       string
		hashFunc       hash.Hash
		err            error
	)

	BeforeEach(func() {
		fakeAuthClient = &httpclientfakes.FakeAuthenticatedClient{}
		getErr = nil
		getStatus = http.StatusOK
		testError = errors.New(errMessage)
	})

	JustBeforeEach(func() {
		fakeAuthClient.DoAuthenticatedGetReturns(ioutil.NopCloser(bytes.NewBufferString(payload)), getStatus, http.Header{}, getErr)
		downloadUrl, checksum, hashFunc, err = SkipperShellDownloadUrl(skipperServerUrl, fakeAuthClient, testAccessToken)
	})

	It("should drive the /about endpoint with the supplied access token", func() {
		Expect(fakeAuthClient.DoAuthenticatedGetCallCount()).To(Equal(1))
		aboutUrl, accessToken := fakeAuthClient.DoAuthenticatedGetArgsForCall(0)
		Expect(aboutUrl).To(Equal(skipperServerUrl + "/about"))
		Expect(accessToken).To(Equal(testAccessToken))
	})

	Context("when driving the /about endpoint returns an error", func() {
		BeforeEach(func() {
			getErr = testError
		})

		It("should propagate the error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(": %s", errMessage)))
		})
	})

	Context("when driving the /about endpoint returns an invalid HTTP status", func() {
		BeforeEach(func() {
			getStatus = http.StatusBadGateway
		})

		It("should propagate the error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(": %d", http.StatusBadGateway)))
		})
	})

	Context("when the /about endpoint returns a response reader which cannot be read", func() {
		JustBeforeEach(func() {
			fakeAuthClient.DoAuthenticatedGetReturns(ioutil.NopCloser(badReader{}), getStatus, http.Header{}, getErr)
			downloadUrl, checksum, hashFunc, err = SkipperShellDownloadUrl(skipperServerUrl, fakeAuthClient, testAccessToken)
		})

		It("should return a suitable error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Cannot read Skipper server response body: read error"))
		})
	})

	Context("when the /about endpoint returns invalid JSON", func() {
		BeforeEach(func() {
			payload = "{"
		})

		It("should return a suitable error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Invalid Skipper server response JSON: unexpected end of JSON input, response body: '{'"))
		})
	})

	Context("when the /about endpoint returns a shell download URL", func() {
		BeforeEach(func() {
			payload = fmt.Sprintf(`
				{"versionInfo":
					{"shell":
						{"url": "%s"
						}
					}
				}`, skipperShellUrl)
		})

		It("should succeed", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return the shell download URL", func() {
			Expect(downloadUrl).To(Equal(skipperShellUrl))
		})
	})

	Context("when the /about endpoint returns a SHA-1 shell checksum", func() {
		BeforeEach(func() {
			payload = fmt.Sprintf(`
				{"versionInfo":
					{"shell":
						{"checksumSha1": "%s"
						}
					}
				}`, testSha1Checksum)
		})

		It("should succeed", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return the SHA-1 checksum", func() {
			Expect(checksum).To(Equal(testSha1Checksum))
		})

		It("should return a SHA-1 hash function", func() {
			Expect(hashFunc).To(BeAssignableToTypeOf(sha1.New()))
		})
	})

	Context("when the /about endpoint returns a SHA-256 shell checksum", func() {
		BeforeEach(func() {
			payload = fmt.Sprintf(`
				{"versionInfo":
					{"shell":
						{"checksumSha256": "%s"
						}
					}
				}`, testSha256Checksum)
		})

		It("should succeed", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return the SHA-256 checksum", func() {
			Expect(checksum).To(Equal(testSha256Checksum))
		})

		It("should return a SHA-256 hash function", func() {
			Expect(hashFunc).To(BeAssignableToTypeOf(sha256.New()))
		})
	})

	Context("when the /about endpoint returns SHA-1 and SHA-256 shell checksums", func() {
		BeforeEach(func() {
			payload = fmt.Sprintf(`
				{"versionInfo":
					{"shell":
						{"checksumSha1": "%s",
						 "checksumSha256": "%s"
						}
					}
				}`, testSha1Checksum, testSha256Checksum)
		})

		It("should succeed", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return the SHA-256 checksum", func() {
			Expect(checksum).To(Equal(testSha256Checksum))
		})

		It("should return a SHA-256 hash function", func() {
			Expect(hashFunc).To(BeAssignableToTypeOf(sha256.New()))
		})
	})
})

type badReader struct{}

func (b badReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}
