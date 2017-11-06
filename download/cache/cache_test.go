package cache_test

import (
	"errors"

	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/download/cache"
)

const (
	cfHomeProperty = "CF_HOME"
	errMessage     = "worse things happen at sea"
	urlValue       = "http://host/path/file.extension"
)

var (
	testCacheFolder = os.Getenv("TMPDIR") + "plugin-testing-cf-home"
	cfHomeWasSet    bool
	oldCfHomeValue  string
)

var _ = BeforeSuite(func() {
	oldCfHomeValue, cfHomeWasSet = os.LookupEnv(cfHomeProperty)
	os.Setenv(cfHomeProperty, testCacheFolder)
})

var _ = AfterSuite(func() {
	if _, err := os.Stat(testCacheFolder); !os.IsNotExist(err) {
		os.RemoveAll(testCacheFolder)
	}

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
		Context("when tests use CF_HOME", func() {
			JustBeforeEach(func() {
				cacheEntry = downloadsCache.Entry(url)
			})

			It("should return a cache entry that has stored the expected path for the cache entries file", func() {
				if cacheEntry, ok := cacheEntry.(cache.FieldGetter); ok {
					cacheEntriesFilePath := cacheEntry.GetCacheEntriesFile()
					Expect(cacheEntriesFilePath).Should(HavePrefix(testCacheFolder))
					Expect(cacheEntriesFilePath).Should(HaveSuffix(".cf/spring-cloud-dataflow-for-pcf/cache/.cachedata"))
				} else {
					Fail("cache entry did not implement FieldGetter")
				}
			})

			It("should return a cache entry that has stored the expected path for the file to download", func() {
				if cacheEntry, ok := cacheEntry.(cache.FieldGetter); ok {
					downloadFilePath := cacheEntry.GetDownloadFile()
					Expect(downloadFilePath).Should(HavePrefix(testCacheFolder))
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
