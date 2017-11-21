package cache

type FieldGetter interface {
	GetChecksumCalculator() ChecksumCalculator
	GetEtagHelper() EtagHelper
	GetDownloadUrl() string
	GetDownloadFile() string
	GetCacheEntriesFile() string
}

func (f *fileCacheEntry) GetChecksumCalculator() ChecksumCalculator {
	return f.checksumCalculator
}

func (f *fileCacheEntry) GetEtagHelper() EtagHelper {
	return f.etagHelper
}

func (f *fileCacheEntry) GetDownloadUrl() string {
	return f.downloadUrl
}

func (f *fileCacheEntry) GetDownloadFile() string {
	return f.downloadFile
}

func (f *fileCacheEntry) GetCacheEntriesFile() string {
	return f.cacheEntriesFile
}

type FieldSetter interface {
	SetChecksumCalculator(calculator ChecksumCalculator)
	SetEtagHelper(helper EtagHelper)
	SetDownloadFile(filePath string)
}

func (f *fileCacheEntry) SetChecksumCalculator(calculator ChecksumCalculator) {
	f.checksumCalculator = calculator
}

func (f *fileCacheEntry) SetEtagHelper(helper EtagHelper) {
	f.etagHelper = helper
}

func (f *fileCacheEntry) SetDownloadFile(filePath string) {
	f.downloadFile = filePath
}
