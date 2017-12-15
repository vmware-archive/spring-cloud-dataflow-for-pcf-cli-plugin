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
package cache_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"

	"fmt"

	"bufio"

	"crypto/sha256"
	"hash"

	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/download/cache"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/download/downloadfakes"
)

const (
	cfHomeProperty = "CF_HOME"
	homeProperty   = "HOME"
	errMessage     = "worse things happen at sea"
	urlValue       = "http://host/path/file.extension"
)

var (
	testCacheUnderCfHomeFolder string
	testCacheUnderHomeFolder   string
	cfHomeWasSet               bool
	oldCfHomeValue             string
	err                        error
)

var _ = BeforeSuite(func() {
	testCacheUnderCfHomeFolder, err = ioutil.TempDir("", "plugin-testing-cf-home")
	if err != nil {
		Fail(fmt.Sprintf("Unable to create temporary test folder: %s", err.Error()))
	}

	oldCfHomeValue, cfHomeWasSet = os.LookupEnv(cfHomeProperty)
	os.Setenv(cfHomeProperty, testCacheUnderCfHomeFolder)
})

var _ = AfterSuite(func() {
	os.RemoveAll(testCacheUnderCfHomeFolder)

	if cfHomeWasSet {
		os.Setenv(cfHomeProperty, oldCfHomeValue)
	} else {
		os.Unsetenv(cfHomeProperty)
	}
})

var _ = Describe("Cache", func() {
	var (
		downloadsCache cache.Cache
		cacheEntry     cache.CacheEntry
		err            error
		testError      error
		url            string
	)

	Describe("NewCache", func() {

		JustBeforeEach(func() {
			downloadsCache, err = cache.NewCache(GinkgoWriter)
		})

		Context("when the downloads directory cannot be created", func() {
			var downloadsDir string

			BeforeEach(func() {
				downloadsParentDir := path.Join(testCacheUnderCfHomeFolder, ".cf", "spring-cloud-dataflow-for-pcf")
				Expect(os.MkdirAll(downloadsParentDir, 0755)).To(Succeed())
				downloadsDir = path.Join(downloadsParentDir, "cache")
				Expect(os.RemoveAll(downloadsDir)).To(Succeed())
				Expect(ioutil.WriteFile(downloadsDir, []byte("x"), 0755)).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.Remove(downloadsDir)).To(Succeed())
			})

			It("should propagate the error", func() {
				Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
			})
		})

		Context("when the cache data file cannot be created", func() {
			var downloadsDir string

			BeforeEach(func() {
				downloadsParentDir := path.Join(testCacheUnderCfHomeFolder, ".cf", "spring-cloud-dataflow-for-pcf")
				Expect(os.MkdirAll(downloadsParentDir, 0755)).To(Succeed())
				downloadsDir = path.Join(downloadsParentDir, "cache")
				Expect(os.RemoveAll(downloadsDir)).To(Succeed())
				Expect(os.Mkdir(downloadsDir, 0555)).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.Remove(downloadsDir)).To(Succeed())
			})

			It("should propagate the error", func() {
				Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
			})
		})
	})

	Describe("Entry", func() {
		BeforeEach(func() {
			testError = errors.New(errMessage)
			url = urlValue

			downloadsCache, err = cache.NewCache(GinkgoWriter)
		})

		Context("when CF_HOME is set", func() {
			JustBeforeEach(func() {
				cacheEntry = downloadsCache.Entry(url)
			})

			It("should return a cache entry that has stored the expected path for the cache entries file", func() {
				cacheEntriesFilePath := path.Join(testCacheUnderCfHomeFolder, ".cf", "spring-cloud-dataflow-for-pcf", "cache", ".cachedata")
				_, err := os.Stat(cacheEntriesFilePath)
				Expect(os.IsNotExist(err)).To(BeFalse())
			})

			It("should return a cache entry that has stored the expected path for the file to download", func() {
				if cacheEntry, ok := cacheEntry.(cache.FieldGetter); ok {
					downloadFilePath := cacheEntry.GetDownloadFile()
					Expect(downloadFilePath).Should(HavePrefix(testCacheUnderCfHomeFolder))
					Expect(downloadFilePath).Should(HaveSuffix(".cf/spring-cloud-dataflow-for-pcf/cache/file.extension"))
				} else {
					Fail("cache entry did not implement FieldGetter")
				}
			})

			It("should return a cache entry that has stored the supplied url value", func() {
				if cacheEntry, ok := cacheEntry.(cache.FieldGetter); ok {
					Expect(cacheEntry.GetDownloadUrl()).To(Equal(url))
				} else {
					Fail("cache entry did not implement FieldGetter")
				}
			})
		})

		Context("when CF_HOME is not set", func() {
			var (
				originalCfHomeValue string
				originalHomeValue   string
				homeWasSet          bool
			)

			BeforeEach(func() {
				testCacheUnderHomeFolder, err = ioutil.TempDir("", "plugin-testing-home")
				if err != nil {
					Fail(fmt.Sprintf("Unable to create temporary test folder: %s", err.Error()))
				}

				originalCfHomeValue = os.Getenv(cfHomeProperty)
				os.Unsetenv(cfHomeProperty)

				originalHomeValue, homeWasSet = os.LookupEnv(homeProperty)
				os.Setenv(homeProperty, testCacheUnderHomeFolder)
			})

			AfterEach(func() {
				os.RemoveAll(testCacheUnderHomeFolder)

				if homeWasSet {
					os.Setenv(homeProperty, originalHomeValue)
				} else {
					os.Unsetenv(homeProperty)
				}

				os.Setenv(cfHomeProperty, originalCfHomeValue)
			})

			JustBeforeEach(func() {
				downloadsCache, err = cache.NewCache(GinkgoWriter)
				cacheEntry = downloadsCache.Entry(url)
			})

			It("should return a cache entry that has stored the expected path for the cache entries file", func() {
				cacheEntriesFilePath := path.Join(testCacheUnderHomeFolder, ".cf", "spring-cloud-dataflow-for-pcf", "cache", ".cachedata")
				_, err := os.Stat(cacheEntriesFilePath)
				Expect(os.IsNotExist(err)).To(BeFalse())
			})

			It("should return a cache entry that has stored the expected path for the file to download", func() {
				if cacheEntry, ok := cacheEntry.(cache.FieldGetter); ok {
					downloadFilePath := cacheEntry.GetDownloadFile()
					Expect(downloadFilePath).Should(HavePrefix(testCacheUnderHomeFolder))
					Expect(downloadFilePath).Should(HaveSuffix(".cf/spring-cloud-dataflow-for-pcf/cache/file.extension"))
				} else {
					Fail("cache entry did not implement FieldGetter")
				}
			})

			It("should return a cache entry that has stored the supplied url value", func() {
				if cacheEntry, ok := cacheEntry.(cache.FieldGetter); ok {
					Expect(cacheEntry.GetDownloadUrl()).To(Equal(url))
				} else {
					Fail("cache entry did not implement FieldGetter")
				}
			})
		})
	})
})

