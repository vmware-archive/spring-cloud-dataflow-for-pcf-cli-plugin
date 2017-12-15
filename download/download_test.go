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
package download_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"

	"net/http"

	"fmt"

	"io/ioutil"

	"bytes"

	"crypto/sha256"
	"hash"

	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/download"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/download/downloadfakes"
)

const (
	errMessage        = "things can only get better"
	ifNoneMatchHeader = "If-None-Match"
	etagHeader        = "ETag"
	etagValue         = "etag"
	urlValue          = "http://some/remote/file"
	checksumValue     = "checksum"
	testFilePath      = "/some/path"
)

var _ = Describe("Download", func() {

	var (
		downloader       download.Downloader
		fakeCache        *downloadfakes.FakeCache
		fakeCacheEntry   *downloadfakes.FakeCacheEntry
		fakeHttpHelper   *downloadfakes.FakeHttpHelper
		fakeHttpRequest  *downloadfakes.FakeHttpRequest
		fakeHttpResponse *downloadfakes.FakeHttpResponse
		hashFunc         hash.Hash
		filePath         string
		etag             string
		testError        error
		err              error
		url              string
	)

	BeforeEach(func() {
		fakeCache = &downloadfakes.FakeCache{}
		fakeCacheEntry = &downloadfakes.FakeCacheEntry{}
		fakeHttpHelper = &downloadfakes.FakeHttpHelper{}
		fakeHttpRequest = &downloadfakes.FakeHttpRequest{}
		fakeHttpResponse = &downloadfakes.FakeHttpResponse{}
		hashFunc = sha256.New()
		etag = etagValue
		testError = errors.New(errMessage)
		url = urlValue

		downloader, err = download.NewDownloader(fakeCache, fakeHttpHelper, GinkgoWriter)
	})

	Describe("DownloadFile", func() {
		JustBeforeEach(func() {
			filePath, err = downloader.DownloadFile(urlValue, checksumValue, hashFunc)
		})

		Context("when it is the normal case", func() {
			BeforeEach(func() {
				fakeCache.EntryReturns(fakeCacheEntry)

				fakeHttpHelper.CreateHttpRequestStub = func(method string, url string) (download.HttpRequest, error) {
					return fakeHttpRequest, nil
				}

				fakeHttpRequest.SendRequestStub = func() (download.HttpResponse, error) {
					return fakeHttpResponse, nil
				}
			})

			It("should request an entry from the cache for the supplied url", func() {
				Expect(fakeCache.EntryCallCount()).To(Equal(1))
				Expect(fakeCache.EntryArgsForCall(0)).To(Equal(url))
			})

			It("should try and retrieve cache details from the received cache entry", func() {
				Expect(fakeCacheEntry.RetrieveCallCount()).To(Equal(1))
			})

			Context("when retrieving details from the cache entry results in an error", func() {
				BeforeEach(func() {
					fakeCacheEntry.RetrieveReturns("", "", testError)
				})

				It("should propagate the error", func() {
					Expect(err).To(MatchError(errMessage))
				})
			})

			Context("when retrieving details from the cache entry is successful", func() {
				It("should prepare a new HTTP GET request with the supplied url", func() {
					Expect(fakeHttpHelper.CreateHttpRequestCallCount()).To(Equal(1))
					requestMethod, requestUrl := fakeHttpHelper.CreateHttpRequestArgsForCall(0)
					Expect(requestMethod).To(Equal(http.MethodGet))
					Expect(requestUrl).To(Equal(url))
				})

				Context("when preparing a new HTTP GET request results in an error", func() {
					BeforeEach(func() {
						fakeHttpHelper.CreateHttpRequestReturns(fakeHttpRequest, testError)
					})

					It("should propagate the error", func() {
						Expect(err).To(MatchError(fmt.Sprintf(`CreateHttpRequest for download URL "http://some/remote/file" failed: %s`, errMessage)))
					})
				})

				Context("when the retrieved cache entry contains a non empty file path", func() {
					Context("when the retrieved cache entry contains a non-empty etag value", func() {
						BeforeEach(func() {
							fakeCacheEntry.RetrieveReturns(testFilePath, etag, nil)
						})

						It("should set the If-None-Match header request header with the etag value", func() {
							Expect(fakeHttpRequest.SetHeaderCallCount()).To(Equal(1))
							headerKey, headerValue := fakeHttpRequest.SetHeaderArgsForCall(0)
							Expect(headerKey).To(Equal(ifNoneMatchHeader))
							Expect(headerValue).To(Equal(etag))
						})
					})

					Context("when the retrieved cache entry contains an empty etag value", func() {
						BeforeEach(func() {
							fakeCacheEntry.RetrieveReturns(testFilePath, "", nil)
						})

						It("should not try to set the If-None-Match request header", func() {
							Expect(fakeHttpRequest.SetHeaderCallCount()).To(Equal(0))
						})
					})
				})

				Context("when the retrieved cache entry contains an empty file path", func() {
					Context("when the retrieved cache entry contains a non-empty etag value", func() {
						BeforeEach(func() {
							fakeCacheEntry.RetrieveReturns("", etag, nil)
						})

						It("should not try to set the If-None-Match request header", func() {
							Expect(fakeHttpRequest.SetHeaderCallCount()).To(Equal(0))
						})
					})

					Context("when the retrieved cache entry contains an empty etag value", func() {
						BeforeEach(func() {
							fakeCacheEntry.RetrieveReturns("", "", nil)
						})

						It("should not try to set the If-None-Match request header", func() {
							Expect(fakeHttpRequest.SetHeaderCallCount()).To(Equal(0))
						})
					})
				})

				It("should make the HTTP GET request", func() {
					Expect(fakeHttpRequest.SendRequestCallCount()).To(Equal(1))
				})

				Context("when sending the HTTP GET request returns an error", func() {
					BeforeEach(func() {
						fakeHttpRequest.SendRequestReturns(nil, testError)
					})

					It("should propagate the error", func() {
						Expect(err).To(MatchError(fmt.Sprintf(`Download from URL "http://some/remote/file" failed: %s`, errMessage)))
					})
				})

				Context("when sending the HTTP GET request is successful and returns a 304 response code", func() {
					BeforeEach(func() {
						fakeHttpRequest.SendRequestStub = func() (download.HttpResponse, error) {
							fakeHttpResponse.GetStatusCodeReturns(http.StatusNotModified)
							return fakeHttpResponse, nil
						}
					})

					It("should not continue to try and store a file in the cache", func() {
						Expect(fakeHttpResponse.GetHeaderCallCount()).To(Equal(0))
						Expect(fakeHttpResponse.GetBodyCallCount()).To(Equal(0))
						Expect(fakeCacheEntry.StoreCallCount()).To(Equal(0))
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("when sending the HTTP GET request is successful and returns a 200 response code", func() {

					var (
						responseBody = ioutil.NopCloser(bytes.NewReader([]byte("whatever")))
					)

					BeforeEach(func() {
						fakeCacheEntry.RetrieveReturns(testFilePath, "", nil)

						fakeHttpResponse.GetStatusCodeReturns(http.StatusOK)
						fakeHttpResponse.GetBodyReturns(responseBody)

						fakeHttpResponse.GetHeaderStub = func(name string) string {
							if name == etagHeader {
								return etagValue
							}
							return ""
						}

						fakeHttpRequest.SendRequestStub = func() (download.HttpResponse, error) {
							return fakeHttpResponse, nil
						}
					})

					It("should query the value of the ETag header in the response", func() {
						Expect(fakeHttpResponse.GetHeaderCallCount()).To(Equal(1))
						Expect(fakeHttpResponse.GetHeaderArgsForCall(0)).To(Equal(etagHeader))
					})

					It("should get the body of the response for passing to the cache", func() {
						Expect(fakeHttpResponse.GetBodyCallCount()).To(Equal(1))
					})

					It("should try and store the file in the cache", func() {
						Expect(fakeCacheEntry.StoreCallCount()).To(Equal(1))

						contentsArg, tagArg, checksumArg, hf := fakeCacheEntry.StoreArgsForCall(0)
						Expect(contentsArg).To(Equal(responseBody))
						Expect(tagArg).To(Equal(etagValue))
						Expect(checksumArg).To(Equal(checksumValue))
						Expect(hf).To(Equal(hashFunc))
					})

					It("should return the file path from the cache entry", func() {
						Expect(filePath).To(Equal(testFilePath))
					})

					Context("when trying to store the file in the cache fails", func() {
						BeforeEach(func() {
							fakeCacheEntry.StoreReturns(testError)
						})

						It("should propagate the error", func() {
							Expect(err).To(MatchError(errMessage))
						})
					})

					Context("when retrieving the file path from the cache entry initially returns an empty file path", func() {
						BeforeEach(func() {
							fakeCacheEntry.RetrieveReturnsOnCall(0, "", etag, nil)
						})

						Context("when retrieving the file path from the updated cache entry succeeds", func() {
							BeforeEach(func() {
								fakeCacheEntry.RetrieveReturnsOnCall(1, testFilePath, etag, nil)
							})

							It("should return the file path from the cache entry", func() {
								Expect(filePath).To(Equal(testFilePath))
							})
						})

						Context("when retrieving the file path from the updated cache entry fails", func() {
							BeforeEach(func() {
								fakeCacheEntry.RetrieveReturnsOnCall(1, "", etag, testError)
							})

							It("should propagate the error", func() {
								Expect(err).To(MatchError(errMessage))
							})
						})
					})
				})

				Context("when sending the HTTP GET request is successful but returns an unexpected response code", func() {
					BeforeEach(func() {
						fakeHttpRequest.SendRequestStub = func() (download.HttpResponse, error) {
							fakeHttpResponse.GetStatusCodeReturns(http.StatusServiceUnavailable)
							return fakeHttpResponse, nil
						}
					})

					It("should return a new error", func() {
						Expect(err).To(MatchError(fmt.Sprintf("Unexpected response '%d' downloading from '%s'", http.StatusServiceUnavailable, url)))
					})
				})
			})
		})
	})
})

var _ = Describe("HttpRequest", func() {
	var (
		testError error
		err       error
		url       string
	)

	Context("with fake dependency", func() {
		var (
			request        download.HttpRequest
			response       download.HttpResponse
			fakeHttpClient *downloadfakes.FakeHttpClient
		)

		BeforeEach(func() {
			fakeHttpClient = &downloadfakes.FakeHttpClient{}
			testError = errors.New(errMessage)

			request, err = download.NewHttpHelper().CreateHttpRequest(http.MethodGet, url)

			if request, ok := request.(download.HttpClientSetter); ok {
				request.SetHttpClient(fakeHttpClient)
			} else {
				Fail("request did not implement HttpClientSetter")
			}
		})

		Describe("SetHeader", func() {
			JustBeforeEach(func() {
				request.SetHeader(etagHeader, etagValue)
			})

			It("should have set the specified header with the supplied value", func() {
				if request, ok := request.(download.RequestFieldGetter); ok {
					Expect(request.GetHeaderMap().Get(etagHeader)).To(Equal(etagValue))
				} else {
					Fail("request did not implement RequestFieldGetter")
				}
			})
		})

		Describe("SendRequest", func() {
			JustBeforeEach(func() {
				response, err = request.SendRequest()
			})

			Context("when sending the HTTP request results in an error", func() {
				BeforeEach(func() {
					fakeHttpClient.DoReturns(nil, testError)
				})

				It("should propagate the error", func() {
					Expect(err).To(MatchError(testError))
				})
			})

			Context("when sending the HTTP request is successful", func() {
				var goodResponse = &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader([]byte("response body"))),
				}

				BeforeEach(func() {
					fakeHttpClient.DoReturns(goodResponse, nil)
				})

				It("should return a new HttpResponse that wraps the low level response", func() {
					buf := new(bytes.Buffer)
					buf.ReadFrom(response.GetBody())
					bodyString := buf.String()
					Expect(bodyString).To(Equal("response body"))

					Expect(response.GetStatusCode()).To(Equal(http.StatusOK))
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})

	Context("with actual dependency", func() {
		var (
			request download.HttpRequest
			err     error
		)

		JustBeforeEach(func() {
			_, err = request.SendRequest()
		})

		Context("when the URL is empty", func() {
			BeforeEach(func() {
				request, err = download.NewHttpHelper().CreateHttpRequest(http.MethodGet, "")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return a suitable error", func() {
				Expect(err.Error()).To(ContainSubstring("unsupported protocol scheme"))
			})
		})
	})
})

var _ = Describe("HttpResponse", func() {
	var (
		response download.HttpResponse
	)

	Describe("GetHeader", func() {
		Context("when the response contains a header", func() {
			BeforeEach(func() {
				response = download.NewHttpResponse(&http.Response{
					Header: http.Header{"Accept-Encoding": []string{"gzip", "deflate"}},
				})
			})

			It("should return the header", func() {
				Expect(response.GetHeader("Accept-Encoding")).To(Equal("gzip"))
			})
		})
	})
})

var _ = Describe("CreateHttpRequest", func() {
	It("should return a suitable error when the method is unknown", func() {
		_, err := download.NewHttpHelper().CreateHttpRequest("bad method", "http://example.com")
		Expect(err).To(MatchError(`net/http: invalid method "bad method"`))
	})
})
