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

	BeforeEach(func() {
		testError = errors.New(errMessage)
		url = urlValue

		downloadsCache, err = cache.NewCache()
	})

	Describe("Entry", func() {
		Context("when CF_HOME is set", func() {
			JustBeforeEach(func() {
				cacheEntry = downloadsCache.Entry(url)
			})

			It("should return a cache entry that has stored the expected path for the cache entries file", func() {
				if cacheEntry, ok := cacheEntry.(cache.FieldGetter); ok {
					cacheEntriesFilePath := cacheEntry.GetCacheEntriesFile()
					Expect(cacheEntriesFilePath).Should(HavePrefix(testCacheUnderCfHomeFolder))
					Expect(cacheEntriesFilePath).Should(HaveSuffix(".cf/spring-cloud-dataflow-for-pcf/cache/.cachedata"))
				} else {
					Fail("cache entry did not implement FieldGetter")
				}
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
				downloadsCache, err = cache.NewCache()
				cacheEntry = downloadsCache.Entry(url)
			})

			It("should return a cache entry that has stored the expected path for the cache entries file", func() {
				if cacheEntry, ok := cacheEntry.(cache.FieldGetter); ok {
					cacheEntriesFilePath := cacheEntry.GetCacheEntriesFile()
					Expect(cacheEntriesFilePath).Should(HavePrefix(testCacheUnderHomeFolder))
					Expect(cacheEntriesFilePath).Should(HaveSuffix(".cf/spring-cloud-dataflow-for-pcf/cache/.cachedata"))
				} else {
					Fail("cache entry did not implement FieldGetter")
				}
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
		checksumValue         = "checksum"
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
	)

	BeforeEach(func() {
		downloadFilePath = path.Join(testCacheUnderCfHomeFolder, ".cf", "spring-cloud-dataflow-for-pcf", "cache", "file.extension")

		downloadsCache, err = cache.NewCache()

		downloadContent = ioutil.NopCloser(bytes.NewReader([]byte(downloadContentString)))

		fakeChecksumCalculator = &downloadfakes.FakeChecksumCalculator{}

		fakeEtagHelper = &downloadfakes.FakeEtagHelper{}

		etagArgument = etagValue

		testError = errors.New(errMessage)

		cacheEntry = downloadsCache.Entry(urlValue)

		if cacheEntry, ok := cacheEntry.(cache.FieldSetter); ok {
			cacheEntry.SetChecksumCalculator(fakeChecksumCalculator)
			cacheEntry.SetEtagHelper(fakeEtagHelper)
		} else {
			Fail("cache entry did not implement FieldSetter")
		}
	})

	Describe("Store", func() {
		JustBeforeEach(func() {
			err = cacheEntry.Store(downloadContent, etagArgument, checksumValue)
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

					urlArg, etagArg, cacheEntriesFilePath := fakeEtagHelper.SetEtagForUrlArgsForCall(0)
					Expect(urlArg).To(Equal(urlValue))
					Expect(etagArg).To(Equal(etagValue))
					Expect(cacheEntriesFilePath).Should(HaveSuffix(".cachedata"))
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