var _ = Describe("CacheEntry", func() {

	const (
		etagValue             = "etag"
		checksumValue         = "79dbdd760b4e80686e81c81466424ca6a21ed70b353a19e2154984e41a3e6e4b"
		downloadContentString = "download content"
	)

	var (
		fakeChecksumCalculator *downloadfakes.FakeChecksumCalculator
		fakeEtagHelper         *downloadfakes.FakeEtagHelper
		downloadsCache         cache.Cache
		cacheEntry             cache.CacheEntry
		downloadContent        io.ReadCloser
		downloadFilePath       string
		etagArgument           string
		testError              error
		err                    error
		hashFunc               hash.Hash
	)

	BeforeEach(func() {
		downloadFilePath = path.Join(testCacheUnderCfHomeFolder, ".cf", "spring-cloud-dataflow-for-pcf", "cache", "file.extension")

		downloadsCache, err = cache.NewCache(GinkgoWriter)

		downloadContent = ioutil.NopCloser(bytes.NewReader([]byte(downloadContentString)))

		fakeChecksumCalculator = &downloadfakes.FakeChecksumCalculator{}

		fakeEtagHelper = &downloadfakes.FakeEtagHelper{}

		etagArgument = etagValue

		testError = errors.New(errMessage)

		cacheEntry = downloadsCache.Entry(urlValue)

		hashFunc = sha256.New()
	})

	Describe("Retrieve", func() {
		It("should return an empty path when the file has not been cached", func() {
			path, _, err := cacheEntry.Retrieve()
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(BeEmpty())
		})
	})

	Describe("Store", func() {
		JustBeforeEach(func() {
			err = cacheEntry.Store(downloadContent, etagArgument, checksumValue, hashFunc)
		})

		Context("with actual dependencies", func() {
			It("should succeed", func() {
				Expect(err).To(Succeed())
			})

			It("should return the correct etag on retrieval", func() {
				_, etag, err := cacheEntry.Retrieve()
				Expect(err).NotTo(HaveOccurred())
				Expect(etag).To(Equal(etagValue))
			})

			Context("when the download content cannot be read", func() {
				BeforeEach(func() {
					downloadContent = ioutil.NopCloser(badReader{})
				})

				It("should percolate the error", func() {
					Expect(err).To(MatchError("read error"))
				})
			})

			Context("when the stored file cannot be read to calculate its checksum", func() {
				BeforeEach(func() {
					var checksumCalculator cache.ChecksumCalculator

					if cacheEntry, ok := cacheEntry.(cache.FieldGetter); ok {
						checksumCalculator = cacheEntry.GetChecksumCalculator()
					} else {
						Fail("cache entry did not implement FieldGetter")
					}

					fakeChecksumCalculator.CalculateChecksumStub = func(filePath string, hashFunc hash.Hash) (string, error) {
						Expect(os.Remove(filePath)).To(Succeed())
						return checksumCalculator.CalculateChecksum(filePath, hashFunc)
					}

					if cacheEntry, ok := cacheEntry.(cache.FieldSetter); ok {
						cacheEntry.SetChecksumCalculator(fakeChecksumCalculator)
					} else {
						Fail("cache entry did not implement FieldSetter")
					}
				})

				It("should percolate the error", func() {
					Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
				})
			})

			Context("when the checksum accumulation fails", func() {
				BeforeEach(func() {
					hashFunc = badHash{}
				})

				It("should percolate the error", func() {
					Expect(err).To(MatchError("write error"))
				})
			})
		})

		Context("with fake dependencies", func() {
			BeforeEach(func() {
				if cacheEntry, ok := cacheEntry.(cache.FieldSetter); ok {
					cacheEntry.SetChecksumCalculator(fakeChecksumCalculator)
					cacheEntry.SetEtagHelper(fakeEtagHelper)
				} else {
					Fail("cache entry did not implement FieldSetter")
				}
			})

			Context("in the normal case", func() {
				BeforeEach(func() {
					fakeChecksumCalculator.CalculateChecksumReturns(checksumValue, nil)
				})

				It("should create the file with the expected name in the expected location", func() {
					Expect(fileExists(downloadFilePath)).To(BeTrue())
				})

				It("should write the supplied data into the download file", func() {
					fileContent, err := readTestFileContent(downloadFilePath)
					if err != nil {
						Fail(fmt.Sprintf("Unable to read test file %s\n", downloadFilePath))
					}
					Expect(fileContent).To(Equal(downloadContentString))
				})

				It("should calculate the checksum value of the downloaded file", func() {
					Expect(fakeChecksumCalculator.CalculateChecksumCallCount()).To(Equal(1))
				})

				Context("when an error occurs calculating the checksum", func() {
					BeforeEach(func() {
						fakeChecksumCalculator.CalculateChecksumReturns("", testError)
					})

					It("should propagate the error", func() {
						Expect(err).To(MatchError(testError))
					})
				})

				Context("when the calculated checksum does not match the supplied value", func() {
					BeforeEach(func() {
						fakeChecksumCalculator.CalculateChecksumReturns("unexpected value", nil)
					})

					It("should raise an error", func() {
						Expect(err).To(MatchError(fmt.Sprintf("Downloaded file '%s' checksum does not match supplied value", downloadFilePath)))
					})
				})

				Context("when the supplied etag value is not an empty string", func() {
					It("should set the etag value in the cache entries file", func() {
						Expect(fakeEtagHelper.SetEtagForUrlCallCount()).To(Equal(1))

						urlArg, etagArg := fakeEtagHelper.SetEtagForUrlArgsForCall(0)
						Expect(urlArg).To(Equal(urlValue))
						Expect(etagArg).To(Equal(etagValue))
					})

					Context("when trying to set the etag value fails with an error", func() {
						BeforeEach(func() {
							fakeEtagHelper.SetEtagForUrlReturns(testError)
						})

						It("should propagate the error", func() {
							Expect(err).To(MatchError(testError))
						})
					})
				})

				Context("when the supplied etag value is an empty string", func() {
					BeforeEach(func() {
						etagArgument = ""
					})

					It("should not try to set the etag value in the cache entries file", func() {
						Expect(fakeEtagHelper.SetEtagForUrlCallCount()).To(Equal(0))
					})
				})
			})

			Context("when it is not possible to create the download file", func() {
				BeforeEach(func() {
					// make the cache entries directory read only
					permChangeErr := os.Chmod(path.Join(testCacheUnderCfHomeFolder, ".cf", "spring-cloud-dataflow-for-pcf", "cache"), 0444)
					if permChangeErr != nil {
						Fail("Unable to make test directory read-only to simulate error situation")
					}
				})

				It("should propagate the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).Should(HaveSuffix("permission denied"))
				})

				AfterEach(func() {
					// make the cache entries directory writable once again
					permChangeErr := os.Chmod(path.Join(testCacheUnderCfHomeFolder, ".cf", "spring-cloud-dataflow-for-pcf", "cache"), 0755)
					if permChangeErr != nil {
						Fail("Unable to make test directory writable")
					}
				})
			})
		})
	})
})

func readTestFileContent(testFilePath string) (string, error) {
	testFile, err := os.Open(testFilePath)
	if err != nil {
		return "", err
	}

	defer testFile.Close()

	scanner := bufio.NewScanner(testFile)
	scanner.Scan()
	return scanner.Text(), nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

type badReader struct{}

func (badReader) Read([]byte) (n int, err error) {
	return 0, errors.New("read error")
}

type badHash struct{}

func (badHash) Write([]byte) (n int, err error) {
	return 0, errors.New("write error")
}

func (badHash) Sum([]byte) []byte {
	panic("not implemented")
}

func (badHash) Reset() {
	panic("not implemented")
}

func (badHash) Size() int {
	panic("not implemented")
}

func (badHash) BlockSize() int {
	panic("not implemented")
}
