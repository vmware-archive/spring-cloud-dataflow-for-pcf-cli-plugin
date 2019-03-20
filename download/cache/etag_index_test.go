package cache_test

import (
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/download/cache"

	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EtagIndex", func() {
	const (
		url1  = "https://url.1"
		url2  = "https://url.2"
		etag1 = "etag1"
		etag2 = "etag2"
	)
	var (
		indexFile string
		etagIndex cache.EtagHelper
	)

	BeforeEach(func() {
		dir, err := ioutil.TempDir("", "etag_index_test")
		Expect(err).NotTo(HaveOccurred())
		indexFile = path.Join(dir, "indexFile")
		etagIndex, err = cache.NewEtagIndex(indexFile)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(indexFile)).To(Succeed())
	})

	It("should record an etag for a given URL", func() {
		err := etagIndex.SetEtagForUrl(url1, etag1)
		Expect(err).NotTo(HaveOccurred())

		e, err := etagIndex.GetETagForUrl(url1)
		Expect(err).NotTo(HaveOccurred())
		Expect(e).To(Equal(etag1))
	})

	It("should cope with an unknown URL", func() {
		e, err := etagIndex.GetETagForUrl(url1)
		Expect(err).NotTo(HaveOccurred())
		Expect(e).To(Equal(""))
	})

	It("should cope with multiple URLs and corresponding etags", func() {
		err := etagIndex.SetEtagForUrl(url1, etag1)
		Expect(err).NotTo(HaveOccurred())

		err = etagIndex.SetEtagForUrl(url2, etag2)
		Expect(err).NotTo(HaveOccurred())

		e, err := etagIndex.GetETagForUrl(url1)
		Expect(err).NotTo(HaveOccurred())
		Expect(e).To(Equal(etag1))

		e, err = etagIndex.GetETagForUrl(url2)
		Expect(err).NotTo(HaveOccurred())
		Expect(e).To(Equal(etag2))
	})

	It("should update and existing etag", func() {
		err := etagIndex.SetEtagForUrl(url1, etag1)
		Expect(err).NotTo(HaveOccurred())

		err = etagIndex.SetEtagForUrl(url1, etag2)
		Expect(err).NotTo(HaveOccurred())

		e, err := etagIndex.GetETagForUrl(url1)
		Expect(err).NotTo(HaveOccurred())
		Expect(e).To(Equal(etag2))
	})

	Context("when the underlying file is deleted", func() {
		BeforeEach(func() {
			Expect(os.Remove(indexFile)).To(Succeed())
		})

		It("should return an error from GetEtagForUrl", func() {
			_, err := etagIndex.GetETagForUrl(url1)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
		})

		It("should return an error from SetEtagForUrl", func() {
			err := etagIndex.SetEtagForUrl(url1, etag1)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
		})
	})

	Context("when the underlying file turns out to be a directory", func() {
		BeforeEach(func() {
			Expect(os.Remove(indexFile)).To(Succeed())
			Expect(os.MkdirAll(indexFile, 0755)).To(Succeed())
		})

		It("should return an error from NewEtagIndex", func() {
			_, err = cache.NewEtagIndex(indexFile)
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
		})
	})
})
